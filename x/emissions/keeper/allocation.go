package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	state "github.com/upshot-tech/protocol-state-machine-module"
)

// BeginBlocker is called at the start of every block. Redirects funds to the upshot pool
func (k *Keeper) BeginBlocker(ctx sdk.Context) error {
	if ctx.BlockHeight() > 1 {
		if err := k.RedirectFeeTokens(ctx); err != nil {
			return err
		}
	}
	return nil
}

// RedirectFeeTokens redirects the collected fees from a block to the upshot pool that pays
// upshot reputers and worker nodes
func (k *Keeper) RedirectFeeTokens(ctx sdk.Context) error {
	feeCollectorName := authtypes.FeeCollectorName
	// fetch and clear the collected fees for distribution, since this is
	// called in BeginBlock, collected fees will be from the previous block
	// (and distributed to the previous proposer)
	feeCollector := k.authKeeper.GetModuleAccount(ctx, feeCollectorName)
	feesCollectedInt := k.bankKeeper.GetAllBalances(ctx, feeCollector.GetAddress())
	fmt.Println("Fees collected from the previous block: ", feesCollectedInt)
	for _, coin := range feesCollectedInt {
		fmt.Println("Coin: ", coin)
		fmt.Println("Coin denom: ", coin.Denom)
	}
	// transfer collected fees to the upshot module account
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, feeCollectorName, state.ModuleName, feesCollectedInt); err != nil {
		fmt.Println("Error sending coins from module to module: ", err)
		return err
	}
	accumulatedEpochRewards := k.GetAccumulatedEpochRewards(ctx)
	fmt.Println("Accumulated fees: ", accumulatedEpochRewards)

	return nil
}
