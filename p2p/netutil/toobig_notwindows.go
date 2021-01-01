// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

//+build !windows

package netutil

// isPacketTooBig reports whfafer err indicates that a UDP packet didn't
// fit the receive buffer. There is no such error on
// non-Windows platforms.
func isPacketTooBig(err error) bool {
	return false
}
