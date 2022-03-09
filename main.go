package main

/* notes
 *
 * remember: getParams() needs appropriate in/out numbers for the devices..
 * why crackling over http?  where is bottleneck?
 *   wait until buffer is full somehow?
 *   maybe do binary instead of straight byte stream?
 */

import (
	"log"
	"os"
	"time"

	"github.com/gordonklaus/portaudio"
)

func chk(msg string, err error) {
	log.Println("chk msg:", msg)
	if err != nil {
		// log.Println(msg + ": " + err.Error())
		panic(msg + ": " + err.Error())
	}
}

func main() {
	badArgs := "u want 'server' or 'client' or 'echo' pls?"
	if len(os.Args) < 2 {
		log.Fatal(badArgs)
	}

	// doneCh := make(chan struct{})
	defer audioInit()()

	// TODO: 2,2 (in,out) works for my clavinova and speakers,
	// but won't work for mic which needs 1,2
	p, err := getParams(2, 2)
	chk("getParams", err)
	// spew.Dump("params", p)
	// return
	log.Println("samplerate:", p.SampleRate)
	log.Println("input:", p.Input.Device.Name)
	log.Println("output:", p.Output.Device.Name)

	// buffer := int(p.SampleRate) * 2
	// buffer := int(p.SampleRate)
	buffer := int(p.SampleRate) / 2
	// buffer := int(p.SampleRate) / 32
	// buffer := int(p.SampleRate) / 128
	// buffer := 1024
	sch := make(chan float32, buffer)
	s := &Streamer{
		ch: sch,
	}

	switch os.Args[1] {

	case "echo":
		defer OpenStream(p, s, s.read)()
		s2 := &Streamer{ch: sch}
		defer OpenStream(p, s2, s2.write)()

	// case "server":
	// 	defer OpenStream(p, s, s.read)()
	// 	go webServer(s.ch, doneCh)

	// case "client":
	// 	// p.Output.Channels = 1
	// 	defer OpenStream(p, s, s.write)()
	// 	go stream(s.ch)

	default:
		log.Fatal(badArgs)
	}

	time.Sleep(time.Second * 60)
	// doneCh <- struct{}{}
}

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
	chk("DefaultHostApi", err)

	p := portaudio.LowLatencyParameters(h.DefaultInputDevice, h.DefaultOutputDevice)
	p.Input.Channels = in
	p.Output.Channels = out
	return p, nil
}

type Streamer struct {
	*portaudio.Stream
	ch chan float32
}

func OpenStream(p portaudio.StreamParameters, s *Streamer, f func([]float32, []float32)) func() {
	var err error
	s.Stream, err = portaudio.OpenStream(p, f)
	chk("open", err)
	chk("start", s.Start())
	return func() {
		defer s.Close()
		chk("stop", s.Stop())
	}
}

func (s *Streamer) read(in, out []float32) {
	// log.Println("read", in)
	for i := range in {
		// log.Println("i", i, in[i])
		s.ch <- in[i]
	}
}

func (s *Streamer) write(in, out []float32) {
	// log.Println("write")
	for i := range out {
		select {
		case v := <-s.ch:
			// log.Println("v", v)
			out[i] = v
		case <-time.After(time.Second):
			log.Println("timeout")
		}
	}
}
