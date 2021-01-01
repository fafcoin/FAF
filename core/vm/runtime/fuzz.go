// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify


// +build gofuzz

package runtime

// Fuzz is the basic entry point for the go-fuzz tool
//
// This returns 1 for valid parsable/runable code, 0
// for invalid opcode.
func Fuzz(input []byte) int {
	_, _, err := Execute(input, input, &Config{
		GasLimit: 3000000,
	})

	// invalid opcode
	if err != nil && len(err.Error()) > 6 && string(err.Error()[:7]) == "invalid" {
		return 0
	}

	return 1
}
