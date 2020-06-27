module github.com/hakobera/go-sora/examples/sdl2

go 1.14

replace github.com/hakobera/go-sora v0.1.0 => ../../../go-sora

require (
	github.com/hakobera/go-ayame/pkg/decoder v0.0.0-20200621143614-85b3b2ba1a4b
	github.com/hakobera/go-sora v0.1.0
	github.com/pion/rtp v1.5.5
	github.com/pion/webrtc/v2 v2.2.17
	github.com/veandco/go-sdl2 v0.4.4
)
