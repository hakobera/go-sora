package sora

import (
	"fmt"

	"github.com/pion/webrtc/v2"
)

// Signaling の型定義は以下のURLを参照
// https://sora-doc.shiguredo.jp/signaling_type

type connectMessage struct {
	Type                    string                 `json:"type"`
	Role                    Role                   `json:"role"`
	ChannelID               string                 `json:"channel_id"`
	ClientID                string                 `json:"client_id,omitempty"`
	Metadata                *Metadata              `json:"metadata,omitempty"`
	SignalingNotifyMetadata map[string]interface{} `json:"signaling_notify_metadata,omitempty"`
	Multistream             bool                   `json:"multistream,omitempty"`
	Spotlight               uint8                  `json:"spotlight,omitempty"`
	Simulcast               *Simulcast             `json:"simulcast,omitempty"`
	Audio                   bool                   `json:"audio"`
	Video                   *Video                 `json:"video"`
	Sdp                     string                 `json:"sdp,omitempty"`
	SoraClient              string                 `json:"sora_client"`
	Environment             string                 `json:"environment"`
}

// Role はクライアント役割を指定します
type Role string

const (
	// SendRecvRole はマルチストリーム、スポットライトで利用できる role で送受信を行います
	SendRecvRole Role = "sendrecv"

	// SendOnlyRole はすべてで利用でき、送信のみを行い、受信を行いません
	SendOnlyRole Role = "sendonly"

	// RecvOnlyRole はすべてで利用でき、受信のみを行い、送信を行いません
	RecvOnlyRole Role = "recvonly"
)

// SimulcastQuality はサイマルキャストの画質を指定します
type SimulcastQuality string

const (
	// SimulcastQualityDefault は Sora の `default_simulcast_quality` で指定された画質。指定がない場合は `low`
	SimulcastQualityDefault SimulcastQuality = ""

	// SimulcastQualityLow は低画質
	SimulcastQualityLow SimulcastQuality = "low"

	// SimulcastQualityMiddle は中画質
	SimulcastQualityMiddle SimulcastQuality = "middle"

	// SimulcastQualityHigh は最高画質
	SimulcastQualityHigh SimulcastQuality = "high"
)

// Simulcast はサイマルキャストの設定
type Simulcast struct {
	Quality SimulcastQuality `json:"quality"`
}

func (s Simulcast) MarshalJSON() ([]byte, error) {
	if s.Quality == SimulcastQualityDefault {
		return []byte("true"), nil
	}
	return []byte(fmt.Sprintf("{quality:%s}", s.Quality)), nil
}

type VideoCodecType string

const (
	VideoCodecTypeVP8  VideoCodecType = "VP8"
	VideoCodecTypeVP9  VideoCodecType = "VP9"
	VideoCodecTypeAV1  VideoCodecType = "AV1"
	VideoCodecTypeH264 VideoCodecType = "H264"
	VideoCodecTypeH265 VideoCodecType = "H265"
)

// ビデオの設定
type Video struct {
	// ビデオコーデックの設定
	CodecType VideoCodecType `json:"codec_type,omitempty"`

	// ビデオのビットレート指定。指定できる値は 1 から 50000 です
	BitRate uint16 `json:"bitrate,omitempty"`
}

// Metadata は認証 Webhook に渡される認証用のメタデータ
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

type pingMessage struct {
	Type  string `json:"type"`
	Stats bool   `json:"stats"`
}

type pongMessage struct {
	Type  string         `json:"type"`
	Stats []webrtc.Stats `json:"stats,omitempty"`
}

type candidateMessage struct {
	Type             string  `json:"type"`
	Candidate        string  `json:"candidate"`
	SDPMid           *string `json:"sdpMid,omitempty"`
	SDPMLineIndex    *uint16 `json:"sdpMLineIndex,omitempty"`
	UsernameFragment string  `json:"usernameFragment,omitempty"`
}

type notifyMessage struct {
	Type      string `json:"type"`
	EventType string `json:"event_type"`
}

// SignalingNotifyMessage はシグナリング通知メッセージ
// https://sora-doc.shiguredo.jp/signaling_notify
type SignalingNotifyMessage struct {
	Type                         string                   `json:"type"`
	EventType                    string                   `json:"event_type"`
	Role                         string                   `json:"role"`
	Minutes                      int                      `json:"minutes"`
	ChannelConnections           int                      `json:"channel_connections"`
	ChannelUpstreamConnections   int                      `json:"channel_upstream_connections"`
	ChannelDownstreamConnections int                      `json:"channel_downstream_connections"`
	ClientID                     string                   `json:"client_id"`
	ConnectionID                 string                   `json:"connection_id"`
	Audio                        bool                     `json:"audio"`
	Video                        bool                     `json:"video"`
	Metadata                     map[string]interface{}   `json:"metadata"`
	MetadataList                 []map[string]interface{} `json:"metadata_list"`
}

// SpotlightNotifyMessage はスポットライト機能を利用した場合のシグナリング通知メッセージ
// https://sora-doc.shiguredo.jp/signaling_notify#id9
type SpotlightNotifyMessage struct {
	Type        string `json:"type"`
	EventType   string `json:"event_type"`
	ChannelID   string `json:"channel_id"`
	ClientID    string `json:"client_id"`
	SpotlightID string `json:"spotlight_id"`
	Audio       bool   `json:"audio"`
	Video       bool   `json:"video"`
	Fixed       bool   `json:"fixed"`
}

// NetworkNotifyMessage はネットワークのシグナリング通知メッセージ
// https://sora-doc.shiguredo.jp/signaling_notify#id9
type NetworkNotifyMessage struct {
	Type          string `json:"type"`
	EventType     string `json:"event_type"`
	UnstableLevel int    `json:"unstable_level"`
}
