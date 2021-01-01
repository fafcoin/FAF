// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

// Package les implements the Light fafereum Subprotocol.
package les

import (
	"fmt"
	"sync"
	"time"

	"github.com/fafereum/go-fafereum/accounts"
	"github.com/fafereum/go-fafereum/common"
	"github.com/fafereum/go-fafereum/common/hexutil"
	"github.com/fafereum/go-fafereum/consensus"
	"github.com/fafereum/go-fafereum/core"
	"github.com/fafereum/go-fafereum/core/bloombits"
	"github.com/fafereum/go-fafereum/core/rawdb"
	"github.com/fafereum/go-fafereum/core/types"
	"github.com/fafereum/go-fafereum/faf"
	"github.com/fafereum/go-fafereum/faf/downloader"
	"github.com/fafereum/go-fafereum/faf/filters"
	"github.com/fafereum/go-fafereum/faf/gasprice"
	"github.com/fafereum/go-fafereum/event"
	"github.com/fafereum/go-fafereum/internal/fafapi"
	"github.com/fafereum/go-fafereum/light"
	"github.com/fafereum/go-fafereum/log"
	"github.com/fafereum/go-fafereum/node"
	"github.com/fafereum/go-fafereum/p2p"
	"github.com/fafereum/go-fafereum/p2p/discv5"
	"github.com/fafereum/go-fafereum/params"
	rpc "github.com/fafereum/go-fafereum/rpc"
)

type Lightfafereum struct {
	lesCommons

	odr         *LesOdr
	relay       *LesTxRelay
	chainConfig *params.ChainConfig
	// Channel for shutting down the service
	shutdownChan chan bool

	// Handlers
	peers      *peerSet
	txPool     *light.TxPool
	blockchain *light.LightChain
	serverPool *serverPool
	reqDist    *requestDistributor
	retriever  *retrieveManager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer

	ApiBackend *LesApiBackend

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	networkId     uint64
	netRPCService *fafapi.PublicNetAPI

	wg sync.WaitGroup
}

