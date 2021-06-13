

package les

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/fafdb"
	"github.com/ethereum/go-ethereum/log"
)

// pruner is responsible for pruning historical light chain data.
type pruner struct {
	db       fafdb.Database
	indexers []*core.ChainIndexer
	closeCh  chan struct{}
	wg       sync.WaitGroup
}

// newPruner returns a light chain pruner instance.
func newPruner(db fafdb.Database, indexers ...*core.ChainIndexer) *pruner {
	pruner := &pruner{
		db:       db,
		indexers: indexers,
		closeCh:  make(chan struct{}),
	}
	pruner.wg.Add(1)
	go pruner.loop()
	return pruner
}

// close notifies all background goroutines belonging to pruner to exit.
func (p *pruner) close() {
	close(p.closeCh)
	p.wg.Wait()
}

// loop periodically queries the status of chain indexers and prunes useless
// historical chain data. Notably, whenever Geth restarts, it will iterate
// all historical sections even they don't exist at all(below checkpoint) so
// that light client can prune cached chain data that was ODRed after pruning
// that section.
func (p *pruner) loop() {
	defer p.wg.Done()

	// cleanTicker is the ticker used to trigger a history clean 2 times a day.
	var cleanTicker = time.NewTicker(12 * time.Hour)

	// pruning finds the sections that have been processed by all indexers
	// and deletes all historical chain data.
	// Note, if some indexers don't support pruning(e.g. eth.BloomIndexer),
	// pruning operations can be silently ignored.
	pruning := func() {
		min := uint64(math.MaxUint64)
		for _, indexer := range p.indexers {
			sections, _, _ := indexer.Sections()
			if sections < min {
				min = sections
			}
		}
		// Always keep the latest section data in database.
		if min < 2 || len(p.indexers) == 0 {
			return
		}
		for _, indexer := range p.indexers {
			if err := indexer.Prune(min - 2); err != nil {
				log.Debug("Failed to prune historical data", "err", err)
				return
			}
		}
		p.db.Compact(nil, nil) // Compact entire database, ensure all removed data are deleted.
	}
	for {
		pruning()
		select {
		case <-cleanTicker.C:
		case <-p.closeCh:
			return
		}
	}
}
