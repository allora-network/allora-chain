package keeper

import (
	"cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TODO: write the actual function that handles vesting locked tokens
// this is just a mocked function for example purposes
// return the uncirculating supply, i.e. tokens on a vesting schedule
func numberLockedTokens() math.Int {
	return math.ZeroInt()
}

/*
	  # TOKEN EMISSION
	        old_emission_per_token = holders['ecosystem']['tokens_emission'][i-1]/network_staked[i-1]
	        target_emission_per_token = np.max(
				[
					0,
					f_emission*holders['ecosystem']['tokens_total'][i]/network_staked[i]*network_unlocked[i]/network_total[i]
				]
				target_emission_per_token = np.max([0,f_emission*(holders['ecosystem']['tokens_total'][i])/network_staked[i]*network_unlocked[i]/network_total[i]])
			)
	        if i == 1:
	            old_emission_per_token = target_emission_per_token
	        holders['ecosystem']['tokens_emission'][i] = network_staked[i]*(alpha_emission_per_token*target_emission_per_token+(1-alpha_emission_per_token)*old_emission_per_token)
	        holders['ecosystem']['tokens_unlocked'][i] += holders['ecosystem']['tokens_emission'][i]
*/
func TargetRewardEmissionPerUnitStakedToken(
	ctx sdk.Context,
	k Keeper,
	f_emission math.LegacyDec,
	alphaEmission math.LegacyDec,
) (math.LegacyDec, error) {
	previousReward, err := k.PreviousReward.Get(ctx)
	if err != nil {
		return math.LegacyDec{}, err
	}
	networkLocked := numberLockedTokens()
	networkTotal := k.GetSupply(ctx).Amount
	networkUnlocked := networkTotal.Sub(networkLocked)
	if networkUnlocked.IsNegative() {
		networkUnlocked = math.ZeroInt()
	}
	cosmosValidatorsStaked, err := k.StakingTokenSupply(ctx)
	if err != nil {
		return math.LegacyDec{}, err
	}
	reputersStaked, err := k.emissionsKeeper.GetTotalStake(ctx)
	if err != nil {
		return math.LegacyDec{}, err
	}
	networkStaked := cosmosValidatorsStaked.Add(math.NewIntFromBigInt(reputersStaked.BigInt()))
	ecosystemAddr := k.accountKeeper.GetModuleAddress(types.EcosystemModuleName)
	params, err := k.Params.Get(ctx)
	if err != nil {
		return math.LegacyDec{}, err
	}
	ecosystemBalance := k.bankKeeper.GetBalance(ctx, ecosystemAddr, params.MintDenom).Amount
	targetEmissionPerToken := f_emission.MulInt(ecosystemBalance).QuoInt(networkStaked).MulInt(networkUnlocked).QuoInt(networkTotal)
	if targetEmissionPerToken.IsNegative() {
		targetEmissionPerToken = math.LegacyZeroDec()
	}
	previousRewardEmissionPerToken := math.LegacyNewDecFromInt(previousReward).QuoInt(networkStaked)
	if ctx.BlockHeight() == 1 {
		previousRewardEmissionPerToken = targetEmissionPerToken
	}
	ema := alphaEmission.Mul(targetEmissionPerToken).Add(math.LegacyOneDec().Sub(alphaEmission).Mul(previousRewardEmissionPerToken))
	ret := ema.MulInt(networkStaked)
	return ret, nil
}
