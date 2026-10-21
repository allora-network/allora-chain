package rewards

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/header"
	cosmosMath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
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
	ctx               sdk.Context
	keeper            keeper.Keeper
	bankKeeper        bankkeeper.BaseKeeper
	emissionsStoreKey *storetypes.KVStoreKey
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

	encCfg := moduletestutil.MakeTestEncodingConfig(auth.AppModule{AppModuleBasic: auth.AppModuleBasic{}}, bank.AppModule{})
	cdc := encCfg.Codec

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

	return payoutTestFixture{ctx: ctx, keeper: emissionsKeeper, bankKeeper: bankKeeper, emissionsStoreKey: keyEmissions}
}

// fundRewardsAccount mints coins into the rewards module account so sends
// past the funding gate can proceed.
func (f payoutTestFixture) fundRewardsAccount(t *testing.T, amount cosmosMath.Int) {
	t.Helper()
	require.NoError(t, f.bankKeeper.MintCoins(
		f.ctx,
		types.AlloraRewardsAccountName,
		sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amount)),
	))
}

func allRewardPayoutFailedEvents(t *testing.T, em *sdk.EventManager) []*types.EventRewardPayoutFailed {
	t.Helper()
	out := make([]*types.EventRewardPayoutFailed, 0)
	for _, ev := range em.Events() {
		msg, err := sdk.ParseTypedEvent(abci.Event(ev))
		if err != nil {
			continue
		}
		if typed, ok := msg.(*types.EventRewardPayoutFailed); ok {
			out = append(out, typed)
		}
	}
	return out
}

// Every payout failure where the actor is not paid emits exactly one
// EventRewardPayoutFailed naming the reason for the site that failed.
func TestPayoutRewardsEmitsFailureReasonPerSite(t *testing.T) {
	nonce := int64(940)
	worker := sdk.AccAddress([]byte("worker______________")).String()
	reputer := sdk.AccAddress([]byte("reputer_____________")).String()
	owner := sdk.AccAddress([]byte("owner_______________")).String()

	cases := []struct {
		name      string
		reward    types.TaskReward
		setup     func(t *testing.T, f payoutTestFixture)
		actorType types.ActorType
		reason    types.RewardPayoutFailureReason
	}{
		{
			name:      "negative reward",
			reward:    types.TaskReward{TopicId: 1, Address: worker, Reward: alloraMath.NewDecFromInt64(-1), Type: types.WorkerInferenceRewardType},
			setup:     func(t *testing.T, f payoutTestFixture) { t.Helper() },
			actorType: types.ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED,
			reason:    types.RewardPayoutFailureReason_REWARD_PAYOUT_FAILURE_REASON_NEGATIVE_REWARD,
		},
		{
			name: "reward conversion",
			// A NaN cannot be converted to an integer coin amount.
			reward:    types.TaskReward{TopicId: 1, Address: worker, Reward: alloraMath.NewNaN(), Type: types.WorkerInferenceRewardType},
			setup:     func(t *testing.T, f payoutTestFixture) { t.Helper() },
			actorType: types.ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED,
			reason:    types.RewardPayoutFailureReason_REWARD_PAYOUT_FAILURE_REASON_REWARD_CONVERSION,
		},
		{
			name: "invalid payout address",
			reward: types.TaskReward{TopicId: 1, Address: "not-a-bech32-address",
				Reward: alloraMath.NewDecFromInt64(10), Type: types.WorkerForecastRewardType},
			setup:     func(t *testing.T, f payoutTestFixture) { t.Helper() },
			actorType: types.ActorType_ACTOR_TYPE_FORECASTER,
			reason:    types.RewardPayoutFailureReason_REWARD_PAYOUT_FAILURE_REASON_INVALID_PAYOUT_ADDRESS,
		},
		{
			name: "worker info lookup",
			reward: types.TaskReward{TopicId: 1, Address: worker,
				Reward: alloraMath.NewDecFromInt64(10), Type: types.WorkerInferenceRewardType},
			// Valid bech32, but no worker registered under it.
			setup:     func(t *testing.T, f payoutTestFixture) { t.Helper() },
			actorType: types.ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED,
			reason:    types.RewardPayoutFailureReason_REWARD_PAYOUT_FAILURE_REASON_WORKER_INFO_LOOKUP,
		},
		{
			name: "account send",
			reward: types.TaskReward{TopicId: 1, Address: worker,
				Reward: alloraMath.NewDecFromInt64(10), Type: types.WorkerForecastRewardType},
			setup: func(t *testing.T, f payoutTestFixture) {
				t.Helper()
				require.NoError(t, f.keeper.GetWorkerKeeper().InsertWorker(f.ctx, 1, worker, types.OffchainNode{
					Owner:       owner,
					NodeAddress: worker,
				}))
				// The rewards module account holds nothing, so the send fails.
			},
			actorType: types.ActorType_ACTOR_TYPE_FORECASTER,
			reason:    types.RewardPayoutFailureReason_REWARD_PAYOUT_FAILURE_REASON_ACCOUNT_SEND,
		},
		{
			name: "module send",
			reward: types.TaskReward{TopicId: 1, Address: reputer,
				Reward: alloraMath.NewDecFromInt64(10), Type: types.ReputerAndDelegatorRewardType},
			// The rewards module account holds nothing, so the module-to-module send fails.
			setup:     func(t *testing.T, f payoutTestFixture) { t.Helper() },
			actorType: types.ActorType_ACTOR_TYPE_REPUTER,
			reason:    types.RewardPayoutFailureReason_REWARD_PAYOUT_FAILURE_REASON_MODULE_SEND,
		},
		{
			name: "add stake",
			reward: types.TaskReward{TopicId: 1, Address: "not-a-bech32-address",
				Reward: alloraMath.NewDecFromInt64(10), Type: types.ReputerAndDelegatorRewardType},
			// The reputer branch never decodes the address, so with the rewards
			// account funded the send succeeds and AddReputerStake fails on
			// SetStakeReputerAuthority's bech32 validation.
			setup: func(t *testing.T, f payoutTestFixture) {
				t.Helper()
				f.fundRewardsAccount(t, cosmosMath.NewInt(1000))
			},
			actorType: types.ActorType_ACTOR_TYPE_REPUTER,
			reason:    types.RewardPayoutFailureReason_REWARD_PAYOUT_FAILURE_REASON_ADD_STAKE,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newPayoutTestFixture(t)
			tc.setup(t, f)
			em := sdk.NewEventManager()
			ctx := f.ctx.WithEventManager(em)

			errs := payoutRewards(ctx, f.keeper, []types.TaskReward{tc.reward}, nonce)
			require.Len(t, errs, 1)

			events := allRewardPayoutFailedEvents(t, em)
			require.Len(t, events, 1)
			event := events[0]
			require.Equal(t, tc.reward.TopicId, event.TopicId)
			require.Equal(t, nonce, event.Nonce)
			require.Equal(t, tc.reward.Address, event.Address)
			require.Equal(t, tc.actorType, event.ActorType)
			require.True(t, tc.reward.Reward.Equal(event.Amount))
			require.Equal(t, tc.reason, event.Reason)
		})
	}
}

