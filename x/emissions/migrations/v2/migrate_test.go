package v2_test

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/header"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	v2 "github.com/allora-network/allora-chain/x/emissions/migrations/v2"
	oldtypes "github.com/allora-network/allora-chain/x/emissions/migrations/v2/oldtypes"
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
	"github.com/stretchr/testify/suite"

	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

type EmissionsV2MigrationsTestSuite struct {
	suite.Suite
	ctx             sdk.Context
	codec           codec.Codec
	storeService    store.KVStoreService
	emissionsKeeper keeper.Keeper
}

func (s *EmissionsV2MigrationsTestSuite) SetupTest() {
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

func TestEmissionsV2MigrationsTestSuite(t *testing.T) {
	suite.Run(t, new(EmissionsV2MigrationsTestSuite))
}

func (s *EmissionsV2MigrationsTestSuite) TestMigrateStore() {
	err := v2.MigrateStore(s.ctx, s.emissionsKeeper)
	s.Require().NoError(err)
}

func (s *EmissionsV2MigrationsTestSuite) TestMigrateTopic() {
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
	}

	bz, err := proto.Marshal(&oldTopic)
	s.Require().NoError(err)

	topicStore := prefix.NewStore(store, types.TopicsKey)
	topicStore.Set([]byte("testKey"), bz)

	err = v2.MigrateTopics(store, cdc)
	s.Require().NoError(err)

	// Verify the store has been updated correctly
	iterator := topicStore.Iterator(nil, nil)
	s.Require().True(iterator.Valid())

	var newMsg types.Topic
	err = proto.Unmarshal(iterator.Value(), &newMsg)
	s.Require().NoError(err)

	s.Require().Equal(oldTopic.Id, newMsg.Id)
	s.Require().Equal(oldTopic.Creator, newMsg.Creator)
	s.Require().Equal(oldTopic.Metadata, newMsg.Metadata)
	s.Require().Equal("mse", newMsg.LossMethod)
	s.Require().Equal(oldTopic.EpochLength, newMsg.EpochLength)
	s.Require().Equal(oldTopic.GroundTruthLag, newMsg.GroundTruthLag)
	s.Require().Equal(oldTopic.PNorm, newMsg.PNorm)
	s.Require().Equal(oldTopic.AlphaRegret, newMsg.AlphaRegret)
	s.Require().Equal(oldTopic.AllowNegative, newMsg.AllowNegative)
	s.Require().Equal(oldTopic.EpochLastEnded, newMsg.EpochLastEnded)
}

func (s *EmissionsV2MigrationsTestSuite) MigrateOffchainNodeStore(store prefix.Store, cdc codec.BinaryCodec, prefixKey collections.Prefix) {
	oldOffchainNode := oldtypes.OffchainNode{
		LibP2PKey:    "testLibP2PKey",
		MultiAddress: "testMultiAddress",
		Owner:        "testOwner",
		NodeAddress:  "testNodeAddress",
		NodeId:       "testNodeId",
	}
	oldOffchainNode2 := oldtypes.OffchainNode{
		LibP2PKey:    "testLibP2PKey2",
		MultiAddress: "testMultiAddress2",
		Owner:        "testOwner2",
		NodeAddress:  "testNodeAddress2",
		NodeId:       "testNodeId2",
	}

	bz, err := proto.Marshal(&oldOffchainNode)
	s.Require().NoError(err)
	bz2, err := proto.Marshal(&oldOffchainNode2)
	s.Require().NoError(err)

	offchainNodeStore := prefix.NewStore(store, prefixKey)
	offchainNodeStore.Set([]byte("testLibP2PKey"), bz)
	offchainNodeStore.Set([]byte("testLibP2PKey2"), bz2)

	err = v2.MigrateOffchainNode(store, cdc)
	s.Require().NoError(err)

	// Verify the store has been updated correctly

	oldObj := offchainNodeStore.Get([]byte("testLibP2PKey"))
	s.Require().Nil(oldObj)
	oldObj2 := offchainNodeStore.Get([]byte("testLibP2PKey2"))
	s.Require().Nil(oldObj2)

	newObj := offchainNodeStore.Get([]byte("testNodeAddress"))
	s.Require().NotNil(newObj)
	newObj2 := offchainNodeStore.Get([]byte("testNodeAddress2"))
	s.Require().NotNil(newObj2)

	var newMsg types.OffchainNode
	err = proto.Unmarshal(newObj, &newMsg)
	s.Require().NoError(err)
	s.Require().Equal(oldOffchainNode.Owner, newMsg.Owner)
	s.Require().Equal(oldOffchainNode.NodeAddress, newMsg.NodeAddress)
	// second object
	err = proto.Unmarshal(newObj2, &newMsg)
	s.Require().NoError(err)
	s.Require().Equal(oldOffchainNode2.Owner, newMsg.Owner)
	s.Require().Equal(oldOffchainNode2.NodeAddress, newMsg.NodeAddress)
}

