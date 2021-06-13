

// Contains initialization code for the mbile library.

package geth

import (
	"os"
	"runtime"

	"github.com/ethereum/go-ethereum/log"
)

func init() {
	// Initialize the logger
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(false))))

	// Initialize the goroutine count
	runtime.GOMAXPROCS(runtime.NumCPU())
}
