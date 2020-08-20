package miniopera

import (
	"github.com/Fantom-foundation/go-mini-opera/miniopera/genesis"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

const (
	FakeNetworkID uint64 = 0xefa3
)

// DagConfig of Lachesis DAG (directed acyclic graph).
type DagConfig struct {
	MaxParents     int `json:"maxParents"`
	MaxFreeParents int `json:"maxFreeParents"` // maximum number of parents with no gas cost

	MaxEpochBlocks idx.Block `json:"maxEpochBlocks"`
}

// Config describes miniopera net.
type Config struct {
	Name      string
	NetworkID uint64

	Genesis genesis.Genesis

	// Graph options
	Dag DagConfig
}

func FakeNetConfig(accs genesis.Validators) Config {
	return Config{
		Name:      "fake",
		NetworkID: FakeNetworkID,
		Genesis:   genesis.FakeGenesis(accs),
		Dag:       FakeNetDagConfig(),
	}
}

func DefaultDagConfig() DagConfig {
	return DagConfig{
		MaxParents:     10,
		MaxFreeParents: 3,
		MaxEpochBlocks: 1000,
	}
}

func FakeNetDagConfig() DagConfig {
	return DefaultDagConfig()
}
