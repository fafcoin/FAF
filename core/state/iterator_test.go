// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify


package state

import (
	"bytes"
	"testing"

	"github.com/fafereum/go-fafereum/common"
	"github.com/fafereum/go-fafereum/fafdb"
)

// Tests that the node iterator indeed walks over the entire database contents.
func TestNodeIteratorCoverage(t *testing.T) {
	// Create some arbitrary test state to iterate
	db, root, _ := makeTestState()

	state, err := New(root, db)
	if err != nil {
		t.Fatalf("failed to create state trie at %x: %v", root, err)
	}
	// Gather all the node hashes found by the iterator
	hashes := make(map[common.Hash]struct{})
	for it := NewNodeIterator(state); it.Next(); {
		if it.Hash != (common.Hash{}) {
			hashes[it.Hash] = struct{}{}
		}
	}
	// Cross check the iterated hashes and the database/nodepool content
	for hash := range hashes {
		if _, err := db.TrieDB().Node(hash); err != nil {
			t.Errorf("failed to retrieve reported node %x", hash)
		}
	}
	for _, hash := range db.TrieDB().Nodes() {
		if _, ok := hashes[hash]; !ok {
			t.Errorf("state entry not reported %x", hash)
		}
	}
	for _, key := range db.TrieDB().DiskDB().(*fafdb.MemDatabase).Keys() {
		if bytes.HasPrefix(key, []byte("secure-key-")) {
			continue
		}
		if _, ok := hashes[common.BytesToHash(key)]; !ok {
			t.Errorf("state entry not reported %x", key)
		}
	}
}
