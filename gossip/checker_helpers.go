package gossip

import (
	"sync/atomic"

	"github.com/Fantom-foundation/go-mini-opera/miniopera/genesis"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

// ValidatorsPubKeys stores info to authenticate validators
type ValidatorsPubKeys struct {
	Epoch   idx.Epoch
	PubKeys map[idx.ValidatorID][]byte
}

// HeavyCheckReader is a helper to run heavy power checks
type HeavyCheckReader struct {
	PubKeys atomic.Value
}

// GetEpochPubKeys is safe for concurrent use
func (r *HeavyCheckReader) GetEpochPubKeys() (map[idx.ValidatorID][]byte, idx.Epoch) {
	auth := r.PubKeys.Load().(*ValidatorsPubKeys)

	return auth.PubKeys, auth.Epoch
}

// NewEpochPubKeys reads fills ValidatorsPubKeys with data from store
func NewEpochPubKeys(epoch idx.Epoch, vv genesis.Validators) *ValidatorsPubKeys {
	pubKeys := make(map[idx.ValidatorID][]byte)
	for _, it := range vv {
		pubKeys[it.ID] = it.PubKey
	}
	return &ValidatorsPubKeys{
		Epoch:   epoch,
		PubKeys: pubKeys,
	}
}
