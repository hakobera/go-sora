module github.com/hakobera/go-sora/examples/multistream

go 1.14

replace github.com/hakobera/go-sora v0.2.0 => ../../../go-sora

require (
	github.com/hakobera/go-sora v0.2.0
	github.com/hakobera/go-webrtc-decoder v0.3.0
	github.com/pion/rtp v1.6.0
	github.com/pion/webrtc/v2 v2.2.23
	github.com/veandco/go-sdl2 v0.4.4
)
