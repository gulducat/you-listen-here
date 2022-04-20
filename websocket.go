package main

// TODO: profile

/* gorilla
 * https://github.com/gorilla/websocket/blob/master/examples/echo/server.go
 * https://github.com/gorilla/websocket/blob/master/examples/command/main.go
 */

// TODO: nhooyr.io/websocket ???  supposedly idiomatic according to Val

import (
	"compress/flate"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{EnableCompression: true}

func ServeWebSocket(ctx context.Context, sampleCh <-chan []float32) {
	clientChans := &SampleChanneler{}

	// send input sample slices to all client channels as they come in
	go FanChannels(ctx, sampleCh, clientChans)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("🌍", r.RequestURI)
		file := strings.TrimPrefix(r.RequestURI, "/")
		switch file {
		case "":
			file = "websocket.html"
		// other allowed files
		case "websocket.js":
		case "you-listen-here.png":
		// case "favicon.ico":
		// do not load anything else (#security hah.)
		default:
			return
		}
		http.ServeFile(w, r, file)
	})

	http.HandleFunc("/websocket", handleWebSocket(clientChans))

	// serve http
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = "0.0.0.0:8080"
	}
	server := &http.Server{Addr: addr, Handler: nil}
	errCh := make(chan error)
	go func() {
		log.Println("http listen: " + addr)
		errCh <- server.ListenAndServe()
	}()

	// wait for server error or context to get done
	select {
	case err := <-errCh:
		log.Println("server err:", err)
	case <-ctx.Done():
		if err := server.Close(); err != nil {
			log.Println("err closing web server:", err)
		}
		log.Println("web server done")
	}

}

func handleWebSocket(chans *SampleChanneler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("🌍", r.RequestURI)

		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("upgrade:", err)
			return
		}
		defer c.Close()

		// max compression - TODO: does this do anything useful at all?  does it increase CPU usage?
		if err := c.SetCompressionLevel(flate.BestCompression); err != nil {
			log.Println("failed to set compression level to 9")
			return
		}

		id, ch := chans.NewChannel(4000) // TODO: is this a reasonable number?
		defer chans.Delete(id)

		log.Printf("✅ new connection id=%d current=%d", id, chans.Len())

		// listen in background for close message
		errCh := make(chan error)
		go func() {
			mt, message, err := c.ReadMessage()
			if err != nil {
				if mt == -1 {
					log.Printf("💔 end connection id=%d current=%d", id, chans.Len()-1)
					err = nil
				} else {
					log.Println("❓ mt:", mt, "message:", message, "sending to errCh")
				}
			}
			errCh <- err
			// close(errCh)
		}()

		for {
			select {
			// main loop
			case samples := <-ch:
				// TODO: more samples per message?  would that help with compression or...?
				msg := samplesToBytes(samples)
				err := c.WriteMessage(websocket.TextMessage, msg)
				if err != nil {
					// TODO: why? `c.WriteMessage err: websocket: close sent`
					log.Println("c.WriteMessage err:", err)
					return
				}

			// catch close message and exit this handler
			case err := <-errCh:
				if err != nil {
					log.Println("errCh:", err)
				}
				return

			// request context might get done'd without a websocket close?
			case <-r.Context().Done():
				err := r.Context().Err()
				if err != nil && err != context.Canceled {
					log.Println(id, "request ctx err:", err)
				}
				return
			}
		}
	}
}

func samplesToBytes(samples []float32) []byte {
	// TODO: this feels inefficient...
	msg := ""
	for _, samp := range samples {
		msg += fmt.Sprintf("%f,", samp)
	}
	msg = strings.Trim(msg, ",") // drop trailing ','
	return []byte(msg)
}
