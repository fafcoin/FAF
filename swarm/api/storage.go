// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package api

import (
	"context"
	"path"

	"github.com/fafereum/go-fafereum/swarm/storage"
)

type Response struct {
	MimeType string
	Status   int
	Size     int64
	// Content  []byte
	Content string
}

// implements a service
//
// DEPRECATED: Use the HTTP API instead
type Storage struct {
	api *API
}

func NewStorage(api *API) *Storage {
	return &Storage{api}
}

// Put uploads the content to the swarm with a simple manifest speficying
// its content type
//
// DEPRECATED: Use the HTTP API instead
func (s *Storage) Put(ctx context.Context, content string, contentType string, toEncrypt bool) (storage.Address, func(context.Context) error, error) {
	return s.api.Put(ctx, content, contentType, toEncrypt)
}

// Get retrieves the content from bzzpath and reads the response in full
// It returns the Response object, which serialises containing the
// response body as the value of the Content field
// NOTE: if error is non-nil, sResponse may still have partial content
// the actual size of which is given in len(resp.Content), while the expected
// size is resp.Size
//
// DEPRECATED: Use the HTTP API instead
func (s *Storage) Get(ctx context.Context, bzzpath string) (*Response, error) {
	uri, err := Parse(path.Join("bzz:/", bzzpath))
	if err != nil {
		return nil, err
	}
	addr, err := s.api.Resolve(ctx, uri.Addr)
	if err != nil {
		return nil, err
	}
	reader, mimeType, status, _, err := s.api.Get(ctx, nil, addr, uri.Path)
	if err != nil {
		return nil, err
	}
	quitC := make(chan bool)
	expsize, err := reader.Size(ctx, quitC)
	if err != nil {
		return nil, err
	}
	body := make([]byte, expsize)
	size, err := reader.Read(body)
	if int64(size) == expsize {
		err = nil
	}
	return &Response{mimeType, status, expsize, string(body[:size])}, err
}
