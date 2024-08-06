package v2_test

import (
	"testing"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	v2 "github.com/allora-network/allora-chain/x/emissions/migrations/v2"
	oldtypes "github.com/allora-network/allora-chain/x/emissions/migrations/v2/types"
	"github.com/allora-network/allora-chain/x/emissions/module"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

type MigrationsTestSuite struct {
	suite.Suite
	ctx             sdk.Context
	codec           codec.Codec
	storeService    store.KVStoreService
	emissionsKeeper keeper.Keeper
}

func (s *MigrationsTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey("emissions")
	storeService := runtime.NewKVStoreService(key)
	s.storeService = storeService
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithHeaderInfo(header.Info{Time: time.Now()})
	s.ctx = ctx
	encCfg := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, module.AppModule{})
	s.codec = encCfg.Codec
	addressCodec := address.NewBech32Codec(params.Bech32PrefixAccAddr)
	maccPerms := map[string][]string{
		"fee_collector":                {"minter"},
		"mint":                         {"minter"},
		types.AlloraStakingAccountName: {"burner", "minter", "staking"},
		types.AlloraRewardsAccountName: {"minter"},
		types.AlloraPendingRewardForDelegatorAccountName: {"minter"},
		minttypes.EcosystemModuleName:                    nil,
		"bonded_tokens_pool":                             {"burner", "staking"},
		"not_bonded_tokens_pool":                         {"burner", "staking"},
		"multiple permissions account":                   {"burner", "minter", "staking"},
		"random permission":                              {"random"},
	}
	accountKeeper := authkeeper.NewAccountKeeper(
		encCfg.Codec,
		storeService,
		authtypes.ProtoBaseAccount,
		maccPerms,
		authcodec.NewBech32Codec(params.Bech32PrefixAccAddr),
		params.Bech32PrefixAccAddr,
		authtypes.NewModuleAddress("gov").String(),
	)
	bankKeeper := bankkeeper.NewBaseKeeper(
		encCfg.Codec,
		storeService,
		accountKeeper,
		map[string]bool{},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		log.NewNopLogger(),
	)
	s.emissionsKeeper = keeper.NewKeeper(
		encCfg.Codec,
		addressCodec,
		storeService,
		accountKeeper,
		bankKeeper,
		authtypes.FeeCollectorName)
}

func (s *MigrationsTestSuite) TestMigrateStore(t *testing.T) {
	err := v2.MigrateStore(s.ctx, s.emissionsKeeper)
	require.NoError(t, err)
}

func (s *MigrationsTestSuite) TestMigrateTopic(t *testing.T) {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	oldTopic := oldtypes.Topic{
		Id:              1,
		Creator:         "creator",
		Metadata:        "metadata",
		LossLogic:       "losslogic",
		LossMethod:      "lossmethod",
		InferenceLogic:  "inferencelogic",
		InferenceMethod: "inferencemethod",
		EpochLastEnded:  0,
		EpochLength:     100,
		GroundTruthLag:  10,
		DefaultArg:      "defaultarg",
		PNorm:           alloraMath.NewDecFromInt64(3),
		AlphaRegret:     alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:   false,
		Epsilon:         alloraMath.MustNewDecFromString("0.0001"),
	}

	bz, err := proto.Marshal(&oldTopic)
	require.NoError(t, err)

	topicStore := prefix.NewStore(store, types.TopicsKey)
	topicStore.Set([]byte("testKey"), bz)

	err = v2.MigrateTopics(store, cdc)
	require.NoError(t, err)

	// Verify the store has been updated correctly
	iterator := topicStore.Iterator(nil, nil)
	require.True(t, iterator.Valid())

	var newMsg types.Topic
	err = proto.Unmarshal(iterator.Value(), &newMsg)
	require.NoError(t, err)

	require.Equal(t, oldTopic.Id, newMsg.Id)
	require.Equal(t, oldTopic.Creator, newMsg.Creator)
	require.Equal(t, oldTopic.Metadata, newMsg.Metadata)
	require.Equal(t, oldTopic.LossMethod, newMsg.LossMethod)
	require.Equal(t, oldTopic.EpochLength, newMsg.EpochLength)
	require.Equal(t, oldTopic.GroundTruthLag, newMsg.GroundTruthLag)
	require.Equal(t, oldTopic.PNorm, newMsg.PNorm)
	require.Equal(t, oldTopic.AlphaRegret, newMsg.AlphaRegret)
	require.Equal(t, oldTopic.AllowNegative, newMsg.AllowNegative)
	require.Equal(t, oldTopic.Epsilon, newMsg.Epsilon)
	require.Equal(t, 0, newMsg.EpochLastEnded)
	require.Equal(t, 0, newMsg.InitialRegret)
}

