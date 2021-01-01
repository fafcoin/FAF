// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package localstore

import (
	"testing"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
)

// TestModeSetAccess validates ModeSetAccess index values on the provided DB.
func TestModeSetAccess(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	chunk := generateRandomChunk()

	wantTimestamp := time.Now().UTC().UnixNano()
	defer setNow(func() (t int64) {
		return wantTimestamp
	})()

	err := db.NewSetter(ModeSetAccess).Set(chunk.Address())
	if err != nil {
		t.Fatal(err)
	}

	t.Run("pull index", newPullIndexTest(db, chunk, wantTimestamp, nil))

	t.Run("pull index count", newItemsCountTest(db.pullIndex, 1))

	t.Run("gc index", newGCIndexTest(db, chunk, wantTimestamp, wantTimestamp))

	t.Run("gc index count", newItemsCountTest(db.gcIndex, 1))

	t.Run("gc size", newIndexGCSizeTest(db))
}

// TestModeSetSync validates ModeSetSync index values on the provided DB.
func TestModeSetSync(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	chunk := generateRandomChunk()

	wantTimestamp := time.Now().UTC().UnixNano()
	defer setNow(func() (t int64) {
		return wantTimestamp
	})()

	err := db.NewPutter(ModePutUpload).Put(chunk)
	if err != nil {
		t.Fatal(err)
	}

	err = db.NewSetter(ModeSetSync).Set(chunk.Address())
	if err != nil {
		t.Fatal(err)
	}

	t.Run("retrieve indexes", newRetrieveIndexesTestWithAccess(db, chunk, wantTimestamp, wantTimestamp))

	t.Run("push index", newPushIndexTest(db, chunk, wantTimestamp, leveldb.ErrNotFound))

	t.Run("gc index", newGCIndexTest(db, chunk, wantTimestamp, wantTimestamp))

	t.Run("gc index count", newItemsCountTest(db.gcIndex, 1))

	t.Run("gc size", newIndexGCSizeTest(db))
}

// TestModeSetRemove validates ModeSetRemove index values on the provided DB.
func TestModeSetRemove(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	chunk := generateRandomChunk()

	err := db.NewPutter(ModePutUpload).Put(chunk)
	if err != nil {
		t.Fatal(err)
	}

	err = db.NewSetter(modeSetRemove).Set(chunk.Address())
	if err != nil {
		t.Fatal(err)
	}

	t.Run("retrieve indexes", func(t *testing.T) {
		wantErr := leveldb.ErrNotFound
		_, err := db.retrievalDataIndex.Get(addressToItem(chunk.Address()))
		if err != wantErr {
			t.Errorf("got error %v, want %v", err, wantErr)
		}
		t.Run("retrieve data index count", newItemsCountTest(db.retrievalDataIndex, 0))

		// access index should not be set
		_, err = db.retrievalAccessIndex.Get(addressToItem(chunk.Address()))
		if err != wantErr {
			t.Errorf("got error %v, want %v", err, wantErr)
		}
		t.Run("retrieve access index count", newItemsCountTest(db.retrievalAccessIndex, 0))
	})

	t.Run("pull index", newPullIndexTest(db, chunk, 0, leveldb.ErrNotFound))

	t.Run("pull index count", newItemsCountTest(db.pullIndex, 0))

	t.Run("gc index count", newItemsCountTest(db.gcIndex, 0))

	t.Run("gc size", newIndexGCSizeTest(db))

}
