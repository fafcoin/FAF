// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify


package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/fafereum/go-fafereum/core"
)

// Tests the go-fafereum to Alfaf chainspec conversion for the Stureby testnet.
func TestAlfafSturebyConverter(t *testing.T) {
	blob, err := ioutil.ReadFile("testdata/stureby_gfaf.json")
	if err != nil {
		t.Fatalf("could not read file: %v", err)
	}
	var genesis core.Genesis
	if err := json.Unmarshal(blob, &genesis); err != nil {
		t.Fatalf("failed parsing genesis: %v", err)
	}
	spec, err := newAlfafGenesisSpec("stureby", &genesis)
	if err != nil {
		t.Fatalf("failed creating chainspec: %v", err)
	}

	expBlob, err := ioutil.ReadFile("testdata/stureby_alfaf.json")
	if err != nil {
		t.Fatalf("could not read file: %v", err)
	}
	expspec := &alfafGenesisSpec{}
	if err := json.Unmarshal(expBlob, expspec); err != nil {
		t.Fatalf("failed parsing genesis: %v", err)
	}
	if !reflect.DeepEqual(expspec, spec) {
		t.Errorf("chainspec mismatch")
		c := spew.ConfigState{
			DisablePointerAddresses: true,
			SortKeys:                true,
		}
		exp := strings.Split(c.Sdump(expspec), "\n")
		got := strings.Split(c.Sdump(spec), "\n")
		for i := 0; i < len(exp) && i < len(got); i++ {
			if exp[i] != got[i] {
				fmt.Printf("got: %v\nexp: %v\n", exp[i], got[i])
			}
		}
	}
}

// Tests the go-fafereum to Parity chainspec conversion for the Stureby testnet.
func TestParitySturebyConverter(t *testing.T) {
	blob, err := ioutil.ReadFile("testdata/stureby_gfaf.json")
	if err != nil {
		t.Fatalf("could not read file: %v", err)
	}
	var genesis core.Genesis
	if err := json.Unmarshal(blob, &genesis); err != nil {
		t.Fatalf("failed parsing genesis: %v", err)
	}
	spec, err := newParityChainSpec("Stureby", &genesis, []string{})
	if err != nil {
		t.Fatalf("failed creating chainspec: %v", err)
	}

	expBlob, err := ioutil.ReadFile("testdata/stureby_parity.json")
	if err != nil {
		t.Fatalf("could not read file: %v", err)
	}
	expspec := &parityChainSpec{}
	if err := json.Unmarshal(expBlob, expspec); err != nil {
		t.Fatalf("failed parsing genesis: %v", err)
	}
	expspec.Nodes = []string{}

	if !reflect.DeepEqual(expspec, spec) {
		t.Errorf("chainspec mismatch")
		c := spew.ConfigState{
			DisablePointerAddresses: true,
			SortKeys:                true,
		}
		exp := strings.Split(c.Sdump(expspec), "\n")
		got := strings.Split(c.Sdump(spec), "\n")
		for i := 0; i < len(exp) && i < len(got); i++ {
			if exp[i] != got[i] {
				fmt.Printf("got: %v\nexp: %v\n", exp[i], got[i])
			}
		}
	}
}
