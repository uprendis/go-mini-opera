package api

import (
	"github.com/ethereum/go-ethereum/rpc"
)

// Backend interface provides the common API services
type Backend interface{}

func GetAPIs(apiBackend Backend) []rpc.API {
	return []rpc.API{}
}
