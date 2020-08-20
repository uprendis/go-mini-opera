package inter

import (
	"github.com/Fantom-foundation/lachesis-base/hash"
)

type Block struct {
	Time    Timestamp
	Atropos hash.Event
	Events  hash.Events
}
