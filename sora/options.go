package sora

// ConnectionOptions は Sora 接続設定です。
type ConnectionOptions struct {
	// Sora の URL
	SoraURL string

	// Role はクライアントの役割の設定
	Role Role

	// 接続する Channel ID
	ChannelID string

	// クライアント ID は、sora.conf で allow_client_id_assignment を true に設定した場合のみ指定することが可能です
	ClientID string

	// Video の設定
	Video *Video

	// Audio の設定
	Audio bool

	// Simulcast の設定
	Simulcast *Simulcast

	// Multistream の設定
	Multistream bool

	// Metadata
	Metadata *Metadata

	// Debug 出力をするかどうかのフラグ
	Debug bool
}
