package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/hakobera/go-sora/sora"
	"github.com/hakobera/go-webrtc-decoder/decoder"
	"github.com/hakobera/go-webrtc-decoder/decoder/vpx"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/veandco/go-sdl2/sdl"
)

func main() {
	signalingURL := flag.String("url", "wss://sora-labo.shiguredo.jp/signaling", "Specify WebRTC SFU Sora signaling URL")
	channelID := flag.String("channel-id", "", "specify channel ID")
	videoCodecName := flag.String("video-codec", "VP8", "Specify video codec type [VP8 | VP9]")
	signalingKey := flag.String("signaling-key", "", "specify signaling key")
	verbose := flag.Bool("verbose", false, "enable verbose log")

	flag.Parse()
	log.Printf("args: url=%s, channel-id=%s, video-codec=%s, signaling-key=%s", *signalingURL, *channelID, *videoCodecName, *signalingKey)

	var err error

	videoCodec, err := sora.CreateVideoCodecByName(*videoCodecName)
	if err != nil {
		log.Fatal(err)
	}

	const windowWidth = 640
	const windowHeight = 480

	window, renderer, texture, err := initSDL("go-sora SDL2 Example", windowWidth, windowHeight)
	if err != nil {
		log.Fatal("Failed to initialize SDL", err)
	}
	defer texture.Destroy()
	defer renderer.Destroy()
	defer window.Destroy()
	defer sdl.Quit()

	renderer.SetDrawColor(0, 0, 0, sdl.ALPHA_OPAQUE)
	renderer.Clear()

	opts := sora.DefaultOptions()
	opts.Metadata.SignalingKey = *signalingKey
	opts.Audio = false
	opts.Video = videoCodec
	opts.Debug = *verbose

	d, err := initVideoDecoder(*videoCodecName)
	if err != nil {
		log.Fatal("Failed to initialize Video Deocder", err)
	}
	defer d.Close()

	videoFrameBuilder := d.NewFrameBuilder()

	videoFrameChan := make(chan *decoder.Frame, 60)
	defer close(videoFrameChan)

	decodedImgChan := make(chan decoder.DecodedImage)

	go d.Process(videoFrameChan, decodedImgChan)

	con := sora.NewConnection(*signalingURL, *channelID, opts)
	defer con.Disconnect()

	con.OnConnect(func() {
		fmt.Println("Connected")
	})

	con.OnTrackPacket(func(track *webrtc.Track, packet *rtp.Packet) {
		switch track.Kind() {
		case webrtc.RTPCodecTypeVideo:
			videoFrameBuilder.Push(packet)

			for {
				frame := videoFrameBuilder.Pop()
				if frame == nil {
					return
				}
				videoFrameChan <- frame
			}
		}
	})

	err = con.Connect()
	if err != nil {
		log.Fatal("failed to connect Sora", err)
	}

	go func() {
		for {
			var err error = nil
			select {
			case img, ok := <-decodedImgChan:
				if !ok {
					return
				}

				err = texture.UpdateYUV(nil, img.Plane(0), img.Stride(0), img.Plane(1), img.Stride(1), img.Plane(2), img.Stride(2))
				if err != nil {
					log.Println("Failed to update SDL Texture", err)
					continue
				}

				// TODO: アスペクト比を維持したままの拡大縮小
				src := &sdl.Rect{0, 0, int32(img.Width()), int32(img.Height())}
				dst := &sdl.Rect{0, 0, windowWidth, windowHeight}

				renderer.Clear()
				renderer.Copy(texture, src, dst)
				renderer.Present()
			}
		}
	}()

	running := true
	for running {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent:
				println("Quit")
				running = false
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func initSDL(name string, width int32, height int32) (*sdl.Window, *sdl.Renderer, *sdl.Texture, error) {
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to init SDL: %w", err)
	}

	window, err := sdl.CreateWindow(name, sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, width, height, sdl.WINDOW_SHOWN|sdl.WINDOW_ALLOW_HIGHDPI)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create SDL window: %w", err)
	}

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create SDL renderer: %w", err)
	}

	texture, err := renderer.CreateTexture(sdl.PIXELFORMAT_YV12, sdl.TEXTUREACCESS_STREAMING, width, height)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create SDL texture: %w", err)
	}

	return window, renderer, texture, nil
}

func initVideoDecoder(codec string) (decoder.Decoder, error) {
	var d decoder.Decoder
	var err error

	switch codec {
	case "VP8":
		d, err = vpx.NewVP8Decoder()
	case "VP9":
		d, err = vpx.NewVP9Decoder()
	default:
		err = fmt.Errorf("Unsupported video codec: %s", codec)
	}

	if err != nil {
		return nil, err
	}

	return d, nil
}
