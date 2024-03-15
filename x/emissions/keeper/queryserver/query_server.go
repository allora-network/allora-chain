package queryserver

import (
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
)

var _ types.QueryServer = queryServer{}

// NewQueryServerImpl returns an implementation of the module QueryServer.
func NewQueryServerImpl(k keeper.Keeper) types.QueryServer {
	return queryServer{k}
}

type queryServer struct {
	k keeper.Keeper
}
