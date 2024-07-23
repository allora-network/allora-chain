package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/mint/types"

	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Keeper of the mint store
type Keeper struct {
	cdc              codec.BinaryCodec
	storeService     storetypes.KVStoreService
	accountKeeper    types.AccountKeeper
	stakingKeeper    types.StakingKeeper
	bankKeeper       types.BankKeeper
	emissionsKeeper  types.EmissionsKeeper
	feeCollectorName string

	Schema                                   collections.Schema
	Params                                   collections.Item[types.Params]
	PreviousRewardEmissionPerUnitStakedToken collections.Item[math.LegacyDec]
	PreviousBlockEmission                    collections.Item[math.Int]
	EcosystemTokensMinted                    collections.Item[math.Int]
}

// NewKeeper creates a new mint Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	sk types.StakingKeeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	ek types.EmissionsKeeper,
	feeCollectorName string,
) Keeper {
	// ensure mint module account is set
	if addr := ak.GetModuleAddress(types.ModuleName); addr == nil {
		panic(fmt.Sprintf("the x/%s module account has not been set", types.ModuleName))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:                                      cdc,
		storeService:                             storeService,
		stakingKeeper:                            sk,
		accountKeeper:                            ak,
		bankKeeper:                               bk,
		emissionsKeeper:                          ek,
		feeCollectorName:                         feeCollectorName,
		Params:                                   collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		PreviousRewardEmissionPerUnitStakedToken: collections.NewItem(sb, types.PreviousRewardEmissionPerUnitStakedTokenKey, "previousrewardsemissionsperunitstakedtoken", alloraMath.LegacyDecValue),
		PreviousBlockEmission:                    collections.NewItem(sb, types.PreviousBlockEmissionKey, "previousblockemission", sdk.IntValue),
		EcosystemTokensMinted:                    collections.NewItem(sb, types.EcosystemTokensMintedKey, "ecosystemtokensminted", sdk.IntValue),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// This function increases the ledger that tracks the total tokens minted by the ecosystem treasury
// over the life of the blockchain.
func (k Keeper) AddEcosystemTokensMinted(ctx context.Context, minted math.Int) error {
	curr, err := k.EcosystemTokensMinted.Get(ctx)
	if err != nil {
		return err
	}
	newTotal := curr.Add(minted)
	return k.EcosystemTokensMinted.Set(ctx, newTotal)
}

/// STAKING KEEPER RELATED FUNCTIONS

// StakingTokenSupply implements an alias call to the underlying staking keeper's
// StakingTokenSupply to be used in BeginBlocker.
func (k Keeper) CosmosValidatorStakedSupply(ctx context.Context) (math.Int, error) {
	return k.stakingKeeper.TotalBondedTokens(ctx)
}

/// BANK KEEPER RELATED FUNCTIONS

// MintCoins implements an alias call to the underlying supply keeper's
// MintCoins to be used in BeginBlocker.
func (k Keeper) MintCoins(ctx context.Context, newCoins sdk.Coins) error {
	if newCoins.Empty() {
		// skip as no coins need to be minted
		return nil
	}

	return k.bankKeeper.MintCoins(ctx, types.ModuleName, newCoins)
}

// MoveCoinsFromMintToEcosystem moves freshly minted tokens from the mint module
// which has permissions to create new tokens, to the ecosystem account which
// only has permissions to hold tokens.
func (k Keeper) MoveCoinsFromMintToEcosystem(ctx context.Context, mintedCoins sdk.Coins) error {
	if mintedCoins.Empty() {
		return nil
	}
	return k.bankKeeper.SendCoinsFromModuleToModule(
		ctx,
		types.ModuleName,
		types.EcosystemModuleName,
		mintedCoins,
	)
}

// PayValidatorsFromEcosystem sends funds from the ecosystem
// treasury account to the cosmos network validators rewards account (fee collector)
// PayValidatorsFromEcosystem to be used in BeginBlocker.
func (k Keeper) PayValidatorsFromEcosystem(ctx context.Context, rewards sdk.Coins) error {
	if rewards.Empty() {
		return nil
	}
	return k.bankKeeper.SendCoinsFromModuleToModule(
		ctx,
		types.EcosystemModuleName,
		k.feeCollectorName,
		rewards,
	)
}

// PayAlloraRewardsFromEcosystem sends funds from the ecosystem
// treasury account to the allora reward payout account used in the emissions module
// PayAlloraRewardsFromEcosystem to be used in BeginBlocker.
func (k Keeper) PayAlloraRewardsFromEcosystem(ctx context.Context, rewards sdk.Coins) error {
	if rewards.Empty() {
		return nil
	}
	return k.bankKeeper.SendCoinsFromModuleToModule(
		ctx,
		types.EcosystemModuleName,
		emissionstypes.AlloraRewardsAccountName,
		rewards,
	)
}

// GetTotalCurrTokenSupply implements an alias call to the underlying supply keeper's
// GetTotalCurrTokenSupply to be used in BeginBlocker.
func (k Keeper) GetTotalCurrTokenSupply(ctx context.Context) sdk.Coin {
	return k.bankKeeper.GetSupply(ctx, params.BaseCoinUnit)
}

// returns the quantity of tokens currenty stored in the "ecosystem" module account
// this module account is paid by inference requests and is drained by this mint module
// when forwarding rewards to fee collector and allorarewards accounts
func (k Keeper) GetEcosystemBalance(ctx context.Context, mintDenom string) (math.Int, error) {
	ecosystemAddr := k.accountKeeper.GetModuleAddress(types.EcosystemModuleName)
	return k.bankKeeper.GetBalance(ctx, ecosystemAddr, mintDenom).Amount, nil
}

// Params getter
func (k Keeper) GetParams(ctx context.Context) (types.Params, error) {
	return k.Params.Get(ctx)
}

// What split of the rewards should be given to cosmos validators vs
// allora participants (reputers, forecaster workers, inferrer workers)
func (k Keeper) GetValidatorsVsAlloraPercentReward(ctx context.Context) (alloraMath.Dec, error) {
	emissionsParams, err := k.emissionsKeeper.GetParams(ctx)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return emissionsParams.ValidatorsVsAlloraPercentReward, nil
}

// The last time we paid out rewards, what was the percentage of those rewards that went to staked reputers
// (as opposed to forecaster workers and inferrer workers)
func (k Keeper) GetPreviousPercentageRewardToStakedReputers(ctx context.Context) (math.LegacyDec, error) {
	stakedPercent, err := k.emissionsKeeper.GetPreviousPercentageRewardToStakedReputers(ctx)
	if err != nil {
		return math.LegacyDec{}, err
	}
	return stakedPercent.SdkLegacyDec(), nil
}

// wrapper around emissions keeper call to get the number of blocks expected in a month
func (k Keeper) GetParamsBlocksPerMonth(ctx context.Context) (uint64, error) {
	emissionsParams, err := k.emissionsKeeper.GetParams(ctx)
	if err != nil {
		return 0, err
	}
	return emissionsParams.BlocksPerMonth, nil
}

// wrapper around emissions keeper call to get if whitelist admin
func (k Keeper) IsWhitelistAdmin(ctx context.Context, admin string) (bool, error) {
	return k.emissionsKeeper.IsWhitelistAdmin(ctx, admin)
}
