// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package feed

import (
	"encoding/json"
	"time"
)

// TimestampProvider sets the time source of the feeds package
var TimestampProvider timestampProvider = NewDefaultTimestampProvider()

// Timestamp encodes a point in time as a Unix epoch
type Timestamp struct {
	Time uint64 `json:"time"` // Unix epoch timestamp, in seconds
}

// timestampProvider interface describes a source of timestamp information
type timestampProvider interface {
	Now() Timestamp // returns the current timestamp information
}

// UnmarshalJSON implements the json.Unmarshaller interface
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &t.Time)
}

// MarshalJSON implements the json.Marshaller interface
func (t *Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Time)
}

// DefaultTimestampProvider is a TimestampProvider that uses system time
// as time source
type DefaultTimestampProvider struct {
}

// NewDefaultTimestampProvider creates a system clock based timestamp provider
func NewDefaultTimestampProvider() *DefaultTimestampProvider {
	return &DefaultTimestampProvider{}
}

// Now returns the current time according to this provider
func (dtp *DefaultTimestampProvider) Now() Timestamp {
	return Timestamp{
		Time: uint64(time.Now().Unix()),
	}
}
