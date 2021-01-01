// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

// +build nopsshandshake

package pss

const (
	IsActiveHandshake = false
)

func NewHandshakeParams() interface{} {
	return nil
}
