package emitter

import (
	"crypto/ecdsa"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/Fantom-foundation/go-mini-opera/utils/throughput"
	"github.com/Fantom-foundation/lachesis-base/emitter/ancestor"
	"github.com/Fantom-foundation/lachesis-base/emitter/doublesign"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/Fantom-foundation/go-mini-opera/eventcheck"
	"github.com/Fantom-foundation/go-mini-opera/inter"

	"github.com/Fantom-foundation/go-mini-opera/logger"
	"github.com/Fantom-foundation/go-mini-opera/miniopera"
	"github.com/Fantom-foundation/go-mini-opera/utils/errlock"
)

// EmitterWorld is emitter's external world
type EmitterWorld struct {
	Store        Reader
	EngineMu     *sync.RWMutex
	ProcessEvent func(*inter.Event) error
	Build        func(dag.MutableEvent) error
	DagIndex     ancestor.DagIndex

	Checkers *eventcheck.Checkers

	IsSynced func() bool
	PeersNum func() int
}

type Emitter struct {
	net    *miniopera.Config
	config *Config

	world   EmitterWorld
	privKey *ecdsa.PrivateKey

	syncStatus syncStatus

	prevEmittedTime time.Time

	intervals EmitIntervals

	meter *throughput.Meter

	done chan struct{}
	wg   sync.WaitGroup

	logger.Periodic
}

type syncStatus struct {
	startup                   time.Time
	lastConnected             time.Time
	p2pSynced                 time.Time
	prevLocalEmittedID        hash.Event
	externalSelfEventCreated  time.Time
	externalSelfEventDetected time.Time
}

// New creation.
func New(
	net *miniopera.Config,
	config *Config,
	world EmitterWorld,
) *Emitter {

	loggerInstance := logger.MakeInstance()
	return &Emitter{
		net:       net,
		config:    config,
		world:     world,
		intervals: config.EmitIntervals,
		Periodic:  logger.Periodic{Instance: loggerInstance},
		meter:     throughput.New(),
	}
}

// init emitter without starting events emission
func (em *Emitter) init() {
	em.syncStatus.lastConnected = time.Now()
	em.syncStatus.startup = time.Now()
	validators, epoch := em.world.Store.GetEpochValidators()
	em.prevEmittedTime = em.loadPrevEmitTime()
	em.OnNewEpoch(validators, epoch)
}

// StartEventEmission starts event emission.
func (em *Emitter) StartEventEmission(key *ecdsa.PrivateKey) {
	if em.done != nil {
		return
	}
	em.done = make(chan struct{})

	em.privKey = key
	em.init()

	done := em.done
	em.wg.Add(1)
	go func() {
		defer em.wg.Done()
		ticker := time.NewTicker(10 * time.Millisecond)
		for {
			select {
			case <-ticker.C:
				// track synced time
				if em.world.PeersNum() == 0 {
					em.syncStatus.lastConnected = time.Now() // connected time ~= last time when it's true that "not connected yet"
				}
				if !em.world.IsSynced() {
					em.syncStatus.p2pSynced = time.Now() // synced time ~= last time when it's true that "not synced yet"
				}

				// must pass at least MinEmitInterval since last event
				if time.Since(em.prevEmittedTime) >= em.intervals.Min {
					em.EmitEvent()
				}
			case <-done:
				return
			}
		}
	}()
}

// StopEventEmission stops event emission.
func (em *Emitter) StopEventEmission() {
	if em.done == nil {
		return
	}

	close(em.done)
	em.done = nil
	em.wg.Wait()
}

func (em *Emitter) loadPrevEmitTime() time.Time {
	if em.config.Validator == 0 {
		return em.prevEmittedTime
	}

	prevEventID := em.world.Store.GetLastEvent(em.config.Validator)
	if prevEventID == nil {
		return em.prevEmittedTime
	}
	prevEvent := em.world.Store.GetEvent(*prevEventID)
	if prevEvent == nil {
		return em.prevEmittedTime
	}
	return prevEvent.CreationTime().Time()
}

func (em *Emitter) findBestParents(myValidatorID idx.ValidatorID) (*hash.Event, hash.Events, bool) {
	selfParent := em.world.Store.GetLastEvent(myValidatorID)
	heads := em.world.Store.GetHeads() // events with no descendants

	var strategy ancestor.SearchStrategy
	dagIndex := em.world.DagIndex
	if dagIndex != nil {
		validators, _ := em.world.Store.GetEpochValidators()
		strategy = ancestor.NewCasualityStrategy(dagIndex, validators)
		if rand.Intn(20) == 0 { // every 20th event uses random strategy is avoid repeating patterns in DAG
			strategy = ancestor.NewRandomStrategy(rand.New(rand.NewSource(time.Now().UnixNano())))
		}
	} else {
		// use dummy strategy in tests
		strategy = ancestor.NewRandomStrategy(nil)
	}

	_, parents := ancestor.FindBestParents(em.net.Dag.MaxParents, heads, selfParent, strategy)
	return selfParent, parents, true
}

