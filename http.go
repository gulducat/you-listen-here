package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TODO: custom sync.Map thing

func webClient(sampleCh chan<- []float32) {
	host := os.Getenv("HOST")
	if host == "" {
		host = "http://127.0.0.1:8080"
	}
	log.Println("connecting to:", host+"/stream")
	resp, err := http.Get(host + "/stream")
	if err != nil {
		log.Fatal("get err:", err)
	}
	reader := bufio.NewReader(resp.Body)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Println("readbytes err:", err)
			return
		}
		// log.Printf("got: %s", line) // this actually slows things down noticably when we're talking 48000hz
		samps := []float32{}
		for _, samp := range strings.Split(line, ",") {
			samp = strings.TrimSpace(samp)
			f, err := strconv.ParseFloat(samp, 32)
			if err != nil {
				log.Fatal("parseFloat:", err)
			}
			samps = append(samps, float32(f))
		}
		sampleCh <- samps
	}
}

func webServer(ctx context.Context, sampleCh <-chan []float32) {
	// add handlers for http routes
	clientChans := addHandlers()

	// send input sample slices to all client channels as they come in
	go fanChannels(ctx, sampleCh, clientChans)

	// serve http
	errCh := make(chan error)
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		log.Println("http listen: " + port)
		errCh <- http.ListenAndServe("0.0.0.0:"+port, nil)
	}()

	select {
	case err := <-errCh:
		log.Println("server err:", err)
	case <-ctx.Done():
		log.Println("web server done")
	}
}

func fanChannels(ctx context.Context, inCh <-chan []float32, outChans *sync.Map) {
	for {
		select {
		case <-ctx.Done():
			log.Println("channel fanner ctx done")
			return

		case samples := <-inCh:
			outChans.Range(func(_, val interface{}) bool {
				ch, ok := val.(chan []float32)
				if !ok {
					log.Printf("not a sample chan?  %#v", ch)
					return true
				}

				// anonymous goroutine to prevent timeout case from blocking other channels
				go func(ch chan []float32) {
					select {
					case ch <- samples: // this right here is the main event in this program.
					case <-time.After(time.Millisecond * 500):
						log.Println("no sendy after 500ms")
					}
				}(ch)

				return true
			})
		}
	}
}

func addHandlers() (chans *sync.Map) {
	chans = &sync.Map{}

	count := 0      // current open connections
	totalCount := 0 // total number of /stream connections, mainly for /stats vanity

	http.HandleFunc("/", tryFiles)

	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		msg := fmt.Sprintf("total: %d\ncurrent: %d\n", totalCount, count)
		if _, err := w.Write([]byte(msg)); err != nil {
			log.Println("error writing stats:", err)
		}
	})

	http.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		if count > 50 {
			w.WriteHeader(http.StatusGatewayTimeout)
			_, err := w.Write([]byte("uh oh too many?"))
			chk("too many", err)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			return
		}
		count++
		totalCount++
		func(id int) {
			ch := make(chan []float32, 48000) // todo: hm.
			// defer close(ch) // this causes panic, shouldn't close here on the receiver side
			log.Println("✅ connecting:", id, "current:", count)

			chans.Store(id, ch)
			handleStream(w, r, id, ch)
			chans.Delete(id)

			count--
			log.Println("❌ completed:", id, "current:", count)
		}(totalCount)
	})

	return chans
}

func tryFiles(w http.ResponseWriter, r *http.Request) {
	uri := strings.TrimPrefix(r.RequestURI, "/")
	if uri == "" {
		uri = "index.html"
	}
	if uri == "favicon.ico" {
		return
	}

	f, err := os.Open(uri)
	if err != nil {
		log.Println("open", uri, err)
		return
	}
	log.Println("loading file:", f.Name())
	bts, err := ioutil.ReadAll(f)
	if err != nil {
		log.Println("readall", uri, err)
		return
	}
	_, err = w.Write(bts)
	if err != nil {
		log.Println("write", uri, err)
	}
}

func handleStream(w http.ResponseWriter, r *http.Request, id int, ch <-chan []float32) {
	// hm. https://medium.com/@valentijnnieman_79984/how-to-build-an-audio-streaming-server-in-go-part-1-1676eed93021
	w.Header().Set("Connection", "Keep-Alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	for {
		select {
		case c := <-ch:
			msg := ""
			for i := range c {
				msg += fmt.Sprintf("%f,", c[i])
				// msg += fmt.Sprintf("%d,", fToInt16(c[i]))
			}
			msg = strings.Trim(msg, ",")
			msg += "\n"
			if _, err := w.Write([]byte(msg)); err != nil {
				log.Println(id, "err:", err)
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

		case <-r.Context().Done():
			err := r.Context().Err()
			if err != nil && err != context.Canceled {
				log.Println(id, "ctx err:", err)
			}
			return
		}
	}
}
