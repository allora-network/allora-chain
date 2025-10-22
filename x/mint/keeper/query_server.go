package keeper

import (
	"context"

	"cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ types.QueryServiceServer = queryServer{} //nolint: exhaustruct

func NewQueryServerImpl(k Keeper) types.QueryServiceServer {
	return queryServer{k}
}

type queryServer struct {
	k Keeper
}

// Params returns params of the mint module.
func (q queryServer) Params(ctx context.Context, _ *types.QueryServiceParamsRequest) (*types.QueryServiceParamsResponse, error) {
	params, err := q.k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryServiceParamsResponse{Params: params}, nil
}

// Inflation returns the annual inflation rate of the mint module.
// note this is the _current_ inflation rate, could change at any time
func (q queryServer) Inflation(ctx context.Context, _ *types.QueryServiceInflationRequest) (*types.QueryServiceInflationResponse, error) {
	// as a crude approximation we take the last blockEmission
	// multiply by the amount of blocks in a year,
	// then use that relative to the current circulating supply as "inflation"
	// Inflation Rate = ((B-A)/A) x 100
	moduleParams, err := q.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	blockHeight := uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())
	startingEmissionsBlockHeight, err := q.k.GetStartingEmissionsBlockHeight(ctx)
	if err != nil {
		return nil, err
	}
	blockCountSinceTGE := blockHeight - (uint64(startingEmissionsBlockHeight))
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
	monthsUnlocked := q.k.GetMonthsAlreadyUnlocked(ctx)
	circulatingSupply, _, _, _, _, err := GetCirculatingSupply(ctx, q.k, moduleParams, blockCountSinceTGE, blocksPerMonth, monthsUnlocked)
	if err != nil {
		return nil, err
	}
	inflation := EmissionPerYearAtCurrentBlockEmissionRate.QuoInt(circulatingSupply).MulInt64(100)
	ret := types.QueryServiceInflationResponse{
		Inflation: inflation,
	}
	return &ret, nil
}

// mint and inflation emission rate endpoint
// nice way to access live chain data
func (q queryServer) EmissionInfo(ctx context.Context, _ *types.QueryServiceEmissionInfoRequest) (*types.QueryServiceEmissionInfoResponse, error) {
	params, eventInfo, err := q.k.GetEmissionInfo(ctx)
	if err != nil {
		return nil, err
	}

	// Convert the event info to the query response
	return &types.QueryServiceEmissionInfoResponse{
		Params:                                   *params,
		EcosystemBalance:                         eventInfo.EcosystemBalance,
		PreviousBlockEmission:                    eventInfo.PreviousBlockEmission,
		EcosystemMintSupplyRemaining:             eventInfo.EcosystemMintSupplyRemaining,
		BlocksPerMonth:                           eventInfo.BlocksPerMonth,
		BlockHeightTargetEILastCalculated:        eventInfo.BlockHeightTargetEILastCalculated,
		BlockHeightTargetEINextCalculated:        eventInfo.BlockHeightTargetEINextCalculated,
		NetworkStakedTokens:                      eventInfo.NetworkStakedTokens,
		LockedVestingTokensTotal:                 eventInfo.LockedVestingTokensTotal,
		LockedVestingTokensInvestorsPreseed:      eventInfo.LockedVestingTokensInvestorsPreseed,
		LockedVestingTokensInvestorsSeed:         eventInfo.LockedVestingTokensInvestorsSeed,
		LockedVestingTokensTeam:                  eventInfo.LockedVestingTokensTeam,
		EcosystemLocked:                          eventInfo.EcosystemLocked,
		CirculatingSupply:                        eventInfo.CirculatingSupply,
		MaxSupply:                                eventInfo.MaxSupply,
		TargetEmissionRatePerUnitStakedToken:     eventInfo.TargetEmissionRatePerUnitStakedToken,
		ReputersPercent:                          eventInfo.ReputersPercent,
		ValidatorsPercent:                        eventInfo.ValidatorsPercent,
		MaximumMonthlyEmissionPerUnitStakedToken: eventInfo.MaximumMonthlyEmissionPerUnitStakedToken,
		TargetRewardEmissionPerUnitStakedToken:   eventInfo.TargetRewardEmissionPerUnitStakedToken,
		EmissionPerUnitStakedToken:               eventInfo.EmissionPerUnitStakedToken,
		EmissionPerMonth:                         eventInfo.EmissionPerMonth,
		BlockEmission:                            eventInfo.BlockEmission,
		ValidatorCut:                             eventInfo.ValidatorCut,
		AlloraRewardsCut:                         eventInfo.AlloraRewardsCut,
		PreviousRewardEmissionPerUnitStakedToken: eventInfo.PreviousRewardEmissionPerUnitStakedToken,
		MonthsAlreadyUnlocked:                    eventInfo.MonthsAlreadyUnlocked,
		UpdatedMonthsUnlocked:                    eventInfo.UpdatedMonthsUnlocked,
		StartingEmissionsBlockHeight:             eventInfo.StartingEmissionsBlockHeight,
	}, nil
}
