package msgserver_test

import (
	"time"

	"cosmossdk.io/core/header"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"

	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/keeper/msgserver"
	emissionstestutil "github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

var (
	nonAdminAccounts = simtestutil.CreateRandomAccounts(4)
	// TODO: Change PKS to accounts here and in all the tests (like the above line)
	PKS     = simtestutil.CreateTestPubKeys(4)
	Addr    = sdk.AccAddress(PKS[0].Address())
	ValAddr = sdk.ValAddress(Addr)
)

type KeeperTestSuite struct {
	suite.Suite

	ctx             sdk.Context
	bankKeeper      *emissionstestutil.MockBankKeeper
	authKeeper      *emissionstestutil.MockAccountKeeper
	emissionsKeeper keeper.Keeper
	msgServer       types.MsgServer
	mockCtrl        *gomock.Controller
	key             *storetypes.KVStoreKey
}

func (s *KeeperTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey("emissions")
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithHeaderInfo(header.Info{Time: time.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()
	addressCodec := address.NewBech32Codec(params.Bech32PrefixAccAddr)
	ctrl := gomock.NewController(s.T())

	s.bankKeeper = emissionstestutil.NewMockBankKeeper(ctrl)
	s.authKeeper = emissionstestutil.NewMockAccountKeeper(ctrl)

	s.ctx = ctx
	s.emissionsKeeper = keeper.NewKeeper(encCfg.Codec, addressCodec, storeService, s.authKeeper, s.bankKeeper, "fee_collector")
	s.msgServer = msgserver.NewMsgServerImpl(s.emissionsKeeper)
	s.mockCtrl = ctrl
	s.key = key

	// Add all tests addresses in whitelists
	for _, addr := range PKS {
		s.emissionsKeeper.AddWhitelistAdmin(ctx, sdk.AccAddress(addr.Address()))
		s.emissionsKeeper.AddToTopicCreationWhitelist(ctx, sdk.AccAddress(addr.Address()))
		s.emissionsKeeper.AddToReputerWhitelist(ctx, sdk.AccAddress(addr.Address()))
	}
}

func (s *KeeperTestSuite) CreateOneTopic() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Create a topic first
	metadata := "Some metadata for the new topic"
	// Create a MsgCreateNewTopic message
	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:         sdk.AccAddress(PKS[0].Address()).String(),
		Metadata:        metadata,
		LossLogic:       "logic",
		EpochLength:     10800,
		InferenceLogic:  "Ilogic",
		InferenceMethod: "Imethod",
		DefaultArg:      "ETH",
	}

	_, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on first creation")
}

func (s *KeeperTestSuite) TestCreateSeveralTopics() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	// Mock setup for metadata and validation steps
	metadata := "Some metadata for the new topic"
	// Create a MsgCreateNewTopic message
	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:         sdk.AccAddress(PKS[0].Address()).String(),
		Metadata:        metadata,
		LossLogic:       "logic",
		EpochLength:     10800,
		InferenceLogic:  "Ilogic",
		InferenceMethod: "Imethod",
		DefaultArg:      "ETH",
	}

	_, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on first creation")

	result, err := s.emissionsKeeper.GetNumTopics(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(result, uint64(1), "Topic count after first topic is not 1.")

	// Create second topic
	_, err = msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on second topic")

	result, err = s.emissionsKeeper.GetNumTopics(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(result, uint64(2), "Topic count after second topic insertion is not 2")
}
