// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify

package faf

import (
	"context"
	"math/big"

	"github.com/fafereum/go-fafereum/accounts"
	"github.com/fafereum/go-fafereum/common"
	"github.com/fafereum/go-fafereum/common/math"
	"github.com/fafereum/go-fafereum/core"
	"github.com/fafereum/go-fafereum/core/bloombits"
	"github.com/fafereum/go-fafereum/core/state"
	"github.com/fafereum/go-fafereum/core/types"
	"github.com/fafereum/go-fafereum/core/vm"
	"github.com/fafereum/go-fafereum/faf/downloader"
	"github.com/fafereum/go-fafereum/faf/gasprice"
	"github.com/fafereum/go-fafereum/fafdb"
	"github.com/fafereum/go-fafereum/event"
	"github.com/fafereum/go-fafereum/params"
	"github.com/fafereum/go-fafereum/rpc"
)

// fafAPIBackend implements fafapi.Backend for full nodes
type fafAPIBackend struct {
	faf *fafereum
	gpo *gasprice.Oracle
}

// ChainConfig returns the active chain configuration.
func (b *fafAPIBackend) ChainConfig() *params.ChainConfig {
	return b.faf.chainConfig
}

func (b *fafAPIBackend) CurrentBlock() *types.Block {
	return b.faf.blockchain.CurrentBlock()
}

func (b *fafAPIBackend) Sfafead(number uint64) {
	b.faf.protocolManager.downloader.Cancel()
	b.faf.blockchain.Sfafead(number)
}

func (b *fafAPIBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.faf.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.faf.blockchain.CurrentBlock().Header(), nil
	}
	return b.faf.blockchain.GfafeaderByNumber(uint64(blockNr)), nil
}

func (b *fafAPIBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.faf.blockchain.GfafeaderByHash(hash), nil
}

func (b *fafAPIBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.faf.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.faf.blockchain.CurrentBlock(), nil
	}
	return b.faf.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

func (b *fafAPIBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block, state := b.faf.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := b.faf.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *fafAPIBackend) GetBlock(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.faf.blockchain.GetBlockByHash(hash), nil
}

func (b *fafAPIBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return b.faf.blockchain.GetReceiptsByHash(hash), nil
}

func (b *fafAPIBackend) GetLogs(ctx context.Context, hash common.Hash) ([][]*types.Log, error) {
	receipts := b.faf.blockchain.GetReceiptsByHash(hash)
	if receipts == nil {
		return nil, nil
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

func (b *fafAPIBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.faf.blockchain.GetTdByHash(blockHash)
}

func (b *fafAPIBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	vmError := func() error { return nil }

	context := core.NewEVMContext(msg, header, b.faf.BlockChain(), nil)
	return vm.NewEVM(context, state, b.faf.chainConfig, *b.faf.blockchain.GetVMConfig()), vmError, nil
}

func (b *fafAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.faf.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *fafAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.faf.BlockChain().SubscribeChainEvent(ch)
}

func (b *fafAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.faf.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *fafAPIBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.faf.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *fafAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.faf.BlockChain().SubscribeLogsEvent(ch)
}

func (b *fafAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.faf.txPool.AddLocal(signedTx)
}

func (b *fafAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.faf.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *fafAPIBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.faf.txPool.Get(hash)
}

func (b *fafAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.faf.txPool.State().GetNonce(addr), nil
}

func (b *fafAPIBackend) Stats() (pending int, queued int) {
	return b.faf.txPool.Stats()
}

func (b *fafAPIBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.faf.TxPool().Content()
}

func (b *fafAPIBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.faf.TxPool().SubscribeNewTxsEvent(ch)
}

func (b *fafAPIBackend) Downloader() *downloader.Downloader {
	return b.faf.Downloader()
}

func (b *fafAPIBackend) ProtocolVersion() int {
	return b.faf.fafVersion()
}

func (b *fafAPIBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *fafAPIBackend) ChainDb() fafdb.Database {
	return b.faf.ChainDb()
}

func (b *fafAPIBackend) EventMux() *event.TypeMux {
	return b.faf.EventMux()
}

func (b *fafAPIBackend) AccountManager() *accounts.Manager {
	return b.faf.AccountManager()
}

func (b *fafAPIBackend) RPCGasCap() *big.Int {
	return b.faf.config.RPCGasCap
}

func (b *fafAPIBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.faf.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *fafAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.faf.bloomRequests)
	}
}
