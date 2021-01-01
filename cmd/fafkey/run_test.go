// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify


package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/docker/docker/pkg/reexec"
	"github.com/fafereum/go-fafereum/internal/cmdtest"
)

type testfafkey struct {
	*cmdtest.TestCmd
}

// spawns fafkey with the given command line args.
func runfafkey(t *testing.T, args ...string) *testfafkey {
	tt := new(testfafkey)
	tt.TestCmd = cmdtest.NewTestCmd(t, tt)
	tt.Run("fafkey-test", args...)
	return tt
}

func TestMain(m *testing.M) {
	// Run the app if we've been exec'd as "fafkey-test" in runfafkey.
	reexec.Register("fafkey-test", func() {
		if err := app.Run(os.Args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	})
	// check if we have been reexec'd
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}
