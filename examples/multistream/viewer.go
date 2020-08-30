package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/hakobera/go-webrtc-decoder/decoder"
	"github.com/hakobera/go-webrtc-decoder/decoder/vpx"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/veandco/go-sdl2/sdl"
)

const maxTracks = 9

type Stream struct {
	width  int
	height int

	id                string
	label             string
	decoder           decoder.VideoDecoder
	videoFrameBuilder *decoder.FrameBuilder
	videoFrameChan    chan *decoder.Frame
	image             decoder.DecodedImage
	texture           *sdl.Texture
	dirty             bool

	mu sync.Mutex
}

func CreateStream(track *webrtc.Track) (*Stream, error) {
	s := &Stream{id: track.ID(), label: track.Label()}
	d, err := s.initVideoDecoder(track.Codec().Name)
	if err != nil {
		return nil, err
	}

	s.decoder = d
	s.videoFrameBuilder = d.NewFrameBuilder()
	s.videoFrameChan = make(chan *decoder.Frame, 60)

	go s.process(d)

	return s, nil
}

func (s *Stream) process(d decoder.VideoDecoder) {
	for result := range d.Process(s.videoFrameChan) {
		if result.Err != nil {
			log.Println("Failed to process video frame:", result.Err)
			continue
		}
		s.mu.Lock()
		s.image = result.Image
		s.dirty = true
		s.mu.Unlock()
	}
}

func (s *Stream) Update(packet *rtp.Packet) {
	s.videoFrameBuilder.Push(packet)

	for {
		frame := s.videoFrameBuilder.Pop()
		if frame == nil {
			return
		}
		s.videoFrameChan <- frame
	}
}

func (s *Stream) Render(renderer *sdl.Renderer, x, y, w, h int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	//log.Printf("Render stream: id=%s, label=%s (%d, %d, %d, %d)", s.id, s.label, x, y, w, h)

	if s.image != nil {
		var err error
		img := s.image

		if s.dirty {
			if s.texture == nil {
				s.texture, err = renderer.CreateTexture(sdl.PIXELFORMAT_YV12, sdl.TEXTUREACCESS_STREAMING, int32(img.Width()), int32(img.Height()))
			} else if s.width != int(img.Width()) || s.height != int(img.Height()) {
				s.texture.Destroy()
				s.texture, err = renderer.CreateTexture(sdl.PIXELFORMAT_YV12, sdl.TEXTUREACCESS_STREAMING, int32(img.Width()), int32(img.Height()))
			}

			if err != nil {
				return fmt.Errorf("failed to create SDL Texture: %w", err)
			}

			s.width = int(img.Width())
			s.height = int(img.Height())

			err = s.texture.UpdateYUV(nil, img.Plane(0), img.Stride(0), img.Plane(1), img.Stride(1), img.Plane(2), img.Stride(2))
			if err != nil {
				return fmt.Errorf("failed to update SDL Texture: %w", err)
			}
			s.dirty = false
		}

		src := &sdl.Rect{X: 0, Y: 0, W: int32(img.Width()), H: int32(img.Height())}
		dst := &sdl.Rect{X: x, Y: y, W: w, H: h}
		if img.Width() > img.Height() {
			dst.H = int32(float64(uint32(w)*img.Height()) / float64(img.Width()))
			dst.Y = y + (h-dst.H)/2
		} else {
			dst.W = int32(float64(uint32(h)*img.Width()) / float64(img.Height()))
			dst.X = x + (w-dst.W)/2
		}

		renderer.Copy(s.texture, src, dst)
	}

	return nil
}

func (s *Stream) Destroy() {
	s.width = 0
	s.height = 0
	s.dirty = false

	if s.texture != nil {
		err := s.texture.Destroy()
		if err != nil {
			log.Printf(err.Error())
		}
		s.texture = nil
	}

	if s.decoder != nil {
		close(s.videoFrameChan)
		s.decoder.Close()
		s.decoder = nil
	}
}