func New(ctx *node.ServiceContext, config *faf.Config) (*Lightfafereum, error) {
	chainDb, err := faf.CreateDB(ctx, config, "lightchaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlockWithOverride(chainDb, config.Genesis, config.ConstantinopleOverride)
	if _, isCompat := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !isCompat {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	peers := newPeerSet()
	quitSync := make(chan struct{})

	lfaf := &Lightfafereum{
		lesCommons: lesCommons{
			chainDb: chainDb,
			config:  config,
			iConfig: light.DefaultClientIndexerConfig,
		},
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		peers:          peers,
		reqDist:        newRequestDistributor(peers, quitSync),
		accountManager: ctx.AccountManager,
		engine:         faf.CreateConsensusEngine(ctx, chainConfig, &config.fafash, nil, false, chainDb),
		shutdownChan:   make(chan bool),
		networkId:      config.NetworkId,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   faf.NewBloomIndexer(chainDb, params.BloomBitsBlocksClient, params.HelperTrieConfirmations),
	}

	lfaf.relay = NewLesTxRelay(peers, lfaf.reqDist)
	lfaf.serverPool = newServerPool(chainDb, quitSync, &lfaf.wg)
	lfaf.retriever = newRetrieveManager(peers, lfaf.reqDist, lfaf.serverPool)

	lfaf.odr = NewLesOdr(chainDb, light.DefaultClientIndexerConfig, lfaf.retriever)
	lfaf.chtIndexer = light.NewChtIndexer(chainDb, lfaf.odr, params.CHTFrequencyClient, params.HelperTrieConfirmations)
	lfaf.bloomTrieIndexer = light.NewBloomTrieIndexer(chainDb, lfaf.odr, params.BloomBitsBlocksClient, params.BloomTrieFrequency)
	lfaf.odr.SetIndexers(lfaf.chtIndexer, lfaf.bloomTrieIndexer, lfaf.bloomIndexer)

	// Note: NewLightChain adds the trusted checkpoint so it needs an ODR with
	// indexers already set but not started yet
	if lfaf.blockchain, err = light.NewLightChain(lfaf.odr, lfaf.chainConfig, lfaf.engine); err != nil {
		return nil, err
	}
	// Note: AddChildIndexer starts the update process for the child
	lfaf.bloomIndexer.AddChildIndexer(lfaf.bloomTrieIndexer)
	lfaf.chtIndexer.Start(lfaf.blockchain)
	lfaf.bloomIndexer.Start(lfaf.blockchain)

	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		lfaf.blockchain.Sfafead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	lfaf.txPool = light.NewTxPool(lfaf.chainConfig, lfaf.blockchain, lfaf.relay)
	if lfaf.protocolManager, err = NewProtocolManager(lfaf.chainConfig, light.DefaultClientIndexerConfig, true, config.NetworkId, lfaf.eventMux, lfaf.engine, lfaf.peers, lfaf.blockchain, nil, chainDb, lfaf.odr, lfaf.relay, lfaf.serverPool, quitSync, &lfaf.wg); err != nil {
		return nil, err
	}
	lfaf.ApiBackend = &LesApiBackend{lfaf, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.MinerGasPrice
	}
	lfaf.ApiBackend.gpo = gasprice.NewOracle(lfaf.ApiBackend, gpoParams)
	return lfaf, nil
}

func lesTopic(genesisHash common.Hash, protocolVersion uint) discv5.Topic {
	var name string
	switch protocolVersion {
	case lpv1:
		name = "LES"
	case lpv2:
		name = "LES2"
	default:
		panic(nil)
	}
	return discv5.Topic(name + "@" + common.Bytes2Hex(genesisHash.Bytes()[0:8]))
}

type LightDummyAPI struct{}

// faferbase is the address that mining rewards will be send to
func (s *LightDummyAPI) faferbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Coinbase is the address that mining rewards will be send to (alias for faferbase)
func (s *LightDummyAPI) Coinbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Hashrate returns the POW hashrate
func (s *LightDummyAPI) Hashrate() hexutil.Uint {
	return 0
}

// Mining returns an indication if this node is currently mining.
func (s *LightDummyAPI) Mining() bool {
	return false
}

// APIs returns the collection of RPC services the fafereum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *Lightfafereum) APIs() []rpc.API {
	return append(fafapi.GetAPIs(s.ApiBackend), []rpc.API{
		{
			Namespace: "faf",
			Version:   "1.0",
			Service:   &LightDummyAPI{},
			Public:    true,
		}, {
			Namespace: "faf",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "faf",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, true),
			Public:    true,
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *Lightfafereum) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *Lightfafereum) BlockChain() *light.LightChain      { return s.blockchain }
func (s *Lightfafereum) TxPool() *light.TxPool              { return s.txPool }
func (s *Lightfafereum) Engine() consensus.Engine           { return s.engine }
func (s *Lightfafereum) LesVersion() int                    { return int(ClientProtocolVersions[0]) }
func (s *Lightfafereum) Downloader() *downloader.Downloader { return s.protocolManager.downloader }
func (s *Lightfafereum) EventMux() *event.TypeMux           { return s.eventMux }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *Lightfafereum) Protocols() []p2p.Protocol {
	return s.makeProtocols(ClientProtocolVersions)
}

// Start implements node.Service, starting all internal goroutines needed by the
// fafereum protocol implementation.
func (s *Lightfafereum) Start(srvr *p2p.Server) error {
	log.Warn("Light client mode is an experimental feature")
	s.startBloomHandlers(params.BloomBitsBlocksClient)
	s.netRPCService = fafapi.NewPublicNetAPI(srvr, s.networkId)
	// clients are searching for the first advertised protocol in the list
	protocolVersion := AdvertiseProtocolVersions[0]
	s.serverPool.start(srvr, lesTopic(s.blockchain.Genesis().Hash(), protocolVersion))
	s.protocolManager.Start(s.config.LightPeers)
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// fafereum protocol.
func (s *Lightfafereum) Stop() error {
	s.odr.Stop()
	s.bloomIndexer.Close()
	s.chtIndexer.Close()
	s.blockchain.Stop()
	s.protocolManager.Stop()
	s.txPool.Stop()
	s.engine.Close()

	s.eventMux.Stop()

	time.Sleep(time.Millisecond * 200)
	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
