package gossip

import (
	"errors"
	"sort"
	"time"

	"github.com/Fantom-foundation/go-mini-opera/eventcheck"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/Fantom-foundation/lachesis-base/lachesis"

	"github.com/ethereum/go-ethereum/log"

	"github.com/Fantom-foundation/go-mini-opera/inter"
)

var (
	errStopped = errors.New("service is stopped")
)

func (s *Service) GetConsensusCallbacks() lachesis.ConsensusCallbacks {
	return lachesis.ConsensusCallbacks{
		BeginBlock: func(cBlock *lachesis.Block) lachesis.BlockCallbacks {
			start := time.Now()

			confirmedEvents := inter.Events{}
			var atropos *inter.Event
			payloadSize := 0

			return lachesis.BlockCallbacks{
				ApplyEvent: func(_e dag.Event) {
					e := _e.(*inter.Event)
					if cBlock.Atropos == e.ID() {
						atropos = e
					}
					if len(e.Payload()) != 0 {
						// non-empty events only
						confirmedEvents.Add(e)
					}
					payloadSize += len(e.Payload())
				},
				EndBlock: func() (sealEpoch *pos.Validators) {
					// sort events by Lamport time
					sort.Sort(confirmedEvents)

					// new block
					bs := s.store.GetBlockState()
					var block = &inter.Block{
						Time:    atropos.CreationTime(), // non-secure, may be easily biased. That's fine only for miniopera
						Atropos: cBlock.Atropos,
						Events:  confirmedEvents.IDs(),
					}
					bs.Block++
					bs.EpochBlocks++
					s.store.SetBlock(bs.Block, block)

					// in miniopera, sealing condition is straightforward, based only on blocks count or cheaters present
					if bs.EpochBlocks >= s.config.Net.Dag.MaxEpochBlocks || cBlock.Cheaters.Len() != 0 {
						// seal epoch
						bs.EpochBlocks = 0
						es := s.store.GetEpochState()
						newEpoch := es.Epoch + 1
						es.Epoch = newEpoch

						// notify event checkers about new validation data
						s.heavyCheckReader.PubKeys.Store(NewEpochPubKeys(newEpoch, s.config.Net.Genesis.Validators))
						s.store.SetEpochState(es)

						// notify about new epoch
						s.emitter.OnNewEpoch(es.Validators, newEpoch)
						s.feed.newEpoch.Send(newEpoch)

						// in miniopera, validators group doesn't change, so just use genesis validators (even if they became cheaters)
						sealEpoch = es.Validators
					}
					s.store.SetBlockState(bs)

					// meter
					s.meter.Mark(uint64(payloadSize))
					log.Info("New block", "index", bs.Block, "atropos", block.Atropos, "payload", payloadSize,
						"payload_MB/s", s.meter.Per(time.Second)/1e6, "t", time.Since(start))
					return sealEpoch
				},
			}
		},
	}
}

// processEvent extends the engine.ProcessEvent with gossip-specific actions on each event processing
func (s *Service) processEvent(e *inter.Event) error {
	// s.engineMu is locked here
	if s.stopped {
		return errStopped
	}

	// repeat the checks under the mutex which may depend on volatile data
	if s.store.HasEvent(e.ID()) {
		return eventcheck.ErrAlreadyConnectedEvent
	}
	if err := s.checkers.Epochcheck.Validate(e); err != nil {
		return err
	}

	oldEpoch := s.store.GetEpoch()

	s.store.SetEvent(e)
	err := s.engine.ProcessEvent(e)
	if err != nil { // TODO make it possible to write only on success
		s.store.DelEvent(e.ID())
		return err
	}

	s.packsOnNewEvent(e, e.Epoch())
	s.emitter.OnNewEvent(e)

	newEpoch := s.store.GetEpoch()

	if newEpoch == oldEpoch {
		// set validator's last event. we don't care about forks, because this index is used only for emitter
		s.store.SetLastEvent(e.Creator(), e.ID())

		// track events with no descendants, i.e. heads
		for _, parent := range e.Parents() {
			s.store.DelHead(parent)
		}
		s.store.AddHead(e.ID())
	} else {
		// epoch is sealed, prune epoch data
		s.packsOnNewEpoch(oldEpoch, newEpoch)
		s.store.resetEpochStore(newEpoch)
	}

	immediately := newEpoch != oldEpoch

	return s.store.Commit(e.ID().Bytes(), immediately)
}
