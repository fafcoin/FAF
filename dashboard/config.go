// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package dashboard

import "time"

// DefaultConfig contains default settings for the dashboard.
var DefaultConfig = Config{
	Host:    "localhost",
	Port:    8080,
	Refresh: 5 * time.Second,
}

// Config contains the configuration parameters of the dashboard.
type Config struct {
	// Host is the host interface on which to start the dashboard server. If this
	// field is empty, no dashboard will be started.
	Host string `toml:",omitempty"`

	// Port is the TCP port number on which to start the dashboard server. The
	// default zero value is/ valid and will pick a port number randomly (useful
	// for ephemeral nodes).
	Port int `toml:",omitempty"`

	// Refresh is the refresh rate of the data updates, the chartEntry will be collected this often.
	Refresh time.Duration `toml:",omitempty"`
}
