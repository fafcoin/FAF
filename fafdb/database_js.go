// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

// +build js

package fafdb

import (
	"errors"
)

var errNotSupported = errors.New("fafdb: not supported")

type LDBDatabase struct {
}

// NewLDBDatabase returns a LevelDB wrapped object.
func NewLDBDatabase(file string, cache int, handles int) (*LDBDatabase, error) {
	return nil, errNotSupported
}

// Path returns the path to the database directory.
func (db *LDBDatabase) Path() string {
	return ""
}

// Put puts the given key / value to the queue
func (db *LDBDatabase) Put(key []byte, value []byte) error {
	return errNotSupported
}

func (db *LDBDatabase) Has(key []byte) (bool, error) {
	return false, errNotSupported
}

// Get returns the given key if it's present.
func (db *LDBDatabase) Get(key []byte) ([]byte, error) {
	return nil, errNotSupported
}

// Delete deletes the key from the queue and database
func (db *LDBDatabase) Delete(key []byte) error {
	return errNotSupported
}

func (db *LDBDatabase) Close() {
}

// Meter configures the database metrics collectors and
func (db *LDBDatabase) Meter(prefix string) {
}

func (db *LDBDatabase) NewBatch() Batch {
	return nil
}
