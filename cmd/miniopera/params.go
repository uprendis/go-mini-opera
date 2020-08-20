package main

import (
	"github.com/ethereum/go-ethereum/params"
)

func overrideParams() {
	// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
	// experimental RLPx v5 topic-discovery miniopera.
	params.DiscoveryV5Bootnodes = []string{}

	// MainnetBootnodes are the enode URLs of the discovery V4 P2P bootstrap nodes running on
	// the main Opera miniopera.
	params.MainnetBootnodes = []string{}

	// TestnetBootnodes are the enode URLs of the discovery V4 P2P bootstrap nodes running on
	//  the test Opera miniopera.
	params.TestnetBootnodes = []string{}

	params.RinkebyBootnodes = []string{}
	params.GoerliBootnodes = []string{}

}