func (s *Stream) initVideoDecoder(codec string) (decoder.VideoDecoder, error) {
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

type Viewer struct {
	name   string
	width  int
	height int
	inited bool

	window   *sdl.Window
	renderer *sdl.Renderer
	streams  []*Stream

	mu sync.Mutex
}

func CreateViewer(name string, width int, height int) (*Viewer, error) {
	v := &Viewer{
		name:   name,
		width:  width,
		height: height,
	}

	err := sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		return nil, fmt.Errorf("failed to init SDL: %w", err)
	}

	v.window, err = sdl.CreateWindow(name, sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, int32(width), int32(height), sdl.WINDOW_SHOWN|sdl.WINDOW_ALLOW_HIGHDPI)
	if err != nil {
		return nil, fmt.Errorf("failed to create SDL window: %w", err)
	}

	v.renderer, err = sdl.CreateRenderer(v.window, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC)
	if err != nil {
		return nil, fmt.Errorf("failed to create SDL renderer: %w", err)
	}

	v.streams = make([]*Stream, 0)

	v.renderer.SetDrawColor(0, 0, 0, sdl.ALPHA_OPAQUE)
	v.renderer.Clear()

	v.inited = true

	return v, nil
}

func (v *Viewer) Destroy() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.inited = false

	for _, s := range v.streams {
		log.Printf("Destory stream[label=%s]", s.label)
		s.Destroy()
	}
	v.streams = make([]*Stream, 0)

	if v.renderer != nil {
		err := v.renderer.Destroy()
		if err != nil {
			log.Printf(err.Error())
		}
		v.renderer = nil
	}

	if v.window != nil {
		err := v.window.Destroy()
		if err != nil {
			log.Printf(err.Error())
		}
		v.window = nil
	}

	sdl.Quit()
}

func (v *Viewer) Update(track *webrtc.Track, packet *rtp.Packet) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.inited {
		return fmt.Errorf("viewer is not intiallized")
	}

	s, _ := v.findStream(track.Label())
	if s == nil {
		return fmt.Errorf("stream not found: label=%s", track.Label())
	}

	s.Update(packet)
	return nil
}

func (v *Viewer) Render() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	err := v.renderer.Clear()
	if err != nil {
		return err
	}

	l := len(v.streams)
	var dw int32 = 1
	var dh int32 = 1

	if l > 1 && l <= 4 {
		dw = 2
		dh = 2
	} else if l > 4 {
		dw = 3
		dh = 3
	}

	var i int32 = 0
	var uw int32 = int32(v.width) / dw
	var uh int32 = int32(v.height) / dh

	for _, s := range v.streams {
		s.Render(v.renderer, (i%dw)*uw, (i/dh)*uh, uw, uh)
		i++
	}

	v.renderer.Present()
	return nil
}

func (v *Viewer) AddTrack(track *webrtc.Track) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if len(v.streams) >= maxTracks {
		return fmt.Errorf("track number is reached to limit, max %d is allowed", maxTracks)
	}

	s, _ := v.findStream(track.Label())
	if s != nil {
		return fmt.Errorf("Track[label=%s] is already added", track.Label())
	}

	s, err := CreateStream(track)
	if err != nil {
		return err
	}
	v.streams = append(v.streams, s)
	log.Printf("Viewer#AddTrack: Track[label=%s] added", track.Label())
	return nil
}

func (v *Viewer) RemoveTrack(label string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	s, i := v.findStream(label)
	if s == nil {
		return fmt.Errorf("stream not found: label=%s", label)
	}

	s.Destroy()
	v.streams = append(v.streams[:i], v.streams[i+1:]...)
	log.Printf("Viewer#RemoveTrack: Track[label=%s] removed", label)
	return nil
}

func (v *Viewer) findStream(label string) (*Stream, int) {
	for i, ss := range v.streams {
		if ss.label == label {
			return ss, i
		}
	}
	return nil, -1
}
