package heavycheck

import (
	"errors"
	"runtime"
	"sync"

	"github.com/Fantom-foundation/lachesis-base/eventcheck/epochcheck"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/Fantom-foundation/go-mini-opera/inter"

	"github.com/Fantom-foundation/go-mini-opera/miniopera"
)

var (
	ErrWrongEventSig = errors.New("event has wrong signature")

	errTerminated = errors.New("terminated") // internal err
)

const (
	maxQueuedTasks = 128 // the maximum number of events to queue up
	maxBatch       = 4   // Maximum number of events in an task batch (batch is divided if exceeded)
)

// OnValidatedFn is a callback type for notifying about validation result.
type OnValidatedFn func(dag.Events, []error)

// Reader is accessed by the validator to get the current state.
type Reader interface {
	GetEpochPubKeys() (map[idx.ValidatorID][]byte, idx.Epoch)
}

// Check which require only parents list + current epoch info
type Checker struct {
	config *miniopera.DagConfig
	reader Reader

	numOfThreads int

	tasksQ chan *TaskData
	quit   chan struct{}
	wg     sync.WaitGroup
}

type TaskData struct {
	Events dag.Events // events to validate
	Result []error    // resulting errors of events, nil if ok

	onValidated OnValidatedFn
}

// NewDefault uses N-1 threads
func NewDefault(config *miniopera.DagConfig, reader Reader) *Checker {
	threads := runtime.NumCPU()
	if threads > 1 {
		threads--
	}
	if threads < 1 {
		threads = 1
	}
	return New(config, reader, threads)
}

// New validator which performs heavy checks, related to signatures validation and Merkle tree validation
func New(config *miniopera.DagConfig, reader Reader, numOfThreads int) *Checker {
	return &Checker{
		config:       config,
		reader:       reader,
		numOfThreads: numOfThreads,
		tasksQ:       make(chan *TaskData, maxQueuedTasks),
		quit:         make(chan struct{}),
	}
}

func (v *Checker) Start() {
	for i := 0; i < v.numOfThreads; i++ {
		v.wg.Add(1)
		go v.loop()
	}
}

func (v *Checker) Stop() {
	close(v.quit)
	v.wg.Wait()
}

func (v *Checker) Overloaded() bool {
	return len(v.tasksQ) > maxQueuedTasks/2
}

func (v *Checker) Enqueue(events dag.Events, onValidated func(dag.Events, []error)) error {
	// divide big batch into smaller ones
	for start := 0; start < len(events); start += maxBatch {
		end := len(events)
		if end > start+maxBatch {
			end = start + maxBatch
		}
		op := &TaskData{
			Events:      events[start:end],
			onValidated: onValidated,
		}
		select {
		case v.tasksQ <- op:
			continue
		case <-v.quit:
			return errTerminated
		}
	}
	return nil
}

// Validate event
func (v *Checker) Validate(de dag.Event) error {
	e := de.(*inter.Event)
	pubKeys, epoch := v.reader.GetEpochPubKeys()
	if e.Epoch() != epoch {
		return epochcheck.ErrNotRelevant
	}
	// validatorID
	pubKey, ok := pubKeys[e.Creator()]
	if !ok {
		return epochcheck.ErrAuth
	}

	// event sig
	signedHash := e.HashToSign()
	sig := e.Sig()
	ok = crypto.VerifySignature(pubKey, signedHash[:], sig[:])
	if !ok {
		return ErrWrongEventSig
	}

	return nil
}

func (v *Checker) loop() {
	defer v.wg.Done()
	for {
		select {
		case <-v.quit:
			return

		case op := <-v.tasksQ:
			op.Result = make([]error, len(op.Events))
			for i, e := range op.Events {
				op.Result[i] = v.Validate(e)
			}
			op.onValidated(op.Events, op.Result)
		}
	}
}
