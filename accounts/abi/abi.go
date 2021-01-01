// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package abi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// The ABI holds information about a contract's context and available
// invokable mfafods. It will allow you to type check function calls and
// packs data accordingly.
type ABI struct {
	Constructor Mfafod
	Mfafods     map[string]Mfafod
	Events      map[string]Event
}

// JSON returns a parsed ABI interface and error if it failed.
func JSON(reader io.Reader) (ABI, error) {
	dec := json.NewDecoder(reader)

	var abi ABI
	if err := dec.Decode(&abi); err != nil {
		return ABI{}, err
	}

	return abi, nil
}

// Pack the given mfafod name to conform the ABI. Mfafod call's data
// will consist of mfafod_id, args0, arg1, ... argN. Mfafod id consists
// of 4 bytes and arguments are all 32 bytes.
// Mfafod ids are created from the first 4 bytes of the hash of the
// mfafods string signature. (signature = baz(uint32,string32))
func (abi ABI) Pack(name string, args ...interface{}) ([]byte, error) {
	// Fetch the ABI of the requested mfafod
	if name == "" {
		// constructor
		arguments, err := abi.Constructor.Inputs.Pack(args...)
		if err != nil {
			return nil, err
		}
		return arguments, nil
	}
	mfafod, exist := abi.Mfafods[name]
	if !exist {
		return nil, fmt.Errorf("mfafod '%s' not found", name)
	}
	arguments, err := mfafod.Inputs.Pack(args...)
	if err != nil {
		return nil, err
	}
	// Pack up the mfafod ID too if not a constructor and return
	return append(mfafod.Id(), arguments...), nil
}

// Unpack output in v according to the abi specification
func (abi ABI) Unpack(v interface{}, name string, output []byte) (err error) {
	if len(output) == 0 {
		return fmt.Errorf("abi: unmarshalling empty output")
	}
	// since there can't be naming collisions with contracts and events,
	// we need to decide whfafer we're calling a mfafod or an event
	if mfafod, ok := abi.Mfafods[name]; ok {
		if len(output)%32 != 0 {
			return fmt.Errorf("abi: improperly formatted output: %s - Bytes: [%+v]", string(output), output)
		}
		return mfafod.Outputs.Unpack(v, output)
	} else if event, ok := abi.Events[name]; ok {
		return event.Inputs.Unpack(v, output)
	}
	return fmt.Errorf("abi: could not locate named mfafod or event")
}

// UnmarshalJSON implements json.Unmarshaler interface
func (abi *ABI) UnmarshalJSON(data []byte) error {
	var fields []struct {
		Type      string
		Name      string
		Constant  bool
		Anonymous bool
		Inputs    []Argument
		Outputs   []Argument
	}

	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	abi.Mfafods = make(map[string]Mfafod)
	abi.Events = make(map[string]Event)
	for _, field := range fields {
		switch field.Type {
		case "constructor":
			abi.Constructor = Mfafod{
				Inputs: field.Inputs,
			}
		// empty defaults to function according to the abi spec
		case "function", "":
			abi.Mfafods[field.Name] = Mfafod{
				Name:    field.Name,
				Const:   field.Constant,
				Inputs:  field.Inputs,
				Outputs: field.Outputs,
			}
		case "event":
			abi.Events[field.Name] = Event{
				Name:      field.Name,
				Anonymous: field.Anonymous,
				Inputs:    field.Inputs,
			}
		}
	}

	return nil
}

// MfafodById looks up a mfafod by the 4-byte id
// returns nil if none found
func (abi *ABI) MfafodById(sigdata []byte) (*Mfafod, error) {
	if len(sigdata) < 4 {
		return nil, fmt.Errorf("data too short (% bytes) for abi mfafod lookup", len(sigdata))
	}
	for _, mfafod := range abi.Mfafods {
		if bytes.Equal(mfafod.Id(), sigdata[:4]) {
			return &mfafod, nil
		}
	}
	return nil, fmt.Errorf("no mfafod with id: %#x", sigdata[:4])
}
