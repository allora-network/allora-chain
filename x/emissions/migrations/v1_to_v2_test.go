package migrations_test

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
	allora_math "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/keeper/msgserver"
	"github.com/allora-network/allora-chain/x/emissions/migrations"
	"github.com/allora-network/allora-chain/x/emissions/module"
	emissionsv1 "github.com/allora-network/allora-chain/x/emissions/types"
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

	cosmosAddress "cosmossdk.io/core/address"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

const (
	multiPerm  = "multiple permissions account"
	randomPerm = "random permission"
)

type MigrationsTestSuite struct {
	suite.Suite

	ctx             sdk.Context
	codec           codec.Codec
	addressCodec    cosmosAddress.Codec
	storeService    store.KVStoreService
	accountKeeper   authkeeper.AccountKeeper
	bankKeeper      bankkeeper.BaseKeeper
	emissionsKeeper keeper.Keeper
	appModule       module.AppModule
	msgServer       emissionsv1.MsgServer
	key             *storetypes.KVStoreKey
	addrs           []sdk.AccAddress
	addrsStr        []string
}

func (s *MigrationsTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey("emissions")
	storeService := runtime.NewKVStoreService(key)
	s.storeService = storeService
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithHeaderInfo(header.Info{Time: time.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, module.AppModule{})
	s.codec = encCfg.Codec
	addressCodec := address.NewBech32Codec(params.Bech32PrefixAccAddr)
	s.addressCodec = addressCodec

	maccPerms := map[string][]string{
		"fee_collector":                      {"minter"},
		"mint":                               {"minter"},
		emissionsv1.AlloraStakingAccountName: {"burner", "minter", "staking"},
		emissionsv1.AlloraRewardsAccountName: {"minter"},
		emissionsv1.AlloraPendingRewardForDelegatorAccountName: {"minter"},
		minttypes.EcosystemModuleName:                          nil,
		"bonded_tokens_pool":                                   {"burner", "staking"},
		"not_bonded_tokens_pool":                               {"burner", "staking"},
		multiPerm:                                              {"burner", "minter", "staking"},
		randomPerm:                                             {"random"},
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

	var addrs []sdk.AccAddress = make([]sdk.AccAddress, 0)
	var addrsStr []string = make([]string, 0)
	pubkeys := simtestutil.CreateTestPubKeys(5)
	for i := 0; i < 5; i++ {
		addrs = append(addrs, sdk.AccAddress(pubkeys[i].Address()))
		addrsStr = append(addrsStr, addrs[i].String())
	}
	s.addrs = addrs
	s.addrsStr = addrsStr

	bankKeeper := bankkeeper.NewBaseKeeper(
		encCfg.Codec,
		storeService,
		accountKeeper,
		map[string]bool{},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		log.NewNopLogger(),
	)

	s.ctx = ctx
	s.accountKeeper = accountKeeper
	s.bankKeeper = bankKeeper
	s.emissionsKeeper = keeper.NewKeeper(
		encCfg.Codec,
		addressCodec,
		storeService,
		accountKeeper,
		bankKeeper,
		authtypes.FeeCollectorName)
	s.key = key
	appModule := module.NewAppModule(encCfg.Codec, s.emissionsKeeper)
	defaultGenesis := appModule.DefaultGenesis(encCfg.Codec)
	appModule.InitGenesis(ctx, encCfg.Codec, defaultGenesis)
	s.msgServer = msgserver.NewMsgServerImpl(s.emissionsKeeper)

	s.appModule = appModule

	// Add all tests addresses in whitelists
	for _, addr := range addrsStr {
		s.emissionsKeeper.AddWhitelistAdmin(ctx, addr)
	}
}

func (s *MigrationsTestSuite) TestMigrateStore(t *testing.T) {
	err := migrations.V1ToV2(s.ctx, s.emissionsKeeper)
	require.NoError(t, err)
}

func (s *MigrationsTestSuite) TestMigrateTopic(t *testing.T) {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	oldTopic := TopicV1{
		ID:                     "testKey",
		Name:                   "testName",
		Description:            "testDescription",
		Creator:                "testCreator",
		Metadata:               "testMetadata",
		LossMethod:             "testLossMethod",
		EpochLength:            100,
		GroundTruthLag:         10,
		PNorm:                  2,
		AlphaRegret:            0.1,
		AllowNegative:          true,
		Epsilon:                0.01,
		WorkerSubmissionWindow: 10,
	}

	bz, err := proto.Marshal(&oldTopic)
	require.NoError(t, err)

	topicStore := prefix.NewStore(store, emissionsv1.TopicsKey)
	topicStore.Set([]byte("testKey"), bz)

	err = migrations.MigrateTopics(store, cdc)
	require.NoError(t, err)

	// Verify the store has been updated correctly
	iterator := topicStore.Iterator(nil, nil)
	require.True(t, iterator.Valid())

	var newMsg emissionsv1.Topic
	err = proto.Unmarshal(iterator.Value(), &newMsg)
	require.NoError(t, err)

	require.Equal(t, oldTopic.ID, newMsg.Id)
	require.Equal(t, oldTopic.Creator, newMsg.Creator)
	require.Equal(t, oldTopic.Metadata, newMsg.Metadata)
	require.Equal(t, oldTopic.LossMethod, newMsg.LossMethod)
	require.Equal(t, oldTopic.EpochLength, newMsg.EpochLength)
	require.Equal(t, oldTopic.GroundTruthLag, newMsg.GroundTruthLag)
	require.Equal(t, oldTopic.PNorm, newMsg.PNorm)
	require.Equal(t, oldTopic.AlphaRegret, newMsg.AlphaRegret)
	require.Equal(t, oldTopic.AllowNegative, newMsg.AllowNegative)
	require.Equal(t, oldTopic.Epsilon, newMsg.Epsilon)
	require.Equal(t, oldTopic.WorkerSubmissionWindow, newMsg.WorkerSubmissionWindow)
	require.Equal(t, 0, newMsg.EpochLastEnded)
	require.Equal(t, 0, newMsg.InitialRegret)
}

func (s *MigrationsTestSuite) TestMigrateOffchainNode(t *testing.T) {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	oldOffchainNode := OffchainNodeV1{
		LibP2PKey:    "testLibP2PKey",
		MultiAddress: "testMultiAddress",
		Owner:        "testOwner",
		NodeAddress:  "testNodeAddress",
		NodeId:       "testNodeId",
	}

	bz, err := proto.Marshal(&oldOffchainNode)
	require.NoError(t, err)

	offchainNodeStore := prefix.NewStore(store, emissionsv1.WorkerNodesKey)
	offchainNodeStore.Set([]byte("testKey"), bz)

	err = migrations.MigrateOffchainNode(store, cdc)
	require.NoError(t, err)

	// Verify the store has been updated correctly
	iterator := offchainNodeStore.Iterator(nil, nil)
	require.True(t, iterator.Valid())

	var newMsg emissionsv1.OffchainNode
	err = proto.Unmarshal(iterator.Value(), &newMsg)
	require.NoError(t, err)

	require.Equal(t, oldOffchainNode.Owner, newMsg.Owner)
	require.Equal(t, oldOffchainNode.NodeAddress, newMsg.NodeAddress)
}

func (s *MigrationsTestSuite) TestMigrateValueBundle(t *testing.T) {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	reputerNonce := &emissionsv1.Nonce{
		BlockHeight: 1,
	}
	oldValueBundle := ValueBundleV1{
		TopicId: 1,
		ReputerRequestNonce: &emissionsv1.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
		},
		ExtraData:     []byte("testExtraData"),
		CombinedValue: allora_math.OneDec(),
		InfererValues: []*emissionsv1.WorkerAttributedValue{
			{
				Worker: "testWorker",
				Value:  allora_math.OneDec(),
			},
		},
		ForecasterValues: []*emissionsv1.WorkerAttributedValue{
			{
				Worker: "testWorker",
				Value:  allora_math.OneDec(),
			},
		},
		NaiveValue: allora_math.OneDec(),
		OneOutInfererValues: []*emissionsv1.WithheldWorkerAttributedValue{
			{
				Worker: "testWorker",
				Value:  allora_math.OneDec(),
			},
		},
		OneOutForecasterValues: []*emissionsv1.WithheldWorkerAttributedValue{
			{
				Worker: "testWorker",
				Value:  allora_math.OneDec(),
			},
		},
		OneInForecasterValues: []*emissionsv1.WorkerAttributedValue{
			{
				Worker: "testWorker",
				Value:  allora_math.OneDec(),
			},
		},
	}

	bz, err := proto.Marshal(&oldValueBundle)
	require.NoError(t, err)

	valueBundleStore := prefix.NewStore(store, emissionsv1.NetworkLossBundlesKey)
	valueBundleStore.Set([]byte("testKey"), bz)

	err = migrations.MigrateNetworkLossBundles(store, cdc)
	require.NoError(t, err)

	// Verify the store has been updated correctly
	iterator := valueBundleStore.Iterator(nil, nil)
	require.True(t, iterator.Valid())

	var newMsg emissionsv1.ValueBundle
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

	reputerNonce := &emissionsv1.Nonce{
		BlockHeight: 1,
	}
	oldValueBundle := ValueBundleV1{
		TopicId: 1,
		ReputerRequestNonce: &emissionsv1.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
		},
		ExtraData:     []byte("testExtraData"),
		CombinedValue: allora_math.OneDec(),
		InfererValues: []*emissionsv1.WorkerAttributedValue{
			{
				Worker: "testWorker",
				Value:  allora_math.OneDec(),
			},
		},
		ForecasterValues: []*emissionsv1.WorkerAttributedValue{
			{
				Worker: "testWorker",
				Value:  allora_math.OneDec(),
			},
		},
		NaiveValue: allora_math.OneDec(),
		OneOutInfererValues: []*emissionsv1.WithheldWorkerAttributedValue{
			{
				Worker: "testWorker",
				Value:  allora_math.OneDec(),
			},
		},
		OneOutForecasterValues: []*emissionsv1.WithheldWorkerAttributedValue{
			{
				Worker: "testWorker",
				Value:  allora_math.OneDec(),
			},
		},
		OneInForecasterValues: []*emissionsv1.WorkerAttributedValue{
			{
				Worker: "testWorker",
				Value:  allora_math.OneDec(),
			},
		},
	}

	reputerValueBundle := ReputerValueBundleV1{
		ValueBundle: &oldValueBundle,
		Signature:   []byte("testSignature"),
		Pubkey:      "testPubkey",
	}

	reputerValueBundles := ReputerValueBundlesV1{
		ReputerValueBundles: []*ReputerValueBundleV1{
			&reputerValueBundle,
		},
	}

	bz := cdc.MustMarshal(&reputerValueBundles)

	allLossBundlesStore := prefix.NewStore(store, emissionsv1.AllLossBundlesKey)
	allLossBundlesStore.Set([]byte("testKey"), bz)

	err := migrations.MigrateNetworkLossBundles(store, cdc)
	require.NoError(t, err)

	// Verify the store has been updated correctly
	iterator := allLossBundlesStore.Iterator(nil, nil)
	require.True(t, iterator.Valid())

	var newMsg emissionsv1.ReputerValueBundles
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

	oldTimestampedActorNonce1 := TimestampedActorNonceV1{
		BlockHeight: 1,
		Actor:       "testActor1",
		Nonce: &emissionsv1.Nonce{
			BlockHeight: 1,
		},
	}

	oldTimestampedActorNonce2 := TimestampedActorNonceV1{
		BlockHeight: 1,
		Actor:       "testActor2",
		Nonce: &emissionsv1.Nonce{
			BlockHeight: 1,
		},
	}

	bz1, err := proto.Marshal(&oldTimestampedActorNonce1)
	require.NoError(t, err)

	timestampedActorNonceStore1 := prefix.NewStore(store, emissionsv1.TopicLastWorkerCommitKey)
	timestampedActorNonceStore1.Set([]byte("testKey"), bz1)

	bz2, err := proto.Marshal(&oldTimestampedActorNonce2)
	require.NoError(t, err)

	timestampedActorNonceStore2 := prefix.NewStore(store, emissionsv1.TopicLastReputerCommitKey)
	timestampedActorNonceStore2.Set([]byte("testKey"), bz2)

	err = migrations.MigrateAllRecordCommits(store, cdc)
	require.NoError(t, err)

	// Verify the store has been updated correctly
	iterator1 := timestampedActorNonceStore1.Iterator(nil, nil)
	require.True(t, iterator1.Valid())

	iterator2 := timestampedActorNonceStore2.Iterator(nil, nil)
	require.True(t, iterator2.Valid())

	var newMsg1 emissionsv1.TimestampedActorNonce
	err = proto.Unmarshal(iterator1.Value(), &newMsg1)
	require.NoError(t, err)

	var newMsg2 emissionsv1.TimestampedActorNonce
	err = proto.Unmarshal(iterator2.Value(), &newMsg2)
	require.NoError(t, err)

	require.Equal(t, oldTimestampedActorNonce1.BlockHeight, newMsg1.BlockHeight)
	require.Equal(t, oldTimestampedActorNonce1.Nonce, newMsg1.Nonce)

	require.Equal(t, oldTimestampedActorNonce2.BlockHeight, newMsg2.BlockHeight)
	require.Equal(t, oldTimestampedActorNonce2.Nonce, newMsg2.Nonce)
}

