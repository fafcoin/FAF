// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package tests

import (
	"fmt"
	"math/big"

	"github.com/fafereum/go-fafereum/common"
	"github.com/fafereum/go-fafereum/common/math"
	"github.com/fafereum/go-fafereum/consensus/fafash"
	"github.com/fafereum/go-fafereum/core/types"
	"github.com/fafereum/go-fafereum/params"
)

//go:generate gencodec -type DifficultyTest -field-override difficultyTestMarshaling -out gen_difficultytest.go

type DifficultyTest struct {
	ParentTimestamp    uint64      `json:"parentTimestamp"`
	ParentDifficulty   *big.Int    `json:"parentDifficulty"`
	UncleHash          common.Hash `json:"parentUncles"`
	CurrentTimestamp   uint64      `json:"currentTimestamp"`
	CurrentBlockNumber uint64      `json:"currentBlockNumber"`
	CurrentDifficulty  *big.Int    `json:"currentDifficulty"`
}

type difficultyTestMarshaling struct {
	ParentTimestamp    math.HexOrDecimal64
	ParentDifficulty   *math.HexOrDecimal256
	CurrentTimestamp   math.HexOrDecimal64
	CurrentDifficulty  *math.HexOrDecimal256
	UncleHash          common.Hash
	CurrentBlockNumber math.HexOrDecimal64
}

func (test *DifficultyTest) Run(config *params.ChainConfig) error {
	parentNumber := big.NewInt(int64(test.CurrentBlockNumber - 1))
	parent := &types.Header{
		Difficulty: test.ParentDifficulty,
		Time:       test.ParentTimestamp,
		Number:     parentNumber,
		UncleHash:  test.UncleHash,
	}

	actual := fafash.CalcDifficulty(config, test.CurrentTimestamp, parent)
	exp := test.CurrentDifficulty

	if actual.Cmp(exp) != 0 {
		return fmt.Errorf("parent[time %v diff %v unclehash:%x] child[time %v number %v] diff %v != expected %v",
			test.ParentTimestamp, test.ParentDifficulty, test.UncleHash,
			test.CurrentTimestamp, test.CurrentBlockNumber, actual, exp)
	}
	return nil

}
