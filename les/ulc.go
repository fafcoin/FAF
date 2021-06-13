

package les

import (
	"errors"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

type ulc struct {
	keys     map[string]bool
	fraction int
}

// newULC creates and returns an ultra light client instance.
func newULC(servers []string, fraction int) (*ulc, error) {
	keys := make(map[string]bool)
	for _, id := range servers {
		node, err := enode.Parse(enode.ValidSchemes, id)
		if err != nil {
			log.Warn("Failed to parse trusted server", "id", id, "err", err)
			continue
		}
		keys[node.ID().String()] = true
	}
	if len(keys) == 0 {
		return nil, errors.New("no trusted servers")
	}
	return &ulc{
		keys:     keys,
		fraction: fraction,
	}, nil
}

// trusted return an indicator that whether the specified peer is trusted.
func (u *ulc) trusted(p enode.ID) bool {
	return u.keys[p.String()]
}
