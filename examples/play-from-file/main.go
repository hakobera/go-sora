package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/hakobera/go-sora/sora"

	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media"
	"github.com/pion/webrtc/v2/pkg/media/ivfreader"
)

func main() {
	signalingURL := flag.String("url", "wss://sora-labo.shiguredo.jp/signaling", "Specify WebRTC SFU Sora signaling URL")
	channelID := flag.String("channel-id", "", "specify channel ID")
	signalingKey := flag.String("signaling-key", "", "specify signaling key")
	videoFilename := flag.String("input", "", "specify input video filename")
	verbose := flag.Bool("verbose", false, "enable verbose log")

	flag.Parse()
	log.Printf("args: url=%s, channel-id=%s, signaling-key=%s", *signalingURL, *channelID, *signalingKey)

	_, err := os.Stat(*videoFilename)
	haveVideoFile := !os.IsNotExist(err)

	if !haveVideoFile {
		log.Fatal("Could not find `" + *videoFilename + "`")
	}

	opts := sora.DefaultOptions()
	opts.Metadata.SignalingKey = *signalingKey
	opts.Role = sora.SendRecvRole
	opts.Audio = false
	opts.Video = &sora.Video{CodecType: sora.VideoCodecTypeVP8}
	opts.Multistream = true
	opts.Debug = *verbose

	con := sora.NewConnection(*signalingURL, *channelID, opts)
	defer con.Disconnect()

	var videoTrack *webrtc.Track
	con.OnOpen(func(pc *webrtc.PeerConnection, m webrtc.MediaEngine) {
		log.Printf("onOpen: adding new video track from %s", *videoFilename)
		track, addTrackErr := pc.NewTrack(getPayloadType(m, webrtc.RTPCodecTypeVideo, "VP8"), rand.Uint32(), "video", "pion")
		if addTrackErr != nil {
			log.Fatal(addTrackErr)
		}
		if _, addTrackErr = pc.AddTrack(track); err != nil {
			log.Fatal(addTrackErr)
		}
		videoTrack = track
	})

	con.OnConnect(func() {
		log.Printf("onConnect: read content from %s", *videoFilename)
		go func() {
			file, ivfErr := os.Open(*videoFilename)
			if ivfErr != nil {
				log.Fatal(ivfErr)
			}

			ivf, header, ivfErr := ivfreader.NewWith(file)
			if ivfErr != nil {
				log.Fatal(ivfErr)
			}

			sleepTime := time.Millisecond * time.Duration((float32(header.TimebaseNumerator)/float32(header.TimebaseDenominator))*1000)
			for {
				if videoTrack != nil {
					frame, _, ivfErr := ivf.ParseNextFrame()
					if ivfErr == io.EOF {
						log.Printf("All video frames parsed and sent, rewind to top.")
						file.Seek(0, io.SeekStart)
						ivf, header, ivfErr = ivfreader.NewWith(file)
						if ivfErr != nil {
							log.Fatal(ivfErr)
						}
						continue
					}

					if ivfErr != nil {
						log.Fatal(ivfErr)
					}

					time.Sleep(sleepTime)
					if ivfErr = videoTrack.WriteSample(media.Sample{Data: frame, Samples: 90000}); ivfErr != nil {
						log.Fatal(ivfErr)
					}
				}
			}
		}()
	})

	err = con.Connect()
	if err != nil {
		log.Fatal("failed to connect Sora", err)
	}

	// Block forever
	select {}
}

// Search for Codec PayloadType
//
// Since we are answering we need to match the remote PayloadType
func getPayloadType(m webrtc.MediaEngine, codecType webrtc.RTPCodecType, codecName string) uint8 {
	for _, codec := range m.GetCodecsByKind(codecType) {
		if codec.Name == codecName {
			return codec.PayloadType
		}
	}
	panic(fmt.Sprintf("Remote peer does not support %s", codecName))
}
