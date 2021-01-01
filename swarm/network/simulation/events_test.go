// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package simulation

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestPeerEvents creates simulation, adds two nodes,
// register for peer events, connects nodes in a chain
// and waits for the number of connection events to
// be received.
func TestPeerEvents(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	_, err := sim.AddNodes(2)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	events := sim.PeerEvents(ctx, sim.NodeIDs())

	// two nodes -> two connection events
	expectedEventCount := 2

	var wg sync.WaitGroup
	wg.Add(expectedEventCount)

	go func() {
		for e := range events {
			if e.Error != nil {
				if e.Error == context.Canceled {
					return
				}
				t.Error(e.Error)
				continue
			}
			wg.Done()
		}
	}()

	err = sim.Net.ConnectNodesChain(sim.NodeIDs())
	if err != nil {
		t.Fatal(err)
	}

	wg.Wait()
}

func TestPeerEventsTimeout(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	_, err := sim.AddNodes(2)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	events := sim.PeerEvents(ctx, sim.NodeIDs())

	done := make(chan struct{})
	errC := make(chan error)
	go func() {
		for e := range events {
			if e.Error == context.Canceled {
				return
			}
			if e.Error == context.DeadlineExceeded {
				close(done)
				return
			} else {
				errC <- e.Error
			}
		}
	}()

	select {
	case <-time.After(time.Second):
		t.Fatal("no context deadline received")
	case err := <-errC:
		t.Fatal(err)
	case <-done:
		// all good, context deadline detected
	}
}
