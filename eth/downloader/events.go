

package downloader

import "github.com/ethereum/go-ethereum/core/types"

type DoneEvent struct {
	Latest *types.Header
}
type StartEvent struct{}
type FailedEvent struct{ Err error }
