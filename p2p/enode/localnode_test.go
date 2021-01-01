// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package enode

import (
	"testing"

	"github.com/fafereum/go-fafereum/crypto"
	"github.com/fafereum/go-fafereum/p2p/enr"
)

func newLocalNodeForTesting() (*LocalNode, *DB) {
	db, _ := OpenDB("")
	key, _ := crypto.GenerateKey()
	return NewLocalNode(db, key), db
}

func TestLocalNode(t *testing.T) {
	ln, db := newLocalNodeForTesting()
	defer db.Close()

	if ln.Node().ID() != ln.ID() {
		t.Fatal("inconsistent ID")
	}

	ln.Set(enr.WithEntry("x", uint(3)))
	var x uint
	if err := ln.Node().Load(enr.WithEntry("x", &x)); err != nil {
		t.Fatal("can't load entry 'x':", err)
	} else if x != 3 {
		t.Fatal("wrong value for entry 'x':", x)
	}
}

func TestLocalNodeSeqPersist(t *testing.T) {
	ln, db := newLocalNodeForTesting()
	defer db.Close()

	if s := ln.Node().Seq(); s != 1 {
		t.Fatalf("wrong initial seq %d, want 1", s)
	}
	ln.Set(enr.WithEntry("x", uint(1)))
	if s := ln.Node().Seq(); s != 2 {
		t.Fatalf("wrong seq %d after set, want 2", s)
	}

	// Create a new instance, it should reload the sequence number.
	// The number increases just after that because a new record is
	// created without the "x" entry.
	ln2 := NewLocalNode(db, ln.key)
	if s := ln2.Node().Seq(); s != 3 {
		t.Fatalf("wrong seq %d on new instance, want 3", s)
	}

	// Create a new instance with a different node key on the same database.
	// This should reset the sequence number.
	key, _ := crypto.GenerateKey()
	ln3 := NewLocalNode(db, key)
	if s := ln3.Node().Seq(); s != 1 {
		t.Fatalf("wrong seq %d on instance with changed key, want 1", s)
	}
}
