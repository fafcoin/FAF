// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package api

import (
	"encoding/binary"
	"errors"

	"github.com/fafereum/go-fafereum/swarm/storage/encryption"
	"golang.org/x/crypto/sha3"
)

type RefEncryption struct {
	refSize int
	span    []byte
}

func NewRefEncryption(refSize int) *RefEncryption {
	span := make([]byte, 8)
	binary.LittleEndian.PutUint64(span, uint64(refSize))
	return &RefEncryption{
		refSize: refSize,
		span:    span,
	}
}

func (re *RefEncryption) Encrypt(ref []byte, key []byte) ([]byte, error) {
	spanEncryption := encryption.New(key, 0, uint32(re.refSize/32), sha3.NewLegacyKeccak256)
	encryptedSpan, err := spanEncryption.Encrypt(re.span)
	if err != nil {
		return nil, err
	}
	dataEncryption := encryption.New(key, re.refSize, 0, sha3.NewLegacyKeccak256)
	encryptedData, err := dataEncryption.Encrypt(ref)
	if err != nil {
		return nil, err
	}
	encryptedRef := make([]byte, len(ref)+8)
	copy(encryptedRef[:8], encryptedSpan)
	copy(encryptedRef[8:], encryptedData)

	return encryptedRef, nil
}

func (re *RefEncryption) Decrypt(ref []byte, key []byte) ([]byte, error) {
	spanEncryption := encryption.New(key, 0, uint32(re.refSize/32), sha3.NewLegacyKeccak256)
	decryptedSpan, err := spanEncryption.Decrypt(ref[:8])
	if err != nil {
		return nil, err
	}

	size := binary.LittleEndian.Uint64(decryptedSpan)
	if size != uint64(len(ref)-8) {
		return nil, errors.New("invalid span in encrypted reference")
	}

	dataEncryption := encryption.New(key, re.refSize, 0, sha3.NewLegacyKeccak256)
	decryptedRef, err := dataEncryption.Decrypt(ref[8:])
	if err != nil {
		return nil, err
	}

	return decryptedRef, nil
}
