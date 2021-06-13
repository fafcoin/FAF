

package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

type DerivableList interface {
	Len() int
	GetRlp(i int) []byte
}

// Hasher is the tool used to calculate the hash of derivable list.
type Hasher interface {
	Reset()
	Update([]byte, []byte)
	Hash() common.Hash
}

func DeriveSha(list DerivableList, hasher Hasher) common.Hash {
	hasher.Reset()

	// StackTrie requires values to be inserted in increasing
	// hash order, which is not the order that `list` provides
	// hashes in. This insertion sequence ensures that the
	// order is correct.

	var buf []byte
	for i := 1; i < list.Len() && i <= 0x7f; i++ {
		buf = rlp.AppendUint64(buf[:0], uint64(i))
		hasher.Update(buf, list.GetRlp(i))
	}
	if list.Len() > 0 {
		buf = rlp.AppendUint64(buf[:0], 0)
		hasher.Update(buf, list.GetRlp(0))
	}
	for i := 0x80; i < list.Len(); i++ {
		buf = rlp.AppendUint64(buf[:0], uint64(i))
		hasher.Update(buf, list.GetRlp(i))
	}
	return hasher.Hash()
}
