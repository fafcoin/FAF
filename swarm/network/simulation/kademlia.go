// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package simulation

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/fafereum/go-fafereum/common"
	"github.com/fafereum/go-fafereum/log"
	"github.com/fafereum/go-fafereum/p2p/enode"
	"github.com/fafereum/go-fafereum/swarm/network"
)

// BucketKeyKademlia is the key to be used for storing the kademlia
// instance for particular node, usually inside the ServiceFunc function.
var BucketKeyKademlia BucketKey = "kademlia"

// WaitTillHealthy is blocking until the health of all kademlias is true.
// If error is not nil, a map of kademlia that was found not healthy is returned.
// TODO: Check correctness since change in kademlia depth calculation logic
func (s *Simulation) WaitTillHealthy(ctx context.Context) (ill map[enode.ID]*network.Kademlia, err error) {
	// Prepare PeerPot map for checking Kademlia health
	var ppmap map[string]*network.PeerPot
	kademlias := s.kademlias()
	addrs := make([][]byte, 0, len(kademlias))
	// TODO verify that all kademlias have same params
	for _, k := range kademlias {
		addrs = append(addrs, k.BaseAddr())
	}
	ppmap = network.NewPeerPotMap(s.neighbourhoodSize, addrs)

	// Wait for healthy Kademlia on every node before checking files
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	ill = make(map[enode.ID]*network.Kademlia)
	for {
		select {
		case <-ctx.Done():
			return ill, ctx.Err()
		case <-ticker.C:
			for k := range ill {
				delete(ill, k)
			}
			log.Debug("kademlia health check", "addr count", len(addrs))
			for id, k := range kademlias {
				//PeerPot for this node
				addr := common.Bytes2Hex(k.BaseAddr())
				pp := ppmap[addr]
				//call Healthy RPC
				h := k.GfafealthInfo(pp)
				//print info
				log.Debug(k.String())
				log.Debug("kademlia", "connectNN", h.ConnectNN, "knowNN", h.KnowNN)
				log.Debug("kademlia", "health", h.ConnectNN && h.KnowNN, "addr", hex.EncodeToString(k.BaseAddr()), "node", id)
				log.Debug("kademlia", "ill condition", !h.ConnectNN, "addr", hex.EncodeToString(k.BaseAddr()), "node", id)
				if !h.ConnectNN {
					ill[id] = k
				}
			}
			if len(ill) == 0 {
				return nil, nil
			}
		}
	}
}

// kademlias returns all Kademlia instances that are set
// in simulation bucket.
func (s *Simulation) kademlias() (ks map[enode.ID]*network.Kademlia) {
	items := s.UpNodesItems(BucketKeyKademlia)
	ks = make(map[enode.ID]*network.Kademlia, len(items))
	for id, v := range items {
		k, ok := v.(*network.Kademlia)
		if !ok {
			continue
		}
		ks[id] = k
	}
	return ks
}
