package main

import (
	"log"
	"os"

	"github.com/aler9/gortsplib"
	"github.com/aler9/gortsplib/pkg/url"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("wrong usage: provide rtsp stream url")
	}

	c := gortsplib.Client{}
	u, err := url.Parse(os.Args[1])
	if err != nil {
		log.Fatalf("error parsing url: %v", err)
	}

	// connect to the server
	err = c.Start(u.Scheme, u.Host)
	if err != nil {
		log.Fatalf("error connecting to server: %v", err)
	}
	defer c.Close()

	// find published tracks
	tracks, baseURL, resp, err := c.Describe(u)
	if err != nil {
		log.Fatalf("error finding published tracks: %v", err)
	}
	log.Printf("RTSP server response:\n%s\n", resp)

	// find the H264 track
	h264TrackID, h264track := -1, gortsplib.TrackH264{}
	for i, track := range tracks {
		if h264track, ok := track.(*gortsplib.TrackH264); ok {
			log.Printf("found H264 track (%d): %#v\n", i, h264track)
			h264TrackID = i
			break
		} else {
			log.Printf("cannot convert track to H264, using general track info")
			log.Printf("general tack (%d): %#v", i, track)
		}
	}
	if h264TrackID < 0 {
		log.Fatal("H264 track not found")
	}

	// setup H264->raw frames decoder
	h264dec, err := newH264Decoder()
	if err != nil {
		log.Fatalf("error decoding H264 stream: %v", err)
	}
	defer h264dec.close()

	// if present, send SPS and PPS from the SDP to the decoder
	sps := h264track.SafeSPS()
	if sps != nil {
		h264dec.decode(sps)
	}
	pps := h264track.SafePPS()
	if pps != nil {
		h264dec.decode(pps)
	}

	// called when a RTP packet arrives
	c.OnPacketRTP = func(ctx *gortsplib.ClientOnPacketRTPCtx) {
		if ctx.TrackID != h264TrackID {
			return
		}
		if ctx.H264NALUs == nil {
			return
		}
		for _, nalu := range ctx.H264NALUs {
			// convert H264 NALUs to RGBA frames
			img, err := h264dec.decode(nalu)
			if err != nil {
				log.Fatalf("error decoding NALU: %v", err)
			}
			// wait for a frame
			if img == nil {
				continue
			}
			log.Printf("decoded frame with size %v", img.Bounds().Max)
		}
	}

	// start reading tracks
	err = c.SetupAndPlay(tracks, baseURL)
	if err != nil {
		log.Fatalf("error playing stream %q: %v", baseURL, err)
	}

	// wait until a fatal error
	log.Fatal(c.Wait())
}