func (s *MigrationsTestSuite) TestMigrateOffchainNode(t *testing.T) {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	oldOffchainNode := oldtypes.OffchainNode{
		LibP2PKey:    "testLibP2PKey",
		MultiAddress: "testMultiAddress",
		Owner:        "testOwner",
		NodeAddress:  "testNodeAddress",
		NodeId:       "testNodeId",
	}

	bz, err := proto.Marshal(&oldOffchainNode)
	require.NoError(t, err)

	offchainNodeStore := prefix.NewStore(store, types.WorkerNodesKey)
	offchainNodeStore.Set([]byte("testKey"), bz)

	err = v2.MigrateOffchainNode(store, cdc)
	require.NoError(t, err)

	// Verify the store has been updated correctly
	iterator := offchainNodeStore.Iterator(nil, nil)
	require.True(t, iterator.Valid())

	var newMsg types.OffchainNode
	err = proto.Unmarshal(iterator.Value(), &newMsg)
	require.NoError(t, err)

	require.Equal(t, oldOffchainNode.Owner, newMsg.Owner)
	require.Equal(t, oldOffchainNode.NodeAddress, newMsg.NodeAddress)
}

func (s *MigrationsTestSuite) TestMigrateValueBundle(t *testing.T) {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	reputerNonce := &oldtypes.Nonce{
		BlockHeight: 1,
	}
	oldValueBundle := oldtypes.ValueBundle{
		TopicId: 1,
		ReputerRequestNonce: &oldtypes.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
		},
		ExtraData:     []byte("testExtraData"),
		CombinedValue: alloraMath.OneDec(),
		InfererValues: []*oldtypes.WorkerAttributedValue{
			{
				Worker: "testWorker",
				Value:  alloraMath.OneDec(),
			},
		},
		ForecasterValues: []*oldtypes.WorkerAttributedValue{
			{
				Worker: "testWorker",
				Value:  alloraMath.OneDec(),
			},
		},
		NaiveValue: alloraMath.OneDec(),
		OneOutInfererValues: []*oldtypes.WithheldWorkerAttributedValue{
			{
				Worker: "testWorker",
				Value:  alloraMath.OneDec(),
			},
		},
		OneOutForecasterValues: []*oldtypes.WithheldWorkerAttributedValue{
			{
				Worker: "testWorker",
				Value:  alloraMath.OneDec(),
			},
		},
		OneInForecasterValues: []*oldtypes.WorkerAttributedValue{
			{
				Worker: "testWorker",
				Value:  alloraMath.OneDec(),
			},
		},
	}

	bz, err := proto.Marshal(&oldValueBundle)
	require.NoError(t, err)

	valueBundleStore := prefix.NewStore(store, types.NetworkLossBundlesKey)
	valueBundleStore.Set([]byte("testKey"), bz)

	err = v2.MigrateNetworkLossBundles(store, cdc)
	require.NoError(t, err)

	// Verify the store has been updated correctly
	iterator := valueBundleStore.Iterator(nil, nil)
	require.True(t, iterator.Valid())

	var newMsg types.ValueBundle
	err = proto.Unmarshal(iterator.Value(), &newMsg)
	require.NoError(t, err)

	require.Equal(t, oldValueBundle.TopicId, newMsg.TopicId)
	require.Equal(t, oldValueBundle.ReputerRequestNonce, newMsg.ReputerRequestNonce)
	require.Equal(t, oldValueBundle.Reputer, newMsg.Reputer)
	require.Equal(t, oldValueBundle.ExtraData, newMsg.ExtraData)
	require.Equal(t, oldValueBundle.CombinedValue, newMsg.CombinedValue)
	require.Equal(t, oldValueBundle.InfererValues, newMsg.InfererValues)
	require.Equal(t, oldValueBundle.ForecasterValues, newMsg.ForecasterValues)
	require.Equal(t, oldValueBundle.NaiveValue, newMsg.NaiveValue)
	require.Equal(t, oldValueBundle.OneOutInfererValues, newMsg.OneOutInfererValues)
	require.Equal(t, oldValueBundle.OneOutForecasterValues, newMsg.OneOutForecasterValues)
	require.Equal(t, oldValueBundle.OneInForecasterValues, newMsg.OneInForecasterValues)
}

