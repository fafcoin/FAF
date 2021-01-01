// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package storage

// this is a clone of an earlier state of the fafereum fafdb/database
// no need for queueing/caching

import (
	"github.com/fafereum/go-fafereum/metrics"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

const openFileLimit = 128

type LDBDatabase struct {
	db *leveldb.DB
}

func NewLDBDatabase(file string) (*LDBDatabase, error) {
	// Open the db
	db, err := leveldb.OpenFile(file, &opt.Options{OpenFilesCacheCapacity: openFileLimit})
	if err != nil {
		return nil, err
	}

	database := &LDBDatabase{db: db}

	return database, nil
}

func (db *LDBDatabase) Put(key []byte, value []byte) error {
	metrics.GetOrRegisterCounter("ldbdatabase.put", nil).Inc(1)

	return db.db.Put(key, value, nil)
}

func (db *LDBDatabase) Get(key []byte) ([]byte, error) {
	metrics.GetOrRegisterCounter("ldbdatabase.get", nil).Inc(1)

	dat, err := db.db.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return dat, nil
}

func (db *LDBDatabase) Delete(key []byte) error {
	return db.db.Delete(key, nil)
}

func (db *LDBDatabase) NewIterator() iterator.Iterator {
	metrics.GetOrRegisterCounter("ldbdatabase.newiterator", nil).Inc(1)

	return db.db.NewIterator(nil, nil)
}

func (db *LDBDatabase) Write(batch *leveldb.Batch) error {
	metrics.GetOrRegisterCounter("ldbdatabase.write", nil).Inc(1)

	return db.db.Write(batch, nil)
}

func (db *LDBDatabase) Close() {
	// Close the leveldb database
	db.db.Close()
}
