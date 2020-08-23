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
	"github.com/pion/webrtc/v2"
	"github.com/veandco/go-sdl2/sdl"
)

func main() {
	signalingURL := flag.String("url", "wss://sora-labo.shiguredo.jp/signaling", "Specify WebRTC SFU Sora signaling URL")
	channelID := flag.String("channel-id", "", "specify channel ID")
	videoCodecName := flag.String("video-codec", "VP8", "Specify video codec type [VP8 | VP9]")
	signalingKey := flag.String("signaling-key", "", "specify signaling key")
	simulcast := flag.Bool("simulcast", false, "enable simulcast")
	verbose := flag.Bool("verbose", false, "enable verbose log")

	flag.Parse()
	log.Printf("args: url=%s, channel-id=%s, video-codec=%s, signaling-key=%s", *signalingURL, *channelID, *videoCodecName, *signalingKey)

	var err error

	video, err := createVideoByName(*videoCodecName, *simulcast)
	if err != nil {
		log.Fatal(err)
	}

	const windowWidth = 640
	const windowHeight = 480

	window, renderer, err := initSDL("go-sora SDL2 Example", windowWidth, windowHeight)
	if err != nil {
		log.Fatal("Failed to initialize SDL", err)
	}
	defer renderer.Destroy()
	defer window.Destroy()
	defer sdl.Quit()

	renderer.SetDrawColor(0, 0, 0, sdl.ALPHA_OPAQUE)
	renderer.Clear()

	opts := sora.DefaultOptions()
	opts.Metadata.SignalingKey = *signalingKey
	opts.Audio = false
	opts.Video = video
	if *simulcast {
		opts.Simulcast = &sora.Simulcast{Quality: sora.SimulcastQualityDefault}
	}
	opts.Debug = *verbose

	d, err := initVideoDecoder(*videoCodecName)
	if err != nil {
		log.Fatal("Failed to initialize Video Deocder", err)
	}
	defer d.Close()

	videoFrameBuilder := d.NewFrameBuilder()

	videoFrameChan := make(chan *decoder.Frame, 60)
	defer close(videoFrameChan)

	con := sora.NewConnection(*signalingURL, *channelID, opts)
	defer con.Disconnect()

	con.OnConnect(func() {
		log.Println("Connected")
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

	con.OnSignalingNotify(func(eventType string, message *sora.SignalingNotifyMessage) {
		switch eventType {
		case "connection.created":
			if con.ConnectionID() == message.ConnectionID {
				log.Printf("Join (Local): connectionID=%s, clientID=%s", message.ConnectionID, message.ClientID)
			} else {
				log.Printf("Join (Remote): connectionID=%s, clientID=%s", message.ConnectionID, message.ClientID)
			}
		case "connection.updated":
			log.Printf("Update: connectionID=%s, clientID=%s, %d minutes connected", message.ConnectionID, message.ClientID, message.Minutes)
		case "connection.destroyed":
			log.Printf("Leave: connectionID=%s, clientID=%s", message.ConnectionID, message.ClientID)
		}
	})

	err = con.Connect()
	if err != nil {
		log.Fatal("failed to connect Sora", err)
	}

	go func() {
		for result := range d.Process(videoFrameChan) {
			if result.Err != nil {
				log.Println("Failed to process video frame:", result.Err)
				continue
			}

			err := renderFrame(renderer, result.Image, windowWidth, windowHeight)
			if err != nil {
				log.Println(err.Error())
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

func initSDL(name string, width int32, height int32) (*sdl.Window, *sdl.Renderer, error) {
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		return nil, nil, fmt.Errorf("failed to init SDL: %w", err)
	}

	window, err := sdl.CreateWindow(name, sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, width, height, sdl.WINDOW_SHOWN|sdl.WINDOW_ALLOW_HIGHDPI)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create SDL window: %w", err)
	}

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create SDL renderer: %w", err)
	}

	return window, renderer, nil
}

func initVideoDecoder(codec string) (decoder.VideoDecoder, error) {
	var d decoder.VideoDecoder
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

func createVideoByName(codecType string, simulcast bool) (*sora.Video, error) {
	v := &sora.Video{}
	switch codecType {
	case string(sora.VideoCodecTypeVP8):
		v.CodecType = sora.VideoCodecTypeVP8
	case string(sora.VideoCodecTypeVP9):
		if simulcast {
			return nil, fmt.Errorf("Simulcast is only supported for VP8")
		}
		v.CodecType = sora.VideoCodecTypeVP9
	default:
		return nil, fmt.Errorf("SDL2 example does not suport '%s'", codecType)
	}
	return v, nil
}

func renderFrame(renderer *sdl.Renderer, img decoder.DecodedImage, windowWidth int32, windowHeight int32) error {
	texture, err := renderer.CreateTexture(sdl.PIXELFORMAT_YV12, sdl.TEXTUREACCESS_STREAMING, int32(img.Width()), int32(img.Height()))
	if err != nil {
		return fmt.Errorf("Failed to create SDL texture: %w", err)
	}
	defer texture.Destroy()

	err = texture.UpdateYUV(nil, img.Plane(0), img.Stride(0), img.Plane(1), img.Stride(1), img.Plane(2), img.Stride(2))
	if err != nil {
		return fmt.Errorf("Failed to update SDL Texture: %w", err)
	}

	src := &sdl.Rect{0, 0, int32(img.Width()), int32(img.Height())}
	dst := &sdl.Rect{0, 0, windowWidth, windowHeight}

	if img.Width() > img.Height() {
		dst.H = int32(float64(windowWidth*int32(img.Height())) / float64(img.Width()))
		dst.Y = (windowHeight - dst.H) / 2
	} else {
		dst.W = int32(float64(windowHeight*int32(img.Width())) / float64(img.Height()))
		dst.X = (windowWidth - dst.W) / 2
	}

	renderer.Clear()
	renderer.Copy(texture, src, dst)
	renderer.Present()

	return nil
}