func (s *EmissionsV2MigrationsTestSuite) TestMigrateOffchainNodeWorkers() {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()
	offchainNodeStoreWorker := prefix.NewStore(store, types.WorkerNodesKey)
	s.MigrateOffchainNodeStore(offchainNodeStoreWorker, cdc, types.WorkerNodesKey)
}

func (s *EmissionsV2MigrationsTestSuite) TestMigrateOffchainNodeReputers() {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()
	offchainNodeStoreReputer := prefix.NewStore(store, types.ReputerNodesKey)
	s.MigrateOffchainNodeStore(offchainNodeStoreReputer, cdc, types.ReputerNodesKey)
}

func areAttributedArraysEqual(oldValues []*oldtypes.WorkerAttributedValue, newValues []*types.WorkerAttributedValue) bool {
	if len(oldValues) != len(newValues) {
		return false
	}
	for i, oldVal := range oldValues {
		if oldVal.Worker != newValues[i].Worker || oldVal.Value != newValues[i].Value {
			return false
		}
	}
	return true
}

func areWithHeldArraysEqual(oldValues []*oldtypes.WithheldWorkerAttributedValue, newValues []*types.WithheldWorkerAttributedValue) bool {
	if len(oldValues) != len(newValues) {
		return false
	}
	for i, oldVal := range oldValues {
		if oldVal.Worker != newValues[i].Worker || oldVal.Value != newValues[i].Value {
			return false
		}
	}
	return true
}

func (s *EmissionsV2MigrationsTestSuite) TestMigrateValueBundle() {
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
	s.Require().NoError(err)

	valueBundleStore := prefix.NewStore(store, types.NetworkLossBundlesKey)
	valueBundleStore.Set([]byte("testKey"), bz)

	err = v2.MigrateNetworkLossBundles(store, cdc)
	s.Require().NoError(err)

	// Verify the store has been updated correctly
	iterator := valueBundleStore.Iterator(nil, nil)
	s.Require().True(iterator.Valid())

	var newMsg types.ValueBundle
	err = proto.Unmarshal(iterator.Value(), &newMsg)
	s.Require().NoError(err)

	s.Require().Equal(oldValueBundle.TopicId, newMsg.TopicId)
	s.Require().Equal(oldValueBundle.ReputerRequestNonce.ReputerNonce.BlockHeight, newMsg.ReputerRequestNonce.ReputerNonce.BlockHeight)
	s.Require().Equal(oldValueBundle.Reputer, newMsg.Reputer)
	s.Require().Equal(oldValueBundle.ExtraData, newMsg.ExtraData)
	s.Require().Equal(oldValueBundle.CombinedValue, newMsg.CombinedValue)
	// check that the infererValues have been migrated correctly in a loop
	s.Require().True(areAttributedArraysEqual(oldValueBundle.InfererValues, newMsg.InfererValues))
	s.Require().True(areAttributedArraysEqual(oldValueBundle.ForecasterValues, newMsg.ForecasterValues))
	s.Require().Equal(oldValueBundle.NaiveValue, newMsg.NaiveValue)

	s.Require().True(areWithHeldArraysEqual(oldValueBundle.OneOutInfererValues, newMsg.OneOutInfererValues))
	s.Require().True(areWithHeldArraysEqual(oldValueBundle.OneOutForecasterValues, newMsg.OneOutForecasterValues))
	s.Require().True(areAttributedArraysEqual(oldValueBundle.OneInForecasterValues, newMsg.OneInForecasterValues))

	s.Require().Empty(newMsg.OneOutInfererForecasterValues)
}

