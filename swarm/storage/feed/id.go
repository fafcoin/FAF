// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package feed

import (
	"fmt"
	"hash"
	"strconv"

	"github.com/fafereum/go-fafereum/common"
	"github.com/fafereum/go-fafereum/swarm/storage/feed/lookup"

	"github.com/fafereum/go-fafereum/swarm/storage"
)

// ID uniquely identifies an update on the network.
type ID struct {
	Feed         `json:"feed"`
	lookup.Epoch `json:"epoch"`
}

// ID layout:
// Feed feedLength bytes
// Epoch EpochLength
const idLength = feedLength + lookup.EpochLength

// Addr calculates the feed update chunk address corresponding to this ID
func (u *ID) Addr() (updateAddr storage.Address) {
	serializedData := make([]byte, idLength)
	var cursor int
	u.Feed.binaryPut(serializedData[cursor : cursor+feedLength])
	cursor += feedLength

	eid := u.Epoch.ID()
	copy(serializedData[cursor:cursor+lookup.EpochLength], eid[:])

	hasher := hashPool.Get().(hash.Hash)
	defer hashPool.Put(hasher)
	hasher.Reset()
	hasher.Write(serializedData)
	return hasher.Sum(nil)
}

// binaryPut serializes this instance into the provided slice
func (u *ID) binaryPut(serializedData []byte) error {
	if len(serializedData) != idLength {
		return NewErrorf(ErrInvalidValue, "Incorrect slice size to serialize ID. Expected %d, got %d", idLength, len(serializedData))
	}
	var cursor int
	if err := u.Feed.binaryPut(serializedData[cursor : cursor+feedLength]); err != nil {
		return err
	}
	cursor += feedLength

	epochBytes, err := u.Epoch.MarshalBinary()
	if err != nil {
		return err
	}
	copy(serializedData[cursor:cursor+lookup.EpochLength], epochBytes[:])
	cursor += lookup.EpochLength

	return nil
}

// binaryLength returns the expected size of this structure when serialized
func (u *ID) binaryLength() int {
	return idLength
}

// binaryGet restores the current instance from the information contained in the passed slice
func (u *ID) binaryGet(serializedData []byte) error {
	if len(serializedData) != idLength {
		return NewErrorf(ErrInvalidValue, "Incorrect slice size to read ID. Expected %d, got %d", idLength, len(serializedData))
	}

	var cursor int
	if err := u.Feed.binaryGet(serializedData[cursor : cursor+feedLength]); err != nil {
		return err
	}
	cursor += feedLength

	if err := u.Epoch.UnmarshalBinary(serializedData[cursor : cursor+lookup.EpochLength]); err != nil {
		return err
	}
	cursor += lookup.EpochLength

	return nil
}

// FromValues deserializes this instance from a string key-value store
// useful to parse query strings
func (u *ID) FromValues(values Values) error {
	level, _ := strconv.ParseUint(values.Get("level"), 10, 32)
	u.Epoch.Level = uint8(level)
	u.Epoch.Time, _ = strconv.ParseUint(values.Get("time"), 10, 64)

	if u.Feed.User == (common.Address{}) {
		return u.Feed.FromValues(values)
	}
	return nil
}

// AppendValues serializes this structure into the provided string key-value store
// useful to build query strings
func (u *ID) AppendValues(values Values) {
	values.Set("level", fmt.Sprintf("%d", u.Epoch.Level))
	values.Set("time", fmt.Sprintf("%d", u.Epoch.Time))
	u.Feed.AppendValues(values)
}