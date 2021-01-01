// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package dashboard

import (
	"encoding/json"
	"time"
)

type Message struct {
	General *GeneralMessage `json:"general,omitempty"`
	Home    *HomeMessage    `json:"home,omitempty"`
	Chain   *ChainMessage   `json:"chain,omitempty"`
	TxPool  *TxPoolMessage  `json:"txpool,omitempty"`
	Network *NetworkMessage `json:"network,omitempty"`
	System  *SystemMessage  `json:"system,omitempty"`
	Logs    *LogsMessage    `json:"logs,omitempty"`
}

type ChartEntries []*ChartEntry

type ChartEntry struct {
	Time  time.Time `json:"time,omitempty"`
	Value float64   `json:"value,omitempty"`
}

type GeneralMessage struct {
	Version string `json:"version,omitempty"`
	Commit  string `json:"commit,omitempty"`
}

type HomeMessage struct {
	/* TODO (kurkomisi) */
}

type ChainMessage struct {
	/* TODO (kurkomisi) */
}

type TxPoolMessage struct {
	/* TODO (kurkomisi) */
}

type NetworkMessage struct {
	/* TODO (kurkomisi) */
}

type SystemMessage struct {
	ActiveMemory   ChartEntries `json:"activeMemory,omitempty"`
	VirtualMemory  ChartEntries `json:"virtualMemory,omitempty"`
	NetworkIngress ChartEntries `json:"networkIngress,omitempty"`
	NetworkEgress  ChartEntries `json:"networkEgress,omitempty"`
	ProcessCPU     ChartEntries `json:"processCPU,omitempty"`
	SystemCPU      ChartEntries `json:"systemCPU,omitempty"`
	DiskRead       ChartEntries `json:"diskRead,omitempty"`
	DiskWrite      ChartEntries `json:"diskWrite,omitempty"`
}

// LogsMessage wraps up a log chunk. If Source isn't present, the chunk is a stream chunk.
type LogsMessage struct {
	Source *LogFile        `json:"source,omitempty"` // Attributes of the log file.
	Chunk  json.RawMessage `json:"chunk"`            // Contains log records.
}

// LogFile contains the attributes of a log file.
type LogFile struct {
	Name string `json:"name"` // The name of the file.
	Last bool   `json:"last"` // Denotes if the actual log file is the last one in the directory.
}

// Request represents the client request.
type Request struct {
	Logs *LogsRequest `json:"logs,omitempty"`
}

type LogsRequest struct {
	Name string `json:"name"` // The request handler searches for log file based on this file name.
	Past bool   `json:"past"` // Denotes whfafer the client wants the previous or the next file.
}
