package gossip

import (
	"fmt"

	"github.com/Fantom-foundation/go-mini-opera/inter"
	"github.com/Fantom-foundation/lachesis-base/abft"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"

	"github.com/Fantom-foundation/go-mini-opera/miniopera"
)

// GenesisMismatchError is raised when trying to overwrite an existing
// genesis block with an incompatible one.
type GenesisMismatchError struct {
	Stored, New hash.Event
}

// Error implements error interface.
func (e *GenesisMismatchError) Error() string {
	return fmt.Sprintf("database contains incompatible gossip genesis (have %s, new %s)", e.Stored.FullID(), e.New.FullID())
}

// ApplyGenesis writes initial state.
func (s *Store) ApplyGenesis(net *miniopera.Config) (genesisAtropos hash.Event, genesisState common.Hash, new bool, err error) {
	storedGenesis := s.GetBlock(0)
	if storedGenesis != nil {
		newHash := calcGenesisHash(net)
		if storedGenesis.Atropos != newHash {
			return genesisAtropos, genesisState, true, &GenesisMismatchError{storedGenesis.Atropos, newHash}
		}

		genesisAtropos = storedGenesis.Atropos
		genesisState = common.Hash(genesisAtropos)
		return genesisAtropos, genesisState, false, nil
	}
	// if we'here, then it's first time genesis is applied
	genesisAtropos, genesisState, err = s.applyGenesis(net)
	if err != nil {
		return genesisAtropos, genesisState, true, err
	}

	return genesisAtropos, genesisState, true, err
}

// calcGenesisHash calcs hash of genesis state.
func calcGenesisHash(net *miniopera.Config) hash.Event {
	s := NewMemStore()
	defer s.Close()

	h, _, _ := s.applyGenesis(net)

	return h
}

func (s *Store) applyGenesis(net *miniopera.Config) (genesisAtropos hash.Event, genesisState common.Hash, err error) {
	prettyHash := func(net *miniopera.Config) hash.Event {
		e := inter.MutableEvent{}
		// for nice-looking ID
		e.SetEpoch(0)
		e.SetLamport(idx.Lamport(net.Dag.MaxEpochBlocks))
		// actual data hashed
		h := net.Genesis.Validators.Hash()
		e.SetPayload(append(net.Genesis.ExtraData, h[:]...))
		e.SetRawTime(dag.RawTimestamp(net.Genesis.Time))

		return e.Build().ID()
	}
	genesisAtropos = prettyHash(net)
	genesisState = common.Hash(genesisAtropos)

	block := &inter.Block{
		Time:    net.Genesis.Time,
		Atropos: genesisAtropos,
		Events:  hash.Events{genesisAtropos},
	}
	s.SetBlock(0, block)

	s.SetEpochState(EpochState{
		Epoch:      abft.FirstEpoch,
		Validators: net.Genesis.Validators.Build(),
	})
	s.SetBlockState(BlockState{
		Block:       0,
		EpochBlocks: 0,
	})

	return genesisAtropos, genesisState, nil
}
