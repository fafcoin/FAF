// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package mem

import (
	"testing"

	"github.com/fafereum/go-fafereum/swarm/storage/mock/test"
)

// TestGlobalStore is running test for a GlobalStore
// using test.MockStore function.
func TestGlobalStore(t *testing.T) {
	test.MockStore(t, NewGlobalStore(), 100)
}

// TestImportExport is running tests for importing and
// exporting data between two GlobalStores
// using test.ImportExport function.
func TestImportExport(t *testing.T) {
	test.ImportExport(t, NewGlobalStore(), NewGlobalStore(), 100)
}
