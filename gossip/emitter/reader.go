package emitter

import (
	"github.com/Fantom-foundation/go-mini-opera/inter"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
)

// Reader is a callback for getting events from an external storage.
type Reader interface {
	GetEpochValidators() (*pos.Validators, idx.Epoch)
	GetEvent(hash.Event) *inter.Event
	GetLastEvent(from idx.ValidatorID) *hash.Event
	GetHeads() hash.Events
}