func (s *EmissionsV2MigrationsTestSuite) TestMigrateAllLossBundles() {
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
	s.Require().NoError(err)

	// Verify the store has been updated correctly
	iterator := allLossBundlesStore.Iterator(nil, nil)
	s.Require().True(iterator.Valid())

	var newMsg types.ReputerValueBundles
	err = proto.Unmarshal(iterator.Value(), &newMsg)
	s.Require().NoError(err)

	s.Require().Equal(reputerValueBundle.ValueBundle.TopicId, newMsg.ReputerValueBundles[0].ValueBundle.TopicId)
	s.Require().Equal(reputerValueBundle.ValueBundle.ReputerRequestNonce.ReputerNonce.BlockHeight, newMsg.ReputerValueBundles[0].ValueBundle.ReputerRequestNonce.ReputerNonce.BlockHeight)
	s.Require().Equal(reputerValueBundle.ValueBundle.Reputer, newMsg.ReputerValueBundles[0].ValueBundle.Reputer)
	s.Require().Equal(reputerValueBundle.ValueBundle.ExtraData, newMsg.ReputerValueBundles[0].ValueBundle.ExtraData)
	s.Require().Equal(reputerValueBundle.ValueBundle.CombinedValue, newMsg.ReputerValueBundles[0].ValueBundle.CombinedValue)

	s.Require().True(areAttributedArraysEqual(oldValueBundle.InfererValues, newMsg.ReputerValueBundles[0].ValueBundle.InfererValues))
	s.Require().True(areAttributedArraysEqual(oldValueBundle.ForecasterValues, newMsg.ReputerValueBundles[0].ValueBundle.ForecasterValues))

	s.Require().Equal(reputerValueBundle.ValueBundle.NaiveValue, newMsg.ReputerValueBundles[0].ValueBundle.NaiveValue)

	s.Require().True(areWithHeldArraysEqual(oldValueBundle.OneOutInfererValues, newMsg.ReputerValueBundles[0].ValueBundle.OneOutInfererValues))
	s.Require().True(areWithHeldArraysEqual(oldValueBundle.OneOutForecasterValues, newMsg.ReputerValueBundles[0].ValueBundle.OneOutForecasterValues))

	s.Require().True(areAttributedArraysEqual(oldValueBundle.OneInForecasterValues, newMsg.ReputerValueBundles[0].ValueBundle.OneInForecasterValues))

	defaultOneOutInfererForecasterValues := []*types.OneOutInfererForecasterValues{}
	s.Require().Equal(len(defaultOneOutInfererForecasterValues), len(newMsg.ReputerValueBundles[0].ValueBundle.OneOutInfererForecasterValues))

	s.Require().Equal(reputerValueBundle.Signature, newMsg.ReputerValueBundles[0].Signature)
	s.Require().Equal(reputerValueBundle.Pubkey, newMsg.ReputerValueBundles[0].Pubkey)
}

func (s *EmissionsV2MigrationsTestSuite) TestMigrateAllRecordCommits() {
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
	s.Require().NoError(err)

	timestampedActorNonceStore1 := prefix.NewStore(store, types.TopicLastWorkerCommitKey)
	timestampedActorNonceStore1.Set([]byte("testKey"), bz1)

	bz2, err := proto.Marshal(&oldTimestampedActorNonce2)
	s.Require().NoError(err)

	timestampedActorNonceStore2 := prefix.NewStore(store, types.TopicLastReputerCommitKey)
	timestampedActorNonceStore2.Set([]byte("testKey"), bz2)

	err = v2.MigrateAllRecordCommits(store, cdc)
	s.Require().NoError(err)

	// Verify the store has been updated correctly
	iterator1 := timestampedActorNonceStore1.Iterator(nil, nil)
	s.Require().True(iterator1.Valid())

	iterator2 := timestampedActorNonceStore2.Iterator(nil, nil)
	s.Require().True(iterator2.Valid())

	var newMsg1 types.TimestampedActorNonce
	err = proto.Unmarshal(iterator1.Value(), &newMsg1)
	s.Require().NoError(err)

	var newMsg2 types.TimestampedActorNonce
	err = proto.Unmarshal(iterator2.Value(), &newMsg2)
	s.Require().NoError(err)

	s.Require().Equal(oldTimestampedActorNonce1.BlockHeight, newMsg1.BlockHeight)
	s.Require().Equal(oldTimestampedActorNonce1.Nonce.BlockHeight, newMsg1.Nonce.BlockHeight)

	s.Require().Equal(oldTimestampedActorNonce2.BlockHeight, newMsg2.BlockHeight)
	s.Require().Equal(oldTimestampedActorNonce2.Nonce.BlockHeight, newMsg2.Nonce.BlockHeight)
}

