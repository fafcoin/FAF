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

func init() {
	reexec.Register("swarm-snapshot", func() {
		if err := newApp().Run(os.Args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	})
}

func runSnapshot(t *testing.T, args ...string) *cmdtest.TestCmd {
	tt := cmdtest.NewTestCmd(t, nil)
	tt.Run("swarm-snapshot", args...)
	return tt
}

func TestMain(m *testing.M) {
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}
