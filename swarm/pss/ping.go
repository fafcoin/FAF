// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

// +build !nopssprotocol,!nopssping

package pss

import (
	"context"
	"errors"
	"time"

	"github.com/fafereum/go-fafereum/p2p"
	"github.com/fafereum/go-fafereum/p2p/protocols"
	"github.com/fafereum/go-fafereum/swarm/log"
)

// Generic ping protocol implementation for
// pss devp2p protocol emulation
type PingMsg struct {
	Created time.Time
	Pong    bool // set if message is pong reply
}

type Ping struct {
	Pong bool      // toggle pong reply upon ping receive
	OutC chan bool // trigger ping
	InC  chan bool // optional, report back to calling code
}

func (p *Ping) pingHandler(ctx context.Context, msg interface{}) error {
	var pingmsg *PingMsg
	var ok bool
	if pingmsg, ok = msg.(*PingMsg); !ok {
		return errors.New("invalid msg")
	}
	log.Debug("ping handler", "msg", pingmsg, "outc", p.OutC)
	if p.InC != nil {
		p.InC <- pingmsg.Pong
	}
	if p.Pong && !pingmsg.Pong {
		p.OutC <- true
	}
	return nil
}

var PingProtocol = &protocols.Spec{
	Name:       "psstest",
	Version:    1,
	MaxMsgSize: 1024,
	Messages: []interface{}{
		PingMsg{},
	},
}

var PingTopic = ProtocolTopic(PingProtocol)

func NewPingProtocol(ping *Ping) *p2p.Protocol {
	return &p2p.Protocol{
		Name:    PingProtocol.Name,
		Version: PingProtocol.Version,
		Length:  uint64(PingProtocol.MaxMsgSize),
		Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
			quitC := make(chan struct{})
			pp := protocols.NewPeer(p, rw, PingProtocol)
			log.Trace("running pss vprotocol", "peer", p, "outc", ping.OutC)
			go func() {
				for {
					select {
					case ispong := <-ping.OutC:
						pp.Send(context.TODO(), &PingMsg{
							Created: time.Now(),
							Pong:    ispong,
						})
					case <-quitC:
					}
				}
			}()
			err := pp.Run(ping.pingHandler)
			quitC <- struct{}{}
			return err
		},
	}
}
