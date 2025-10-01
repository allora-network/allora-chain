package types // noalias

import (
	context "context"

	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"

	"cosmossdk.io/core/address"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

// StakingKeeper defines the expected staking keeper
type StakingKeeper interface {
	TotalBondedTokens(ctx context.Context) (math.Int, error)
}

// AccountKeeper defines the contract required for account APIs.
type AccountKeeper interface {
	AddressCodec() address.Codec
	GetModuleAddress(name string) sdk.AccAddress

	SetModuleAccount(context.Context, sdk.ModuleAccountI)
	GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI
}

// BankKeeper defines the contract needed to be fulfilled for banking and supply
// dependencies.
type BankKeeper interface {
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	MintCoins(ctx context.Context, name string, amt sdk.Coins) error
	GetSupply(ctx context.Context, denom string) sdk.Coin
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
}

type EmissionsKeeper interface {
	GetTotalStake(ctx context.Context) (math.Int, error)
	GetPreviousPercentageRewardToStakedReputers(ctx context.Context) (alloraMath.Dec, error)
	GetParams(ctx context.Context) (emissionstypes.Params, error)
	SetRewardCurrentBlockEmission(ctx context.Context, emission math.Int) error
	IsWhitelistAdmin(ctx context.Context, admin string) (bool, error)
	SetParams(ctx context.Context, params emissionstypes.Params) error
}

// SchedulerKeeper defines the expected interface for the scheduler keeper.
type SchedulerKeeper interface {
	RegisterTaskHandlers(taskHandlers schedulertypes.TaskHandlers) error
	ScheduleTask(ctx context.Context, typename string, id schedulertypes.TaskID, args proto.Message, scheduleOpts ...schedulertypes.SchedulingOption) error
}

// used for testing
type MintKeeper interface {
	CosmosValidatorStakedSupply(ctx context.Context) (math.Int, error)
	GetEmissionsKeeperTotalStake(ctx context.Context) (math.Int, error)
	GetTotalCurrTokenSupply(ctx context.Context) sdk.Coin
	GetPreviousPercentageRewardToStakedReputers(ctx context.Context) (math.LegacyDec, error)
	GetPreviousRewardEmissionPerUnitStakedToken(ctx context.Context) (math.LegacyDec, error)
	GetEcosystemMintSupplyRemaining(ctx context.Context, params Params) (math.Int, error)
	GetEcosystemBalance(ctx context.Context, mintDenom string) (math.Int, error)
	GetMonthsAlreadyUnlocked(ctx context.Context) math.Int
	SetMonthsAlreadyUnlocked(ctx context.Context, monthsAlreadyUnlocked math.Int) error
}
