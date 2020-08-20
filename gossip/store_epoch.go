package gossip

import (
	"errors"
	"fmt"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/skiperrors"
	"github.com/Fantom-foundation/lachesis-base/kvdb/table"
)

type (
	epochStore struct {
		db    kvdb.DropableStore
		table struct {
			Tips  kvdb.Store `table:"t"`
			Heads kvdb.Store `table:"H"`
		}
	}
)

func newEpochStore(db kvdb.DropableStore) *epochStore {
	es := &epochStore{}
	es.db = db
	table.MigrateTables(&es.table, db)

	err := errors.New("database closed")

	// wrap with skiperrors to skip errors on reading from a dropped DB
	es.table.Tips = skiperrors.Wrap(es.table.Tips, err)
	es.table.Heads = skiperrors.Wrap(es.table.Heads, err)

	return es
}

// getEpochStore safe for concurrent use.
func (s *Store) getEpochStore() *epochStore {
	es := s.epochStore.Load()
	if es == nil {
		return nil
	}
	return es.(*epochStore)
}

func (s *Store) resetEpochStore(newEpoch idx.Epoch) {
	oldEs := s.epochStore.Load()
	// create new DB
	s.createEpochStore(newEpoch)
	// drop previous DB
	// there may be race condition with threads which hold this DB, so wrap tables with skiperrors
	if oldEs != nil {
		err := oldEs.(*epochStore).db.Close()
		if err != nil {
			s.Log.Error("Failed to close epoch DB", "err", err)
			return
		}
		oldEs.(*epochStore).db.Drop()
	}
}

func (s *Store) loadEpochStore(epoch idx.Epoch) {
	if s.epochStore.Load() != nil {
		return
	}
	s.createEpochStore(epoch)
}

func (s *Store) createEpochStore(epoch idx.Epoch) {
	// create new DB
	db := s.dbs.GetDb(fmt.Sprintf("gossip-%d", epoch))
	s.epochStore.Store(newEpochStore(db))
}
