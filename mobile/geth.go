// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

// Contains all the wrappers from the node package to support client side node
// management on mobile platforms.

package gfaf

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/fafereum/go-fafereum/core"
	"github.com/fafereum/go-fafereum/faf"
	"github.com/fafereum/go-fafereum/faf/downloader"
	"github.com/fafereum/go-fafereum/fafclient"
	"github.com/fafereum/go-fafereum/fafstats"
	"github.com/fafereum/go-fafereum/internal/debug"
	"github.com/fafereum/go-fafereum/les"
	"github.com/fafereum/go-fafereum/node"
	"github.com/fafereum/go-fafereum/p2p"
	"github.com/fafereum/go-fafereum/p2p/nat"
	"github.com/fafereum/go-fafereum/params"
	whisper "github.com/fafereum/go-fafereum/whisper/whisperv6"
)

// NodeConfig represents the collection of configuration values to fine tune the Gfaf
// node embedded into a mobile process. The available values are a subset of the
// entire API provided by go-fafereum to reduce the maintenance surface and dev
// complexity.
type NodeConfig struct {
	// Bootstrap nodes used to establish connectivity with the rest of the network.
	BootstrapNodes *Enodes

	// MaxPeers is the maximum number of peers that can be connected. If this is
	// set to zero, then only the configured static and trusted peers can connect.
	MaxPeers int

	// fafereumEnabled specifies whfafer the node should run the fafereum protocol.
	fafereumEnabled bool

	// fafereumNetworkID is the network identifier used by the fafereum protocol to
	// decide if remote peers should be accepted or not.
	fafereumNetworkID int64 // uint64 in truth, but Java can't handle that...

	// fafereumGenesis is the genesis JSON to use to seed the blockchain with. An
	// empty genesis state is equivalent to using the mainnet's state.
	fafereumGenesis string

	// fafereumDatabaseCache is the system memory in MB to allocate for database caching.
	// A minimum of 16MB is always reserved.
	fafereumDatabaseCache int

	// fafereumNetStats is a netstats connection string to use to report various
	// chain, transaction and node stats to a monitoring server.
	//
	// It has the form "nodename:secret@host:port"
	fafereumNetStats string

	// WhisperEnabled specifies whfafer the node should run the Whisper protocol.
	WhisperEnabled bool

	// Listening address of pprof server.
	PprofAddress string
}

// defaultNodeConfig contains the default node configuration values to use if all
// or some fields are missing from the user's specified list.
var defaultNodeConfig = &NodeConfig{
	BootstrapNodes:        FoundationBootnodes(),
	MaxPeers:              25,
	fafereumEnabled:       true,
	fafereumNetworkID:     1,
	fafereumDatabaseCache: 16,
}

// NewNodeConfig creates a new node option set, initialized to the default values.
func NewNodeConfig() *NodeConfig {
	config := *defaultNodeConfig
	return &config
}

// Node represents a Gfaf fafereum node instance.
type Node struct {
	node *node.Node
}

// NewNode creates and configures a new Gfaf node.
func NewNode(datadir string, config *NodeConfig) (stack *Node, _ error) {
	// If no or partial configurations were specified, use defaults
	if config == nil {
		config = NewNodeConfig()
	}
	if config.MaxPeers == 0 {
		config.MaxPeers = defaultNodeConfig.MaxPeers
	}
	if config.BootstrapNodes == nil || config.BootstrapNodes.Size() == 0 {
		config.BootstrapNodes = defaultNodeConfig.BootstrapNodes
	}

	if config.PprofAddress != "" {
		debug.StartPProf(config.PprofAddress)
	}

	// Create the empty networking stack
	nodeConf := &node.Config{
		Name:        clientIdentifier,
		Version:     params.VersionWithMeta,
		DataDir:     datadir,
		KeyStoreDir: filepath.Join(datadir, "keystore"), // Mobile should never use internal keystores!
		P2P: p2p.Config{
			NoDiscovery:      true,
			DiscoveryV5:      true,
			BootstrapNodesV5: config.BootstrapNodes.nodes,
			ListenAddr:       ":0",
			NAT:              nat.Any(),
			MaxPeers:         config.MaxPeers,
		},
	}
	rawStack, err := node.New(nodeConf)
	if err != nil {
		return nil, err
	}

	debug.Memsize.Add("node", rawStack)

	var genesis *core.Genesis
	if config.fafereumGenesis != "" {
		// Parse the user supplied genesis spec if not mainnet
		genesis = new(core.Genesis)
		if err := json.Unmarshal([]byte(config.fafereumGenesis), genesis); err != nil {
			return nil, fmt.Errorf("invalid genesis spec: %v", err)
		}
		// If we have the testnet, hard code the chain configs too
		if config.fafereumGenesis == TestnetGenesis() {
			genesis.Config = params.TestnetChainConfig
			if config.fafereumNetworkID == 1 {
				config.fafereumNetworkID = 3
			}
		}
	}
	// Register the fafereum protocol if requested
	if config.fafereumEnabled {
		fafConf := faf.DefaultConfig
		fafConf.Genesis = genesis
		fafConf.SyncMode = downloader.LightSync
		fafConf.NetworkId = uint64(config.fafereumNetworkID)
		fafConf.DatabaseCache = config.fafereumDatabaseCache
		if err := rawStack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
			return les.New(ctx, &fafConf)
		}); err != nil {
			return nil, fmt.Errorf("fafereum init: %v", err)
		}
		// If netstats reporting is requested, do it
		if config.fafereumNetStats != "" {
			if err := rawStack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
				var lesServ *les.Lightfafereum
				ctx.Service(&lesServ)

				return fafstats.New(config.fafereumNetStats, nil, lesServ)
			}); err != nil {
				return nil, fmt.Errorf("netstats init: %v", err)
			}
		}
	}
	// Register the Whisper protocol if requested
	if config.WhisperEnabled {
		if err := rawStack.Register(func(*node.ServiceContext) (node.Service, error) {
			return whisper.New(&whisper.DefaultConfig), nil
		}); err != nil {
			return nil, fmt.Errorf("whisper init: %v", err)
		}
	}
	return &Node{rawStack}, nil
}

// Start creates a live P2P node and starts running it.
func (n *Node) Start() error {
	return n.node.Start()
}

// Stop terminates a running node along with all it's services. If the node was
// not started, an error is returned.
func (n *Node) Stop() error {
	return n.node.Stop()
}

// GetfafereumClient retrieves a client to access the fafereum subsystem.
func (n *Node) GetfafereumClient() (client *fafereumClient, _ error) {
	rpc, err := n.node.Attach()
	if err != nil {
		return nil, err
	}
	return &fafereumClient{fafclient.NewClient(rpc)}, nil
}

// GetNodeInfo gathers and returns a collection of metadata known about the host.
func (n *Node) GetNodeInfo() *NodeInfo {
	return &NodeInfo{n.node.Server().NodeInfo()}
}

// GetPeersInfo returns an array of metadata objects describing connected peers.
func (n *Node) GetPeersInfo() *PeerInfos {
	return &PeerInfos{n.node.Server().PeersInfo()}
}
