// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package intervals

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/fafereum/go-fafereum/swarm/state"
)

// TestDBStore tests basic functionality of DBStore.
func TestDBStore(t *testing.T) {
	dir, err := ioutil.TempDir("", "intervals_test_db_store")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	store, err := state.NewDBStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	testStore(t, store)
}