func (s *MigrationsTestSuite) TestMigrateParams(t *testing.T) {
	prevParams := emissionsv1.DefaultParams()
	prevParams.DataSendingFee = math.ZeroInt()
	err := s.emissionsKeeper.SetParams(s.ctx, prevParams)
	s.Require().NoError(err)

	// Check params before migration
	params, err := s.emissionsKeeper.GetParams(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(params.DataSendingFee, math.ZeroInt())

	// Run migration
	migrations.MigrateParams(s.ctx, s.emissionsKeeper)
	newParams, err := s.emissionsKeeper.GetParams(s.ctx)
	s.Require().NoError(err)

	// Check params after migration
	s.Require().NotEqual(newParams.DataSendingFee, math.ZeroInt())
}

// Old types
//
// types.Topic
// types.OffchainNode
// types.ValueBundle
// types.ReputerValueBundles
// types.TimestampedActorNonce
type TopicV1 struct {
	ID                     string
	Name                   string
	Description            string
	Creator                string
	Metadata               string
	LossMethod             string
	EpochLength            int64
	GroundTruthLag         int64
	PNorm                  int64
	AlphaRegret            float64
	AllowNegative          bool
	Epsilon                float64
	WorkerSubmissionWindow int64
}

func (*TopicV1) ProtoMessage()    {}
func (m *TopicV1) Reset()         { *m = TopicV1{} }
func (m *TopicV1) String() string { return proto.CompactTextString(m) }

type OffchainNodeV1 struct {
	LibP2PKey    string
	MultiAddress string
	Owner        string
	NodeAddress  string
	NodeId       string
}

func (*OffchainNodeV1) ProtoMessage()    {}
func (m *OffchainNodeV1) Reset()         { *m = OffchainNodeV1{} }
func (m *OffchainNodeV1) String() string { return proto.CompactTextString(m) }

type ValueBundleV1 struct {
	TopicId                uint64
	ReputerRequestNonce    *emissionsv1.ReputerRequestNonce
	Reputer                string
	ExtraData              []byte
	CombinedValue          allora_math.Dec
	InfererValues          []*emissionsv1.WorkerAttributedValue
	ForecasterValues       []*emissionsv1.WorkerAttributedValue
	NaiveValue             allora_math.Dec
	OneOutInfererValues    []*emissionsv1.WithheldWorkerAttributedValue
	OneOutForecasterValues []*emissionsv1.WithheldWorkerAttributedValue
	OneInForecasterValues  []*emissionsv1.WorkerAttributedValue
}

func (*ValueBundleV1) ProtoMessage()    {}
func (m *ValueBundleV1) Reset()         { *m = ValueBundleV1{} }
func (m *ValueBundleV1) String() string { return proto.CompactTextString(m) }

type ReputerValueBundleV1 struct {
	ValueBundle *ValueBundleV1
	Signature   []byte
	Pubkey      string
}

func (*ReputerValueBundleV1) ProtoMessage()    {}
func (m *ReputerValueBundleV1) Reset()         { *m = ReputerValueBundleV1{} }
func (m *ReputerValueBundleV1) String() string { return proto.CompactTextString(m) }

type ReputerValueBundlesV1 struct {
	ReputerValueBundles []*ReputerValueBundleV1
}

func (*ReputerValueBundlesV1) ProtoMessage()    {}
func (m *ReputerValueBundlesV1) Reset()         { *m = ReputerValueBundlesV1{} }
func (m *ReputerValueBundlesV1) String() string { return proto.CompactTextString(m) }

type TimestampedActorNonceV1 struct {
	BlockHeight int64
	Actor       string
	Nonce       *emissionsv1.Nonce
}

func (*TimestampedActorNonceV1) ProtoMessage()    {}
func (m *TimestampedActorNonceV1) Reset()         { *m = TimestampedActorNonceV1{} }
func (m *TimestampedActorNonceV1) String() string { return proto.CompactTextString(m) }
