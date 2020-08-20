package inter

import (
	"bytes"
	"math"
	"math/rand"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"

	"github.com/Fantom-foundation/go-mini-opera/utils/fast"
)

func TestEventHeaderDataSerialization(t *testing.T) {
	empty := MutableEvent{}
	empty.SetParents(hash.Events{})
	empty.SetPayload([]byte{})

	max := MutableEvent{}
	max.SetEpoch(math.MaxUint32)
	max.SetParents(hash.Events{hash.BytesToEvent(bytes.Repeat([]byte{math.MaxUint8}, 32))})
	max.SetSig(BytesToSignature(bytes.Repeat([]byte{math.MaxUint8}, 64)))
	max.SetPayload(bytes.Repeat([]byte{math.MaxUint8}, 100))
	max.SetIsRoot(true)

	ee := map[string]Event{
		"empty":  *empty.Build(),
		"max":    *max.Build(),
		"random": *FakeEvent(),
	}

	t.Run("ok", func(t *testing.T) {
		assertar := assert.New(t)

		for name, header0 := range ee {
			buf, err := rlp.EncodeToBytes(&header0)
			if !assertar.NoError(err) {
				return
			}

			var header1 Event
			err = rlp.DecodeBytes(buf, &header1)
			if !assertar.NoError(err, name) {
				return
			}

			if !assert.EqualValues(t, header0, header1, name) {
				return
			}
			if !assert.EqualValues(t, header0.ID(), header1.ID(), name) {
				return
			}
			if !assert.EqualValues(t, header0.HashToSign(), header1.HashToSign(), name) {
				return
			}
			if !assert.EqualValues(t, header0.Size(), header1.Size(), name) {
				return
			}
		}
	})

	t.Run("err", func(t *testing.T) {
		assertar := assert.New(t)

		for name, header0 := range ee {
			bin, err := header0.MarshalBinary()
			if !assertar.NoError(err, name) {
				return
			}

			n := rand.Intn(len(bin) - len(header0.Payload()) - 1)
			bin = bin[0:n]

			buf, err := rlp.EncodeToBytes(bin)
			if !assertar.NoError(err, name) {
				return
			}

			var header1 Event
			err = rlp.DecodeBytes(buf, &header1)
			if !assertar.Error(err, name) {
				return
			}
			//t.Log(err)
		}
	})
}

func BenchmarkEventHeaderData_EncodeRLP(b *testing.B) {
	header := FakeEvent()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf, err := rlp.EncodeToBytes(&header)
		if err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(len(buf)), "size")
	}
}

func BenchmarkEventHeaderData_DecodeRLP(b *testing.B) {
	header := FakeEvent()

	buf, err := rlp.EncodeToBytes(&header)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err = rlp.DecodeBytes(buf, &header)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestReadUintCompact(t *testing.T) {
	assertar := assert.New(t)

	// canonical
	for exp, bb := range map[uint64][]byte{
		0x000000: []byte{0x00},
		0x0000FF: []byte{0xFF},
		0x010000: []byte{0x00, 0x00, 0x01},
	} {
		got, err := readUintCompact(fast.NewBuffer(bb), len(bb))
		if !assertar.NoError(err) {
			return
		}
		if !assertar.Equal(exp, got, bb) {
			return
		}
	}

	// non canonical
	for _, bb := range [][]byte{
		[]byte{0x00, 0x00},
		[]byte{0xFF, 0x00},
		[]byte{0x00, 0x00, 0x01, 0x00},
	} {
		_, err := readUintCompact(fast.NewBuffer(bb), len(bb))
		if !assertar.Error(err) {
			return
		}
		if !assertar.Equal(ErrNonCanonicalEncoding, err, bb) {
			return
		}
	}
}

// FakeEvent generates random event for testing purpose.
func FakeEvent() *Event {
	r := rand.New(rand.NewSource(int64(0)))
	random := MutableEvent{}
	random.SetLamport(idx.Lamport(r.Uint32()))
	random.SetPayload([]byte{byte(r.Uint32())})
	random.SetSeq(idx.Event(r.Uint32() >> 8))
	random.SetCreator(idx.ValidatorID(r.Uint32()))
	random.SetFrame(idx.Frame(r.Uint32() >> 16))
	random.SetRawTime(dag.RawTimestamp(r.Uint64()))
	random.SetParents(hash.Events{})

	return random.Build()
}
