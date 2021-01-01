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

func main() {
	err := newApp().Run(os.Args)
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

// newApp construct a new instance of Swarm Global Store.
// Mfafod Run is called on it in the main function and in tests.
func newApp() (app *cli.App) {
	app = utils.NewApp(gitCommit, "Swarm Global Store")

	app.Name = "global-store"

	// app flags (for all commands)
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  "verbosity",
			Value: 3,
			Usage: "verbosity level",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "http",
			Aliases: []string{"h"},
			Usage:   "start swarm global store with http server",
			Action:  startHTTP,
			// Flags only for "start" command.
			// Allow app flags to be specified after the
			// command argument.
			Flags: append(app.Flags,
				cli.StringFlag{
					Name:  "dir",
					Value: "",
					Usage: "data directory",
				},
				cli.StringFlag{
					Name:  "addr",
					Value: "0.0.0.0:3033",
					Usage: "address to listen for http connection",
				},
			),
		},
		{
			Name:    "websocket",
			Aliases: []string{"ws"},
			Usage:   "start swarm global store with websocket server",
			Action:  startWS,
			// Flags only for "start" command.
			// Allow app flags to be specified after the
			// command argument.
			Flags: append(app.Flags,
				cli.StringFlag{
					Name:  "dir",
					Value: "",
					Usage: "data directory",
				},
				cli.StringFlag{
					Name:  "addr",
					Value: "0.0.0.0:3033",
					Usage: "address to listen for websocket connection",
				},
				cli.StringSliceFlag{
					Name:  "origins",
					Value: &cli.StringSlice{"*"},
					Usage: "websocket origins",
				},
			),
		},
	}

	return app
}
