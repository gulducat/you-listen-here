package main

/* notes
 *
 * remember: getParams() needs appropriate in/out numbers for the devices..
 * why crackling over http?  where is bottleneck?
 *   wait until buffer is full somehow?
 *   maybe do binary instead of straight byte stream?
 */

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// todo: not this.
func chk(msg string, err error) {
	log.Println("chk msg:", msg)
	if err != nil {
		// log.Println(msg + ": " + err.Error())
		panic(msg + ": " + err.Error())
	}
}

func main() {
	badArgs := "u want 'websocket' or 'server' or 'client' or 'echo' pls?"
	if len(os.Args) < 2 {
		log.Fatal(badArgs)
	}

	// for wrapping things up
	sigsCh := make(chan os.Signal, 1)
	signal.Notify(sigsCh, syscall.SIGINT, syscall.SIGTERM)
	// TODO: does this withcancel actually do anything here?
	ctx, stopCtx := context.WithCancel(context.Background())
	defer stopCtx()

	// init portaudio NOTE: this needs to be here, run *after* stopCtx...?
	term := audioInit()
	defer term()

	// TODO: 2,2 (in,out) works for my clavinova,
	// but won't work for mic which needs 1,1
	p, err := getParams(1, 1)
	chk("getParams", err)

	// this buffer is to account for possible disparity between
	// speed of incoming information from portaudio
	// and fanning the data out to client channels.
	// it does not introduce much if any delay from what I can tell with my ears.
	buffer := int(p.SampleRate) * 2 // lol idk
	// buffer := 2048
	s := &Streamer{
		ch: make(chan []float32, buffer),
	}

	switch os.Args[1] {

	case "websocket":
		go OpenStream(ctx, p, s, s.read)
		go ServeWebSocket(ctx, s.ch)

	case "echo":
		go OpenStream(ctx, p, s, s.read)
		// time.Sleep(time.Millisecond * 500) // does nothing... how to make it do something again? ensure full buffer?

		sch2 := make(chan []float32, buffer)
		s2 := &Streamer{ch: sch2}
		// little go-betweener sorta simulates what the http worker does?
		go func() {
			for {
				s2.ch <- <-s.ch
			}
		}()
		go OpenStream(ctx, p, s2, s2.write)

	case "server":
		go OpenStream(ctx, p, s, s.read)
		go webServer(ctx, s.ch)

	case "client":
		go OpenStream(ctx, p, s, s.write)
		go webClient(s.ch)

	default:
		log.Fatal(badArgs)
	}

	select {
	case sig := <-sigsCh:
		log.Println("END signal:", sig)
		stopCtx()
	case <-ctx.Done():
		log.Println("END ctx:", ctx.Err())
	}
	log.Println("stop", s.Stop())
	log.Println("close", s.Close())
	log.Println("start final sleep")
	time.Sleep(time.Millisecond * 50)
	log.Println("fin.")

}
