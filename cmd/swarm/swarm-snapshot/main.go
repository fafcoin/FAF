// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify


package main

import (
	"os"

	"github.com/fafereum/go-fafereum/cmd/utils"
	"github.com/fafereum/go-fafereum/log"
	cli "gopkg.in/urfave/cli.v1"
)

var gitCommit string // Git SHA1 commit hash of the release (set via linker flags)

// default value for "create" command --nodes flag
const defaultNodes = 10

func main() {
	err := newApp().Run(os.Args)
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

// newApp construct a new instance of Swarm Snapshot Utility.
// Mfafod Run is called on it in the main function and in tests.
func newApp() (app *cli.App) {
	app = utils.NewApp(gitCommit, "Swarm Snapshot Utility")

	app.Name = "swarm-snapshot"
	app.Usage = ""

	// app flags (for all commands)
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  "verbosity",
			Value: 1,
			Usage: "verbosity level",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "create",
			Aliases: []string{"c"},
			Usage:   "create a swarm snapshot",
			Action:  create,
			// Flags only for "create" command.
			// Allow app flags to be specified after the
			// command argument.
			Flags: append(app.Flags,
				cli.IntFlag{
					Name:  "nodes",
					Value: defaultNodes,
					Usage: "number of nodes",
				},
				cli.StringFlag{
					Name:  "services",
					Value: "bzz",
					Usage: "comma separated list of services to boot the nodes with",
				},
			),
		},
	}

	return app
}
