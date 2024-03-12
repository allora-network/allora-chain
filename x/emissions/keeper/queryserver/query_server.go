package queryserver

import (
	state "github.com/allora-network/allora-chain/x/emissions"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
)

var _ state.QueryServer = queryServer{}

// NewQueryServerImpl returns an implementation of the module QueryServer.
func NewQueryServerImpl(k keeper.Keeper) state.QueryServer {
	return queryServer{k}
}

type queryServer struct {
	k keeper.Keeper
}
