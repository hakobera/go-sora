package sora

import (
	"encoding/json"
	"math/rand"
	"regexp"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/pion/webrtc/v2"
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
