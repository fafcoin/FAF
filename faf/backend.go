// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

// Package faf implements the fafereum protocol.
package faf

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/fafereum/go-fafereum/accounts"
	"github.com/fafereum/go-fafereum/common"
	"github.com/fafereum/go-fafereum/common/hexutil"
	"github.com/fafereum/go-fafereum/consensus"
	"github.com/fafereum/go-fafereum/consensus/clique"
	"github.com/fafereum/go-fafereum/consensus/fafash"
	"github.com/fafereum/go-fafereum/core"
	"github.com/fafereum/go-fafereum/core/bloombits"
	"github.com/fafereum/go-fafereum/core/rawdb"
	"github.com/fafereum/go-fafereum/core/types"
	"github.com/fafereum/go-fafereum/core/vm"
	"github.com/fafereum/go-fafereum/faf/downloader"
	"github.com/fafereum/go-fafereum/faf/filters"
	"github.com/fafereum/go-fafereum/faf/gasprice"
	"github.com/fafereum/go-fafereum/fafdb"
	"github.com/fafereum/go-fafereum/event"
	"github.com/fafereum/go-fafereum/internal/fafapi"
	"github.com/fafereum/go-fafereum/log"
	"github.com/fafereum/go-fafereum/miner"
	"github.com/fafereum/go-fafereum/node"
	"github.com/fafereum/go-fafereum/p2p"
	"github.com/fafereum/go-fafereum/params"
	"github.com/fafereum/go-fafereum/rlp"
	"github.com/fafereum/go-fafereum/rpc"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
}

// fafereum implements the fafereum full node service.
type fafereum struct {
	config      *Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan chan bool // Channel for shutting down the fafereum

	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer

	// DB interfaces
	chainDb fafdb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	APIBackend *fafAPIBackend

	miner     *miner.Miner
	gasPrice  *big.Int
	faferbase common.Address

	networkID     uint64
	netRPCService *fafapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and faferbase)
}

func (s *fafereum) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}

// New creates a new fafereum object (including the
// initialisation of the common fafereum object)
func New(ctx *node.ServiceContext, config *Config) (*fafereum, error) {
	// Ensure configuration values are compatible and sane
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run faf.fafereum in light sync mode, use les.Lightfafereum")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	if config.MinerGasPrice == nil || config.MinerGasPrice.Cmp(common.Big0) <= 0 {
		log.Warn("Sanitizing invalid miner gas price", "provided", config.MinerGasPrice, "updated", DefaultConfig.MinerGasPrice)
		config.MinerGasPrice = new(big.Int).Set(DefaultConfig.MinerGasPrice)
	}
	// Assemble the fafereum object
	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlockWithOverride(chainDb, config.Genesis, config.ConstantinopleOverride)
	////fmt.Println(chainConfig,genesisHash,genesisErr)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	faf := &fafereum{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, chainConfig, &config.fafash, config.MinerNotify, config.MinerNoverify, chainDb),
		shutdownChan:   make(chan bool),
		networkID:      config.NetworkId,
		gasPrice:       config.MinerGasPrice,
		faferbase:      config.faferbase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks, params.BloomConfirms),
	}
	log.Info("Initialising VIMM protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	if !config.SkipBcVersionCheck {
		bcVersion := rawdb.ReadDatabaseVersion(chainDb)
		if bcVersion != nil && *bcVersion > core.BlockChainVersion {
			return nil, fmt.Errorf("database version is v%d, Gfaf %s only supports v%d", *bcVersion, params.VersionWithMeta, core.BlockChainVersion)
		} else if bcVersion != nil && *bcVersion < core.BlockChainVersion {
			log.Warn("Upgrade blockchain database version", "from", *bcVersion, "to", core.BlockChainVersion)
		}
		rawdb.WriteDatabaseVersion(chainDb, core.BlockChainVersion)
	}
	var (
		vmConfig = vm.Config{
			EnablePreimageRecording: config.EnablePreimageRecording,
			EWASMInterpreter:        config.EWASMInterpreter,
			EVMInterpreter:          config.EVMInterpreter,
		}
		cacheConfig = &core.CacheConfig{Disabled: config.NoPruning, TrieCleanLimit: config.TrieCleanCache, TrieDirtyLimit: config.TrieDirtyCache, TrieTimeLimit: config.TrieTimeout}
	)
	faf.blockchain, err = core.NewBlockChain(chainDb, cacheConfig, faf.chainConfig, faf.engine, vmConfig, faf.shouldPreserve)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		faf.blockchain.Sfafead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	faf.bloomIndexer.Start(faf.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	faf.txPool = core.NewTxPool(config.TxPool, faf.chainConfig, faf.blockchain)

	if faf.protocolManager, err = NewProtocolManager(faf.chainConfig, config.SyncMode, config.NetworkId, faf.eventMux, faf.txPool, faf.engine, faf.blockchain, chainDb, config.Whitelist); err != nil {
		return nil, err
	}
	//会创建一个Miner实例 用与挖矿
	faf.miner = miner.New(faf, faf.chainConfig, faf.EventMux(), faf.engine, config.MinerRecommit, config.MinerGasFloor, config.MinerGasCeil, faf.isLocalBlock)
	faf.miner.SetExtra(makeExtraData(config.MinerExtraData))

	faf.APIBackend = &fafAPIBackend{faf, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.MinerGasPrice
	}
	faf.APIBackend.gpo = gasprice.NewOracle(faf.APIBackend, gpoParams)

	return faf, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"gfaf",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (fafdb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*fafdb.LDBDatabase); ok {
		db.Meter("faf/db/chaindata/")
	}
	return db, nil
}

