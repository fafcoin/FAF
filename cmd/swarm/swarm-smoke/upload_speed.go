// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify


package main

import (
	"bytes"
	"fmt"
	"time"

	"github.com/fafereum/go-fafereum/log"
	"github.com/fafereum/go-fafereum/metrics"
	"github.com/fafereum/go-fafereum/swarm/testutil"

	cli "gopkg.in/urfave/cli.v1"
)

func uploadSpeedCmd(ctx *cli.Context, tuid string) error {
	log.Info("uploading to "+hosts[0], "tuid", tuid, "seed", seed)
	randomBytes := testutil.RandomBytes(seed, filesize*1000)

	errc := make(chan error)

	go func() {
		errc <- uploadSpeed(ctx, tuid, randomBytes)
	}()

	select {
	case err := <-errc:
		if err != nil {
			metrics.GetOrRegisterCounter(fmt.Sprintf("%s.fail", commandName), nil).Inc(1)
		}
		return err
	case <-time.After(time.Duration(timeout) * time.Second):
		metrics.GetOrRegisterCounter(fmt.Sprintf("%s.timeout", commandName), nil).Inc(1)

		// trigger debug functionality on randomBytes

		return fmt.Errorf("timeout after %v sec", timeout)
	}
}

func uploadSpeed(c *cli.Context, tuid string, data []byte) error {
	t1 := time.Now()
	hash, err := upload(data, hosts[0])
	if err != nil {
		log.Error(err.Error())
		return err
	}
	metrics.GetOrRegisterCounter("upload-speed.upload-time", nil).Inc(int64(time.Since(t1)))

	fhash, err := digest(bytes.NewReader(data))
	if err != nil {
		log.Error(err.Error())
		return err
	}

	log.Info("uploaded successfully", "hash", hash, "digest", fmt.Sprintf("%x", fhash))
	return nil
}
