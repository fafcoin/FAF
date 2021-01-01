// +build go1.8
//
// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package db

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/fafereum/go-fafereum/swarm/storage/mock/test"
)

// TestDBStore is running a test.MockStore tests
// using test.MockStore function.
func TestDBStore(t *testing.T) {
	dir, err := ioutil.TempDir("", "mock_"+t.Name())
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	store, err := NewGlobalStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	test.MockStore(t, store, 100)
}

// TestImportExport is running a test.ImportExport tests
// using test.MockStore function.
func TestImportExport(t *testing.T) {
	dir1, err := ioutil.TempDir("", "mock_"+t.Name()+"_exporter")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir1)

	store1, err := NewGlobalStore(dir1)
	if err != nil {
		t.Fatal(err)
	}
	defer store1.Close()

	dir2, err := ioutil.TempDir("", "mock_"+t.Name()+"_importer")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir2)

	store2, err := NewGlobalStore(dir2)
	if err != nil {
		t.Fatal(err)
	}
	defer store2.Close()

	test.ImportExport(t, store1, store2, 100)
}
