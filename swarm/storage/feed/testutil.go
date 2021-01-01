// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package feed

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/fafereum/go-fafereum/p2p/enode"
	"github.com/fafereum/go-fafereum/swarm/storage"
)

const (
	testDbDirName = "feeds"
)

type TestHandler struct {
	*Handler
}

func (t *TestHandler) Close() {
	t.chunkStore.Close()
}

type mockNetFetcher struct{}

func (m *mockNetFetcher) Request(hopCount uint8) {
}
func (m *mockNetFetcher) Offer(source *enode.ID) {
}

func newFakeNetFetcher(context.Context, storage.Address, *sync.Map) storage.NetFetcher {
	return &mockNetFetcher{}
}

// NewTestHandler creates Handler object to be used for testing purposes.
func NewTestHandler(datadir string, params *HandlerParams) (*TestHandler, error) {
	path := filepath.Join(datadir, testDbDirName)
	fh := NewHandler(params)
	localstoreparams := storage.NewDefaultLocalStoreParams()
	localstoreparams.Init(path)
	localStore, err := storage.NewLocalStore(localstoreparams, nil)
	if err != nil {
		return nil, fmt.Errorf("localstore create fail, path %s: %v", path, err)
	}
	localStore.Validators = append(localStore.Validators, storage.NewContentAddressValidator(storage.MakeHashFunc(feedsHashAlgorithm)))
	localStore.Validators = append(localStore.Validators, fh)
	netStore, err := storage.NewNetStore(localStore, nil)
	if err != nil {
		return nil, err
	}
	netStore.NewNetFetcherFunc = newFakeNetFetcher
	fh.SetStore(netStore)
	return &TestHandler{fh}, nil
}
