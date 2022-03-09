package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

func stream(msgCh chan<- []float32) {
	resp, err := http.Get("http://127.0.0.1:8080/stream")
	if err != nil {
		log.Fatal("get err:", err)
	}
	reader := bufio.NewReader(resp.Body)
	// reader := bufio.NewReaderSize(resp.Body, 10)

	// this did not help.
	// buf := []float32{}
	// bufPrimed := false

	for {
		ticker := time.NewTicker(time.Second / 48000)

		line, err := reader.ReadString('\n')
		if err != nil {
			log.Println("readbytes err:", err)
			return
		}
		// log.Printf("got: %s", line) // this actually slows things down noticably when we're talking 48000hz
		samps := []float32{}
		for _, samp := range strings.Split(line, ",") {
			trimmed := strings.TrimSpace(samp)
			if trimmed == "" {
				continue
			}
			f, err := strconv.ParseFloat(trimmed, 32)
			if err != nil {
				log.Fatal("parseFloat:", err)
			}
			samps = append(samps, float32(f))
		}
		msgCh <- samps
		// log.Printf("got: %s", samps) // this actually slows things down noticably when we're talking 48000hz

		// f, err := strconv.ParseFloat(strings.TrimSpace(line), 32)
		// if err != nil {
		// 	log.Fatal("parseFloat:", err)
		// }

		// msgCh <- float32(f)
		// if bufPrimed {
		// 	msgCh <- float32(f)
		// } else {
		// 	buf = append(buf, float32(f))
		// 	// fmt.Println("len:", len(buf), "len2:", len(msgCh))
		// 	// if len(buf) == len(msgCh) { msgCh len is 0...
		// 	if len(buf) == 48000 {
		// 		log.Println("primed yeah!")
		// 		bufPrimed = true
		// 		for _, f := range buf {
		// 			msgCh <- f
		// 		}
		// 		log.Println("gotem")
		// 	}
		// }

		<-ticker.C
	}
}

func webServer(ctx context.Context, msgCh <-chan []float32) {
	chans := make(map[int]chan []float32)
	var chLock sync.RWMutex

	// send messages from msgCh to all worker chans
	go func(ctx context.Context) {
		// t := time.NewTicker(time.Second / 10)
		// t := time.NewTicker(time.Second / len(msgCh)) // lol.  need to buffer.
		// t := time.NewTicker(time.Second / len(msgCh)) // lol.  need to buffer.
		// t := time.NewTicker(time.Second / 52000) // it just needs to be *bigger* than device's samplerate? nope lol bigger number makes slow down sound
		// t := time.NewTicker(time.Second / 48000) // TODO
		// t := time.NewTicker(time.Second / 16000) // TODO
		stopCh := make(chan struct{}, 1)
	loop:
		for {
			select {
			case <-ctx.Done():
				log.Println("worker messenger done")
				break loop
			case <-stopCh:
				log.Println("stopCh")
				break loop
			case msg := <-msgCh:
				chLock.RLock()
				for _, ch := range chans {
					// ch <- msg

					// this does something, but does it help?
					go func(ch chan []float32) {
						select {
						case ch <- msg:
						case <-time.After(time.Millisecond * 500):
							log.Println("no sendy after 500ms")
							stopCh <- struct{}{}
							return
						}
					}(ch)
					// if ch, ok <- msg; !ok {
					// 	log.Println("a channel must be closed")
					// }
				}
				chLock.RUnlock()

				// log.Println("block")
				// <-t.C
			}
		}
	}(ctx)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open("index.html")
		chk("open index", err)
		bts, err := ioutil.ReadAll(f)
		chk("ioutil.read", err)
		_, err = w.Write(bts)
		chk("write index", err)
	})

	http.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		ch := make(chan []float32, 48000) // todo: hm.
		defer close(ch)
		id := rand.Int() // todo hm.

		chLock.Lock()
		chans[id] = ch
		chLock.Unlock()

		log.Println("connected:", id)
		handle(w, r, id, ch)
		log.Println("completed:", id)

		chLock.Lock()
		delete(chans, id)
		chLock.Unlock()
	})

	errCh := make(chan error)
	go func() {
		fmt.Println("starting server")
		errCh <- http.ListenAndServe("0.0.0.0:8080", nil)
	}()

	select {
	case err := <-errCh:
		log.Println("server err:", err)
	case <-ctx.Done():
		log.Println("web server done")
	}
}

func handle(w http.ResponseWriter, r *http.Request, id int, ch <-chan []float32) {

	// hm. https://medium.com/@valentijnnieman_79984/how-to-build-an-audio-streaming-server-in-go-part-1-1676eed93021
	w.Header().Set("Connection", "Keep-Alive")
	w.Header().Set("Transfer-Encoding", "chunked")
	// TODO: something something binary.Write(w) instead of w.Write() ???

	for {
		select {
		case c := <-ch:
			msg := ""
			for i := range c {
				msg += fmt.Sprintf("%f,", c[i])
			}
			msg += "\n"
			// msg := fmt.Sprintf("%f\n", c)
			// msg := fmt.Sprintf("%d hi %f\n", id, c)
			// log.Printf("writing %s", msg)
			if _, err := w.Write([]byte(msg)); err != nil {
				log.Println(id, "err:", err)
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			// return

		case <-r.Context().Done():
			err := r.Context().Err()
			if err != nil && err != context.Canceled {
				log.Println(id, "ctx err:", err)
			}
			// log.Println(id, "ctx done")
			return
		}
	}
}

/*
 * eddie:
 * what all are they doing?  duct for duct, maybe 5-6k
 * this is more of a redesign, which requires some expertise?
 * national comfort institute - industry leaders for design standards
 *
 * he has the guys, labor is ok
 * not as sure of materials, national shortage of duct work
 */
