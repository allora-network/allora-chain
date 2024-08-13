package keeper

import (
	"context"

	"cosmossdk.io/errors"

	"cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

// mint and inflation emission rate endpoint
// nice way to access live chain data
func (q queryServer) EmissionInfo(ctx context.Context, _ *types.QueryEmissionInfoRequest) (*types.QueryEmissionInfoResponse, error) {
	moduleParams, err := q.k.Params.Get(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get module params")
	}

	ecosystemBalance, err := q.k.GetEcosystemBalance(ctx, moduleParams.MintDenom)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ecosystem balance")
	}

	previousBlockEmission, err := q.k.PreviousBlockEmission.Get(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get previous block emission")
	}

	ecosystemMintSupplyRemaining, err := q.k.GetEcosystemMintSupplyRemaining(ctx, moduleParams)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ecosystem mint supply remaining")
	}

	blocksPerMonth, err := q.k.GetParamsBlocksPerMonth(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get blocks per month")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := uint64(sdkCtx.BlockHeight())
	numberOfRecalcs := blockHeight / blocksPerMonth
	blockHeightTarget_e_i_LastCalculated := numberOfRecalcs*blocksPerMonth + 1
	blockHeightTarget_e_i_Next := blockHeightTarget_e_i_LastCalculated + blocksPerMonth

	networkStakedTokens, err := GetNumStakedTokens(ctx, q.k)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get number of staked tokens")
	}
	lockedVestingTokensTotal, lockedVestingTokensPreseed,
		lockedVestingTokensSeed, lockedVestingTokensTeam := GetLockedVestingTokens(
		blocksPerMonth,
		math.NewIntFromUint64(blockHeight),
		moduleParams)
	ecosystemLocked := ecosystemBalance.Add(ecosystemMintSupplyRemaining)
	circulatingSupply := moduleParams.MaxSupply.Sub(lockedVestingTokensTotal).Sub(ecosystemLocked)
	targetRewardEmissionPerUnitStakedToken,
		err := GetTargetRewardEmissionPerUnitStakedToken(
		moduleParams.FEmission,
		ecosystemLocked,
		networkStakedTokens,
		circulatingSupply,
		moduleParams.MaxSupply,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get target reward emission per unit staked token")
	}
	reputersPercent, err := q.k.GetPreviousPercentageRewardToStakedReputers(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get previous percentage reward to staked reputers")
	}
	vPercentADec, err := q.k.GetValidatorsVsAlloraPercentReward(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get validators vs allora percent reward")
	}
	vPercent := vPercentADec.SdkLegacyDec()
	maximumMonthlyEmissionPerUnitStakedToken := GetMaximumMonthlyEmissionPerUnitStakedToken(
		moduleParams.MaximumMonthlyPercentageYield,
		reputersPercent,
		vPercent,
	)
	targetRewardEmissionPerUnitStakedToken = GetCappedTargetEmissionPerUnitStakedToken(
		targetRewardEmissionPerUnitStakedToken,
		maximumMonthlyEmissionPerUnitStakedToken,
	)
	var previousRewardEmissionPerUnitStakedToken math.LegacyDec
	// if this is the first month/time we're calculating the target emission...
	if blockHeight < blocksPerMonth {
		previousRewardEmissionPerUnitStakedToken = targetRewardEmissionPerUnitStakedToken
	} else {
		previousRewardEmissionPerUnitStakedToken, err = q.k.GetPreviousRewardEmissionPerUnitStakedToken(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get previous reward emission per unit staked token")
		}
	}
	emissionPerUnitStakedToken := GetExponentialMovingAverage(
		targetRewardEmissionPerUnitStakedToken,
		moduleParams.OneMonthSmoothingDegree,
		previousRewardEmissionPerUnitStakedToken,
	)
	emissionPerMonth := GetTotalEmissionPerMonth(emissionPerUnitStakedToken, networkStakedTokens)
	blockEmission := emissionPerMonth.
		Quo(math.NewIntFromUint64(blocksPerMonth))
	validatorCut := vPercent.Mul(blockEmission.ToLegacyDec()).TruncateInt()
	alloraRewardsCut := blockEmission.Sub(validatorCut)

	return &types.QueryEmissionInfoResponse{
		Params:                                   moduleParams,
		EcosystemBalance:                         ecosystemBalance,
		PreviousBlockEmission:                    previousBlockEmission,
		EcosystemMintSupplyRemaining:             ecosystemMintSupplyRemaining,
		BlocksPerMonth:                           blocksPerMonth,
		BlockHeightTargetEILastCalculated:        blockHeightTarget_e_i_LastCalculated,
		BlockHeightTargetEINextCalculated:        blockHeightTarget_e_i_Next,
		NetworkStakedTokens:                      networkStakedTokens,
		LockedVestingTokensTotal:                 lockedVestingTokensTotal,
		LockedVestingTokensInvestorsPreseed:      lockedVestingTokensPreseed,
		LockedVestingTokensInvestorsSeed:         lockedVestingTokensSeed,
		LockedVestingTokensTeam:                  lockedVestingTokensTeam,
		EcosystemLocked:                          ecosystemLocked,
		CirculatingSupply:                        circulatingSupply,
		MaxSupply:                                moduleParams.MaxSupply,
		TargetEmissionRatePerUnitStakedToken:     targetRewardEmissionPerUnitStakedToken,
		ReputersPercent:                          reputersPercent,
		ValidatorsPercent:                        vPercent,
		MaximumMonthlyEmissionPerUnitStakedToken: maximumMonthlyEmissionPerUnitStakedToken,
		TargetRewardEmissionPerUnitStakedToken:   targetRewardEmissionPerUnitStakedToken,
		EmissionPerUnitStakedToken:               emissionPerUnitStakedToken,
		EmissionPerMonth:                         emissionPerMonth,
		BlockEmission:                            blockEmission,
		ValidatorCut:                             validatorCut,
		AlloraRewardsCut:                         alloraRewardsCut,
		PreviousRewardEmissionPerUnitStakedToken: previousRewardEmissionPerUnitStakedToken,
	}, nil
}
