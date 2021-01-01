// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify


package main

import (
	"crypto/rand"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fafereum/go-fafereum/params"
)

const (
	ipcAPIs  = "admin:1.0 debug:1.0 faf:1.0 fafash:1.0 miner:1.0 net:1.0 personal:1.0 rpc:1.0 shh:1.0 txpool:1.0 web3:1.0"
	httpAPIs = "faf:1.0 net:1.0 rpc:1.0 web3:1.0"
)

// Tests that a node embedded within a console can be started up properly and
// then terminated by closing the input stream.
func TestConsoleWelcome(t *testing.T) {
	coinbase := "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"

	// Start a gfaf console, make sure it's cleaned up and terminate the console
	gfaf := runGfaf(t,
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none",
		"--faferbase", coinbase, "--shh",
		"console")

	// Gather all the infos the welcome message needs to contain
	gfaf.SetTemplateFunc("goos", func() string { return runtime.GOOS })
	gfaf.SetTemplateFunc("goarch", func() string { return runtime.GOARCH })
	gfaf.SetTemplateFunc("gover", runtime.Version)
	gfaf.SetTemplateFunc("gfafver", func() string { return params.VersionWithMeta })
	gfaf.SetTemplateFunc("niltime", func() string { return time.Unix(0, 0).Format(time.RFC1123) })
	gfaf.SetTemplateFunc("apis", func() string { return ipcAPIs })

	// Verify the actual welcome message to the required template
	gfaf.Expect(`
Welcome to the Gfaf JavaScript console!

instance: Gfaf/v{{gfafver}}/{{goos}}-{{goarch}}/{{gover}}
coinbase: {{.faferbase}}
at block: 0 ({{niltime}})
 datadir: {{.Datadir}}
 modules: {{apis}}

> {{.InputLine "exit"}}
`)
	gfaf.ExpectExit()
}

// Tests that a console can be attached to a running node via various means.
func TestIPCAttachWelcome(t *testing.T) {
	// Configure the instance for IPC attachement
	coinbase := "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"
	var ipc string
	if runtime.GOOS == "windows" {
		ipc = `\\.\pipe\gfaf` + strconv.Itoa(trulyRandInt(100000, 999999))
	} else {
		ws := tmpdir(t)
		defer os.RemoveAll(ws)
		ipc = filepath.Join(ws, "gfaf.ipc")
	}
	// Note: we need --shh because testAttachWelcome checks for default
	// list of ipc modules and shh is included there.
	gfaf := runGfaf(t,
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none",
		"--faferbase", coinbase, "--shh", "--ipcpath", ipc)

	time.Sleep(2 * time.Second) // Simple way to wait for the RPC endpoint to open
	testAttachWelcome(t, gfaf, "ipc:"+ipc, ipcAPIs)

	gfaf.Interrupt()
	gfaf.ExpectExit()
}

func TestHTTPAttachWelcome(t *testing.T) {
	coinbase := "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"
	port := strconv.Itoa(trulyRandInt(1024, 65536)) // Yeah, sometimes this will fail, sorry :P
	gfaf := runGfaf(t,
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none",
		"--faferbase", coinbase, "--rpc", "--rpcport", port)

	time.Sleep(2 * time.Second) // Simple way to wait for the RPC endpoint to open
	testAttachWelcome(t, gfaf, "http://localhost:"+port, httpAPIs)

	gfaf.Interrupt()
	gfaf.ExpectExit()
}

func TestWSAttachWelcome(t *testing.T) {
	coinbase := "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"
	port := strconv.Itoa(trulyRandInt(1024, 65536)) // Yeah, sometimes this will fail, sorry :P

	gfaf := runGfaf(t,
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none",
		"--faferbase", coinbase, "--ws", "--wsport", port)

	time.Sleep(2 * time.Second) // Simple way to wait for the RPC endpoint to open
	testAttachWelcome(t, gfaf, "ws://localhost:"+port, httpAPIs)

	gfaf.Interrupt()
	gfaf.ExpectExit()
}

func testAttachWelcome(t *testing.T, gfaf *testgfaf, endpoint, apis string) {
	// Attach to a running gfaf note and terminate immediately
	attach := runGfaf(t, "attach", endpoint)
	defer attach.ExpectExit()
	attach.CloseStdin()

	// Gather all the infos the welcome message needs to contain
	attach.SetTemplateFunc("goos", func() string { return runtime.GOOS })
	attach.SetTemplateFunc("goarch", func() string { return runtime.GOARCH })
	attach.SetTemplateFunc("gover", runtime.Version)
	attach.SetTemplateFunc("gfafver", func() string { return params.VersionWithMeta })
	attach.SetTemplateFunc("faferbase", func() string { return gfaf.faferbase })
	attach.SetTemplateFunc("niltime", func() string { return time.Unix(0, 0).Format(time.RFC1123) })
	attach.SetTemplateFunc("ipc", func() bool { return strings.HasPrefix(endpoint, "ipc") })
	attach.SetTemplateFunc("datadir", func() string { return gfaf.Datadir })
	attach.SetTemplateFunc("apis", func() string { return apis })

	// Verify the actual welcome message to the required template
	attach.Expect(`
Welcome to the Gfaf JavaScript console!

instance: Gfaf/v{{gfafver}}/{{goos}}-{{goarch}}/{{gover}}
coinbase: {{faferbase}}
at block: 0 ({{niltime}}){{if ipc}}
 datadir: {{datadir}}{{end}}
 modules: {{apis}}

> {{.InputLine "exit" }}
`)
	attach.ExpectExit()
}

// trulyRandInt generates a crypto random integer used by the console tests to
// not clash network ports with other tests running cocurrently.
func trulyRandInt(lo, hi int) int {
	num, _ := rand.Int(rand.Reader, big.NewInt(int64(hi-lo)))
	return int(num.Int64()) + lo
}
