// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package les

import (
	"github.com/fafereum/go-fafereum/metrics"
	"github.com/fafereum/go-fafereum/p2p"
)

var (
	/*	propTxnInPacketsMeter     = metrics.NewMeter("faf/prop/txns/in/packets")
		propTxnInTrafficMeter     = metrics.NewMeter("faf/prop/txns/in/traffic")
		propTxnOutPacketsMeter    = metrics.NewMeter("faf/prop/txns/out/packets")
		propTxnOutTrafficMeter    = metrics.NewMeter("faf/prop/txns/out/traffic")
		propHashInPacketsMeter    = metrics.NewMeter("faf/prop/hashes/in/packets")
		propHashInTrafficMeter    = metrics.NewMeter("faf/prop/hashes/in/traffic")
		propHashOutPacketsMeter   = metrics.NewMeter("faf/prop/hashes/out/packets")
		propHashOutTrafficMeter   = metrics.NewMeter("faf/prop/hashes/out/traffic")
		propBlockInPacketsMeter   = metrics.NewMeter("faf/prop/blocks/in/packets")
		propBlockInTrafficMeter   = metrics.NewMeter("faf/prop/blocks/in/traffic")
		propBlockOutPacketsMeter  = metrics.NewMeter("faf/prop/blocks/out/packets")
		propBlockOutTrafficMeter  = metrics.NewMeter("faf/prop/blocks/out/traffic")
		reqHashInPacketsMeter     = metrics.NewMeter("faf/req/hashes/in/packets")
		reqHashInTrafficMeter     = metrics.NewMeter("faf/req/hashes/in/traffic")
		reqHashOutPacketsMeter    = metrics.NewMeter("faf/req/hashes/out/packets")
		reqHashOutTrafficMeter    = metrics.NewMeter("faf/req/hashes/out/traffic")
		reqBlockInPacketsMeter    = metrics.NewMeter("faf/req/blocks/in/packets")
		reqBlockInTrafficMeter    = metrics.NewMeter("faf/req/blocks/in/traffic")
		reqBlockOutPacketsMeter   = metrics.NewMeter("faf/req/blocks/out/packets")
		reqBlockOutTrafficMeter   = metrics.NewMeter("faf/req/blocks/out/traffic")
		reqHeaderInPacketsMeter   = metrics.NewMeter("faf/req/headers/in/packets")
		reqHeaderInTrafficMeter   = metrics.NewMeter("faf/req/headers/in/traffic")
		reqHeaderOutPacketsMeter  = metrics.NewMeter("faf/req/headers/out/packets")
		reqHeaderOutTrafficMeter  = metrics.NewMeter("faf/req/headers/out/traffic")
		reqBodyInPacketsMeter     = metrics.NewMeter("faf/req/bodies/in/packets")
		reqBodyInTrafficMeter     = metrics.NewMeter("faf/req/bodies/in/traffic")
		reqBodyOutPacketsMeter    = metrics.NewMeter("faf/req/bodies/out/packets")
		reqBodyOutTrafficMeter    = metrics.NewMeter("faf/req/bodies/out/traffic")
		reqStateInPacketsMeter    = metrics.NewMeter("faf/req/states/in/packets")
		reqStateInTrafficMeter    = metrics.NewMeter("faf/req/states/in/traffic")
		reqStateOutPacketsMeter   = metrics.NewMeter("faf/req/states/out/packets")
		reqStateOutTrafficMeter   = metrics.NewMeter("faf/req/states/out/traffic")
		reqReceiptInPacketsMeter  = metrics.NewMeter("faf/req/receipts/in/packets")
		reqReceiptInTrafficMeter  = metrics.NewMeter("faf/req/receipts/in/traffic")
		reqReceiptOutPacketsMeter = metrics.NewMeter("faf/req/receipts/out/packets")
		reqReceiptOutTrafficMeter = metrics.NewMeter("faf/req/receipts/out/traffic")*/
	miscInPacketsMeter  = metrics.NewRegisteredMeter("les/misc/in/packets", nil)
	miscInTrafficMeter  = metrics.NewRegisteredMeter("les/misc/in/traffic", nil)
	miscOutPacketsMeter = metrics.NewRegisteredMeter("les/misc/out/packets", nil)
	miscOutTrafficMeter = metrics.NewRegisteredMeter("les/misc/out/traffic", nil)
)

// meteredMsgReadWriter is a wrapper around a p2p.MsgReadWriter, capable of
// accumulating the above defined metrics based on the data stream contents.
type meteredMsgReadWriter struct {
	p2p.MsgReadWriter     // Wrapped message stream to meter
	version           int // Protocol version to select correct meters
}

// newMeteredMsgWriter wraps a p2p MsgReadWriter with metering support. If the
// metrics system is disabled, this function returns the original object.
func newMeteredMsgWriter(rw p2p.MsgReadWriter) p2p.MsgReadWriter {
	if !metrics.Enabled {
		return rw
	}
	return &meteredMsgReadWriter{MsgReadWriter: rw}
}

// Init sets the protocol version used by the stream to know which meters to
// increment in case of overlapping message ids between protocol versions.
func (rw *meteredMsgReadWriter) Init(version int) {
	rw.version = version
}

func (rw *meteredMsgReadWriter) ReadMsg() (p2p.Msg, error) {
	// Read the message and short circuit in case of an error
	msg, err := rw.MsgReadWriter.ReadMsg()
	if err != nil {
		return msg, err
	}
	// Account for the data traffic
	packets, traffic := miscInPacketsMeter, miscInTrafficMeter
	packets.Mark(1)
	traffic.Mark(int64(msg.Size))

	return msg, err
}

func (rw *meteredMsgReadWriter) WriteMsg(msg p2p.Msg) error {
	// Account for the data traffic
	packets, traffic := miscOutPacketsMeter, miscOutTrafficMeter
	packets.Mark(1)
	traffic.Mark(int64(msg.Size))

	// Send the packet to the p2p layer
	return rw.MsgReadWriter.WriteMsg(msg)
}