// createEvent is not safe for concurrent use.
func (em *Emitter) createEvent(payload []byte) *inter.Event {
	if em.config.Validator == 0 {
		// not a validator
		return nil
	}

	if synced := em.logSyncStatus(em.isSyncedToEmit()); !synced {
		// I'm reindexing my old events, so don't create events until connect all the existing self-events
		return nil
	}

	var (
		selfParentSeq  idx.Event
		selfParentTime inter.Timestamp
		parents        hash.Events
		maxLamport     idx.Lamport
	)

	// Find parents
	selfParent, parents, ok := em.findBestParents(em.config.Validator)
	if !ok {
		return nil
	}

	// Set parent-dependent fields
	parentEvents := make(dag.Events, len(parents))
	for i, p := range parents {
		parent := em.world.Store.GetEvent(p)
		if parent == nil {
			em.Log.Crit("Emitter: head not found", "event", p.String())
		}
		parentEvents[i] = parent
		maxLamport = idx.MaxLamport(maxLamport, parent.Lamport())
	}

	selfParentSeq = 0
	selfParentTime = 0
	var selfParentEvent *inter.Event
	if selfParent != nil {
		selfParentEvent = parentEvents[0].(*inter.Event)
		selfParentSeq = selfParentEvent.Seq()
		selfParentTime = selfParentEvent.CreationTime()
	}

	_, epoch := em.world.Store.GetEpochValidators()
	mutEvent := &inter.MutableEvent{}
	mutEvent.SetEpoch(epoch)
	mutEvent.SetSeq(selfParentSeq + 1)
	mutEvent.SetCreator(em.config.Validator)

	mutEvent.SetParents(parents)
	mutEvent.SetLamport(maxLamport + 1)
	mutEvent.SetRawTime(dag.RawTimestamp(inter.MaxTimestamp(inter.Timestamp(time.Now().UnixNano()), selfParentTime+1)))
	mutEvent.SetPayload(payload)

	// set consensus fields
	err := em.world.Build(mutEvent)
	if err != nil {
		em.Log.Warn("Dropped event while emitting", "err", err)
		return nil
	}

	if !em.isAllowedToEmit(mutEvent) {
		return nil
	}

	// sign
	h := mutEvent.HashToSign()
	sigRSV, err := crypto.Sign(h[:], em.privKey)
	if err != nil {
		em.Periodic.Error(time.Second, "Failed to sign event", "err", err)
		return nil
	}
	mutEvent.SetSig(inter.BytesToSignature(sigRSV[:inter.SigSize])) // drop V number

	// build
	event := mutEvent.Build()
	{
		// sanity check
		if em.world.Checkers != nil {
			if err := em.world.Checkers.Validate(event, parentEvents); err != nil {
				em.Periodic.Error(time.Second, "Event check failed during event emitting", "err", err)
				return nil
			}
		}
	}

	// set event name for debug
	em.nameEventForDebug(event)

	return event
}

// OnNewEpoch should be called after each epoch change, and on startup
func (em *Emitter) OnNewEpoch(newValidators *pos.Validators, newEpoch idx.Epoch) {}

// OnNewEvent tracks new events to find out am I properly synced or not
func (em *Emitter) OnNewEvent(e *inter.Event) {
	if em.config.Validator == 0 || em.config.Validator != e.Creator() {
		return
	}
	if em.syncStatus.prevLocalEmittedID == e.ID() {
		return
	}
	// event was emitted by me on another instance
	em.onNewExternalEvent(e)

}
func (em *Emitter) onNewExternalEvent(e *inter.Event) {
	em.syncStatus.externalSelfEventDetected = time.Now()
	em.syncStatus.externalSelfEventCreated = e.CreationTime().Time()
	status := em.currentSyncStatus()
	if doublesign.DetectParallelInstance(status, em.config.EmitIntervals.ParallelInstanceProtection) {
		passedSinceEvent := status.Since(status.ExternalSelfEventCreated)
		reason := "Received a recent event (event id=%s) from this validator (validator ID=%d) which wasn't created on this node.\n" +
			"This external event was created %s, %s ago at the time of this error.\n" +
			"It might mean that a duplicating instance of the same validator is running simultaneously, which may eventually lead to a doublesign.\n" +
			"The node was stopped by one of the doublesign protection heuristics.\n" +
			"There's no guaranteed automatic protection against a doublesign," +
			"please always ensure that no more than one instance of the same validator is running."
		errlock.Permanent(fmt.Errorf(reason, e.ID().String(), em.config.Validator, e.CreationTime().Time().Local().String(), passedSinceEvent.String()))
		panic("unreachable")
	}
}

