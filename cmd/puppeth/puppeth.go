// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify


// puppfaf is a command to assemble and maintain private networks.
package main

import (
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/fafereum/go-fafereum/log"
	"gopkg.in/urfave/cli.v1"
)

// main is just a boring entry point to set up the CLI app.
func main() {
	app := cli.NewApp()
	app.Name = "puppfaf"
	app.Usage = "assemble and maintain private fafereum networks"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "network",
			Usage: "name of the network to administer (no spaces or hyphens, please)",
		},
		cli.IntFlag{
			Name:  "loglevel",
			Value: 3,
			Usage: "log level to emit to the screen",
		},
	}
	app.Before = func(c *cli.Context) error {
		// Set up the logger to print everything and the random generator
		log.Root().Sfafandler(log.LvlFilterHandler(log.Lvl(c.Int("loglevel")), log.StreamHandler(os.Stdout, log.TerminalFormat(true))))
		rand.Seed(time.Now().UnixNano())

		return nil
	}
	app.Action = runWizard
	app.Run(os.Args)
}

// runWizard start the wizard and relinquish control to it.
func runWizard(c *cli.Context) error {
	network := c.String("network")
	if strings.Contains(network, " ") || strings.Contains(network, "-") || strings.ToLower(network) != network {
		log.Crit("No spaces, hyphens or capital letters allowed in network name")
	}
	makeWizard(c.String("network")).run()
	return nil
}
