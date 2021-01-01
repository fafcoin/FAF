// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package les

import (
	"fmt"
	"math/big"

	"github.com/fafereum/go-fafereum/common"
	"github.com/fafereum/go-fafereum/core"
	"github.com/fafereum/go-fafereum/faf"
	"github.com/fafereum/go-fafereum/fafdb"
	"github.com/fafereum/go-fafereum/light"
	"github.com/fafereum/go-fafereum/p2p"
	"github.com/fafereum/go-fafereum/p2p/enode"
	"github.com/fafereum/go-fafereum/params"
)

// lesCommons contains fields needed by both server and client.
type lesCommons struct {
	config                       *faf.Config
	iConfig                      *light.IndexerConfig
	chainDb                      fafdb.Database
	protocolManager              *ProtocolManager
	chtIndexer, bloomTrieIndexer *core.ChainIndexer
}

// NodeInfo represents a short summary of the fafereum sub-protocol metadata
// known about the host peer.
type NodeInfo struct {
	Network    uint64                   `json:"network"`    // fafereum network ID (1=Frontier, 2=Morden, Ropsten=3, Rinkeby=4)
	Difficulty *big.Int                 `json:"difficulty"` // Total difficulty of the host's blockchain
	Genesis    common.Hash              `json:"genesis"`    // SHA3 hash of the host's genesis block
	Config     *params.ChainConfig      `json:"config"`     // Chain configuration for the fork rules
	Head       common.Hash              `json:"head"`       // SHA3 hash of the host's best owned block
	CHT        params.TrustedCheckpoint `json:"cht"`        // Trused CHT checkpoint for fast catchup
}

// makeProtocols creates protocol descriptors for the given LES versions.
func (c *lesCommons) makeProtocols(versions []uint) []p2p.Protocol {
	protos := make([]p2p.Protocol, len(versions))
	for i, version := range versions {
		version := version
		protos[i] = p2p.Protocol{
			Name:     "les",
			Version:  version,
			Length:   ProtocolLengths[version],
			NodeInfo: c.nodeInfo,
			Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				return c.protocolManager.runPeer(version, p, rw)
			},
			PeerInfo: func(id enode.ID) interface{} {
				if p := c.protocolManager.peers.Peer(fmt.Sprintf("%x", id[:8])); p != nil {
					return p.Info()
				}
				return nil
			},
		}
	}
	return protos
}

// nodeInfo retrieves some protocol metadata about the running host node.
func (c *lesCommons) nodeInfo() interface{} {
	var cht params.TrustedCheckpoint
	sections, _, _ := c.chtIndexer.Sections()
	sections2, _, _ := c.bloomTrieIndexer.Sections()

	if !c.protocolManager.lightSync {
		// convert to client section size if running in server mode
		sections /= c.iConfig.PairChtSize / c.iConfig.ChtSize
	}

	if sections2 < sections {
		sections = sections2
	}
	if sections > 0 {
		sectionIndex := sections - 1
		sectionHead := c.bloomTrieIndexer.SectionHead(sectionIndex)
		var chtRoot common.Hash
		if c.protocolManager.lightSync {
			chtRoot = light.GetChtRoot(c.chainDb, sectionIndex, sectionHead)
		} else {
			idxV2 := (sectionIndex+1)*c.iConfig.PairChtSize/c.iConfig.ChtSize - 1
			chtRoot = light.GetChtRoot(c.chainDb, idxV2, sectionHead)
		}
		cht = params.TrustedCheckpoint{
			SectionIndex: sectionIndex,
			SectionHead:  sectionHead,
			CHTRoot:      chtRoot,
			BloomRoot:    light.GetBloomTrieRoot(c.chainDb, sectionIndex, sectionHead),
		}
	}

	chain := c.protocolManager.blockchain
	head := chain.CurrentHeader()
	hash := head.Hash()
	return &NodeInfo{
		Network:    c.config.NetworkId,
		Difficulty: chain.GetTd(hash, head.Number.Uint64()),
		Genesis:    chain.Genesis().Hash(),
		Config:     chain.Config(),
		Head:       chain.CurrentHeader().Hash(),
		CHT:        cht,
	}
}
