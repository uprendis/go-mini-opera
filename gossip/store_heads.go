package gossip

import (
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

func (s *Store) DelHead(id hash.Event) {
	es := s.getEpochStore()
	if es == nil {
		return
	}

	key := id.Bytes()

	if err := es.table.Heads.Delete(key); err != nil {
		s.Log.Crit("Failed to delete key", "err", err)
	}
}

func (s *Store) AddHead(id hash.Event) {
	es := s.getEpochStore()
	if es == nil {
		return
	}

	key := id.Bytes()

	if err := es.table.Heads.Put(key, []byte{}); err != nil {
		s.Log.Crit("Failed to put key-value", "err", err)
	}
}

func (s *Store) IsHead(id hash.Event) bool {
	es := s.getEpochStore()
	if es == nil {
		return false
	}

	key := id.Bytes()

	ok, err := es.table.Heads.Has(key)
	if err != nil {
		s.Log.Crit("Failed to get key", "err", err)
	}
	return ok
}

// GetHeads returns IDs of all the epoch events with no descendants
func (s *Store) GetHeads() hash.Events {
	es := s.getEpochStore()
	if es == nil {
		return nil
	}

	res := make(hash.Events, 0, 100)

	it := es.table.Heads.NewIterator()
	defer it.Release()
	for it.Next() {
		res.Add(hash.BytesToEvent(it.Key()))
	}
	if it.Error() != nil {
		s.Log.Crit("Failed to iterate keys", "err", it.Error())
	}

	return res
}

// SetLastEvent stores last unconfirmed event from a validator (off-chain)
func (s *Store) SetLastEvent(from idx.ValidatorID, id hash.Event) {
	es := s.getEpochStore()
	if es == nil {
		return
	}

	key := from.Bytes()
	if err := es.table.Tips.Put(key, id.Bytes()); err != nil {
		s.Log.Crit("Failed to put key-value", "err", err)
	}
}

// GetLastEvent returns stored last unconfirmed event from a validator (off-chain)
func (s *Store) GetLastEvent(from idx.ValidatorID) *hash.Event {
	es := s.getEpochStore()
	if es == nil {
		return nil
	}

	key := from.Bytes()
	idBytes, err := es.table.Tips.Get(key)
	if err != nil {
		s.Log.Crit("Failed to get key-value", "err", err)
	}
	if idBytes == nil {
		return nil
	}
	id := hash.BytesToEvent(idBytes)
	return &id
}
