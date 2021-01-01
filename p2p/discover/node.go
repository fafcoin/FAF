// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package discover

import (
	"crypto/ecdsa"
	"errors"
	"math/big"
	"net"
	"time"

	"github.com/fafereum/go-fafereum/common/math"
	"github.com/fafereum/go-fafereum/crypto"
	"github.com/fafereum/go-fafereum/crypto/secp256k1"
	"github.com/fafereum/go-fafereum/p2p/enode"
)

// node represents a host on the network.
// The fields of Node may not be modified.
type node struct {
	enode.Node
	addedAt        time.Time // time when the node was added to the table
	livenessChecks uint      // how often liveness was checked
}

type encPubkey [64]byte

func encodePubkey(key *ecdsa.PublicKey) encPubkey {
	var e encPubkey
	math.ReadBits(key.X, e[:len(e)/2])
	math.ReadBits(key.Y, e[len(e)/2:])
	return e
}

func decodePubkey(e encPubkey) (*ecdsa.PublicKey, error) {
	p := &ecdsa.PublicKey{Curve: crypto.S256(), X: new(big.Int), Y: new(big.Int)}
	half := len(e) / 2
	p.X.SetBytes(e[:half])
	p.Y.SetBytes(e[half:])
	if !p.Curve.IsOnCurve(p.X, p.Y) {
		return nil, errors.New("invalid secp256k1 curve point")
	}
	return p, nil
}

func (e encPubkey) id() enode.ID {
	return enode.ID(crypto.Keccak256Hash(e[:]))
}

// recoverNodeKey computes the public key used to sign the
// given hash from the signature.
func recoverNodeKey(hash, sig []byte) (key encPubkey, err error) {
	pubkey, err := secp256k1.RecoverPubkey(hash, sig)
	if err != nil {
		return key, err
	}
	copy(key[:], pubkey[1:])
	return key, nil
}

func wrapNode(n *enode.Node) *node {
	return &node{Node: *n}
}

func wrapNodes(ns []*enode.Node) []*node {
	result := make([]*node, len(ns))
	for i, n := range ns {
		result[i] = wrapNode(n)
	}
	return result
}

func unwrapNode(n *node) *enode.Node {
	return &n.Node
}

func unwrapNodes(ns []*node) []*enode.Node {
	result := make([]*enode.Node, len(ns))
	for i, n := range ns {
		result[i] = unwrapNode(n)
	}
	return result
}

func (n *node) addr() *net.UDPAddr {
	return &net.UDPAddr{IP: n.IP(), Port: n.UDP()}
}

func (n *node) String() string {
	return n.Node.String()
}
