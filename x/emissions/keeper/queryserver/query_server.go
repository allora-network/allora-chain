package queryserver

import (
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

var _ types.QueryServiceServer = queryServer{k: keeper.Keeper{}}

// NewQueryServerImpl returns an implementation of the module QueryServer.
func NewQueryServerImpl(k keeper.Keeper) types.QueryServiceServer {
	return queryServer{k}
}

type queryServer struct {
	k keeper.Keeper
}
