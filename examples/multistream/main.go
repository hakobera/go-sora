package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/hakobera/go-sora/sora"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
	"github.com/veandco/go-sdl2/sdl"
)

func main() {
	signalingURL := flag.String("url", "wss://sora-labo.shiguredo.jp/signaling", "Specify WebRTC SFU Sora signaling URL")
	channelID := flag.String("channel-id", "", "specify channel ID")
	videoCodecName := flag.String("video-codec", "VP8", "Specify video codec type [VP8 | VP9]")
	signalingKey := flag.String("signaling-key", "", "specify signaling key")
	windowWidth := flag.Int("width", 640, "specify window width")
	windowHeight := flag.Int("height", 480, "specify window height")
	verbose := flag.Bool("verbose", false, "enable verbose log")

	flag.Parse()
	log.Printf("args: url=%s, channel-id=%s, video-codec=%s, signaling-key=%s", *signalingURL, *channelID, *videoCodecName, *signalingKey)

	var err error

	video, err := createVideoByName(*videoCodecName)
	if err != nil {
		log.Fatal(err)
	}

	viewer, err := CreateViewer("go-sora multistream", *windowWidth, *windowHeight)
	if err != nil {
		log.Fatal("Failed to initialize engine", err)
	}

	opts := sora.DefaultOptions()
	opts.Metadata.SignalingKey = *signalingKey
	opts.Audio = false
	opts.Video = video
	opts.Multistream = true
	opts.Debug = *verbose

	con := sora.NewConnection(*signalingURL, *channelID, opts)
	defer con.Disconnect()

	con.OnConnect(func() {
		log.Println("Connected")
	})

	con.OnTrack(func(track *webrtc.Track) {
		log.Printf("OnTrack: label=%s", track.Label())
		err := viewer.AddTrack(track)
		if err != nil {
			log.Printf("OnTrack Error: %s", err.Error())
		}
	})

	con.OnTrackPacket(func(track *webrtc.Track, packet *rtp.Packet) {
		switch track.Kind() {
		case webrtc.RTPCodecTypeVideo:
			viewer.Update(track, packet)
			return
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
			if message.Role != "recvonly" {
				err := viewer.RemoveTrack(message.ConnectionID)
				if err != nil {
					log.Println(err)
				}
			}
		}
	})

	err = con.Connect()
	if err != nil {
		log.Fatal("failed to connect Sora", err)
	}

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
		viewer.Render()
		time.Sleep(10 * time.Millisecond)
	}
}

func createVideoByName(codecType string) (*sora.Video, error) {
	v := &sora.Video{}
	switch codecType {
	case string(sora.VideoCodecTypeVP8):
		v.CodecType = sora.VideoCodecTypeVP8
	case string(sora.VideoCodecTypeVP9):
		v.CodecType = sora.VideoCodecTypeVP9
	default:
		return nil, fmt.Errorf("SDL2 example does not suport '%s'", codecType)
	}
	return v, nil
}
