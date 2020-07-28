package sora

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"runtime"
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

const (
	readTimeout  = 90 * time.Second
	readLimit    = 1048576
	writeTimeout = 10 * time.Second
)

// Connection は PeerConnection 接続を管理します。
type Connection struct {
	Options *ConnectionOptions

	connectionID string
	clientID     string
	soraVersion  string

	ws              *websocket.Conn
	pc              *webrtc.PeerConnection
	pcConfig        webrtc.Configuration
	connectionState webrtc.ICEConnectionState
	answerSent      bool

	onOpenHandler        func()
	onConnectHandler     func()
	onDisconnectHandler  func(reason string, err error)
	onTrackPacketHandler func(track *webrtc.Track, packet *rtp.Packet)

	callbackMu sync.Mutex
}

func (c *Connection) Connect() error {
	if c.ws != nil || c.pc != nil {
		c.trace("connection already exists")
		return fmt.Errorf("connection alreay exists")
	}
	c.signaling()
	return nil
}

func (c *Connection) Disconnect() {
	c.callbackMu.Lock()
	defer c.callbackMu.Unlock()

	c.sendDisconnectMessage()
	c.closePeerConnection()
	c.closeWebSocketConnection()
	c.connectionID = ""
	c.clientID = ""
	c.connectionState = webrtc.ICEConnectionStateNew
	c.answerSent = false

	c.onOpenHandler = func() {}
	c.onConnectHandler = func() {}
	c.onDisconnectHandler = func(reason string, err error) {}
	c.onTrackPacketHandler = func(track *webrtc.Track, packet *rtp.Packet) {}
}

// OnOpen は open イベント発生時のコールバック関数を設定します。
func (c *Connection) OnOpen(f func()) {
	c.callbackMu.Lock()
	defer c.callbackMu.Unlock()
	c.onOpenHandler = f
}

// OnConnect は connect イベント発生時のコールバック関数を設定します。
func (c *Connection) OnConnect(f func()) {
	c.callbackMu.Lock()
	defer c.callbackMu.Unlock()
	c.onConnectHandler = f
}

// OnDisconnect は disconnect イベント発生時のコールバック関数を設定します。
func (c *Connection) OnDisconnect(f func(reason string, err error)) {
	c.callbackMu.Lock()
	defer c.callbackMu.Unlock()
	c.onDisconnectHandler = f
}

// OnTrackPacket は RTP Packet 受診時に発生するコールバック関数を設定します。
func (c *Connection) OnTrackPacket(f func(track *webrtc.Track, packet *rtp.Packet)) {
	c.callbackMu.Lock()
	defer c.callbackMu.Unlock()
	c.onTrackPacketHandler = f
}

func (c *Connection) trace(format string, v ...interface{}) {
	if c.Options.Debug {
		logf(format, v...)
	}
}

func (c *Connection) signaling() error {
	if c.ws != nil {
		return fmt.Errorf("WS-ALREADY-EXISTS")
	}

	ctx := context.Background()

	ws, err := c.openWS(ctx)
	if err != nil {
		return fmt.Errorf("WS-OPEN-ERROR: %w", err)
	}
	c.ws = ws

	ctx, cancel := context.WithCancel(ctx)
	messageChannel := make(chan []byte, 100)

	go c.recv(ctx, messageChannel)
	go c.main(cancel, messageChannel)

	return c.sendConnectMessage()
}

func (c *Connection) openWS(ctx context.Context) (*websocket.Conn, error) {
	c.trace("Connecting to %s", c.Options.SoraURL)
	u, err := url.Parse(c.Options.SoraURL)
	if err != nil {
		return nil, err
	}

	conn, _, err := websocket.Dial(ctx, u.String(), nil)
	if err != nil {
		return nil, err
	}
	c.trace("Connected to %s", u.String())
	return conn, nil
}

func (c *Connection) sendMsg(v interface{}) error {
	if c.ws != nil {
		ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
		defer cancel()
		c.trace("send %+v", v)
		if err := wsjson.Write(ctx, c.ws, v); err != nil {
			c.trace("failed to send %v: %v", v, err)
			return err
		}
	}
	return nil
}

func (c *Connection) sendPongMessage() error {
	msg := &pongMessage{
		Type: "pong",
	}

	if err := c.sendMsg(msg); err != nil {
		return err
	}
	return nil
}

func (c *Connection) sendConnectMessage() error {
	msg := &connectMessage{
		Type:        "connect",
		SoraClient:  ClientVersion,
		Environment: fmt.Sprintf("Pion WebRTC on %s %s", runtime.GOOS, runtime.GOARCH),
		Role:        c.Options.Role,
		ChannelID:   c.Options.ChannelID,
		Sdp:         "",
		Audio:       c.Options.Audio,
		Video: video{
			CodecType: c.Options.Video.Name,
		},
		Simulcast: c.Options.Simulcast,
		Metadata:  c.Options.Metadata,
	}

	if err := c.sendMsg(msg); err != nil {
		return err
	}
	return nil
}

