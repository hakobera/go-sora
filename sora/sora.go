// Package sora provides WebRTC signaling feature for WebRTC SFU Sora
package sora

import (
	"fmt"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

const (
	ClientVersion = "go-sora v0.1.0"
)

// DefaultOptions は Sora 接続設定のデフォルト値を生成して返します。
func DefaultOptions() *ConnectionOptions {
	return &ConnectionOptions{
		Role:     "recvonly",
		Audio:    true,
		Video:    webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000),
		Debug:    false,
		Metadata: Metadata{},
	}
}

// CreateVideoCodecByName はコーデック名に対応する webrtc.RTPCodec を生成して返します。
func CreateVideoCodecByName(name string) (*webrtc.RTPCodec, error) {
	var codec *webrtc.RTPCodec

	switch name {
	case webrtc.VP8:
		codec = webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000)
	case webrtc.VP9:
		codec = webrtc.NewRTPVP9Codec(webrtc.DefaultPayloadTypeVP9, 90000)
	default:
		return nil, fmt.Errorf("go-sora does not suport codec name=%s", name)
	}

	return codec, nil
}

// NewConnection は Sora Connection を生成して返します。
func NewConnection(soraURL string, channelID string, options *ConnectionOptions) *Connection {
	if options == nil {
		options = DefaultOptions()
	}

	options.SoraURL = soraURL
	options.ChannelID = channelID

	c := &Connection{
		Options: options,

		connectionID: "",
		clientID:     "",

		ws:              nil,
		pc:              nil,
		pcConfig:        webrtc.Configuration{},
		connectionState: webrtc.ICEConnectionStateNew,
		answerSent:      false,

		onOpenHandler:        func() {},
		onConnectHandler:     func() {},
		onDisconnectHandler:  func(reason string, err error) {},
		onTrackPacketHandler: func(track *webrtc.Track, packet *rtp.Packet) {},
		onByeHandler:         func() {},
	}

	return c
}
