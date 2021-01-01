// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package whisperv6

// Config represents the configuration state of a whisper node.
type Config struct {
	MaxMessageSize                        uint32  `toml:",omitempty"`
	MinimumAcceptedPOW                    float64 `toml:",omitempty"`
	RestrictConnectionBetweenLightClients bool    `toml:",omitempty"`
}

// DefaultConfig represents (shocker!) the default configuration.
var DefaultConfig = Config{
	MaxMessageSize:                        DefaultMaxMessageSize,
	MinimumAcceptedPOW:                    DefaultMinimumPoW,
	RestrictConnectionBetweenLightClients: true,
}
