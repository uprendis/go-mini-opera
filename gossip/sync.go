package gossip

// syncer is responsible for periodically synchronising with the miniopera, both
// downloading hashes and events as well as handling the announcement handler.
func (pm *ProtocolManager) syncer() {
	// Start and ensure cleanup of sync mechanisms
	for {
		select {
		case <-pm.newPeerCh:
		case <-pm.noMorePeers:
			return
		}
	}
}
