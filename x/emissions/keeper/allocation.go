package keeper

import (
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

	// transfer collected fees to the upshot module account
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, feeCollectorName, state.ModuleName, feesCollectedInt); err != nil {
		return err
	}

	return nil
}
