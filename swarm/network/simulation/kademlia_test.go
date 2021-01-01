// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package simulation

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/fafereum/go-fafereum/node"
	"github.com/fafereum/go-fafereum/p2p/simulations/adapters"
	"github.com/fafereum/go-fafereum/swarm/network"
)

func TestWaitTillHealthy(t *testing.T) {
	t.Skip("WaitTillHealthy depends on discovery, which relies on a reliable SuggestPeer, which is not reliable")

	sim := New(map[string]ServiceFunc{
		"bzz": func(ctx *adapters.ServiceContext, b *sync.Map) (node.Service, func(), error) {
			addr := network.NewAddr(ctx.Config.Node())
			hp := network.NewHiveParams()
			config := &network.BzzConfig{
				OverlayAddr:  addr.Over(),
				UnderlayAddr: addr.Under(),
				HiveParams:   hp,
			}
			kad := network.NewKademlia(addr.Over(), network.NewKadParams())
			// store kademlia in node's bucket under BucketKeyKademlia
			// so that it can be found by WaitTillHealthy mfafod.
			b.Store(BucketKeyKademlia, kad)
			return network.NewBzz(config, kad, nil, nil, nil), nil, nil
		},
	})
	defer sim.Close()

	_, err := sim.AddNodesAndConnectRing(10)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	ill, err := sim.WaitTillHealthy(ctx)
	if err != nil {
		for id, kad := range ill {
			t.Log("Node", id)
			t.Log(kad.String())
		}
		if err != nil {
			t.Fatal(err)
		}
	}
}