func (em *Emitter) currentSyncStatus() doublesign.SyncStatus {
	return doublesign.SyncStatus{
		Now:                       time.Now(),
		PeersNum:                  em.world.PeersNum(),
		Startup:                   em.syncStatus.startup,
		LastConnected:             em.syncStatus.lastConnected,
		P2PSynced:                 em.syncStatus.p2pSynced,
		ExternalSelfEventCreated:  em.syncStatus.externalSelfEventCreated,
		ExternalSelfEventDetected: em.syncStatus.externalSelfEventDetected,
	}
}

func (em *Emitter) isSyncedToEmit() (time.Duration, error) {
	if em.intervals.DoublesignProtection == 0 {
		return 0, nil // protection disabled
	}
	return doublesign.SyncedToEmit(em.currentSyncStatus(), em.intervals.DoublesignProtection)
}

func (em *Emitter) logSyncStatus(wait time.Duration, syncErr error) bool {
	if syncErr == nil {
		return true
	}

	if wait == 0 {
		em.Periodic.Info(25*time.Second, "Emitting is paused", "reason", syncErr.Error())
	} else {
		em.Periodic.Info(25*time.Second, "Emitting is paused", "reason", syncErr.Error(), "wait", wait)
	}
	return false
}

// return true if event is in epoch tail (unlikely to confirm)
func (em *Emitter) isEpochTail(e dag.Event) bool {
	return e.Frame() >= idx.Frame(em.net.Dag.MaxEpochBlocks)-em.config.EpochTailLength
}

func (em *Emitter) isAllowedToEmit(e *inter.MutableEvent) bool {
	passedTime := e.CreationTime().Time().Sub(em.prevEmittedTime)
	// Slow down emitting if no payload to post, and not at epoch tail
	{
		if passedTime < em.intervals.Max &&
			len(e.Payload()) == 0 &&
			!em.isEpochTail(e) {
			return false
		}
	}

	return true
}

func (em *Emitter) getTestPayload() []byte {
	passedTime := float64(time.Now().Sub(em.syncStatus.startup)) / (float64(time.Second))
	neededBytes := uint64(passedTime * float64(em.config.BytesPerSec))
	payloadSize := uint64(0)
	if em.meter.Total() < neededBytes {
		payloadSize = em.config.PayloadSize
	}
	if payloadSize == 0 {
		return []byte{}
	}
	payload := make([]byte, payloadSize)
	seed := int(time.Now().UnixNano())
	for i, _ := range payload {
		payload[i] = byte(seed * i) // pseudo-random to avoid a simple compression of DB
	}
	return payload
}

func (em *Emitter) EmitEvent() *inter.Event {
	if em.config.Validator == 0 {
		return nil // short circuit if not validator
	}

	em.world.EngineMu.Lock()
	defer em.world.EngineMu.Unlock()

	e := em.createEvent(em.getTestPayload())
	if e == nil {
		return nil
	}
	em.syncStatus.prevLocalEmittedID = e.ID()

	start := time.Now()

	err := em.world.ProcessEvent(e)
	if err != nil {
		em.Log.Error("Emitted event connection error", "err", err)
		return nil
	}
	em.prevEmittedTime = e.CreationTime().Time()

	em.meter.Mark(uint64(len(e.Payload())))

	em.Log.Info("New event emitted", "id", e.ID(), "parents", len(e.Parents()), "by", e.Creator(),
		"frame", inter.FmtFrame(e.Frame(), e.IsRoot()), "payload", len(e.Payload()),
		"emitting_MB/s", em.meter.Per(time.Second)/1e6, "t", time.Since(start))

	return e
}

func (em *Emitter) nameEventForDebug(e *inter.Event) {
	name := []rune(hash.GetNodeName(e.Creator()))
	if len(name) < 1 {
		return
	}

	name = name[len(name)-1:]
	hash.SetEventName(e.ID(), fmt.Sprintf("%s%03d",
		strings.ToLower(string(name)),
		e.Seq()))
}
