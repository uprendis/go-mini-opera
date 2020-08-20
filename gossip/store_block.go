package gossip

import (
	"github.com/Fantom-foundation/go-mini-opera/inter"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

// SetBlock stores chain block.
func (s *Store) SetBlock(n idx.Block, b *inter.Block) {
	s.set(s.table.Blocks, n.Bytes(), b)

	// Add to LRU cache.
	if b != nil && s.cache.Blocks != nil {
		s.cache.Blocks.Add(n, b)
	}
}

// GetBlock returns stored block.
func (s *Store) GetBlock(n idx.Block) *inter.Block {
	// Get block from LRU cache first.
	if s.cache.Blocks != nil {
		if c, ok := s.cache.Blocks.Get(n); ok {
			if b, ok := c.(*inter.Block); ok {
				return b
			}
		}
	}

	block, _ := s.get(s.table.Blocks, n.Bytes(), &inter.Block{}).(*inter.Block)

	// Add to LRU cache.
	if block != nil && s.cache.Blocks != nil {
		s.cache.Blocks.Add(n, block)
	}

	return block
}
