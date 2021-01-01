// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify


package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/fafereum/go-fafereum/core/asm"
	cli "gopkg.in/urfave/cli.v1"
)

var disasmCommand = cli.Command{
	Action:    disasmCmd,
	Name:      "disasm",
	Usage:     "disassembles evm binary",
	ArgsUsage: "<file>",
}

func disasmCmd(ctx *cli.Context) error {
	if len(ctx.Args().First()) == 0 {
		return errors.New("filename required")
	}

	fn := ctx.Args().First()
	in, err := ioutil.ReadFile(fn)
	if err != nil {
		return err
	}

	code := strings.TrimSpace(string(in))
	fmt.Printf("%v\n", code)
	return asm.PrintDisassembled(code)
}
