package rewards

import (
	"testing"
	"time"

	"cosmossdk.io/core/header"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	codecAddress "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// payoutTestFixture wires a real emissions keeper with just enough account and
// bank scaffolding for payoutRewards to run.
type payoutTestFixture struct {
	ctx    sdk.Context
	keeper keeper.Keeper
}

func newPayoutTestFixture(t *testing.T) payoutTestFixture {
	t.Helper()

	params.SetAddressPrefixes()

	keyEmissions := storetypes.NewKVStoreKey("emissions")
	keyAccount := storetypes.NewKVStoreKey("account")
	keyBank := storetypes.NewKVStoreKey("bank")

	testCtx := sdktestutil.DefaultContextWithKeys(map[string]*storetypes.KVStoreKey{
		"emissions": keyEmissions,
		"account":   keyAccount,
		"bank":      keyBank,
	}, map[string]*storetypes.TransientStoreKey{
		"transient_test": storetypes.NewTransientStoreKey("transient_test"),
	}, nil)
	ctx := testCtx.WithHeaderInfo(header.Info{Height: 1, Time: time.Now()}).WithLogger(log.NewNopLogger()) //nolint:exhaustruct

	encCfg := moduletestutil.MakeTestEncodingConfig(auth.AppModule{}, bank.AppModule{})
	var cdc codec.Codec = encCfg.Codec

	maccPerms := map[string][]string{
		"fee_collector":                                  {"minter"},
		types.AlloraStakingAccountName:                   {"burner", "minter", "staking"},
		types.AlloraRewardsAccountName:                   {"minter"},
		types.AlloraPendingRewardForDelegatorAccountName: {"minter"},
	}
	accountKeeper := authkeeper.NewAccountKeeper(
		cdc,
		runtime.NewKVStoreService(keyAccount),
		authtypes.ProtoBaseAccount,
		maccPerms,
		authcodec.NewBech32Codec(params.Bech32PrefixAccAddr),
		params.Bech32PrefixAccAddr,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	bankKeeper := bankkeeper.NewBaseKeeper(
		cdc,
		runtime.NewKVStoreService(keyBank),
		accountKeeper,
		map[string]bool{},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		log.NewNopLogger(),
	)
	emissionsKeeper := keeper.NewKeeper(
		cdc,
		codecAddress.NewBech32Codec(params.Bech32PrefixAccAddr),
		runtime.NewKVStoreService(keyEmissions),
		accountKeeper,
		bankKeeper,
		authtypes.FeeCollectorName,
	)

	return payoutTestFixture{ctx: ctx, keeper: emissionsKeeper}
}

func lastRewardPayoutFailedEvent(t *testing.T, em *sdk.EventManager) *types.EventRewardPayoutFailed {
	t.Helper()
	var out *types.EventRewardPayoutFailed
	for _, ev := range em.Events() {
		msg, err := sdk.ParseTypedEvent(abci.Event(ev))
		if err != nil {
			continue
		}
		if typed, ok := msg.(*types.EventRewardPayoutFailed); ok {
			out = typed
		}
	}
	return out
}

// A reward that cannot be converted to coins emits EventRewardPayoutFailed
// with the reward nonce, the actor and the conversion reason, and is still
// returned as an error.
func TestPayoutRewardsEmitsConversionFailureEvent(t *testing.T) {
	f := newPayoutTestFixture(t)
	em := sdk.NewEventManager()
	ctx := f.ctx.WithEventManager(em)
	nonce := int64(940)

	rewards := []types.TaskReward{
		{
			TopicId: 1,
			Address: "not-a-bech32-address",
			Reward:  alloraMath.NewDecFromInt64(-1),
			Type:    types.WorkerInferenceRewardType,
		},
	}
	errs := payoutRewards(ctx, f.keeper, rewards, nonce)
	require.Len(t, errs, 1)

	event := lastRewardPayoutFailedEvent(t, em)
	require.NotNil(t, event)
	require.Equal(t, uint64(1), event.TopicId)
	require.Equal(t, nonce, event.Nonce)
	require.Equal(t, "not-a-bech32-address", event.Address)
	require.Equal(t, types.ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED, event.ActorType)
	require.True(t, alloraMath.NewDecFromInt64(-1).Equal(event.Amount))
	require.Equal(t, types.RewardPayoutFailureReason_REWARD_PAYOUT_FAILURE_REASON_REWARD_CONVERSION, event.Reason)
}

// A reward whose address does not decode emits EventRewardPayoutFailed with
// the invalid-address reason.
func TestPayoutRewardsEmitsInvalidPayoutAddressEvent(t *testing.T) {
	f := newPayoutTestFixture(t)
	em := sdk.NewEventManager()
	ctx := f.ctx.WithEventManager(em)
	nonce := int64(940)

	rewards := []types.TaskReward{
		{
			TopicId: 1,
			Address: "not-a-bech32-address",
			Reward:  alloraMath.NewDecFromInt64(10),
			Type:    types.WorkerForecastRewardType,
		},
	}
	errs := payoutRewards(ctx, f.keeper, rewards, nonce)
	require.Len(t, errs, 1)

	event := lastRewardPayoutFailedEvent(t, em)
	require.NotNil(t, event)
	require.Equal(t, uint64(1), event.TopicId)
	require.Equal(t, nonce, event.Nonce)
	require.Equal(t, "not-a-bech32-address", event.Address)
	require.Equal(t, types.ActorType_ACTOR_TYPE_FORECASTER, event.ActorType)
	require.True(t, alloraMath.NewDecFromInt64(10).Equal(event.Amount))
	require.Equal(t, types.RewardPayoutFailureReason_REWARD_PAYOUT_FAILURE_REASON_INVALID_PAYOUT_ADDRESS, event.Reason)
}

// A worker whose registration info is missing emits EventRewardPayoutFailed
// with the worker-info-lookup reason.
func TestPayoutRewardsEmitsWorkerInfoLookupFailureEvent(t *testing.T) {
	f := newPayoutTestFixture(t)
	em := sdk.NewEventManager()
	ctx := f.ctx.WithEventManager(em)
	nonce := int64(940)

	// Valid bech32, but no worker registered under it.
	worker := sdk.AccAddress([]byte("worker1_____________")).String()
	rewards := []types.TaskReward{
		{
			TopicId: 1,
			Address: worker,
			Reward:  alloraMath.NewDecFromInt64(10),
			Type:    types.WorkerInferenceRewardType,
		},
	}
	errs := payoutRewards(ctx, f.keeper, rewards, nonce)
	require.Len(t, errs, 1)

	event := lastRewardPayoutFailedEvent(t, em)
	require.NotNil(t, event)
	require.Equal(t, uint64(1), event.TopicId)
	require.Equal(t, nonce, event.Nonce)
	require.Equal(t, worker, event.Address)
	require.Equal(t, types.ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED, event.ActorType)
	require.True(t, alloraMath.NewDecFromInt64(10).Equal(event.Amount))
	require.Equal(t, types.RewardPayoutFailureReason_REWARD_PAYOUT_FAILURE_REASON_WORKER_INFO_LOOKUP, event.Reason)
}

// A worker payout whose module-to-account send fails (empty rewards account)
// emits EventRewardPayoutFailed with the account-send reason.
func TestPayoutRewardsEmitsAccountSendFailureEvent(t *testing.T) {
	f := newPayoutTestFixture(t)
	em := sdk.NewEventManager()
	ctx := f.ctx.WithEventManager(em)
	nonce := int64(940)

	owner := sdk.AccAddress([]byte("owner_______________")).String()
	worker := sdk.AccAddress([]byte("worker2_____________")).String()
	topicId := uint64(1)
	require.NoError(t, f.keeper.GetWorkerKeeper().InsertWorker(ctx, topicId, worker, types.OffchainNode{
		Owner:       owner,
		NodeAddress: worker,
	}))

	rewards := []types.TaskReward{
		{
			TopicId: topicId,
			Address: worker,
			Reward:  alloraMath.NewDecFromInt64(10),
			Type:    types.WorkerForecastRewardType,
		},
	}
	// The rewards module account holds nothing, so the send fails.
	errs := payoutRewards(ctx, f.keeper, rewards, nonce)
	require.Len(t, errs, 1)

	event := lastRewardPayoutFailedEvent(t, em)
	require.NotNil(t, event)
	require.Equal(t, topicId, event.TopicId)
	require.Equal(t, nonce, event.Nonce)
	require.Equal(t, worker, event.Address)
	require.Equal(t, types.ActorType_ACTOR_TYPE_FORECASTER, event.ActorType)
	require.True(t, alloraMath.NewDecFromInt64(10).Equal(event.Amount))
	require.Equal(t, types.RewardPayoutFailureReason_REWARD_PAYOUT_FAILURE_REASON_ACCOUNT_SEND, event.Reason)
}

// A reputer reward whose restake fails (zero stake to add after conversion is
// not possible here, so use a reward that converts to a positive amount but an
// empty module account) emits EventRewardPayoutFailed with the module-send
// reason.
func TestPayoutRewardsEmitsModuleSendFailureEvent(t *testing.T) {
	f := newPayoutTestFixture(t)
	em := sdk.NewEventManager()
	ctx := f.ctx.WithEventManager(em)
	nonce := int64(940)

	reputer := sdk.AccAddress([]byte("reputer_____________")).String()
	rewards := []types.TaskReward{
		{
			TopicId: 1,
			Address: reputer,
			Reward:  alloraMath.NewDecFromInt64(10),
			Type:    types.ReputerAndDelegatorRewardType,
		},
	}
	// The rewards module account holds nothing, so the module-to-module send fails.
	errs := payoutRewards(ctx, f.keeper, rewards, nonce)
	require.Len(t, errs, 1)

	event := lastRewardPayoutFailedEvent(t, em)
	require.NotNil(t, event)
	require.Equal(t, uint64(1), event.TopicId)
	require.Equal(t, nonce, event.Nonce)
	require.Equal(t, reputer, event.Address)
	require.Equal(t, types.ActorType_ACTOR_TYPE_REPUTER, event.ActorType)
	require.Equal(t, types.RewardPayoutFailureReason_REWARD_PAYOUT_FAILURE_REASON_MODULE_SEND, event.Reason)
}
