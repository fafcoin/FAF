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

var hashesCommand = cli.Command{
	Action:             hashes,
	CustomHelpTemplate: helpTemplate,
	Name:               "hashes",
	Usage:              "print all hashes of a file to STDOUT",
	ArgsUsage:          "<file>",
	Description:        "Prints all hashes of a file to STDOUT",
}

func hashes(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		utils.Fatalf("Usage: swarm hashes <file name>")
	}
	f, err := os.Open(args[0])
	if err != nil {
		utils.Fatalf("Error opening file " + args[1])
	}
	defer f.Close()

	fileStore := storage.NewFileStore(&storage.FakeChunkStore{}, storage.NewFileStoreParams())
	refs, err := fileStore.GetAllReferences(context.TODO(), f, false)
	if err != nil {
		utils.Fatalf("%v\n", err)
	} else {
		for _, r := range refs {
			//fmt.Println(r.String())
		}
	}
}
