package keeper

import (
	"context"

	"cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/mint/types"
)

var _ types.QueryServer = queryServer{}

func NewQueryServerImpl(k Keeper) types.QueryServer {
	return queryServer{k}
}

type queryServer struct {
	k Keeper
}

// Params returns params of the mint module.
func (q queryServer) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params, err := q.k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{Params: params}, nil
}

// Inflation returns the annual inflation rate of the mint module.
// note this is the _current_ inflation rate, could change at any time
func (q queryServer) Inflation(ctx context.Context, _ *types.QueryInflationRequest) (*types.QueryInflationResponse, error) {
	// as a crude approximation we take the last blockEmission
	// multiply by the amount of blocks in a year,
	// then use that relative to the current supply as "inflation"
	// Inflation Rate = ((B-A)/A) x 100
	blockEmission, err := q.k.PreviousBlockEmission.Get(ctx)
	if err != nil {
		return nil, err
	}
	blocksPerMonth, err := q.k.GetParamsBlocksPerMonth(ctx)
	if err != nil {
		return nil, err
	}
	EmissionPerYearAtCurrentBlockEmissionRate := blockEmission.
		Mul(math.NewIntFromUint64(blocksPerMonth)).
		Mul(math.NewInt(12)).
		ToLegacyDec()
	totalSupply := q.k.GetTotalCurrTokenSupply(ctx).Amount.ToLegacyDec()
	inflation := EmissionPerYearAtCurrentBlockEmissionRate.Quo(totalSupply).MulInt64(100)
	ret := types.QueryInflationResponse{
		Inflation: inflation,
	}
	return &ret, nil
}