func (c *Connection) sendDisconnectMessage() error {
	msg := &signalingMessage{
		Type: "disconnect",
	}

	if err := c.sendMsg(msg); err != nil {
		return err
	}
	return nil
}

func (c *Connection) createPeerConnection(offer *offerMessage) error {
	c.trace("Start createPeerConnection")
	m := webrtc.MediaEngine{}
	codecs, err := populateFromSDP(createOfferSessionDescription(offer.Sdp))
	if err != nil {
		return err
	}
	for _, codec := range codecs {
		m.RegisterCodec(codec)
	}

	vcs := m.GetCodecsByName(c.Options.Video.Name)
	if len(vcs) == 0 {
		return fmt.Errorf("Remote peer does not support %s", c.Options.Video.Name)
	}
	c.trace("%+v", *vcs[0])

	if c.Options.Audio {
		acs := m.GetCodecsByName(webrtc.Opus)
		if len(acs) == 0 {
			return fmt.Errorf("Remote peer does not support %s", webrtc.Opus)
		}
		c.trace("%+v", *acs[0])
	}

	s := webrtc.SettingEngine{}
	s.SetTrickle(c.Options.UseTrickeICE)
	s.SetAnsweringDTLSRole(webrtc.DTLSRoleClient)

	api := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithSettingEngine(s))

	c.trace("RTCConfiguration: %v", c.pcConfig)
	c.pcConfig.ICEServers = *offer.Config.IceServers
	c.pcConfig.ICETransportPolicy = webrtc.NewICETransportPolicy(offer.Config.IceTransportPolicy)
	c.trace("RTCConfiguration: %+v", c.pcConfig)

	pc, err := api.NewPeerConnection(c.pcConfig)
	if err != nil {
		return err
	}

	rtpTransceiverInit := webrtc.RtpTransceiverInit{
		Direction: webrtc.RTPTransceiverDirectionRecvonly,
	}
	switch c.Options.Role {
	case "sendonly":
		rtpTransceiverInit.Direction = webrtc.RTPTransceiverDirectionSendonly
	case "sendrecv":
		rtpTransceiverInit.Direction = webrtc.RTPTransceiverDirectionSendrecv
	}

	if c.Options.Audio {
		_, err = pc.AddTransceiver(webrtc.RTPCodecTypeAudio, rtpTransceiverInit)
		if err != nil {
			return err
		}
	}

	_, err = pc.AddTransceiver(webrtc.RTPCodecTypeVideo, rtpTransceiverInit)
	if err != nil {
		return err
	}

	// Set a Handler for when a new remote track starts, this Handler copies inbound RTP packets,
	// replaces the SSRC and sends them back
	pc.OnTrack(func(track *webrtc.Track, receiver *webrtc.RTPReceiver) {
		// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
		// This is a temporary fix until we implement incoming RTCP events, then we would push a PLI only when a viewer requests it
		go func() {
			ticker := time.NewTicker(time.Second * 3)
			for range ticker.C {
				if c.pc == nil || c.pc.SignalingState() == webrtc.SignalingStateClosed {
					return
				}

				errSend := pc.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: track.SSRC()}})
				if errSend != nil {
					c.trace("Failed to write RTCP packet: %s", errSend.Error())
				}
			}
		}()

		c.trace("peerConnection.ontrack(): %d, codec: %s", track.PayloadType(), track.Codec().Name)
		go func() {
			for {
				rtp, readErr := track.ReadRTP()
				if readErr != nil {
					if readErr == io.EOF {
						return
					}
					c.trace("read RTP error %v", readErr)
					c.Disconnect()
					c.onDisconnectHandler("READ-RTP-ERROR", err)
					return
				}
				c.onTrackPacketHandler(track, rtp)

				if c.pc == nil || c.pc.SignalingState() == webrtc.SignalingStateClosed {
					return
				}
			}
		}()
	})
	// Set the Handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		c.trace("ICE connection Status has changed to %s", connectionState.String())
		if c.connectionState != connectionState {
			c.connectionState = connectionState
			switch c.connectionState {
			case webrtc.ICEConnectionStateConnected:
				c.onConnectHandler()
			case webrtc.ICEConnectionStateDisconnected:
				fallthrough
			case webrtc.ICEConnectionStateFailed:
				c.Disconnect()
				c.onDisconnectHandler("ICE-CONNECTION-STATE-FAILED", nil)
			}
		}
	})
	// Set the Handler for Signaling connection state
	pc.OnSignalingStateChange(func(signalingState webrtc.SignalingState) {
		c.trace("signaling state changes: %s", signalingState.String())
	})

	if c.Options.UseTrickeICE {
		pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
			if candidate == nil {
				return
			}

			candidateJSON := candidate.ToJSON()
			candidateMsg := &candidateMessage{
				Type:             "candidate",
				Candidate:        candidateJSON.Candidate,
				SDPMid:           candidateJSON.SDPMid,
				SDPMLineIndex:    candidateJSON.SDPMLineIndex,
				UsernameFragment: candidateJSON.UsernameFragment,
			}
			c.sendMsg(candidateMsg)
		})
	}

	if c.pc == nil {
		c.pc = pc
		c.onOpenHandler()
	} else {
		c.pc = pc
	}

	c.clientID = offer.ClientID
	c.connectionID = offer.ConnectionID
	c.soraVersion = offer.Version

	return nil
}