func (s *MigrationsTestSuite) TestMigrateAllLossBundles(t *testing.T) {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	reputerNonce := &oldtypes.Nonce{
		BlockHeight: 1,
	}
	oldValueBundle := oldtypes.ValueBundle{
		TopicId: 1,
		ReputerRequestNonce: &oldtypes.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
		},
		ExtraData:     []byte("testExtraData"),
		CombinedValue: alloraMath.OneDec(),
		InfererValues: []*oldtypes.WorkerAttributedValue{
			{
				Worker: "testWorker",
				Value:  alloraMath.OneDec(),
			},
		},
		ForecasterValues: []*oldtypes.WorkerAttributedValue{
			{
				Worker: "testWorker",
				Value:  alloraMath.OneDec(),
			},
		},
		NaiveValue: alloraMath.OneDec(),
		OneOutInfererValues: []*oldtypes.WithheldWorkerAttributedValue{
			{
				Worker: "testWorker",
				Value:  alloraMath.OneDec(),
			},
		},
		OneOutForecasterValues: []*oldtypes.WithheldWorkerAttributedValue{
			{
				Worker: "testWorker",
				Value:  alloraMath.OneDec(),
			},
		},
		OneInForecasterValues: []*oldtypes.WorkerAttributedValue{
			{
				Worker: "testWorker",
				Value:  alloraMath.OneDec(),
			},
		},
	}

	reputerValueBundle := oldtypes.ReputerValueBundle{
		ValueBundle: &oldValueBundle,
		Signature:   []byte("testSignature"),
		Pubkey:      "testPubkey",
	}

	reputerValueBundles := oldtypes.ReputerValueBundles{
		ReputerValueBundles: []*oldtypes.ReputerValueBundle{
			&reputerValueBundle,
		},
	}

	bz := cdc.MustMarshal(&reputerValueBundles)

	allLossBundlesStore := prefix.NewStore(store, types.AllLossBundlesKey)
	allLossBundlesStore.Set([]byte("testKey"), bz)

	err := v2.MigrateNetworkLossBundles(store, cdc)
	require.NoError(t, err)

	// Verify the store has been updated correctly
	iterator := allLossBundlesStore.Iterator(nil, nil)
	require.True(t, iterator.Valid())

	var newMsg types.ReputerValueBundles
	err = proto.Unmarshal(iterator.Value(), &newMsg)
	require.NoError(t, err)

	require.Equal(t, reputerValueBundle.ValueBundle.TopicId, newMsg.ReputerValueBundles[0].ValueBundle.TopicId)
	require.Equal(t, reputerValueBundle.ValueBundle.ReputerRequestNonce, newMsg.ReputerValueBundles[0].ValueBundle.ReputerRequestNonce)
	require.Equal(t, reputerValueBundle.ValueBundle.Reputer, newMsg.ReputerValueBundles[0].ValueBundle.Reputer)
	require.Equal(t, reputerValueBundle.ValueBundle.ExtraData, newMsg.ReputerValueBundles[0].ValueBundle.ExtraData)
	require.Equal(t, reputerValueBundle.ValueBundle.CombinedValue, newMsg.ReputerValueBundles[0].ValueBundle.CombinedValue)
	require.Equal(t, reputerValueBundle.ValueBundle.InfererValues, newMsg.ReputerValueBundles[0].ValueBundle.InfererValues)
	require.Equal(t, reputerValueBundle.ValueBundle.ForecasterValues, newMsg.ReputerValueBundles[0].ValueBundle.ForecasterValues)
	require.Equal(t, reputerValueBundle.ValueBundle.NaiveValue, newMsg.ReputerValueBundles[0].ValueBundle.NaiveValue)
	require.Equal(t, reputerValueBundle.ValueBundle.OneOutInfererValues, newMsg.ReputerValueBundles[0].ValueBundle.OneOutInfererValues)
	require.Equal(t, reputerValueBundle.ValueBundle.OneOutForecasterValues, newMsg.ReputerValueBundles[0].ValueBundle.OneOutForecasterValues)
	require.Equal(t, reputerValueBundle.ValueBundle.OneInForecasterValues, newMsg.ReputerValueBundles[0].ValueBundle.OneInForecasterValues)
	require.Equal(t, reputerValueBundle.Signature, newMsg.ReputerValueBundles[0].Signature)
	require.Equal(t, reputerValueBundle.Pubkey, newMsg.ReputerValueBundles[0].Pubkey)
}

