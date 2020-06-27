package sora

import "github.com/pion/webrtc/v2"

type connectMessage struct {
	Type        string   `json:"type"`
	SoraClient  string   `json:"sora_client"`
	Environment string   `json:"environment"`
	Role        string   `json:"role"`
	ChannelID   string   `json:"channel_id"`
	Sdp         string   `json:"sdp"`
	Audio       bool     `json:"audio"`
	Video       video    `json:"video"`
	Metadata    Metadata `json:"metadata"`
}

type video struct {
	CodecType string `json:"codec_type"`
}

type Metadata struct {
	SignalingKey string `json:"signaling_key"`
	TurnTCPOnly  bool   `json:"turn_tcp_only"`
	TurnTLSOnly  bool   `json:"turn_tls_only"`
}

type signalingConfig struct {
	IceServers         *[]webrtc.ICEServer `json:"iceServers"`
	IceTransportPolicy string              `json:"iceTransportPolicy"`
}

type offerMessage struct {
	Type         string          `json:"type"`
	Version      string          `json:"version"`
	ClientID     string          `json:"client_id"`
	Config       signalingConfig `json:"config"`
	ConnectionID string          `json:"connection_id"`
	Sdp          string          `json:"sdp"`
}

type answerMessage struct {
	Type string `json:"type"`
	Sdp  string `json:"sdp"`
}

type signalingMessage struct {
	Type string `json:"type"`
}

type pongMessage struct {
	Type string `json:"type"`
}

type candidateMessage struct {
	Type             string  `json:"type"`
	Candidate        string  `json:"candidate"`
	SDPMid           *string `json:"sdpMid,omitempty"`
	SDPMLineIndex    *uint16 `json:"sdpMLineIndex,omitempty"`
	UsernameFragment string  `json:"usernameFragment,omitempty"`
}
