package sora

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/pion/sdp"
	"github.com/pion/webrtc/v3"
)

func getULID() string {
	t := time.Now()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	return ulid.MustNew(ulid.Timestamp(t), entropy).String()
}

func unmarshalMessage(c *Connection, rawMessage []byte, v interface{}) error {
	if err := json.Unmarshal(rawMessage, v); err != nil {
		c.trace("invalid JSON, rawMessage: %s, error: %v", rawMessage, err)
		return errorInvalidJSON
	}
	return nil
}

func strPtr(s string) *string {
	return &s
}

func createOfferSessionDescription(sdp string) webrtc.SessionDescription {
	return webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  sdp,
	}
}

func cleanupSDP(sdp string) string {
	rep := regexp.MustCompile("b=TIAS:\\d+\\r\\n")
	return rep.ReplaceAllString(sdp, "")
}

func populateFromSDP(sd webrtc.SessionDescription) ([]*webrtc.RTPCodec, error) {
	sdp := sdp.SessionDescription{}
	if err := sdp.Unmarshal(sd.SDP); err != nil {
		return nil, err
	}

	var codecs []*webrtc.RTPCodec
	for _, md := range sdp.MediaDescriptions {
		if md.MediaName.Media != "audio" && md.MediaName.Media != "video" {
			continue
		}

		for _, format := range md.MediaName.Formats {
			pt, err := strconv.Atoi(format)
			if err != nil {
				return nil, fmt.Errorf("format parse error")
			}

			payloadType := uint8(pt)
			payloadCodec, err := sdp.GetCodecForPayloadType(payloadType)
			if err != nil {
				// ignore codec not found for payload type
				continue
			}

			var codec *webrtc.RTPCodec
			switch {
			case strings.EqualFold(payloadCodec.Name, webrtc.PCMA):
				codec = webrtc.NewRTPPCMACodec(payloadType, payloadCodec.ClockRate)
			case strings.EqualFold(payloadCodec.Name, webrtc.PCMU):
				codec = webrtc.NewRTPPCMUCodec(payloadType, payloadCodec.ClockRate)
			case strings.EqualFold(payloadCodec.Name, webrtc.G722):
				codec = webrtc.NewRTPG722Codec(payloadType, payloadCodec.ClockRate)
			case strings.EqualFold(payloadCodec.Name, webrtc.Opus):
				codec = webrtc.NewRTPOpusCodec(payloadType, payloadCodec.ClockRate)
			case strings.EqualFold(payloadCodec.Name, webrtc.VP8):
				codec = webrtc.NewRTPVP8Codec(payloadType, payloadCodec.ClockRate)
			case strings.EqualFold(payloadCodec.Name, webrtc.VP9):
				codec = webrtc.NewRTPVP9Codec(payloadType, payloadCodec.ClockRate)
			case strings.EqualFold(payloadCodec.Name, webrtc.H264):
				codec = webrtc.NewRTPH264Codec(payloadType, payloadCodec.ClockRate)
			default:
				// ignoring other codecs
				continue
			}

			codec.SDPFmtpLine = payloadCodec.Fmtp
			codecs = append(codecs, codec)
		}
	}
	return codecs, nil
}
