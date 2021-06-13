

package snapshot

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
)

// hashes is a helper to implement sort.Interface.
type hashes []common.Hash

// Len is the number of elements in the collection.
func (hs hashes) Len() int { return len(hs) }

// Less reports whether the element with index i should sort before the element
// with index j.
func (hs hashes) Less(i, j int) bool { return bytes.Compare(hs[i][:], hs[j][:]) < 0 }

// Swap swaps the elements with indexes i and j.
func (hs hashes) Swap(i, j int) { hs[i], hs[j] = hs[j], hs[i] }
