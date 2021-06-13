

package tests

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
)

func TestVM(t *testing.T) {
	t.Parallel()
	vmt := new(testMatcher)
	vmt.slow("^vmPerformance")
	vmt.fails("^vmSystemOperationsTest.json/createNameRegistrator$", "fails without parallel execution")

	vmt.walk(t, vmTestDir, func(t *testing.T, name string, test *VMTest) {
		withTrace(t, test.json.Exec.GasLimit, func(vmconfig vm.Config) error {
			return vmt.checkFailure(t, name+"/trie", test.Run(vmconfig, false))
		})
		withTrace(t, test.json.Exec.GasLimit, func(vmconfig vm.Config) error {
			return vmt.checkFailure(t, name+"/snap", test.Run(vmconfig, true))
		})
	})
}
