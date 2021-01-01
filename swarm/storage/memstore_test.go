// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package storage

import (
	"context"
	"testing"

	"github.com/fafereum/go-fafereum/swarm/log"
)

func newTestMemStore() *MemStore {
	storeparams := NewDefaultStoreParams()
	return NewMemStore(storeparams, nil)
}

func testMemStoreRandom(n int, t *testing.T) {
	m := newTestMemStore()
	defer m.Close()
	testStoreRandom(m, n, t)
}

func testMemStoreCorrect(n int, t *testing.T) {
	m := newTestMemStore()
	defer m.Close()
	testStoreCorrect(m, n, t)
}

func TestMemStoreRandom_1(t *testing.T) {
	testMemStoreRandom(1, t)
}

func TestMemStoreCorrect_1(t *testing.T) {
	testMemStoreCorrect(1, t)
}

func TestMemStoreRandom_1k(t *testing.T) {
	testMemStoreRandom(1000, t)
}

func TestMemStoreCorrect_1k(t *testing.T) {
	testMemStoreCorrect(100, t)
}

func TestMemStoreNotFound(t *testing.T) {
	m := newTestMemStore()
	defer m.Close()

	_, err := m.Get(context.TODO(), ZeroAddr)
	if err != ErrChunkNotFound {
		t.Errorf("Expected ErrChunkNotFound, got %v", err)
	}
}

func benchmarkMemStorePut(n int, b *testing.B) {
	m := newTestMemStore()
	defer m.Close()
	benchmarkStorePut(m, n, b)
}

func benchmarkMemStoreGet(n int, b *testing.B) {
	m := newTestMemStore()
	defer m.Close()
	benchmarkStoreGet(m, n, b)
}

func BenchmarkMemStorePut_500(b *testing.B) {
	benchmarkMemStorePut(500, b)
}

func BenchmarkMemStoreGet_500(b *testing.B) {
	benchmarkMemStoreGet(500, b)
}

func TestMemStoreAndLDBStore(t *testing.T) {
	ldb, cleanup := newLDBStore(t)
	ldb.setCapacity(4000)
	defer cleanup()

	cacheCap := 200
	memStore := NewMemStore(NewStoreParams(4000, 200, nil, nil), nil)

	tests := []struct {
		n         int   // number of chunks to push to memStore
		chunkSize int64 // size of chunk (by default in Swarm - 4096)
	}{
		{
			n:         1,
			chunkSize: 4096,
		},
		{
			n:         101,
			chunkSize: 4096,
		},
		{
			n:         501,
			chunkSize: 4096,
		},
		{
			n:         1100,
			chunkSize: 4096,
		},
	}

	for i, tt := range tests {
		log.Info("running test", "idx", i, "tt", tt)
		var chunks []Chunk

		for i := 0; i < tt.n; i++ {
			c := GenerateRandomChunk(tt.chunkSize)
			chunks = append(chunks, c)
		}

		for i := 0; i < tt.n; i++ {
			err := ldb.Put(context.TODO(), chunks[i])
			if err != nil {
				t.Fatal(err)
			}
			err = memStore.Put(context.TODO(), chunks[i])
			if err != nil {
				t.Fatal(err)
			}

			if got := memStore.cache.Len(); got > cacheCap {
				t.Fatalf("expected to get cache capacity less than %v, but got %v", cacheCap, got)
			}

		}

		for i := 0; i < tt.n; i++ {
			_, err := memStore.Get(context.TODO(), chunks[i].Address())
			if err != nil {
				if err == ErrChunkNotFound {
					_, err := ldb.Get(context.TODO(), chunks[i].Address())
					if err != nil {
						t.Fatalf("couldn't get chunk %v from ldb, got error: %v", i, err)
					}
				} else {
					t.Fatalf("got error from memstore: %v", err)
				}
			}
		}
	}
}