// A successful payout emits no EventRewardPayoutFailed.
func TestPayoutRewardsSuccessEmitsNoFailureEvent(t *testing.T) {
	f := newPayoutTestFixture(t)
	f.fundRewardsAccount(t, cosmosMath.NewInt(1000))
	em := sdk.NewEventManager()
	ctx := f.ctx.WithEventManager(em)
	nonce := int64(940)

	reputer := sdk.AccAddress([]byte("reputer_____________")).String()
	rewards := []types.TaskReward{
		{TopicId: 1, Address: reputer, Reward: alloraMath.NewDecFromInt64(10), Type: types.ReputerAndDelegatorRewardType},
	}
	errs := payoutRewards(ctx, f.keeper, rewards, nonce)
	require.Empty(t, errs)
	require.Empty(t, allRewardPayoutFailedEvents(t, em))
}

// An inclusion-count failure happens after the actor is paid, so it emits no
// EventRewardPayoutFailed; the actor still lands in the settled list.
func TestPayoutRewardsInclusionCountFailureEmitsNoFailureEvent(t *testing.T) {
	f := newPayoutTestFixture(t)
	f.fundRewardsAccount(t, cosmosMath.NewInt(1000))
	em := sdk.NewEventManager()
	ctx := f.ctx.WithEventManager(em)
	nonce := int64(940)

	owner := sdk.AccAddress([]byte("owner_______________")).String()
	worker := sdk.AccAddress([]byte("worker______________")).String()
	topicId := uint64(1)
	require.NoError(t, f.keeper.GetWorkerKeeper().InsertWorker(ctx, topicId, worker, types.OffchainNode{
		Owner:       owner,
		NodeAddress: worker,
	}))

	// Corrupt the inclusion count so the post-send increment fails on read:
	// the count is a uint64 collection, and an undecodable value errors the
	// Get inside the increment. The key is built with the same prefix and
	// pair codec the keeper's map uses.
	countKey := collections.Join(topicId, worker)
	keyCodec := collections.PairKeyCodec(collections.Uint64Key, collections.StringKey)
	encodedKey := make([]byte, keyCodec.Size(countKey))
	_, err := keyCodec.Encode(encodedKey, countKey)
	require.NoError(t, err)
	storeKey := append(types.CountInfererInclusionsInTopicKey.Bytes(), encodedKey...)
	kvStore := f.ctx.KVStore(f.emissionsStoreKey)
	kvStore.Set(storeKey, []byte{0xff})

	rewards := []types.TaskReward{
		{TopicId: topicId, Address: worker, Reward: alloraMath.NewDecFromInt64(10), Type: types.WorkerInferenceRewardType},
	}
	errs := payoutRewards(ctx, f.keeper, rewards, nonce)
	require.Len(t, errs, 1, "the inclusion-count failure is still returned")
	require.Empty(t, allRewardPayoutFailedEvents(t, em), "no payout failure event: the actor was paid")
}
