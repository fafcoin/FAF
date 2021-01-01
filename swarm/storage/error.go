// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package storage

import (
	"errors"
)

const (
	ErrInit = iota
	ErrNotFound
	ErrUnauthorized
	ErrInvalidValue
	ErrDataOverflow
	ErrNothingToReturn
	ErrInvalidSignature
	ErrNotSynced
)

var (
	ErrChunkNotFound = errors.New("chunk not found")
	ErrChunkInvalid  = errors.New("invalid chunk")
)
