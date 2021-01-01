// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify


package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/fafereum/go-fafereum/cmd/utils"
	"github.com/fafereum/go-fafereum/console"
	"github.com/fafereum/go-fafereum/crypto"
	"gopkg.in/urfave/cli.v1"
)

// promptPassphrase prompts the user for a passphrase.  Set confirmation to true
// to require the user to confirm the passphrase.
func promptPassphrase(confirmation bool) string {
	passphrase, err := console.Stdin.PromptPassword("Passphrase: ")
	if err != nil {
		utils.Fatalf("Failed to read passphrase: %v", err)
	}

	if confirmation {
		confirm, err := console.Stdin.PromptPassword("Repeat passphrase: ")
		if err != nil {
			utils.Fatalf("Failed to read passphrase confirmation: %v", err)
		}
		if passphrase != confirm {
			utils.Fatalf("Passphrases do not match")
		}
	}

	return passphrase
}

// getPassphrase obtains a passphrase given by the user.  It first checks the
// --passfile command line flag and ultimately prompts the user for a
// passphrase.
func getPassphrase(ctx *cli.Context) string {
	// Look for the --passwordfile flag.
	passphraseFile := ctx.String(passphraseFlag.Name)
	if passphraseFile != "" {
		content, err := ioutil.ReadFile(passphraseFile)
		if err != nil {
			utils.Fatalf("Failed to read passphrase file '%s': %v",
				passphraseFile, err)
		}
		return strings.TrimRight(string(content), "\r\n")
	}

	// Otherwise prompt the user for the passphrase.
	return promptPassphrase(false)
}

// signHash is a helper function that calculates a hash for the given message
// that can be safely used to calculate a signature from.
//
// The hash is calulcated as
//   keccak256("\x19fafereum Signed Message:\n"${message length}${message}).
//
// This gives context to the signed message and prevents signing of transactions.
func signHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19fafereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg))
}

// mustPrintJSON prints the JSON encoding of the given object and
// exits the program with an error message when the marshaling fails.
func mustPrintJSON(jsonObject interface{}) {
	str, err := json.MarshalIndent(jsonObject, "", "  ")
	if err != nil {
		utils.Fatalf("Failed to marshal JSON object: %v", err)
	}
	//fmt.Println(string(str))
}
