// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

// +build js

package fafdb_test

import (
	"github.com/fafereum/go-fafereum/fafdb"
)

var _ fafdb.Database = &fafdb.LDBDatabase{}
