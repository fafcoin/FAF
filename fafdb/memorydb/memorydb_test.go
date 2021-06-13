

package memorydb

import (
	"testing"

	"github.com/ethereum/go-ethereum/fafdb"
	"github.com/ethereum/go-ethereum/fafdb/dbtest"
)

func TestMemoryDB(t *testing.T) {
	t.Run("DatabaseSuite", func(t *testing.T) {
		dbtest.TestDatabaseSuite(t, func() fafdb.KeyValueStore {
			return New()
		})
	})
}
