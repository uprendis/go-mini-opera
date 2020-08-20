package main

import (
	"github.com/Fantom-foundation/go-mini-opera/gossip/emitter"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	cli "gopkg.in/urfave/cli.v1"
)

var validatorFlag = cli.IntFlag{
	Name:  "validator",
	Usage: "ID of a validator to create events from",
	Value: 0,
}

// setValidator retrieves the validator address either from the directly specified
// command line flags or from the keystore if CLI indexed.
func setValidator(ctx *cli.Context, cfg *emitter.Config) *emitter.Config {
	// Extract the current validator address, new flag overriding legacy one
	var validatorID idx.ValidatorID
	switch {
	case ctx.GlobalIsSet(validatorFlag.Name):
		validatorID = idx.ValidatorID(ctx.GlobalInt(validatorFlag.Name))
	case ctx.GlobalIsSet(FakeNetFlag.Name):
		validatorID, _, _ = parseFakeGen(ctx.GlobalString(FakeNetFlag.Name))
	}

	// Convert the validator into an address and configure it
	if validatorID == 0 {
		return cfg
	}

	cfg.Validator = validatorID
	return cfg
}
