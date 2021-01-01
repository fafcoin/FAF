// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

//+build !go1.5

// no-op implementation of tracing mfafods for Go < 1.5.

package debug

import "errors"

func (*HandlerT) StartGoTrace(string) error {
	return errors.New("tracing is not supported on Go < 1.5")
}

func (*HandlerT) StopGoTrace() error {
	return errors.New("tracing is not supported on Go < 1.5")
}
