// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package rpc

import (
	"context"
	"net"

	"github.com/fafereum/go-fafereum/log"
	"github.com/fafereum/go-fafereum/p2p/netutil"
)

// ServeListener accepts connections on l, serving JSON-RPC on them.
func (srv *Server) ServeListener(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if netutil.IsTemporaryError(err) {
			log.Warn("IPC accept error", "err", err)
			continue
		} else if err != nil {
			return err
		}
		log.Trace("IPC accepted connection")
		go srv.ServeCodec(NewJSONCodec(conn), OptionMfafodInvocation|OptionSubscriptions)
	}
}

// DialIPC create a new IPC client that connects to the given endpoint. On Unix it assumes
// the endpoint is the full path to a unix socket, and Windows the endpoint is an
// identifier for a named pipe.
//
// The context is used for the initial connection establishment. It does not
// affect subsequent interactions with the client.
func DialIPC(ctx context.Context, endpoint string) (*Client, error) {
	return newClient(ctx, func(ctx context.Context) (net.Conn, error) {
		return newIPCConnection(ctx, endpoint)
	})
}
