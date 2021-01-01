// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify


package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/fafereum/go-fafereum/accounts/keystore"
	"github.com/fafereum/go-fafereum/common"
	"github.com/fafereum/go-fafereum/log"
)

// deployNode creates a new node configuration based on some user input.
func (w *wizard) deployNode(boot bool) {
	// Do some sanity check before the user wastes time on input
	if w.conf.Genesis == nil {
		log.Error("No genesis block configured")
		return
	}
	if w.conf.fafstats == "" {
		log.Error("No fafstats server configured")
		return
	}
	// Select the server to interact with
	server := w.selectServer()
	if server == "" {
		return
	}
	client := w.servers[server]

	// Retrieve any active node configurations from the server
	infos, err := checkNode(client, w.network, boot)
	if err != nil {
		if boot {
			infos = &nodeInfos{port: 30303, peersTotal: 512, peersLight: 256}
		} else {
			infos = &nodeInfos{port: 30303, peersTotal: 50, peersLight: 0, gasTarget: 7.5, gasLimit: 10, gasPrice: 1}
		}
	}
	existed := err == nil

	infos.genesis, _ = json.MarshalIndent(w.conf.Genesis, "", "  ")
	infos.network = w.conf.Genesis.Config.ChainID.Int64()

	// Figure out where the user wants to store the persistent data
	//fmt.Println()
	if infos.datadir == "" {
		fmt.Printf("Where should data be stored on the remote machine?\n")
		infos.datadir = w.readString()
	} else {
		fmt.Printf("Where should data be stored on the remote machine? (default = %s)\n", infos.datadir)
		infos.datadir = w.readDefaultString(infos.datadir)
	}
	if w.conf.Genesis.Config.fafash != nil && !boot {
		//fmt.Println()
		if infos.fafashdir == "" {
			fmt.Printf("Where should the fafash mining DAGs be stored on the remote machine?\n")
			infos.fafashdir = w.readString()
		} else {
			fmt.Printf("Where should the fafash mining DAGs be stored on the remote machine? (default = %s)\n", infos.fafashdir)
			infos.fafashdir = w.readDefaultString(infos.fafashdir)
		}
	}
	// Figure out which port to listen on
	//fmt.Println()
	fmt.Printf("Which TCP/UDP port to listen on? (default = %d)\n", infos.port)
	infos.port = w.readDefaultInt(infos.port)

	// Figure out how many peers to allow (different based on node type)
	//fmt.Println()
	fmt.Printf("How many peers to allow connecting? (default = %d)\n", infos.peersTotal)
	infos.peersTotal = w.readDefaultInt(infos.peersTotal)

	// Figure out how many light peers to allow (different based on node type)
	//fmt.Println()
	fmt.Printf("How many light peers to allow connecting? (default = %d)\n", infos.peersLight)
	infos.peersLight = w.readDefaultInt(infos.peersLight)

	// Set a proper name to report on the stats page
	//fmt.Println()
	if infos.fafstats == "" {
		fmt.Printf("What should the node be called on the stats page?\n")
		infos.fafstats = w.readString() + ":" + w.conf.fafstats
	} else {
		fmt.Printf("What should the node be called on the stats page? (default = %s)\n", infos.fafstats)
		infos.fafstats = w.readDefaultString(infos.fafstats) + ":" + w.conf.fafstats
	}
	// If the node is a miner/signer, load up needed credentials
	if !boot {
		if w.conf.Genesis.Config.fafash != nil {
			// fafash based miners only need an faferbase to mine against
			//fmt.Println()
			if infos.faferbase == "" {
				fmt.Printf("What address should the miner use?\n")
				for {
					if address := w.readAddress(); address != nil {
						infos.faferbase = address.Hex()
						break
					}
				}
			} else {
				fmt.Printf("What address should the miner use? (default = %s)\n", infos.faferbase)
				infos.faferbase = w.readDefaultAddress(common.HexToAddress(infos.faferbase)).Hex()
			}
		} else if w.conf.Genesis.Config.Clique != nil {
			// If a previous signer was already set, offer to reuse it
			if infos.keyJSON != "" {
				if key, err := keystore.DecryptKey([]byte(infos.keyJSON), infos.keyPass); err != nil {
					infos.keyJSON, infos.keyPass = "", ""
				} else {
					//fmt.Println()
					fmt.Printf("Reuse previous (%s) signing account (y/n)? (default = yes)\n", key.Address.Hex())
					if !w.readDefaultYesNo(true) {
						infos.keyJSON, infos.keyPass = "", ""
					}
				}
			}
			// Clique based signers need a keyfile and unlock password, ask if unavailable
			if infos.keyJSON == "" {
				//fmt.Println()
				//fmt.Println("Please paste the signer's key JSON:")
				infos.keyJSON = w.readJSON()

				//fmt.Println()
				//fmt.Println("What's the unlock password for the account? (won't be echoed)")
				infos.keyPass = w.readPassword()

				if _, err := keystore.DecryptKey([]byte(infos.keyJSON), infos.keyPass); err != nil {
					log.Error("Failed to decrypt key with given passphrase")
					return
				}
			}
		}
		// Establish the gas dynamics to be enforced by the signer
		//fmt.Println()
		fmt.Printf("What gas limit should empty blocks target (MGas)? (default = %0.3f)\n", infos.gasTarget)
		infos.gasTarget = w.readDefaultFloat(infos.gasTarget)

		//fmt.Println()
		fmt.Printf("What gas limit should full blocks target (MGas)? (default = %0.3f)\n", infos.gasLimit)
		infos.gasLimit = w.readDefaultFloat(infos.gasLimit)

		//fmt.Println()
		fmt.Printf("What gas price should the signer require (GWei)? (default = %0.3f)\n", infos.gasPrice)
		infos.gasPrice = w.readDefaultFloat(infos.gasPrice)
	}
	// Try to deploy the full node on the host
	nocache := false
	if existed {
		//fmt.Println()
		fmt.Printf("Should the node be built from scratch (y/n)? (default = no)\n")
		nocache = w.readDefaultYesNo(false)
	}
	if out, err := deployNode(client, w.network, w.conf.bootnodes, infos, nocache); err != nil {
		log.Error("Failed to deploy fafereum node container", "err", err)
		if len(out) > 0 {
			fmt.Printf("%s\n", out)
		}
		return
	}
	// All ok, run a network scan to pick any changes up
	log.Info("Waiting for node to finish booting")
	time.Sleep(3 * time.Second)

	w.networkStats()
}
