package integration

import (
	"fmt"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/utils/adapters"
	"github.com/Fantom-foundation/lachesis-base/vector"

	"github.com/Fantom-foundation/lachesis-base/abft"
	"github.com/Fantom-foundation/lachesis-base/kvdb/flushable"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"

	"github.com/Fantom-foundation/go-mini-opera/gossip"
)

func panics(name string) func(error) {
	return func(err error) {
		log.Crit(fmt.Sprintf("%s error", name), "err", err)
	}
}

type GossipStoreAdapter struct {
	*gossip.Store
}

func (g *GossipStoreAdapter) GetEvent(id hash.Event) dag.Event {
	e := g.Store.GetEvent(id)
	if e == nil {
		return nil
	}
	return e
}

// MakeEngine makes consensus engine from config.
func MakeEngine(dataDir string, gossipCfg *gossip.Config) (*abft.Lachesis, *flushable.SyncedPool, *gossip.Store) {
	dbs := flushable.NewSyncedPool(DBProducer(dataDir))

	gdb := gossip.NewStore(dbs, gossipCfg.StoreConfig)

	cMainDb := dbs.GetDb("lachesis")
	cGetEpochDB := func(epoch idx.Epoch) kvdb.DropableStore {
		return dbs.GetDb(fmt.Sprintf("lachesis-%d", epoch))
	}
	cdb := abft.NewStore(cMainDb, cGetEpochDB, panics("Lachesis store"), abft.DefaultStoreConfig())

	// write genesis

	err := gdb.Migrate()
	if err != nil {
		utils.Fatalf("Failed to migrate Gossip DB: %v", err)
	}
	genesisAtropos, _, isNew, err := gdb.ApplyGenesis(&gossipCfg.Net)
	if err != nil {
		utils.Fatalf("Failed to write Gossip genesis state: %v", err)
	}

	if isNew {
		err = cdb.ApplyGenesis(&abft.Genesis{
			Epoch:      gdb.GetEpoch(),
			Validators: gossipCfg.Net.Genesis.Validators.Build(),
			Atropos:    genesisAtropos,
		})
		if err != nil {
			utils.Fatalf("Failed to write Miniopera genesis state: %v", err)
		}
	}

	err = dbs.Flush(genesisAtropos.Bytes())
	if err != nil {
		utils.Fatalf("Failed to flush genesis state: %v", err)
	}

	if isNew {
		log.Info("Applied genesis state", "hash", genesisAtropos.FullID())
	} else {
		log.Info("Genesis state is already written", "hash", genesisAtropos.FullID())
	}

	// create consensus
	vecClock := vector.NewIndex(panics("Vector clock"), vector.DefaultConfig())
	engine := abft.NewLachesis(cdb, &GossipStoreAdapter{gdb}, &adapters.VectorToDagIndexer{vecClock}, panics("Lachesis"), abft.DefaultConfig())

	return engine, dbs, gdb
}
