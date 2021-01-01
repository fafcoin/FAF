// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify


package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/fafereum/go-fafereum/cmd/utils"
	gfafmetrics "github.com/fafereum/go-fafereum/metrics"
	"github.com/fafereum/go-fafereum/metrics/influxdb"
	swarmmetrics "github.com/fafereum/go-fafereum/swarm/metrics"
	"github.com/fafereum/go-fafereum/swarm/tracing"

	"github.com/fafereum/go-fafereum/log"

	cli "gopkg.in/urfave/cli.v1"
)

var (
	gitCommit string // Git SHA1 commit hash of the release (set via linker flags)
)

var (
	allhosts     string
	hosts        []string
	filesize     int
	syncDelay    int
	httpPort     int
	wsPort       int
	verbosity    int
	timeout      int
	single       bool
	trackTimeout int
)

func main() {

	app := cli.NewApp()
	app.Name = "smoke-test"
	app.Usage = ""

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "hosts",
			Value:       "",
			Usage:       "comma-separated list of swarm hosts",
			Destination: &allhosts,
		},
		cli.IntFlag{
			Name:        "http-port",
			Value:       80,
			Usage:       "http port",
			Destination: &httpPort,
		},
		cli.IntFlag{
			Name:        "ws-port",
			Value:       8546,
			Usage:       "ws port",
			Destination: &wsPort,
		},
		cli.IntFlag{
			Name:        "filesize",
			Value:       1024,
			Usage:       "file size for generated random file in KB",
			Destination: &filesize,
		},
		cli.IntFlag{
			Name:        "sync-delay",
			Value:       5,
			Usage:       "duration of delay in seconds to wait for content to be synced",
			Destination: &syncDelay,
		},
		cli.IntFlag{
			Name:        "verbosity",
			Value:       1,
			Usage:       "verbosity",
			Destination: &verbosity,
		},
		cli.IntFlag{
			Name:        "timeout",
			Value:       120,
			Usage:       "timeout in seconds after which kill the process",
			Destination: &timeout,
		},
		cli.BoolFlag{
			Name:        "single",
			Usage:       "whfafer to fetch content from a single node or from all nodes",
			Destination: &single,
		},
		cli.IntFlag{
			Name:        "track-timeout",
			Value:       5,
			Usage:       "timeout in seconds to wait for GetAllReferences to return",
			Destination: &trackTimeout,
		},
	}

	app.Flags = append(app.Flags, []cli.Flag{
		utils.MetricsEnabledFlag,
		swarmmetrics.MetricsInfluxDBEndpointFlag,
		swarmmetrics.MetricsInfluxDBDatabaseFlag,
		swarmmetrics.MetricsInfluxDBUsernameFlag,
		swarmmetrics.MetricsInfluxDBPasswordFlag,
		swarmmetrics.MetricsInfluxDBTagsFlag,
	}...)

	app.Flags = append(app.Flags, tracing.Flags...)

	app.Commands = []cli.Command{
		{
			Name:    "upload_and_sync",
			Aliases: []string{"c"},
			Usage:   "upload and sync",
			Action:  wrapCliCommand("upload-and-sync", uploadAndSyncCmd),
		},
		{
			Name:    "feed_sync",
			Aliases: []string{"f"},
			Usage:   "feed update generate, upload and sync",
			Action:  wrapCliCommand("feed-and-sync", feedUploadAndSyncCmd),
		},
		{
			Name:    "upload_speed",
			Aliases: []string{"u"},
			Usage:   "measure upload speed",
			Action:  wrapCliCommand("upload-speed", uploadSpeedCmd),
		},
		{
			Name:    "sliding_window",
			Aliases: []string{"s"},
			Usage:   "measure network aggregate capacity",
			Action:  wrapCliCommand("sliding-window", slidingWindowCmd),
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))

	app.Before = func(ctx *cli.Context) error {
		tracing.Setup(ctx)
		return nil
	}

	app.After = func(ctx *cli.Context) error {
		return emitMetrics(ctx)
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Error(err.Error())

		os.Exit(1)
	}
}

func emitMetrics(ctx *cli.Context) error {
	if gfafmetrics.Enabled {
		var (
			endpoint = ctx.GlobalString(swarmmetrics.MetricsInfluxDBEndpointFlag.Name)
			database = ctx.GlobalString(swarmmetrics.MetricsInfluxDBDatabaseFlag.Name)
			username = ctx.GlobalString(swarmmetrics.MetricsInfluxDBUsernameFlag.Name)
			password = ctx.GlobalString(swarmmetrics.MetricsInfluxDBPasswordFlag.Name)
			tags     = ctx.GlobalString(swarmmetrics.MetricsInfluxDBTagsFlag.Name)
		)

		tagsMap := utils.SplitTagsFlag(tags)
		tagsMap["version"] = gitCommit
		tagsMap["filesize"] = fmt.Sprintf("%v", filesize)

		return influxdb.InfluxDBWithTagsOnce(gfafmetrics.DefaultRegistry, endpoint, database, username, password, "swarm-smoke.", tagsMap)
	}

	return nil
}
