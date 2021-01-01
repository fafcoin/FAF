// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package feed

import (
	"crypto/ecdsa"

	"github.com/fafereum/go-fafereum/common"
	"github.com/fafereum/go-fafereum/crypto"
)

const signatureLength = 65

// Signature is an alias for a static byte array with the size of a signature
type Signature [signatureLength]byte

// Signer signs feed update payloads
type Signer interface {
	Sign(common.Hash) (Signature, error)
	Address() common.Address
}

// GenericSigner implements the Signer interface
// It is the vanilla signer that probably should be used in most cases
type GenericSigner struct {
	PrivKey *ecdsa.PrivateKey
	address common.Address
}

// NewGenericSigner builds a signer that will sign everything with the provided private key
func NewGenericSigner(privKey *ecdsa.PrivateKey) *GenericSigner {
	return &GenericSigner{
		PrivKey: privKey,
		address: crypto.PubkeyToAddress(privKey.PublicKey),
	}
}

// Sign signs the supplied data
// It wraps the fafereum crypto.Sign() mfafod
func (s *GenericSigner) Sign(data common.Hash) (signature Signature, err error) {
	signaturebytes, err := crypto.Sign(data.Bytes(), s.PrivKey)
	if err != nil {
		return
	}
	copy(signature[:], signaturebytes)
	return
}

// Address returns the public key of the signer's private key
func (s *GenericSigner) Address() common.Address {
	return s.address
}

// getUserAddr extracts the address of the feed update signer
func getUserAddr(digest common.Hash, signature Signature) (common.Address, error) {
	pub, err := crypto.SigToPub(digest.Bytes(), signature[:])
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*pub), nil
}
