// Copyright 2020 The go-fafjiadong wang
// This file is part of the go-faf library.
// The go-faf library is free software: you can redistribute it and/or modify


package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/fafereum/go-fafereum/log"
	"math/big"
	"os"
	"reflect"
	"unicode"

	cli "gopkg.in/urfave/cli.v1"

	"github.com/fafereum/go-fafereum/cmd/utils"
	"github.com/fafereum/go-fafereum/dashboard"
	"github.com/fafereum/go-fafereum/faf"
	"github.com/fafereum/go-fafereum/node"
	"github.com/fafereum/go-fafereum/params"
	whisper "github.com/fafereum/go-fafereum/whisper/whisperv6"
	"github.com/naoina/toml"
)

var (
	dumpConfigCommand = cli.Command{
		Action:      utils.MigrateFlags(dumpConfig),
		Name:        "dumpconfig",
		Usage:       "Show configuration values",
		ArgsUsage:   "",
		Flags:       append(append(nodeFlags, rpcFlags...), whisperFlags...),
		Category:    "MISCELLANEOUS COMMANDS",
		Description: `The dumpconfig command shows configuration values.`,
	}

	configFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "TOML configuration file",
	}
)

// These settings ensure that TOML keys use the same names as Go struct fields.
var tomlSettings = toml.Config{
	NormFieldName: func(rt reflect.Type, key string) string {
		return key
	},
	FieldToKey: func(rt reflect.Type, field string) string {
		return field
	},
	MissingField: func(rt reflect.Type, field string) error {
		link := ""
		if unicode.IsUpper(rune(rt.Name()[0])) && rt.PkgPath() != "main" {
			link = fmt.Sprintf(", see https://godoc.org/%s#%s for available fields", rt.PkgPath(), rt.Name())
		}
		return fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
	},
}

type fafstatsConfig struct {
	URL string `toml:",omitempty"`
}

type gfafConfig struct {
	faf       faf.Config
	Shh       whisper.Config
	Node      node.Config
	fafstats  fafstatsConfig
	Dashboard dashboard.Config
}

func loadConfig(file string, cfg *gfafConfig) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	err = tomlSettings.NewDecoder(bufio.NewReader(f)).Decode(cfg)
	// Add file name to errors that have a line number.
	if _, ok := err.(*toml.LineError); ok {
		err = errors.New(file + ", " + err.Error())
	}
	return err
}

func defaultNodeConfig() node.Config {
	cfg := node.DefaultConfig
	cfg.Name = clientIdentifier
	cfg.Version = params.VersionWithCommit(gitCommit)
	cfg.HTTPModules = append(cfg.HTTPModules, "faf", "shh")
	cfg.WSModules = append(cfg.WSModules, "faf", "shh")
	cfg.IPCPath = "gfaf.ipc"
	return cfg
}

func makeConfigNode(ctx *cli.Context) (*node.Node, gfafConfig) {
	// Load defaults.
	cfg := gfafConfig{
		faf:       faf.DefaultConfig,
		Shh:       whisper.DefaultConfig,
		Node:      defaultNodeConfig(),
		Dashboard: dashboard.DefaultConfig,
	}

	// Load config file.
	if file := ctx.GlobalString(configFileFlag.Name); file != "" {
		if err := loadConfig(file, &cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}

	// Apply flags.
	log.Info("SetNodeConfig")
	utils.SetNodeConfig(ctx, &cfg.Node)
	stack, err := node.New(&cfg.Node)
	if err != nil {
		utils.Fatalf("Failed to create the protocol stack: %v", err)
	}
	log.Info("SetfafConfig")
	utils.SetfafConfig(ctx, stack, &cfg.faf)
	if ctx.GlobalIsSet(utils.fafStatsURLFlag.Name) {
		cfg.fafstats.URL = ctx.GlobalString(utils.fafStatsURLFlag.Name)
	}
	log.Info("SetShhConfig")
	utils.SetShhConfig(ctx, stack, &cfg.Shh)
	log.Info("SetDashboardConfig")
	utils.SetDashboardConfig(ctx, &cfg.Dashboard)
	return stack, cfg
}

// enableWhisper returns true in case one of the whisper flags is set.
func enableWhisper(ctx *cli.Context) bool {
	for _, flag := range whisperFlags {
		if ctx.GlobalIsSet(flag.GetName()) {
			return true
		}
	}
	return false
}
// 根据命令行参数和一些特殊的配置来创建一个node
func makeFullNode(ctx *cli.Context) *node.Node {
	//加载配置文件
	log.Info(`makeConfigNode`)
	stack, cfg := makeConfigNode(ctx)
	if ctx.GlobalIsSet(utils.ConstantinopleOverrideFlag.Name) {
		cfg.faf.ConstantinopleOverride = new(big.Int).SetUint64(ctx.GlobalUint64(utils.ConstantinopleOverrideFlag.Name))
	}
	//注册一个以太坊服务
	utils.RegisterfafService(stack, &cfg.faf)

	if ctx.GlobalBool(utils.DashboardEnabledFlag.Name) {
		utils.RegisterDashboardService(stack, &cfg.Dashboard, gitCommit)
	}
	// Whisper must be explicitly enabled by specifying at least 1 whisper flag or in dev mode
	shhEnabled := enableWhisper(ctx)
	shhAutoEnabled := !ctx.GlobalIsSet(utils.WhisperEnabledFlag.Name) && ctx.GlobalIsSet(utils.DeveloperFlag.Name)
	if shhEnabled || shhAutoEnabled {
		if ctx.GlobalIsSet(utils.WhisperMaxMessageSizeFlag.Name) {
			cfg.Shh.MaxMessageSize = uint32(ctx.Int(utils.WhisperMaxMessageSizeFlag.Name))
		}
		if ctx.GlobalIsSet(utils.WhisperMinPOWFlag.Name) {
			cfg.Shh.MinimumAcceptedPOW = ctx.Float64(utils.WhisperMinPOWFlag.Name)
		}
		if ctx.GlobalIsSet(utils.WhisperRestrictConnectionBetweenLightClientsFlag.Name) {
			cfg.Shh.RestrictConnectionBetweenLightClients = true
		}
		utils.RegisterShhService(stack, &cfg.Shh)
	}

	// Add the fafereum Stats daemon if requested.
	if cfg.fafstats.URL != "" {
		utils.RegisterfafStatsService(stack, cfg.fafstats.URL)
	}
	return stack
}

// dumpConfig is the dumpconfig command.
func dumpConfig(ctx *cli.Context) error {
	_, cfg := makeConfigNode(ctx)
	comment := ""

	if cfg.faf.Genesis != nil {
		cfg.faf.Genesis = nil
		comment += "# Note: this config doesn't contain the genesis block.\n\n"
	}

	out, err := tomlSettings.Marshal(&cfg)
	if err != nil {
		return err
	}

	dump := os.Stdout
	if ctx.NArg() > 0 {
		dump, err = os.OpenFile(ctx.Args().Get(0), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer dump.Close()
	}
	dump.WriteString(comment)
	dump.Write(out)

	return nil
}
