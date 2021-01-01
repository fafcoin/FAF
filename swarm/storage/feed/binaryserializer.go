// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package feed

import "github.com/fafereum/go-fafereum/common/hexutil"

type binarySerializer interface {
	binaryPut(serializedData []byte) error
	binaryLength() int
	binaryGet(serializedData []byte) error
}

// Values interface represents a string key-value store
// useful for building query strings
type Values interface {
	Get(key string) string
	Set(key, value string)
}

type valueSerializer interface {
	FromValues(values Values) error
	AppendValues(values Values)
}

// Hex serializes the structure and converts it to a hex string
func Hex(bin binarySerializer) string {
	b := make([]byte, bin.binaryLength())
	bin.binaryPut(b)
	return hexutil.Encode(b)
}