// CreateConsensusEngine creates the required type of consensus engine instance for an fafereum service
func CreateConsensusEngine(ctx *node.ServiceContext, chainConfig *params.ChainConfig, config *fafash.Config, notify []string, noverify bool, db fafdb.Database) consensus.Engine {
	// If proof-of-authority is requested, set it up
	if chainConfig.Clique != nil {
		return clique.New(chainConfig.Clique, db)
	}
	// Otherwise assume proof-of-work
	switch config.PowMode {
	case fafash.ModeFake:
		log.Warn("fafash used in fake mode")
		return fafash.NewFaker()
	case fafash.ModeTest:
		log.Warn("fafash used in test mode")
		return fafash.NewTester(nil, noverify)
	case fafash.ModeShared:
		log.Warn("fafash used in shared mode")
		return fafash.NewShared()
	default:
		engine := fafash.New(fafash.Config{
			CacheDir:       ctx.ResolvePath(config.CacheDir),
			CachesInMem:    config.CachesInMem,
			CachesOnDisk:   config.CachesOnDisk,
			DatasetDir:     config.DatasetDir,
			DatasetsInMem:  config.DatasetsInMem,
			DatasetsOnDisk: config.DatasetsOnDisk,
		}, notify, noverify)
		engine.SetThreads(-1) // Disable CPU mining
		return engine
	}
}

// APIs return the collection of RPC services the fafereum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *fafereum) APIs() []rpc.API {
	apis := fafapi.GetAPIs(s.APIBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "faf",
			Version:   "1.0",
			Service:   NewPublicfafereumAPI(s),
			Public:    true,
		}, {
			Namespace: "faf",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "faf",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "faf",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.APIBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *fafereum) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *fafereum) faferbase() (eb common.Address, err error) {
	s.lock.RLock()
	faferbase := s.faferbase
	s.lock.RUnlock()

	if faferbase != (common.Address{}) {
		return faferbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			faferbase := accounts[0].Address

			s.lock.Lock()
			s.faferbase = faferbase
			s.lock.Unlock()

			log.Info("faferbase automatically configured", "address", faferbase)
			return faferbase, nil
		}
	}
	return common.Address{}, fmt.Errorf("faferbase must be explicitly specified")
}

// isLocalBlock checks whfafer the specified block is mined
// by local miner accounts.
//
// We regard two types of accounts as local miner account: faferbase
// and accounts specified via `txpool.locals` flag.
func (s *fafereum) isLocalBlock(block *types.Block) bool {
	author, err := s.engine.Author(block.Header())
	if err != nil {
		log.Warn("Failed to retrieve block author", "number", block.NumberU64(), "hash", block.Hash(), "err", err)
		return false
	}
	// Check whfafer the given address is faferbase.
	s.lock.RLock()
	faferbase := s.faferbase
	s.lock.RUnlock()
	if author == faferbase {
		return true
	}
	// Check whfafer the given address is specified by `txpool.local`
	// CLI flag.
	for _, account := range s.config.TxPool.Locals {
		if account == author {
			return true
		}
	}
	return false
}

