package gossip

import (
	"github.com/Fantom-foundation/go-mini-opera/gossip/emitter"
	"github.com/Fantom-foundation/go-mini-opera/miniopera"
)

type (
	// ProtocolConfig is config for p2p protocol
	ProtocolConfig struct {
		// 0/M means "optimize only for throughput", N/0 means "optimize only for latency", N/M is a balanced mode

		LatencyImportance    int
		ThroughputImportance int
	}
	// Config for the gossip service.
	Config struct {
		Net     miniopera.Config
		Emitter emitter.Config
		StoreConfig

		// Protocol options
		Protocol ProtocolConfig
	}

	// StoreConfig is a config for store db.
	StoreConfig struct {
		// Cache size for Events.
		EventsCacheSize int
		// Cache size for Block.
		BlockCacheSize int
		// Cache size for PackInfos.
		PackInfosCacheSize int
	}
)

// DefaultConfig returns the default configurations for the gossip service.
func DefaultConfig(network miniopera.Config) Config {
	cfg := Config{
		Net:         network,
		Emitter:     emitter.DefaultEmitterConfig(),
		StoreConfig: DefaultStoreConfig(),

		Protocol: ProtocolConfig{
			LatencyImportance:    60,
			ThroughputImportance: 40,
		},
	}

	if network.NetworkID == miniopera.FakeNetworkID {
		cfg.Emitter = emitter.FakeEmitterConfig()
		// disable self-fork protection if fakenet 1/1
		if len(network.Genesis.Validators) == 1 {
			cfg.Emitter.EmitIntervals.DoublesignProtection = 0
		}
	}

	return cfg
}

// DefaultStoreConfig for product.
func DefaultStoreConfig() StoreConfig {
	return StoreConfig{
		EventsCacheSize:    500,
		BlockCacheSize:     100,
		PackInfosCacheSize: 100,
	}
}

// LiteStoreConfig is for tests or inmemory.
func LiteStoreConfig() StoreConfig {
	return StoreConfig{
		EventsCacheSize:    100,
		BlockCacheSize:     100,
		PackInfosCacheSize: 100,
	}
}
