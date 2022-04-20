package main

// probably cleverer (more complicated) than is necessary, but i had fun.

import (
	"context"
	"log"
	"sync"
	"time"
)

func FanChannels(ctx context.Context, inCh <-chan []float32, outChs *SampleChanneler) {
	for {
		select {
		case samples := <-inCh:
			outChs.WriteToAll(samples, time.Millisecond*250) // TODO: good magic?

		case <-ctx.Done():
			log.Println("channel fanner ctx done")
			return
		}
	}
}

type SampleChanneler struct {
	sync.Map

	mu    sync.RWMutex
	count int
	len   int
}

/*
 * things sync.Map doesn't have
 */

func (s *SampleChanneler) NewChannel(buffer int) (id int, ch chan []float32) {
	ch = make(chan []float32, buffer)

	s.mu.Lock()
	s.count++
	id = s.count
	s.mu.Unlock()

	s.Store(id, ch)
	return id, ch
}

func (s *SampleChanneler) WriteToAll(samples []float32, timeout time.Duration) {
	s.Range(func(key int, dest chan []float32) {
		// goroutine so that one slow channel doesn't hold up the whole bunch
		go func() {
			select {
			case dest <- samples:
			case <-time.After(timeout):
				log.Printf("failed to write to %d within %v", key, timeout)
				// delete me because im probably dead? TODO: is this safe? nope
				// s.Delete(key)
			}
		}()
	})
}

func (s *SampleChanneler) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.count
}

func (s *SampleChanneler) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.len
}

/*
 * overloading sync.Map methods, roughly
 */

func (s *SampleChanneler) Store(key int, ch chan []float32) {
	s.Map.Store(key, ch)

	s.mu.Lock()
	s.len++ // assume we're playing nice and only ever writing a key once.
	s.mu.Unlock()
}

func (s *SampleChanneler) Delete(key int) {
	s.Map.Delete(key)

	s.mu.Lock()
	s.len--
	s.mu.Unlock()
}

func (s *SampleChanneler) Load(key int) chan []float32 {
	val, ok := s.Map.Load(key)
	if !ok {
		return nil // nil channels are dangerous...?
	}
	return val.(chan []float32)
}

// this signature diverges a decent bit from sync.Map.
// we always want to hit all channels, so don't need no extra bool.
func (s *SampleChanneler) Range(f func(key int, ch chan []float32)) {
	s.Map.Range(func(key, val interface{}) bool {
		f(key.(int), val.(chan []float32))
		return true // keep going
	})
}
