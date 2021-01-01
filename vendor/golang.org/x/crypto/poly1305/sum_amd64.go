// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

// +build amd64,!gccgo,!appengine

package poly1305

// This function is implemented in sum_amd64.s
//go:noescape
func poly1305(out *[16]byte, m *byte, mlen uint64, key *[32]byte)

// Sum generates an authenticator for m using a one-time key and puts the
// 16-byte result into out. Authenticating two different messages with the same
// key allows an attacker to forge messages at will.
func Sum(out *[16]byte, m []byte, key *[32]byte) {
	var mPtr *byte
	if len(m) > 0 {
		mPtr = &m[0]
	}
	poly1305(out, mPtr, uint64(len(m)), key)
}
