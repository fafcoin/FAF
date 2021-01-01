// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

// Contains initialization code for the mbile library.

package gfaf

import (
	"os"
	"runtime"

	"github.com/fafereum/go-fafereum/log"
)

func init() {
	// Initialize the logger
	log.Root().Sfafandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(false))))

	// Initialize the goroutine count
	runtime.GOMAXPROCS(runtime.NumCPU())
}
