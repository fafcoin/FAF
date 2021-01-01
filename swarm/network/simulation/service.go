// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package simulation

import (
	"github.com/fafereum/go-fafereum/node"
	"github.com/fafereum/go-fafereum/p2p/enode"
	"github.com/fafereum/go-fafereum/p2p/simulations/adapters"
)

// Service returns a single Service by name on a particular node
// with provided id.
func (s *Simulation) Service(name string, id enode.ID) node.Service {
	simNode, ok := s.Net.GetNode(id).Node.(*adapters.SimNode)
	if !ok {
		return nil
	}
	services := simNode.ServiceMap()
	if len(services) == 0 {
		return nil
	}
	return services[name]
}

// RandomService returns a single Service by name on a
// randomly chosen node that is up.
func (s *Simulation) RandomService(name string) node.Service {
	n := s.Net.GetRandomUpNode().Node.(*adapters.SimNode)
	if n == nil {
		return nil
	}
	return n.Service(name)
}

// Services returns all services with a provided name
// from nodes that are up.
func (s *Simulation) Services(name string) (services map[enode.ID]node.Service) {
	nodes := s.Net.GetNodes()
	services = make(map[enode.ID]node.Service)
	for _, node := range nodes {
		if !node.Up() {
			continue
		}
		simNode, ok := node.Node.(*adapters.SimNode)
		if !ok {
			continue
		}
		services[node.ID()] = simNode.Service(name)
	}
	return services
}
