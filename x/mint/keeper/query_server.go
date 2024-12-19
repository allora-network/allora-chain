package keeper

import (
	"context"

	"cosmossdk.io/errors"

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
	circulatingSupply, _, _, _, _, err := GetCirculatingSupply(ctx, q.k, moduleParams, blockHeight, blocksPerMonth, monthsUnlocked)
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
	blockHeightTarget_e_i_LastCalculated := numberOfRecalcs*blocksPerMonth + 1          //nolint:revive // var-naming: don't use underscores in Go names
	blockHeightTarget_e_i_Next := blockHeightTarget_e_i_LastCalculated + blocksPerMonth //nolint:revive // var-naming: don't use underscores in Go names

	networkStakedTokens, err := GetNumStakedTokens(ctx, q.k)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get number of staked tokens")
	}
	monthsAlreadyUnlocked := q.k.GetMonthsAlreadyUnlocked(ctx)
	_, lockedVestingTokensPreseed,
		lockedVestingTokensSeed, lockedVestingTokensTeam, _ := GetLockedVestingTokens(
		blocksPerMonth,
		math.NewIntFromUint64(blockHeight),
		moduleParams,
		monthsAlreadyUnlocked.ToLegacyDec(),
	)
	circulatingSupply,
		totalSupply,
		lockedVestingTokensTotal,
		ecosystemLocked,
		updatedMonthsUnlocked,
		err := GetCirculatingSupply(ctx, q.k, moduleParams, blockHeight, blocksPerMonth, monthsAlreadyUnlocked)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get circulating supply")
	}
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
	vPercent, err := vPercentADec.SdkLegacyDec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert validators vs allora percent reward to legacy dec")
	}
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

	return &types.QueryServiceEmissionInfoResponse{
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
		MaxSupply:                                totalSupply,
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
		MonthsAlreadyUnlocked:                    monthsAlreadyUnlocked,
		UpdatedMonthsUnlocked:                    updatedMonthsUnlocked,
	}, nil
}
