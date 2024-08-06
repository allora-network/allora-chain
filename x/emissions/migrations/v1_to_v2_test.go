package migrations_test

import (
	"testing"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/keeper/msgserver"
	"github.com/allora-network/allora-chain/x/emissions/migrations"
	"github.com/allora-network/allora-chain/x/emissions/module"
	"github.com/allora-network/allora-chain/x/emissions/types"
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
	msgServer       types.MsgServer
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
		"fee_collector":                {"minter"},
		"mint":                         {"minter"},
		types.AlloraStakingAccountName: {"burner", "minter", "staking"},
		types.AlloraRewardsAccountName: {"minter"},
		types.AlloraPendingRewardForDelegatorAccountName: {"minter"},
		minttypes.EcosystemModuleName:                    nil,
		"bonded_tokens_pool":                             {"burner", "staking"},
		"not_bonded_tokens_pool":                         {"burner", "staking"},
		multiPerm:                                        {"burner", "minter", "staking"},
		randomPerm:                                       {"random"},
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

// Previous version of the message
type MsgCreateNewTopicV1 struct {
	Creator         string
	Metadata        string
	LossLogic       string
	LossMethod      string
	InferenceLogic  string
	InferenceMethod string
	EpochLength     int64
	GroundTruthLag  int64
	DefaultArg      string
	PNorm           math.Dec
	AlphaRegret     math.Dec
	AllowNegative   bool
	Epsilon         math.Dec
}

func (*MsgCreateNewTopicV1) ProtoMessage()    {}
func (m *MsgCreateNewTopicV1) Reset()         { *m = MsgCreateNewTopicV1{} }
func (m *MsgCreateNewTopicV1) String() string { return proto.CompactTextString(m) }

func (s *MigrationsTestSuite) TestMigrateMsgCreateNewTopic(t *testing.T) {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))

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

	err = migrations.V1ToV2(s.ctx, s.emissionsKeeper)
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
	// New fields
	// EpochLastEnded
	// InitialRegret

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
	CombinedValue          math.Dec
	InfererValues          []*emissionsv1.WorkerAttributedValue
	ForecasterValues       []*emissionsv1.WorkerAttributedValue
	NaiveValue             math.Dec
	OneOutInfererValues    []*emissionsv1.WithheldWorkerAttributedValue
	OneOutForecasterValues []*emissionsv1.WithheldWorkerAttributedValue
	OneInForecasterValues  []*emissionsv1.WorkerAttributedValue
}

func (*ValueBundleV1) ProtoMessage()    {}
func (m *ValueBundleV1) Reset()         { *m = ValueBundleV1{} }
func (m *ValueBundleV1) String() string { return proto.CompactTextString(m) }
