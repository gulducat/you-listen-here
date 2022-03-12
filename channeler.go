package main

// probably cleverer (more complicated) than is necessary, but i had fun.

import (
	"log"
	"sync"
	"time"
)

type SampleChanneler struct {
	sync.Map

	// do these threaten thread safety?
	count int
	len   int
}

/*
 * things sync.Map doesn't have
 */

func (s *SampleChanneler) NewChannel(buffer int) (id int, ch chan []float32) {
	ch = make(chan []float32, buffer)
	s.count++
	id = s.count
	s.Store(id, ch)
	return id, ch
}

func (s *SampleChanneler) WriteToAll(samples []float32, timeout time.Duration) {
	s.Range(func(key int, dest chan []float32) {
		// goroutine so that one slow channel doesn't hold up the whole bunch
		// TODO: does this matter with very very low timeout?  goroutines are cheap right?
		go func() {
			select {
			case dest <- samples:
			case <-time.After(timeout):
				log.Println("failed to write within", timeout)
				// delete me because im probably dead? TODO: is this safe?
				// s.Delete(key)
			}
		}()
	})
}

/*
 * overloading sync.Map methods, roughly
 */

func (s *SampleChanneler) Store(key int, ch chan []float32) {
	s.len++ // assume we're playing nice and only ever writing a key once.
	s.Map.Store(key, ch)
}

func (s *SampleChanneler) Delete(key int) {
	s.Map.Delete(key)
	s.len--
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
