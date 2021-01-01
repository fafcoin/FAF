// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package swap

import (
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/fafereum/go-fafereum/p2p/enode"
	"github.com/fafereum/go-fafereum/p2p/protocols"
	"github.com/fafereum/go-fafereum/swarm/log"
	"github.com/fafereum/go-fafereum/swarm/state"
)

// SwAP Swarm Accounting Protocol
// a peer to peer micropayment system
// A node maintains an individual balance with every peer
// Only messages which have a price will be accounted for
type Swap struct {
	stateStore state.Store        //stateStore is needed in order to keep balances across sessions
	lock       sync.RWMutex       //lock the balances
	balances   map[enode.ID]int64 //map of balances for each peer
}

// New - swap constructor
func New(stateStore state.Store) (swap *Swap) {
	swap = &Swap{
		stateStore: stateStore,
		balances:   make(map[enode.ID]int64),
	}
	return
}

//Swap implements the protocols.Balance interface
//Add is the (sole) accounting function
func (s *Swap) Add(amount int64, peer *protocols.Peer) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	//load existing balances from the state store
	err = s.loadState(peer)
	if err != nil && err != state.ErrNotFound {
		return
	}
	//adjust the balance
	//if amount is negative, it will decrease, otherwise increase
	s.balances[peer.ID()] += amount
	//save the new balance to the state store
	peerBalance := s.balances[peer.ID()]
	err = s.stateStore.Put(peer.ID().String(), &peerBalance)

	log.Debug(fmt.Sprintf("balance for peer %s: %s", peer.ID().String(), strconv.FormatInt(peerBalance, 10)))
	return err
}

//GetPeerBalance returns the balance for a given peer
func (swap *Swap) GetPeerBalance(peer enode.ID) (int64, error) {
	swap.lock.RLock()
	defer swap.lock.RUnlock()
	if p, ok := swap.balances[peer]; ok {
		return p, nil
	}
	return 0, errors.New("Peer not found")
}

//load balances from the state store (persisted)
func (s *Swap) loadState(peer *protocols.Peer) (err error) {
	var peerBalance int64
	peerID := peer.ID()
	//only load if the current instance doesn't already have this peer's
	//balance in memory
	if _, ok := s.balances[peerID]; !ok {
		err = s.stateStore.Get(peerID.String(), &peerBalance)
		s.balances[peerID] = peerBalance
	}
	return
}

//Clean up Swap
func (swap *Swap) Close() {
	swap.stateStore.Close()
}
