// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify


package main

import (
	"fmt"
	"sort"

	"github.com/fafereum/go-fafereum/log"
)

// deployfafstats queries the user for various input on deploying an fafstats
// monitoring server, after which it executes it.
func (w *wizard) deployfafstats() {
	// Select the server to interact with
	server := w.selectServer()
	if server == "" {
		return
	}
	client := w.servers[server]

	// Retrieve any active fafstats configurations from the server
	infos, err := checkfafstats(client, w.network)
	if err != nil {
		infos = &fafstatsInfos{
			port:   80,
			host:   client.server,
			secret: "",
		}
	}
	existed := err == nil

	// Figure out which port to listen on
	//fmt.Println()
	fmt.Printf("Which port should fafstats listen on? (default = %d)\n", infos.port)
	infos.port = w.readDefaultInt(infos.port)

	// Figure which virtual-host to deploy fafstats on
	if infos.host, err = w.ensureVirtualHost(client, infos.port, infos.host); err != nil {
		log.Error("Failed to decide on fafstats host", "err", err)
		return
	}
	// Port and proxy settings retrieved, figure out the secret and boot fafstats
	//fmt.Println()
	if infos.secret == "" {
		fmt.Printf("What should be the secret password for the API? (must not be empty)\n")
		infos.secret = w.readString()
	} else {
		fmt.Printf("What should be the secret password for the API? (default = %s)\n", infos.secret)
		infos.secret = w.readDefaultString(infos.secret)
	}
	// Gather any blacklists to ban from reporting
	if existed {
		//fmt.Println()
		fmt.Printf("Keep existing IP %v blacklist (y/n)? (default = yes)\n", infos.banned)
		if !w.readDefaultYesNo(true) {
			// The user might want to clear the entire list, although generally probably not
			//fmt.Println()
			fmt.Printf("Clear out blacklist and start over (y/n)? (default = no)\n")
			if w.readDefaultYesNo(false) {
				infos.banned = nil
			}
			// Offer the user to explicitly add/remove certain IP addresses
			//fmt.Println()
			//fmt.Println("Which additional IP addresses should be blacklisted?")
			for {
				if ip := w.readIPAddress(); ip != "" {
					infos.banned = append(infos.banned, ip)
					continue
				}
				break
			}
			//fmt.Println()
			//fmt.Println("Which IP addresses should not be blacklisted?")
			for {
				if ip := w.readIPAddress(); ip != "" {
					for i, addr := range infos.banned {
						if ip == addr {
							infos.banned = append(infos.banned[:i], infos.banned[i+1:]...)
							break
						}
					}
					continue
				}
				break
			}
			sort.Strings(infos.banned)
		}
	}
	// Try to deploy the fafstats server on the host
	nocache := false
	if existed {
		//fmt.Println()
		fmt.Printf("Should the fafstats be built from scratch (y/n)? (default = no)\n")
		nocache = w.readDefaultYesNo(false)
	}
	trusted := make([]string, 0, len(w.servers))
	for _, client := range w.servers {
		if client != nil {
			trusted = append(trusted, client.address)
		}
	}
	if out, err := deployfafstats(client, w.network, infos.port, infos.secret, infos.host, trusted, infos.banned, nocache); err != nil {
		log.Error("Failed to deploy fafstats container", "err", err)
		if len(out) > 0 {
			fmt.Printf("%s\n", out)
		}
		return
	}
	// All ok, run a network scan to pick any changes up
	w.networkStats()
}
