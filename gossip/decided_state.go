package gossip

import (
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
)

type BlockState struct {
	Block       idx.Block
	EpochBlocks idx.Block
}

type EpochState struct {
	Epoch      idx.Epoch
	Validators *pos.Validators
}
