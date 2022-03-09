package main

import (
	"context"
	"log"
	"time"

	"github.com/gordonklaus/portaudio"
)

func audioInit() (terminate func()) {
	log.Println("portaudio.Initialize()")
	chk("init", portaudio.Initialize())

	return func() {
		log.Println("portaudio.Terminate()")
		chk("term", portaudio.Terminate())
	}
}

func getParams(in, out int) (portaudio.StreamParameters, error) {
	h, err := portaudio.DefaultHostApi()
	if err != nil {
		return portaudio.StreamParameters{}, err
	}

	p := portaudio.LowLatencyParameters(h.DefaultInputDevice, h.DefaultOutputDevice)
	p.Input.Channels = in
	p.Output.Channels = out

	log.Println("samplerate:", p.SampleRate)
	log.Println("input:", p.Input.Device.Name)
	log.Println("output:", p.Output.Device.Name)

	return p, nil
}

type Streamer struct {
	*portaudio.Stream
	ch chan float32
}

// type StreamProcessor func() ([]float32, []float32)

func OpenStream(ctx context.Context, p portaudio.StreamParameters, s *Streamer, f func([]float32, []float32)) {
	var err error
	s.Stream, err = portaudio.OpenStream(p, f)
	chk("open", err)
	chk("start", s.Start())
	<-ctx.Done()
	chk("stop", s.Stop())
	chk("close", s.Close())
}

func (s *Streamer) read(in, out []float32) {
	// log.Println("read", in)
	for i := range in {
		// log.Println("i", i, in[i])
		select {
		case s.ch <- in[i]:
		case <-time.After(time.Millisecond * 20):
			log.Println("read timeout")
		}
	}
}

func (s *Streamer) write(in, out []float32) {
	// log.Println("write")
	for i := range out {
		select {
		case v := <-s.ch:
			// log.Println("v", v)
			out[i] = v
		case <-time.After(time.Millisecond * 20):
			log.Println("write timeout")
		}
	}
}

// func (s *Streamer) read(ctx context.Context) func([]float32, []float32) {
// 	return func(in, out []float32) {
// 		// log.Println("read", in)
// 		for i := range in {
// 			// log.Println("i", i, in[i])
// 			select {
// 			case s.ch <- in[i]:
// 			case <-time.After(time.Millisecond * 20):
// 				log.Println("read timeout")
// 			case <-ctx.Done():
// 				log.Println("read ctx done")
// 				return
// 			}
// 		}
// 	}
// }

// func (s *Streamer) write(ctx context.Context) func([]float32, []float32) {
// 	return func(in, out []float32) {
// 		// log.Println("write")
// 		for i := range out {
// 			select {
// 			case v := <-s.ch:
// 				// log.Println("v", v)
// 				out[i] = v
// 			case <-time.After(time.Millisecond * 20):
// 				log.Println("write timeout")
// 			case <-ctx.Done():
// 				log.Println("write ctx done")
// 				return
// 			}
// 		}
// 	}
// }
