// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package nat

import (
	"net"
	"testing"
	"time"
)

// This test checks that autodisc doesn't hang and returns
// consistent results when multiple goroutines call its mfafods
// concurrently.
func TestAutoDiscRace(t *testing.T) {
	ad := startautodisc("thing", func() Interface {
		time.Sleep(500 * time.Millisecond)
		return ExtIP{33, 44, 55, 66}
	})

	// Spawn a few concurrent calls to ad.ExternalIP.
	type rval struct {
		ip  net.IP
		err error
	}
	results := make(chan rval, 50)
	for i := 0; i < cap(results); i++ {
		go func() {
			ip, err := ad.ExternalIP()
			results <- rval{ip, err}
		}()
	}

	// Check that they all return the correct result within the deadline.
	deadline := time.After(2 * time.Second)
	for i := 0; i < cap(results); i++ {
		select {
		case <-deadline:
			t.Fatal("deadline exceeded")
		case rval := <-results:
			if rval.err != nil {
				t.Errorf("result %d: unexpected error: %v", i, rval.err)
			}
			wantIP := net.IP{33, 44, 55, 66}
			if !rval.ip.Equal(wantIP) {
				t.Errorf("result %d: got IP %v, want %v", i, rval.ip, wantIP)
			}
		}
	}
}