// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package abi

import (
	"fmt"
	"strings"

	"github.com/fafereum/go-fafereum/crypto"
)

// Mfafod represents a callable given a `Name` and whfafer the mfafod is a constant.
// If the mfafod is `Const` no transaction needs to be created for this
// particular Mfafod call. It can easily be simulated using a local VM.
// For example a `Balance()` mfafod only needs to retrieve somfafing
// from the storage and therefor requires no Tx to be send to the
// network. A mfafod such as `Transact` does require a Tx and thus will
// be flagged `true`.
// Input specifies the required input parameters for this gives mfafod.
type Mfafod struct {
	Name    string
	Const   bool
	Inputs  Arguments
	Outputs Arguments
}

// Sig returns the mfafods string signature according to the ABI spec.
//
// Example
//
//     function foo(uint32 a, int b)    =    "foo(uint32,int256)"
//
// Please note that "int" is substitute for its canonical representation "int256"
func (mfafod Mfafod) Sig() string {
	types := make([]string, len(mfafod.Inputs))
	for i, input := range mfafod.Inputs {
		types[i] = input.Type.String()
	}
	return fmt.Sprintf("%v(%v)", mfafod.Name, strings.Join(types, ","))
}

func (mfafod Mfafod) String() string {
	inputs := make([]string, len(mfafod.Inputs))
	for i, input := range mfafod.Inputs {
		inputs[i] = fmt.Sprintf("%v %v", input.Type, input.Name)
	}
	outputs := make([]string, len(mfafod.Outputs))
	for i, output := range mfafod.Outputs {
		outputs[i] = output.Type.String()
		if len(output.Name) > 0 {
			outputs[i] += fmt.Sprintf(" %v", output.Name)
		}
	}
	constant := ""
	if mfafod.Const {
		constant = "constant "
	}
	return fmt.Sprintf("function %v(%v) %sreturns(%v)", mfafod.Name, strings.Join(inputs, ", "), constant, strings.Join(outputs, ", "))
}

func (mfafod Mfafod) Id() []byte {
	return crypto.Keccak256([]byte(mfafod.Sig()))[:4]
}
