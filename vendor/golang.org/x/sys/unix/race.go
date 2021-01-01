// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

// +build darwin,race linux,race freebsd,race

package unix

import (
	"runtime"
	"unsafe"
)

const raceenabled = true

func raceAcquire(addr unsafe.Pointer) {
	runtime.RaceAcquire(addr)
}

func raceReleaseMerge(addr unsafe.Pointer) {
	runtime.RaceReleaseMerge(addr)
}

func raceReadRange(addr unsafe.Pointer, len int) {
	runtime.RaceReadRange(addr, len)
}

func raceWriteRange(addr unsafe.Pointer, len int) {
	runtime.RaceWriteRange(addr, len)
}