// shouldPreserve checks whfafer we should preserve the given block
// during the chain reorg depending on whfafer the author of block
// is a local account.
func (s *fafereum) shouldPreserve(block *types.Block) bool {
	// The reason we need to disable the self-reorg preserving for clique
	// is it can be probable to introduce a deadlock.
	//
	// e.g. If there are 7 available signers
	//
	// r1   A
	// r2     B
	// r3       C
	// r4         D
	// r5   A      [X] F G
	// r6    [X]
	//
	// In the round5, the inturn signer E is offline, so the worst case
	// is A, F and G sign the block of round5 and reject the block of opponents
	// and in the round6, the last available signer B is offline, the whole
	// network is stuck.
	if _, ok := s.engine.(*clique.Clique); ok {
		return false
	}
	return s.isLocalBlock(block)
}

// Setfaferbase sets the mining reward address.
func (s *fafereum) Setfaferbase(faferbase common.Address) {
	s.lock.Lock()
	s.faferbase = faferbase
	s.lock.Unlock()

	s.miner.Setfaferbase(faferbase)
}

// StartMining starts the miner with the given number of CPU threads. If mining
// is already running, this mfafod adjust the number of threads allowed to use
// and updates the minimum price required by the transaction pool.
func (s *fafereum) StartMining(threads int) error {
	// Update the thread count within the consensus engine
	type threaded interface {
		SetThreads(threads int)
	}
	if th, ok := s.engine.(threaded); ok {
		log.Info("Updated mining threads", "threads", threads)
		if threads == 0 {
			threads = -1 // Disable the miner from within
		}
		th.SetThreads(threads)
	}
	// If the miner was not running, initialize it
	if !s.IsMining() {
		// Propagate the initial price point to the transaction pool
		s.lock.RLock()
		price := s.gasPrice
		s.lock.RUnlock()
		s.txPool.SetGasPrice(price)

		// Configure the local mining address
		eb, err := s.faferbase()
		if err != nil {
			log.Error("Cannot start mining without faferbase", "err", err)
			return fmt.Errorf("faferbase missing: %v", err)
		}
		if clique, ok := s.engine.(*clique.Clique); ok {
			wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
			if wallet == nil || err != nil {
				log.Error("faferbase account unavailable locally", "err", err)
				return fmt.Errorf("signer missing: %v", err)
			}
			clique.Authorize(eb, wallet.SignHash)
		}
		// If mining is started, we can disable the transaction rejection mechanism
		// introduced to speed sync times.
		atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)

		go s.miner.Start(eb)
	}
	return nil
}

// StopMining terminates the miner, both at the consensus engine level as well as
// at the block creation level.
func (s *fafereum) StopMining() {
	// Update the thread count within the consensus engine
	type threaded interface {
		SetThreads(threads int)
	}
	if th, ok := s.engine.(threaded); ok {
		th.SetThreads(-1)
	}
	// Stop the block creating itself
	s.miner.Stop()
}

func (s *fafereum) IsMining() bool      { return s.miner.Mining() }
func (s *fafereum) Miner() *miner.Miner { return s.miner }

func (s *fafereum) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *fafereum) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *fafereum) TxPool() *core.TxPool               { return s.txPool }
func (s *fafereum) EventMux() *event.TypeMux           { return s.eventMux }
func (s *fafereum) Engine() consensus.Engine           { return s.engine }
func (s *fafereum) ChainDb() fafdb.Database            { return s.chainDb }
func (s *fafereum) IsListening() bool                  { return true } // Always listening
func (s *fafereum) fafVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *fafereum) NetVersion() uint64                 { return s.networkID }
func (s *fafereum) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *fafereum) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	}
	return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
}

// Start implements node.Service, starting all internal goroutines needed by the
// fafereum protocol implementation.
func (s *fafereum) Start(srvr *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	s.startBloomHandlers(params.BloomBitsBlocks)

	// Start the RPC service
	s.netRPCService = fafapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		if s.config.LightPeers >= srvr.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", s.config.LightPeers, srvr.MaxPeers)
		}
		maxPeers -= s.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// fafereum protocol.
func (s *fafereum) Stop() error {
	s.bloomIndexer.Close()
	s.blockchain.Stop()
	s.engine.Close()
	s.protocolManager.Stop()
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)
	return nil
}
