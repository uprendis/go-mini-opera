package inter

import (
	"crypto/sha256"
	"fmt"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

type Event struct {
	dag.BaseEvent
	extEvent
}

func (e *Event) CreationTime() Timestamp { return Timestamp(e.RawTime()) }

func eventHashToSign(b []byte) hash.Hash {
	hasher := sha256.New()
	_, err := hasher.Write(b[:len(b)-SigSize]) // don't hash the signature
	if err != nil {
		panic("can't hash: " + err.Error())
	}
	return hash.BytesToHash(hasher.Sum(nil))
}

func (e *Event) HashToSign() hash.Hash {
	return *e._hashToSign
}

func (e *Event) Size() int {
	return e._size
}

type extEvent struct {
	version uint8 // serialization version

	payload []byte
	sig     Signature

	// cache
	_size       int
	_hashToSign *hash.Hash
}

func (e *extEvent) Version() uint8 { return e.version }

func (e *extEvent) Payload() []byte { return e.payload }

func (e *extEvent) Sig() Signature { return e.sig }

type mutableExtEvent struct {
	extEvent
}

func (e *mutableExtEvent) SetVersion(v uint8) { e.version = v }

func (e *mutableExtEvent) SetPayload(v []byte) { e.payload = v }

func (e *mutableExtEvent) SetSig(v Signature) { e.sig = v }

type MutableEvent struct {
	dag.MutableBaseEvent
	mutableExtEvent
}

func MutableFrom(e *Event) *MutableEvent {
	return &MutableEvent{dag.MutableBaseEvent{e.BaseEvent}, mutableExtEvent{e.extEvent}}
}

func (e *MutableEvent) CreationTime() Timestamp { return Timestamp(e.RawTime()) }

func calcEventID(hashToSign hash.Hash) (id [24]byte) {
	copy(id[:], hashToSign[:24])
	return id
}

func (e *MutableEvent) hashToSign() hash.Hash {
	b, err := e.immutable().MarshalBinary()
	if err != nil {
		panic("can't encode: " + err.Error())
	}
	return eventHashToSign(b)
}

func (e *MutableEvent) size() int {
	b, err := e.immutable().MarshalBinary()
	if err != nil {
		panic("can't encode: " + err.Error())
	}
	return len(b)
}

func (e *MutableEvent) HashToSign() hash.Hash {
	return e.hashToSign()
}

func (e *MutableEvent) Size() int {
	return e.size()
}

func (e *MutableEvent) immutable() *Event {
	return &Event{e.MutableBaseEvent.BaseEvent, e.extEvent}
}

func (e *MutableEvent) fillCaches(b []byte) {
	e._size = len(b)
	h := eventHashToSign(b)
	e._hashToSign = &h
}

func (e *MutableEvent) Build() *Event {
	c := *e
	// set ID
	if c._hashToSign != nil {
		// cached
		c.SetID(calcEventID(*c._hashToSign))
	} else {
		c.SetID(calcEventID(e.hashToSign()))
	}

	if c._size != 0 && c._hashToSign != nil {
		return c.immutable()
	}

	// set caches
	b, err := e.immutable().MarshalBinary()
	if err != nil {
		panic("can't encode: " + err.Error())
	}
	c.fillCaches(b)
	return c.immutable()
}

// fmtFrame returns frame string representation.
func FmtFrame(frame idx.Frame, isRoot bool) string {
	if isRoot {
		return fmt.Sprintf("%d:y", frame)
	}
	return fmt.Sprintf("%d:n", frame)
}
