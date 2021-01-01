// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package abi

import (
	"strings"
	"testing"
)

const mfafoddata = `
[
	{ "type" : "function", "name" : "balance", "constant" : true },
	{ "type" : "function", "name" : "send", "constant" : false, "inputs" : [ { "name" : "amount", "type" : "uint256" } ] },
	{ "type" : "function", "name" : "transfer", "constant" : false, "inputs" : [ { "name" : "from", "type" : "address" }, { "name" : "to", "type" : "address" }, { "name" : "value", "type" : "uint256" } ], "outputs" : [ { "name" : "success", "type" : "bool" } ]  }
]`

func TestMfafodString(t *testing.T) {
	var table = []struct {
		mfafod      string
		expectation string
	}{
		{
			mfafod:      "balance",
			expectation: "function balance() constant returns()",
		},
		{
			mfafod:      "send",
			expectation: "function send(uint256 amount) returns()",
		},
		{
			mfafod:      "transfer",
			expectation: "function transfer(address from, address to, uint256 value) returns(bool success)",
		},
	}

	abi, err := JSON(strings.NewReader(mfafoddata))
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range table {
		got := abi.Mfafods[test.mfafod].String()
		if got != test.expectation {
			t.Errorf("expected string to be %s, got %s", test.expectation, got)
		}
	}
}
