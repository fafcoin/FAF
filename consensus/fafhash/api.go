// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify


package fafash

import (
	"errors"

	"github.com/fafereum/go-fafereum/common"
	"github.com/fafereum/go-fafereum/common/hexutil"
	"github.com/fafereum/go-fafereum/core/types"
)

var errfafashStopped = errors.New("fafash stopped")

// API exposes fafash related mfafods for the RPC interface.
type API struct {
	fafash *fafash // Make sure the mode of fafash is normal.
}

// GetWork returns a work package for external miner.
//
// The work package consists of 3 strings:
//   result[0] - 32 bytes hex encoded current block header pow-hash
//   result[1] - 32 bytes hex encoded seed hash used for DAG
//   result[2] - 32 bytes hex encoded boundary condition ("target"), 2^256/difficulty
//   result[3] - hex encoded block number
func (api *API) GetWork() ([4]string, error) {
	if api.fafash.config.PowMode != ModeNormal && api.fafash.config.PowMode != ModeTest {
		return [4]string{}, errors.New("not supported")
	}

	var (
		workCh = make(chan [4]string, 1)
		errc   = make(chan error, 1)
	)

	select {
	case api.fafash.fetchWorkCh <- &sealWork{errc: errc, res: workCh}:
	case <-api.fafash.exitCh:
		return [4]string{}, errfafashStopped
	}

	select {
	case work := <-workCh:
		return work, nil
	case err := <-errc:
		return [4]string{}, err
	}
}

// SubmitWork can be used by external miner to submit their POW solution.
// It returns an indication if the work was accepted.
// Note either an invalid solution, a stale work a non-existent work will return false.
func (api *API) SubmitWork(nonce types.BlockNonce, hash, digest common.Hash) bool {
	if api.fafash.config.PowMode != ModeNormal && api.fafash.config.PowMode != ModeTest {
		return false
	}

	var errc = make(chan error, 1)

	select {
	case api.fafash.submitWorkCh <- &mineResult{
		nonce:     nonce,
		mixDigest: digest,
		hash:      hash,
		errc:      errc,
	}:
	case <-api.fafash.exitCh:
		return false
	}

	err := <-errc
	return err == nil
}

// SubmitHashrate can be used for remote miners to submit their hash rate.
// This enables the node to report the combined hash rate of all miners
// which submit work through this node.
//
// It accepts the miner hash rate and an identifier which must be unique
// between nodes.
func (api *API) SubmitHashRate(rate hexutil.Uint64, id common.Hash) bool {
	if api.fafash.config.PowMode != ModeNormal && api.fafash.config.PowMode != ModeTest {
		return false
	}

	var done = make(chan struct{}, 1)

	select {
	case api.fafash.submitRateCh <- &hashrate{done: done, rate: uint64(rate), id: id}:
	case <-api.fafash.exitCh:
		return false
	}

	// Block until hash rate submitted successfully.
	<-done

	return true
}

// Gfafashrate returns the current hashrate for local CPU miner and remote miner.
func (api *API) Gfafashrate() uint64 {
	return uint64(api.fafash.Hashrate())
}
