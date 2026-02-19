package queryserver

import (
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

//nolint:exhaustruct
var _ types.QueryServiceServer = queryServer{k: keeper.Keeper{}}

// NewQueryServerImpl returns an implementation of the module QueryServer.
func NewQueryServerImpl(
	k keeper.Keeper,
) types.QueryServiceServer {
	return queryServer{
		k:   k,
		pk:  k.GetParamsKeeper(),
		tk:  k.GetTopicKeeper(),
		wlk: k.GetWhitelistsKeeper(),
		rlk: k.GetReputerLossKeeper(),
		bk:  k.GetBankingKeeper(),
		sk:  k.GetStakingKeeper(),
		sck: k.GetScoresKeeper(),
		wk:  k.GetWorkerKeeper(),
		wtk: k.GetWeightsKeeper(),
		nk:  k.GetNonceKeeper(),
		rk:  k.GetRegretsKeeper(),
	}
}

type queryServer struct {
	k keeper.Keeper

	pk  *keeper.ParamsKeeper
	tk  *keeper.TopicKeeper
	wlk *keeper.WhitelistsKeeper
	rlk *keeper.ReputerLossKeeper
	bk  *keeper.BankingKeeper
	sk  *keeper.StakingKeeper
	sck *keeper.ScoresKeeper
	wk  *keeper.WorkerKeeper
	wtk *keeper.WeightsKeeper
	nk  *keeper.NonceKeeper
	rk  *keeper.RegretsKeeper
}
