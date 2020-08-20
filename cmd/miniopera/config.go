package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/naoina/toml"
	"gopkg.in/urfave/cli.v1"

	"github.com/Fantom-foundation/go-mini-opera/gossip"
	"github.com/Fantom-foundation/go-mini-opera/miniopera"
)

var (
	dumpConfigCommand = cli.Command{
		Action:      utils.MigrateFlags(dumpConfig),
		Name:        "dumpconfig",
		Usage:       "Show configuration values",
		ArgsUsage:   "",
		Flags:       append(nodeFlags, testFlags...),
		Category:    "MISCELLANEOUS COMMANDS",
		Description: `The dumpconfig command shows configuration values.`,
	}

	configFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "TOML configuration file",
	}

	// DataDirFlag defines directory to store Lachesis state and user's wallets
	DataDirFlag = utils.DirectoryFlag{
		Name:  "datadir",
		Usage: "Data directory for the databases and keystore",
		Value: utils.DirectoryString(DefaultDataDir()),
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
		return fmt.Errorf("field '%s' is not defined in %s", field, rt.String())
	},
}

type config struct {
	Node     node.Config
	Lachesis gossip.Config
}

func loadAllConfigs(file string, cfg *config) error {
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
	if err != nil {
		return errors.New(fmt.Sprintf("TOML config file error: %v.\n"+
			"Use 'dumpconfig' command to get an example config file.\n"+
			"If node was recently upgraded and a previous miniopera config file is used, then check updates for the config file.", err))
	}
	return err
}

func defaultMinioperaConfig(ctx *cli.Context) miniopera.Config {
	var cfg miniopera.Config

	n := "1/1"
	switch {
	case ctx.GlobalIsSet(FakeNetFlag.Name):
		n = ctx.GlobalString(FakeNetFlag.Name)
	default:
	}

	_, num, err := parseFakeGen(n)
	if err != nil {
		log.Crit("Invalid flag", "flag", FakeNetFlag.Name, "err", err)
	}
	cfg = miniopera.FakeNetConfig(getFakeValidators(num))

	return cfg
}

func setDataDir(ctx *cli.Context, cfg *node.Config) {
	defaultDataDir := DefaultDataDir()

	switch {
	case ctx.GlobalIsSet(utils.DataDirFlag.Name):
		cfg.DataDir = ctx.GlobalString(utils.DataDirFlag.Name)
	case ctx.GlobalIsSet(FakeNetFlag.Name):
		_, num, err := parseFakeGen(ctx.GlobalString(FakeNetFlag.Name))
		if err != nil {
			log.Crit("Invalid flag", "flag", FakeNetFlag.Name, "err", err)
		}
		cfg.DataDir = filepath.Join(defaultDataDir, fmt.Sprintf("fakenet-%d", num))
	case ctx.GlobalBool(utils.TestnetFlag.Name):
		cfg.DataDir = filepath.Join(defaultDataDir, "testnet")
	default:
		cfg.DataDir = defaultDataDir
	}
}

func gossipConfigWithFlags(ctx *cli.Context, src gossip.Config) gossip.Config {
	cfg := src

	// Avoid conflicting miniopera flags
	utils.CheckExclusive(ctx, FakeNetFlag, utils.DeveloperFlag, utils.TestnetFlag)
	utils.CheckExclusive(ctx, FakeNetFlag, utils.DeveloperFlag, utils.ExternalSignerFlag) // Can't use both ephemeral unlocked and external signer

	if ctx.GlobalIsSet(PayloadFlag.Name) {
		cfg.Emitter.PayloadSize = ctx.GlobalUint64(PayloadFlag.Name)
	}
	if ctx.GlobalIsSet(BytesPerSecondFlag.Name) {
		cfg.Emitter.BytesPerSec = ctx.GlobalUint64(BytesPerSecondFlag.Name)
	}
	if ctx.GlobalIsSet(utils.NetworkIdFlag.Name) {
		cfg.Net.NetworkID = ctx.GlobalUint64(utils.NetworkIdFlag.Name)
	}

	return cfg
}

func nodeConfigWithFlags(ctx *cli.Context, cfg node.Config) node.Config {
	utils.SetNodeConfig(ctx, &cfg)
	setDataDir(ctx, &cfg)
	return cfg
}

func makeAllConfigs(ctx *cli.Context) *config {
	// Defaults (low priority)
	net := defaultMinioperaConfig(ctx)
	cfg := config{Lachesis: gossip.DefaultConfig(net), Node: defaultNodeConfig()}

	// Load config file (medium priority)
	if file := ctx.GlobalString(configFileFlag.Name); file != "" {
		if err := loadAllConfigs(file, &cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}

	// Apply flags (high priority)
	cfg.Lachesis = gossipConfigWithFlags(ctx, cfg.Lachesis)
	cfg.Node = nodeConfigWithFlags(ctx, cfg.Node)

	return &cfg
}

func defaultNodeConfig() node.Config {
	cfg := NodeDefaultConfig
	cfg.Name = clientIdentifier
	cfg.Version = params.VersionWithCommit(gitCommit, gitDate)
	cfg.HTTPModules = append(cfg.HTTPModules, "eth", "ftm", "sfc", "web3")
	cfg.WSModules = append(cfg.WSModules, "eth", "ftm", "sfc", "web3")
	cfg.IPCPath = "miniopera.ipc"
	cfg.DataDir = DefaultDataDir()
	return cfg
}

// dumpConfig is the dumpconfig command.
func dumpConfig(ctx *cli.Context) error {
	cfg := makeAllConfigs(ctx)
	comment := ""

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
