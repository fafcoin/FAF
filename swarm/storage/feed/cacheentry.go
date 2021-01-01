// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package feed

import (
	"bytes"
	"context"
	"time"

	"github.com/fafereum/go-fafereum/swarm/storage"
)

const (
	hasherCount            = 8
	feedsHashAlgorithm     = storage.SHA3Hash
	defaultRetrieveTimeout = 100 * time.Millisecond
)

// cacheEntry caches the last known update of a specific Swarm feed.
type cacheEntry struct {
	Update
	*bytes.Reader
	lastKey storage.Address
}

// implements storage.LazySectionReader
func (r *cacheEntry) Size(ctx context.Context, _ chan bool) (int64, error) {
	return int64(len(r.Update.data)), nil
}

//returns the feed's topic
func (r *cacheEntry) Topic() Topic {
	return r.Feed.Topic
}