func (s *EmissionsV2MigrationsTestSuite) TestMigrateParams() {
	// Create a Params with garbage in it
	prevParams := types.Params{
		Version:                             "v1",
		MaxSerializedMsgLength:              1,
		MinTopicWeight:                      alloraMath.OneDec(),
		RequiredMinimumStake:                cosmosMath.OneInt(),
		RemoveStakeDelayWindow:              1,
		MinEpochLength:                      2341,
		BetaEntropy:                         alloraMath.MustNewDecFromString("0.1337"),
		LearningRate:                        alloraMath.MustNewDecFromString("0.1337"),
		MaxGradientThreshold:                alloraMath.MustNewDecFromString("0.1337"),
		MinStakeFraction:                    alloraMath.MustNewDecFromString("0.1337"),
		MaxUnfulfilledWorkerRequests:        1,
		MaxUnfulfilledReputerRequests:       1,
		TopicRewardStakeImportance:          alloraMath.MustNewDecFromString("0.1337"),
		TopicRewardFeeRevenueImportance:     alloraMath.MustNewDecFromString("0.1337"),
		TopicRewardAlpha:                    alloraMath.MustNewDecFromString("0.1337"),
		TaskRewardAlpha:                     alloraMath.MustNewDecFromString("0.1337"),
		ValidatorsVsAlloraPercentReward:     alloraMath.MustNewDecFromString("0.1337"),
		MaxSamplesToScaleScores:             123,
		MaxTopInferersToReward:              123,
		MaxTopForecastersToReward:           123,
		MaxTopReputersToReward:              123,
		CreateTopicFee:                      cosmosMath.OneInt(),
		GradientDescentMaxIters:             123,
		RegistrationFee:                     cosmosMath.OneInt(),
		DefaultPageLimit:                    123,
		MaxPageLimit:                        123,
		MinEpochLengthRecordLimit:           123,
		BlocksPerMonth:                      123,
		PRewardInference:                    alloraMath.MustNewDecFromString("0.1337"),
		PRewardForecast:                     alloraMath.MustNewDecFromString("0.1337"),
		PRewardReputer:                      alloraMath.MustNewDecFromString("0.1337"),
		CRewardInference:                    alloraMath.MustNewDecFromString("0.1337"),
		CRewardForecast:                     alloraMath.MustNewDecFromString("0.1337"),
		CNorm:                               alloraMath.MustNewDecFromString("0.1337"),
		EpsilonReputer:                      alloraMath.MustNewDecFromString("0.1337"),
		HalfMaxProcessStakeRemovalsEndBlock: 123,
		EpsilonSafeDiv:                      alloraMath.MustNewDecFromString("0.1337"),
		DataSendingFee:                      cosmosMath.OneInt(),
		MaxElementsPerForecast:              123,
		MaxActiveTopicsPerBlock:             123,
		MaxStringLength:                     123,
	}
	err := s.emissionsKeeper.SetParams(s.ctx, prevParams)
	s.Require().NoError(err)

	// Run migration
	err = v2.MigrateParams(s.ctx, s.emissionsKeeper)
	s.Require().NoError(err)
	newParams, err := s.emissionsKeeper.GetParams(s.ctx)
	s.Require().NoError(err)

	defaultParams := types.DefaultParams()
	// Check params after migration
	s.Require().Equal(newParams, defaultParams)
}
