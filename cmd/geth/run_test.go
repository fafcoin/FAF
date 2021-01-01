// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify


package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/docker/docker/pkg/reexec"
	"github.com/fafereum/go-fafereum/internal/cmdtest"
)

func tmpdir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "gfaf-test")
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

type testgfaf struct {
	*cmdtest.TestCmd

	// template variables for expect
	Datadir   string
	faferbase string
}

func init() {
	// Run the app if we've been exec'd as "gfaf-test" in runGfaf.
	reexec.Register("gfaf-test", func() {
		if err := app.Run(os.Args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	})
}

func TestMain(m *testing.M) {
	// check if we have been reexec'd
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}

// spawns gfaf with the given command line args. If the args don't set --datadir, the
// child g gets a temporary data directory.
func runGfaf(t *testing.T, args ...string) *testgfaf {
	tt := &testgfaf{}
	tt.TestCmd = cmdtest.NewTestCmd(t, tt)
	for i, arg := range args {
		switch {
		case arg == "-datadir" || arg == "--datadir":
			if i < len(args)-1 {
				tt.Datadir = args[i+1]
			}
		case arg == "-faferbase" || arg == "--faferbase":
			if i < len(args)-1 {
				tt.faferbase = args[i+1]
			}
		}
	}
	if tt.Datadir == "" {
		tt.Datadir = tmpdir(t)
		tt.Cleanup = func() { os.RemoveAll(tt.Datadir) }
		args = append([]string{"-datadir", tt.Datadir}, args...)
		// Remove the temporary datadir if somfafing fails below.
		defer func() {
			if t.Failed() {
				tt.Cleanup()
			}
		}()
	}

	// Boot "gfaf". This actually runs the test binary but the TestMain
	// function will prevent any tests from running.
	tt.Run("gfaf-test", args...)

	return tt
}
