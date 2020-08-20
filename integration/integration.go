package integration

import (
	"time"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"

	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"

	"github.com/Fantom-foundation/go-mini-opera/gossip"
	"github.com/Fantom-foundation/go-mini-opera/miniopera"
)

// NewIntegration creates gossip service for the integration test
func NewIntegration(ctx *adapters.ServiceContext, network miniopera.Config, validator idx.ValidatorID) *gossip.Service {
	gossipCfg := gossip.DefaultConfig(network)

	engine, _, gdb := MakeEngine(ctx.Config.DataDir, &gossipCfg)

	gossipCfg.Emitter.Validator = validator
	gossipCfg.Emitter.EmitIntervals.Max = 3 * time.Second
	gossipCfg.Emitter.EmitIntervals.DoublesignProtection = 0

	svc, err := gossip.NewService(ctx.NodeContext, &gossipCfg, gdb, engine)
	if err != nil {
		panic(err)
	}
	err = engine.Bootstrap(svc.GetConsensusCallbacks())
	if err != nil {
		panic(err)
	}

	return svc
}
