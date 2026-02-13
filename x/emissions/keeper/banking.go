package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
)

func NewBankingKeeper(bankKeeper BankKeeper, authKeeper AccountKeeper) *BankingKeeper {
	return &BankingKeeper{
		bankKeeper: bankKeeper,
		authKeeper: authKeeper,
	}
}

type BankingKeeper struct {
	bankKeeper BankKeeper
	authKeeper AccountKeeper
}

// wrapper around bank keeper SendCoinsFromModuleToAccount
func (k *BankingKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipient ActorId, amt sdk.Coins) error {
	recipientAddr, err := sdk.AccAddressFromBech32(recipient)
	if err != nil {
		return errorsmod.Wrap(err, "error getting recipient address")
	}
	return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, senderModule, recipientAddr, amt)
}

// wrapper around bank keeper SendCoinsFromAccountToModule
func (k *BankingKeeper) SendCoinsFromAccountToModule(ctx context.Context, sender ActorId, recipientModule string, amt sdk.Coins) error {
	senderAddr, err := sdk.AccAddressFromBech32(sender)
	if err != nil {
		return errorsmod.Wrap(err, "error getting sender address")
	}
	return k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, recipientModule, amt)
}

// wrapper around bank keeper SendCoinsFromModuleToModule
func (k *BankingKeeper) SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, senderModule, recipientModule, amt)
}

// wrapper around bank keeper GetBalance
func (k *BankingKeeper) GetBankBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return k.bankKeeper.GetBalance(ctx, addr, denom)
}

// MoveCoinsFromAlloraRewardsToEcosystem moves funds from the allora rewards account to the ecosystem account
func (k *BankingKeeper) MoveCoinsFromAlloraRewardsToEcosystem(ctx context.Context, amount alloraMath.Dec) error {
	if amount.IsZero() {
		return nil
	}
	amountInt, err := amount.SdkIntTrim()
	if err != nil {
		return errorsmod.Wrap(err, "failed to sdk int trim amount")
	}
	return k.bankKeeper.SendCoinsFromModuleToModule(
		ctx,
		types.AlloraRewardsAccountName,
		minttypes.EcosystemModuleName,
		sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amountInt)),
	)
}

// Gets the total rewards available to be distributed from the Allora Rewards Module Account
func (k *BankingKeeper) GetTotalRewardToDistribute(ctx context.Context) (alloraMath.Dec, error) {
	// Get Allora Rewards Account
	alloraRewardsAccountAddr := k.authKeeper.GetModuleAccount(ctx, types.AlloraRewardsAccountName).GetAddress()
	// Get Total Allocation
	totalReward := k.GetBankBalance(
		ctx,
		alloraRewardsAccountAddr,
		params.DefaultBondDenom).Amount
	totalRewardDec, err := alloraMath.NewDecFromSdkInt(totalReward)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrap(err, "error getting total reward to distribute")
	}
	return totalRewardDec, nil
}
