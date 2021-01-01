// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify


// Command bzzhash computes a swarm tree hash.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/fafereum/go-fafereum/cmd/utils"
	"github.com/fafereum/go-fafereum/swarm/storage"
	"gopkg.in/urfave/cli.v1"
)

var hashCommand = cli.Command{
	Action:             hash,
	CustomHelpTemplate: helpTemplate,
	Name:               "hash",
	Usage:              "print the swarm hash of a file or directory",
	ArgsUsage:          "<file>",
	Description:        "Prints the swarm hash of file or directory",
}

func hash(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		utils.Fatalf("Usage: swarm hash <file name>")
	}
	f, err := os.Open(args[0])
	if err != nil {
		utils.Fatalf("Error opening file " + args[1])
	}
	defer f.Close()

	stat, _ := f.Stat()
	fileStore := storage.NewFileStore(&storage.FakeChunkStore{}, storage.NewFileStoreParams())
	addr, _, err := fileStore.Store(context.TODO(), f, stat.Size(), false)
	if err != nil {
		utils.Fatalf("%v\n", err)
	} else {
		fmt.Printf("%v\n", addr)
	}
}
