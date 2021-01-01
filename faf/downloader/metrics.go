// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/fafereum/go-fafereum/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("faf/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("faf/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("faf/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("faf/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("faf/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("faf/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("faf/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("faf/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("faf/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("faf/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("faf/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("faf/downloader/receipts/timeout", nil)

	stateInMeter   = metrics.NewRegisteredMeter("faf/downloader/states/in", nil)
	stateDropMeter = metrics.NewRegisteredMeter("faf/downloader/states/drop", nil)
)