func (c *Connection) createAnswer() error {
	if c.pc == nil {
		return nil
	}

	answer, err := c.pc.CreateAnswer(nil)
	if err != nil {
		c.Disconnect()
		c.onDisconnectHandler("CREATE-ANSWER-ERROR", err)
		return err
	}
	c.trace("create answer sdp=%s", answer.SDP)
	c.pc.SetLocalDescription(answer)
	if c.pc.LocalDescription() != nil {
		msgType := "answer"
		if c.answerSent {
			msgType = "update"
		}
		answerMsg := &answerMessage{
			Type: msgType,
			Sdp:  answer.SDP,
		}
		err = c.sendMsg(answerMsg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Connection) setOffer(sessionDescription webrtc.SessionDescription) error {
	if c.pc == nil {
		return nil
	}
	err := c.pc.SetRemoteDescription(sessionDescription)
	if err != nil {
		c.Disconnect()
		c.onDisconnectHandler("CREATE-OFFER-ERROR", err)
		return err
	}
	c.trace("set offer sdp=%s", sessionDescription.SDP)
	err = c.createAnswer()
	if err != nil {
		return err
	}
	return nil
}

func (c *Connection) closePeerConnection() {
	if c.pc == nil {
		return
	}
	if c.pc != nil && c.pc.SignalingState() == webrtc.SignalingStateClosed {
		c.pc = nil
		return
	}
	c.pc.OnICEConnectionStateChange(func(_ webrtc.ICEConnectionState) {})

	go func() {
		ticker := time.NewTicker(400 * time.Millisecond)
		for range ticker.C {
			if c.pc == nil {
				ticker.Stop()
				return
			}
			if c.pc != nil && c.pc.SignalingState() == webrtc.SignalingStateClosed {
				ticker.Stop()
				c.pc = nil
				return
			}
		}
	}()
	if c.pc != nil {
		c.pc.Close()
	}
}

func (c *Connection) closeWebSocketConnection() {
	if c.ws == nil {
		return
	}

	if err := c.ws.Close(websocket.StatusNormalClosure, ""); err != nil {
		c.trace("FAILED-SEND-CLOSE-MESSAGE")
	}
	c.trace("SENT-CLOSE-MESSAGE")
	c.ws = nil
}

func (c *Connection) main(cancel context.CancelFunc, messageChannel chan []byte) {
	defer func() {
		cancel()
		c.trace("EXIT-MAIN")
	}()

loop:
	for {
		select {
		case rawMessage, ok := <-messageChannel:
			if !ok {
				c.trace("CLOSED-MESSAGE-CHANNEL")
				return
			}
			if err := c.handleMessage(rawMessage); err != nil {
				c.trace("handleMessage error: %s", err.Error())
				break loop
			}
		}
	}
}

func (c *Connection) recv(ctx context.Context, messageChannel chan []byte) {
loop:
	for {
		if c.ws == nil {
			break loop
		}

		cctx, cancel := context.WithTimeout(ctx, readTimeout)
		_, rawMessage, err := c.ws.Read(cctx)
		cancel()
		if err != nil {
			c.trace("failed to ReadMessage: %v", err)
			break loop
		}
		messageChannel <- rawMessage
	}
	close(messageChannel)
	c.trace("CLOSE-MESSAGE-CHANNEL")
	<-ctx.Done()
	c.trace("EXITED-MAIN")
	c.Disconnect()
	c.onDisconnectHandler("EXIT-RECV", nil)
	c.trace("EXIT-RECV")
}

func (c *Connection) handleMessage(rawMessage []byte) error {
	message := &signalingMessage{}
	if err := json.Unmarshal(rawMessage, &message); err != nil {
		c.trace("invalid JSON, rawMessage: %s, error: %v", rawMessage, err)
		return errorInvalidJSON
	}

	c.trace("recv type: %s, rawMessage: %s", message.Type, string(rawMessage))

	var err error

	switch message.Type {
	case "ping":
		c.sendPongMessage()
	case "notify":
		// Do nothing
		return nil
	case "offer":
		offerMsg := &offerMessage{}
		if err := unmarshalMessage(c, rawMessage, &offerMsg); err != nil {
			return err
		}

		osdp := offerMsg.Sdp
		offerMsg.Sdp = cleanupSDP(osdp)

		err = c.createPeerConnection(offerMsg)
		if err != nil {
			return err
		}
		return c.setOffer(createOfferSessionDescription(offerMsg.Sdp))
	case "update":
		updateMsg := webrtc.SessionDescription{}
		if err := unmarshalMessage(c, rawMessage, &updateMsg); err != nil {
			return err
		}
		return c.setOffer(updateMsg)
	default:
		c.trace("invalid message type %s", message.Type)
		return errorInvalidMessageType
	}
	return nil
}