func (s *MigrationsTestSuite) TestMigrateAllRecordCommits(t *testing.T) {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	oldTimestampedActorNonce1 := oldtypes.TimestampedActorNonce{
		BlockHeight: 1,
		Actor:       "testActor1",
		Nonce: &oldtypes.Nonce{
			BlockHeight: 1,
		},
	}

	oldTimestampedActorNonce2 := oldtypes.TimestampedActorNonce{
		BlockHeight: 1,
		Actor:       "testActor2",
		Nonce: &oldtypes.Nonce{
			BlockHeight: 1,
		},
	}

	bz1, err := proto.Marshal(&oldTimestampedActorNonce1)
	require.NoError(t, err)

	timestampedActorNonceStore1 := prefix.NewStore(store, types.TopicLastWorkerCommitKey)
	timestampedActorNonceStore1.Set([]byte("testKey"), bz1)

	bz2, err := proto.Marshal(&oldTimestampedActorNonce2)
	require.NoError(t, err)

	timestampedActorNonceStore2 := prefix.NewStore(store, types.TopicLastReputerCommitKey)
	timestampedActorNonceStore2.Set([]byte("testKey"), bz2)

	err = v2.MigrateAllRecordCommits(store, cdc)
	require.NoError(t, err)

	// Verify the store has been updated correctly
	iterator1 := timestampedActorNonceStore1.Iterator(nil, nil)
	require.True(t, iterator1.Valid())

	iterator2 := timestampedActorNonceStore2.Iterator(nil, nil)
	require.True(t, iterator2.Valid())

	var newMsg1 types.TimestampedActorNonce
	err = proto.Unmarshal(iterator1.Value(), &newMsg1)
	require.NoError(t, err)

	var newMsg2 types.TimestampedActorNonce
	err = proto.Unmarshal(iterator2.Value(), &newMsg2)
	require.NoError(t, err)

	require.Equal(t, oldTimestampedActorNonce1.BlockHeight, newMsg1.BlockHeight)
	require.Equal(t, oldTimestampedActorNonce1.Nonce, newMsg1.Nonce)

	require.Equal(t, oldTimestampedActorNonce2.BlockHeight, newMsg2.BlockHeight)
	require.Equal(t, oldTimestampedActorNonce2.Nonce, newMsg2.Nonce)
}

func (s *MigrationsTestSuite) TestMigrateParams(t *testing.T) {
	prevParams := types.DefaultParams()
	prevParams.DataSendingFee = math.ZeroInt()
	err := s.emissionsKeeper.SetParams(s.ctx, prevParams)
	s.Require().NoError(err)

	// Check params before migration
	params, err := s.emissionsKeeper.GetParams(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(params.DataSendingFee, math.ZeroInt())

	// Run migration
	v2.MigrateParams(s.ctx, s.emissionsKeeper)
	newParams, err := s.emissionsKeeper.GetParams(s.ctx)
	s.Require().NoError(err)

	// Check params after migration
	s.Require().NotEqual(newParams.DataSendingFee, math.ZeroInt())
}
