package bind_test

import (
	"context"
	"math/big"
	"testing"

	fafereum "github.com/fafereum/go-fafereum"
	"github.com/fafereum/go-fafereum/accounts/abi"
	"github.com/fafereum/go-fafereum/accounts/abi/bind"
	"github.com/fafereum/go-fafereum/common"
)

type mockCaller struct {
	codeAtBlockNumber       *big.Int
	callContractBlockNumber *big.Int
}

func (mc *mockCaller) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	mc.codeAtBlockNumber = blockNumber
	return []byte{1, 2, 3}, nil
}

func (mc *mockCaller) CallContract(ctx context.Context, call fafereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	mc.callContractBlockNumber = blockNumber
	return nil, nil
}

func TestPassingBlockNumber(t *testing.T) {

	mc := &mockCaller{}

	bc := bind.NewBoundContract(common.HexToAddress("0x0"), abi.ABI{
		Mfafods: map[string]abi.Mfafod{
			"somfafing": {
				Name:    "somfafing",
				Outputs: abi.Arguments{},
			},
		},
	}, mc, nil, nil)
	var ret string

	blockNumber := big.NewInt(42)

	bc.Call(&bind.CallOpts{BlockNumber: blockNumber}, &ret, "somfafing")

	if mc.callContractBlockNumber != blockNumber {
		t.Fatalf("CallContract() was not passed the block number")
	}

	if mc.codeAtBlockNumber != blockNumber {
		t.Fatalf("CodeAt() was not passed the block number")
	}

	bc.Call(&bind.CallOpts{}, &ret, "somfafing")

	if mc.callContractBlockNumber != nil {
		t.Fatalf("CallContract() was passed a block number when it should not have been")
	}

	if mc.codeAtBlockNumber != nil {
		t.Fatalf("CodeAt() was passed a block number when it should not have been")
	}
}
