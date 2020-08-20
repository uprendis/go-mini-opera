package inter

import (
	"errors"
	"io"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/Fantom-foundation/go-mini-opera/utils/fast"
)

var (
	ErrNonCanonicalEncoding = errors.New("non canonical event encoding")
	ErrInvalidEncoding      = errors.New("invalid encoded event")
)

// EncodeRLP implements rlp.Encoder interface.
func (e *Event) EncodeRLP(w io.Writer) error {
	bytes, err := e.MarshalBinary()
	if err != nil {
		return err
	}

	err = rlp.Encode(w, &bytes)

	return err
}

// DecodeRLP implements rlp.Decoder interface.
func (e *MutableEvent) DecodeRLP(src *rlp.Stream) error {
	bytes, err := src.Bytes()
	if err != nil {
		return err
	}

	return e.UnmarshalBinary(bytes)
}

// DecodeRLP implements rlp.Decoder interface.
func (e *Event) DecodeRLP(src *rlp.Stream) error {
	mutE := MutableEvent{}
	err := mutE.DecodeRLP(src)
	if err != nil {
		return err
	}
	*e = *mutE.Build()
	return nil
}

// MarshalBinary implements encoding.BinaryMarshaler interface.
func (e *Event) MarshalBinary() ([]byte, error) {
	fields64 := []uint64{
		uint64(e.RawTime()),
	}
	fields32 := []uint32{
		uint32(e.Epoch()),
		uint32(e.Seq()),
		uint32(e.Frame()),
		uint32(e.Creator()),
		uint32(e.Lamport()),
		uint32(len(e.Parents())),
	}
	fieldsBool := []bool{
		e.IsRoot(),
	}

	header3 := fast.NewBitArray(
		4,                     // bits for storing sizes 1-8 of uint64 fields (forced to 4 because fast.BitArray)
		1+uint(len(fields64)), // version + fields64
	)
	header2 := fast.NewBitArray(
		2, // bits for storing sizes 1-4 of uint32 fields
		uint(len(fields32)+len(fieldsBool)),
	)

	maxBytes := header3.Size() +
		header2.Size() +
		len(fields64)*8 +
		len(fields32)*4 +
		len(e.Parents())*(32-4) + // without idx.Epoch
		len(e.Sig()) +
		len(e.Payload())

	raw := make([]byte, maxBytes, maxBytes)

	offset := 0
	header3w := header3.Writer(raw[offset : offset+header3.Size()])
	offset += header3.Size()
	header2w := header2.Writer(raw[offset : offset+header2.Size()])
	offset += header2.Size()
	buf := fast.NewBuffer(raw[offset:])

	// write version first
	header3w.Push(int(e.Version()))

	for _, i64 := range fields64 {
		n := writeUintCompact(buf, i64, 8)
		header3w.Push(n - 1)
	}
	for _, i32 := range fields32 {
		n := writeUintCompact(buf, uint64(i32), 4)
		header2w.Push(n - 1)
	}
	for _, f := range fieldsBool {
		if f {
			header2w.Push(1)
		} else {
			header2w.Push(0)
		}
	}

	for _, p := range e.Parents() {
		buf.Write(p.Bytes()[4:]) // without epoch
	}

	buf.Write(e.Payload())
	sig := e.Sig()
	buf.Write(sig.Bytes())

	length := header3.Size() + header2.Size() + buf.Position()
	return raw[:length], nil
}

func writeUintCompact(buf *fast.Buffer, v uint64, size int) (bytes int) {
	for i := 0; i < size; i++ {
		buf.WriteByte(byte(v))
		bytes++
		v = v >> 8
		if v == 0 {
			break
		}
	}
	return
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler interface.
func (e *MutableEvent) UnmarshalBinary(raw []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ErrInvalidEncoding
		}
	}()

	origRaw := raw

	var rawTime dag.RawTimestamp
	fields64 := []*uint64{
		(*uint64)(&rawTime),
	}
	var epoch idx.Epoch
	var seq idx.Event
	var frame idx.Frame
	var creator idx.ValidatorID
	var lamport idx.Lamport
	var parentsCount uint32
	fields32 := []*uint32{
		(*uint32)(&epoch),
		(*uint32)(&seq),
		(*uint32)(&frame),
		(*uint32)(&creator),
		(*uint32)(&lamport),
		&parentsCount,
	}
	var isRoot bool
	fieldsBool := []*bool{
		&isRoot,
	}

	header3 := fast.NewBitArray(
		4,                     // bits for storing sizes 1-8 of uint64 fields (forced to 4 because fast.BitArray)
		1+uint(len(fields64)), // version + fields64
	)
	header2 := fast.NewBitArray(
		2, // bits for storing sizes 1-4 of uint32 fields
		uint(len(fields32)+len(fieldsBool)),
	)

	offset := 0
	header3r := header3.Reader(raw[offset : offset+header3.Size()])
	offset += header3.Size()
	header2r := header2.Reader(raw[offset : offset+header2.Size()])
	offset += header2.Size()
	raw = raw[offset:]
	buf := fast.NewBuffer(raw)

	// read version first
	ver := header3r.Pop()
	if ver != 0 {
		return errors.New("wrong version")
	}

	for _, i64 := range fields64 {
		n := header3r.Pop() + 1
		*i64, err = readUintCompact(buf, n)
		if err != nil {
			return
		}
	}
	var x uint64
	for _, i32 := range fields32 {
		n := header2r.Pop() + 1
		x, err = readUintCompact(buf, n)
		if err != nil {
			return
		}
		*i32 = uint32(x)
	}
	for _, f := range fieldsBool {
		n := header2r.Pop()
		*f = (n != 0)
	}

	parents := make(hash.Events, parentsCount, parentsCount)
	for i := uint32(0); i < parentsCount; i++ {
		copy(parents[i][:4], epoch.Bytes())
		copy(parents[i][4:], buf.Read(common.HashLength-4)) // without epoch
	}

	if len(raw)-buf.Position() < SigSize {
		return errors.New("malformed signature")
	}
	payload := buf.Read(len(raw) - buf.Position() - SigSize)
	sig := buf.Read(SigSize)

	e.SetVersion(uint8(ver))
	e.SetRawTime(rawTime)
	e.SetEpoch(epoch)
	e.SetSeq(seq)
	e.SetFrame(frame)
	e.SetIsRoot(isRoot)
	e.SetCreator(creator)
	e.SetLamport(lamport)
	e.SetParents(parents)
	e.SetPayload(payload)
	e.SetSig(BytesToSignature(sig))

	// set caches
	e.fillCaches(origRaw)

	return nil
}

func readUintCompact(buf *fast.Buffer, bytes int) (uint64, error) {
	var (
		v    uint64
		last byte
	)
	for i, b := range buf.Read(bytes) {
		v += uint64(b) << uint(8*i)
		last = b
	}

	if bytes > 1 && last == 0 {
		return 0, ErrNonCanonicalEncoding
	}

	return v, nil
}
