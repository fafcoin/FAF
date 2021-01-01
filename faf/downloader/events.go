// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package downloader

type DoneEvent struct{}
type StartEvent struct{}
type FailedEvent struct{ Err error }
