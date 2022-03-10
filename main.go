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

	// for wrapping things up
	defer log.Println("really fin.")
	sigsCh := make(chan os.Signal, 1)
	signal.Notify(sigsCh, syscall.SIGINT, syscall.SIGTERM)
	ctx, stopCtx := context.WithCancel(context.Background())
	defer stopCtx()

	// init portaudio
	term := audioInit()
	defer term()

	// TODO: 2,2 (in,out) works for my clavinova,
	// but won't work for mic which needs 1,1
	p, err := getParams(1, 1)
	chk("getParams", err)
	// spew.Dump("params", p)
	// return

	buffer := int(p.SampleRate) * 2
	// buffer := int(p.SampleRate)
	// buffer := int(p.SampleRate) / 2
	// buffer := int(p.SampleRate) / 32
	// buffer := int(p.SampleRate) / 128
	// buffer := 1024
	sch := make(chan []float32, buffer)
	s := &Streamer{
		ch: sch,
	}

	switch os.Args[1] {

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
		go stream(s.ch)

	default:
		log.Fatal(badArgs)
	}

	select {
	// case <-time.After(time.Second * 300):
	// 	log.Println("SELECT after")
	case sig := <-sigsCh:
		log.Println("SELECT signal:", sig)
	case <-ctx.Done():
		log.Println("SELECT ctx:", ctx.Err())
	}
	log.Println("start final sleep")
	time.Sleep(time.Millisecond * 50)
	log.Println("fin.")

}
