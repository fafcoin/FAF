

package les

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/fafdb"
	"github.com/ethereum/go-ethereum/light"
)

// LesOdr implements light.OdrBackend
type LesOdr struct {
	db                                         fafdb.Database
	indexerConfig                              *light.IndexerConfig
	chtIndexer, bloomTrieIndexer, bloomIndexer *core.ChainIndexer
	retriever                                  *retrieveManager
	stop                                       chan struct{}
}

func NewLesOdr(db fafdb.Database, config *light.IndexerConfig, retriever *retrieveManager) *LesOdr {
	return &LesOdr{
		db:            db,
		indexerConfig: config,
		retriever:     retriever,
		stop:          make(chan struct{}),
	}
}

// Stop cancels all pending retrievals
func (odr *LesOdr) Stop() {
	close(odr.stop)
}

// Database returns the backing database
func (odr *LesOdr) Database() fafdb.Database {
	return odr.db
}

// SetIndexers adds the necessary chain indexers to the ODR backend
func (odr *LesOdr) SetIndexers(chtIndexer, bloomTrieIndexer, bloomIndexer *core.ChainIndexer) {
	odr.chtIndexer = chtIndexer
	odr.bloomTrieIndexer = bloomTrieIndexer
	odr.bloomIndexer = bloomIndexer
}

// ChtIndexer returns the CHT chain indexer
func (odr *LesOdr) ChtIndexer() *core.ChainIndexer {
	return odr.chtIndexer
}

// BloomTrieIndexer returns the bloom trie chain indexer
func (odr *LesOdr) BloomTrieIndexer() *core.ChainIndexer {
	return odr.bloomTrieIndexer
}

// BloomIndexer returns the bloombits chain indexer
func (odr *LesOdr) BloomIndexer() *core.ChainIndexer {
	return odr.bloomIndexer
}

// IndexerConfig returns the indexer config.
func (odr *LesOdr) IndexerConfig() *light.IndexerConfig {
	return odr.indexerConfig
}

const (
	MsgBlockHeaders = iota
	MsgBlockBodies
	MsgCode
	MsgReceipts
	MsgProofsV2
	MsgHelperTrieProofs
	MsgTxStatus
)

// Msg encodes a LES message that delivers reply data for a request
type Msg struct {
	MsgType int
	ReqID   uint64
	Obj     interface{}
}

// Retrieve tries to fetch an object from the LES network.
// If the network retrieval was successful, it stores the object in local db.
func (odr *LesOdr) Retrieve(ctx context.Context, req light.OdrRequest) (err error) {
	lreq := LesRequest(req)

	reqID := genReqID()
	rq := &distReq{
		getCost: func(dp distPeer) uint64 {
			return lreq.GetCost(dp.(*serverPeer))
		},
		canSend: func(dp distPeer) bool {
			p := dp.(*serverPeer)
			if !p.onlyAnnounce {
				return lreq.CanSend(p)
			}
			return false
		},
		request: func(dp distPeer) func() {
			p := dp.(*serverPeer)
			cost := lreq.GetCost(p)
			p.fcServer.QueuedRequest(reqID, cost)
			return func() { lreq.Request(reqID, p) }
		},
	}

	defer func(sent mclock.AbsTime) {
		if err != nil {
			return
		}
		requestRTT.Update(time.Duration(mclock.Now() - sent))
	}(mclock.Now())

	if err := odr.retriever.retrieve(ctx, reqID, rq, func(p distPeer, msg *Msg) error { return lreq.Validate(odr.db, msg) }, odr.stop); err != nil {
		return err
	}
	req.StoreResult(odr.db)
	return nil
}
