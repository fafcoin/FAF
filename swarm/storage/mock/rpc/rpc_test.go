// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package rpc

import (
	"testing"

	"github.com/fafereum/go-fafereum/rpc"
	"github.com/fafereum/go-fafereum/swarm/storage/mock/mem"
	"github.com/fafereum/go-fafereum/swarm/storage/mock/test"
)

// TestDBStore is running test for a GlobalStore
// using test.MockStore function.
func TestRPCStore(t *testing.T) {
	serverStore := mem.NewGlobalStore()

	server := rpc.NewServer()
	if err := server.RegisterName("mockStore", serverStore); err != nil {
		t.Fatal(err)
	}

	store := NewGlobalStore(rpc.DialInProc(server))
	defer store.Close()

	test.MockStore(t, store, 30)
}
