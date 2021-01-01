// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package simulation

import (
	"sync"
	"testing"

	"github.com/fafereum/go-fafereum/node"
	"github.com/fafereum/go-fafereum/p2p/simulations/adapters"
)

// TestServiceBucket tests all bucket functionality using subtests.
// It constructs a simulation of two nodes by adding items to their buckets
// in ServiceFunc constructor, then by SetNodeItem. Testing UpNodesItems
// is done by stopping one node and validating availability of its items.
func TestServiceBucket(t *testing.T) {
	testKey := "Key"
	testValue := "Value"

	sim := New(map[string]ServiceFunc{
		"noop": func(ctx *adapters.ServiceContext, b *sync.Map) (node.Service, func(), error) {
			b.Store(testKey, testValue+ctx.Config.ID.String())
			return newNoopService(), nil, nil
		},
	})
	defer sim.Close()

	id1, err := sim.AddNode()
	if err != nil {
		t.Fatal(err)
	}

	id2, err := sim.AddNode()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("ServiceFunc bucket Store", func(t *testing.T) {
		v, ok := sim.NodeItem(id1, testKey)
		if !ok {
			t.Fatal("bucket item not found")
		}
		s, ok := v.(string)
		if !ok {
			t.Fatal("bucket item value is not string")
		}
		if s != testValue+id1.String() {
			t.Fatalf("expected %q, got %q", testValue+id1.String(), s)
		}

		v, ok = sim.NodeItem(id2, testKey)
		if !ok {
			t.Fatal("bucket item not found")
		}
		s, ok = v.(string)
		if !ok {
			t.Fatal("bucket item value is not string")
		}
		if s != testValue+id2.String() {
			t.Fatalf("expected %q, got %q", testValue+id2.String(), s)
		}
	})

	customKey := "anotherKey"
	customValue := "anotherValue"

	t.Run("SetNodeItem", func(t *testing.T) {
		sim.SetNodeItem(id1, customKey, customValue)

		v, ok := sim.NodeItem(id1, customKey)
		if !ok {
			t.Fatal("bucket item not found")
		}
		s, ok := v.(string)
		if !ok {
			t.Fatal("bucket item value is not string")
		}
		if s != customValue {
			t.Fatalf("expected %q, got %q", customValue, s)
		}

		_, ok = sim.NodeItem(id2, customKey)
		if ok {
			t.Fatal("bucket item should not be found")
		}
	})

	if err := sim.StopNode(id2); err != nil {
		t.Fatal(err)
	}

	t.Run("UpNodesItems", func(t *testing.T) {
		items := sim.UpNodesItems(testKey)

		v, ok := items[id1]
		if !ok {
			t.Errorf("node 1 item not found")
		}
		s, ok := v.(string)
		if !ok {
			t.Fatal("node 1 item value is not string")
		}
		if s != testValue+id1.String() {
			t.Fatalf("expected %q, got %q", testValue+id1.String(), s)
		}

		_, ok = items[id2]
		if ok {
			t.Errorf("node 2 item should not be found")
		}
	})

	t.Run("NodeItems", func(t *testing.T) {
		items := sim.NodesItems(testKey)

		v, ok := items[id1]
		if !ok {
			t.Errorf("node 1 item not found")
		}
		s, ok := v.(string)
		if !ok {
			t.Fatal("node 1 item value is not string")
		}
		if s != testValue+id1.String() {
			t.Fatalf("expected %q, got %q", testValue+id1.String(), s)
		}

		v, ok = items[id2]
		if !ok {
			t.Errorf("node 2 item not found")
		}
		s, ok = v.(string)
		if !ok {
			t.Fatal("node 1 item value is not string")
		}
		if s != testValue+id2.String() {
			t.Fatalf("expected %q, got %q", testValue+id2.String(), s)
		}
	})
}
