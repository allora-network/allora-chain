package mint

import (
	"context"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/mint/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func BeginBlocker(ctx context.Context, k keeper.Keeper) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	minter, err := k.Minter.Get(ctx)
	if err != nil {
		return err
	}
	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}

	inflationBeforeUpdates := minter.Inflation
	totalCirculatingSupply := k.GetSupply(ctx).Amount

	maxSupply := cosmosMath.NewIntFromBigInt(cosmosMath.NewUintFromString(params.MaxSupply).BigInt())
	currentBlockProvision := cosmosMath.LegacyNewDecFromBigInt(cosmosMath.NewUintFromString(params.CurrentBlockProvision).BigInt())

	// Calculate the new total supply if coins were minted this block
	newTotalCirculatingSupply := totalCirculatingSupply.Add(currentBlockProvision.TruncateInt())

	// Only mint new coins if new total supply would not exceed max supply
	if newTotalCirculatingSupply.LTE(maxSupply) {
		mintedCoin := sdk.NewCoin(params.MintDenom, currentBlockProvision.TruncateInt())
		mintedCoins := sdk.NewCoins(mintedCoin)

		err = k.MintCoins(ctx, mintedCoins)
		if err != nil {
			return err
		}

		// Send the minted coins to the fee collector account
		err = k.AddCollectedFees(ctx, mintedCoins)
		if err != nil {
			return err
		}

		// Halving logic: Check if the current block height is a multiple of the halving interval.
		if sdkCtx.BlockHeight()%int64(params.HalvingInterval) == 0 {
			currentBlockProvision = currentBlockProvision.QuoInt64(2)
			params.CurrentBlockProvision = currentBlockProvision.String()
			minter.AnnualProvisions = minter.AnnualProvisions.QuoInt64(2)
		}

		// Recalculate the inflation rate based on the updated circulating supply and provisions.
		calculatedInflationRate := minter.NextInflationRate(totalCirculatingSupply, currentBlockProvision, params.BlocksPerYear)
		params.InflationRateChange = calculatedInflationRate
		minter.Inflation = calculatedInflationRate

	} else if inflationBeforeUpdates.GT(cosmosMath.LegacyZeroDec()) {
		// If the max supply is reached, set the inflation and annual provisions to zero.
		minter.Inflation = cosmosMath.LegacyZeroDec()
		minter.AnnualProvisions = cosmosMath.LegacyZeroDec()
		params.CurrentBlockProvision = cosmosMath.LegacyZeroDec().String()
	}

	if !inflationBeforeUpdates.Equal(cosmosMath.LegacyZeroDec()) {
		// set new minter
		err = k.Minter.Set(ctx, minter)
		if err != nil {
			return err
		}

		//set new params
		err = k.Params.Set(ctx, params)
		if err != nil {
			return err
		}
	}

	return nil
}
