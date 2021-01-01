// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package feed

import (
	"fmt"
)

const (
	ErrInit = iota
	ErrNotFound
	ErrIO
	ErrUnauthorized
	ErrInvalidValue
	ErrDataOverflow
	ErrNothingToReturn
	ErrCorruptData
	ErrInvalidSignature
	ErrNotSynced
	ErrPeriodDepth
	ErrCnt
)

// Error is a the typed error object used for Swarm feeds
type Error struct {
	code int
	err  string
}

// Error implements the error interface
func (e *Error) Error() string {
	return e.err
}

// Code returns the error code
// Error codes are enumerated in the error.go file within the feeds package
func (e *Error) Code() int {
	return e.code
}

// NewError creates a new Swarm feeds Error object with the specified code and custom error message
func NewError(code int, s string) error {
	if code < 0 || code >= ErrCnt {
		panic("no such error code!")
	}
	r := &Error{
		err: s,
	}
	switch code {
	case ErrNotFound, ErrIO, ErrUnauthorized, ErrInvalidValue, ErrDataOverflow, ErrNothingToReturn, ErrInvalidSignature, ErrNotSynced, ErrPeriodDepth, ErrCorruptData:
		r.code = code
	}
	return r
}

// NewErrorf is a convenience version of NewError that incorporates printf-style formatting
func NewErrorf(code int, format string, args ...interface{}) error {
	return NewError(code, fmt.Sprintf(format, args...))
}
