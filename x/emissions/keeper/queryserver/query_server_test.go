package queryserver_test

import (
	"crypto/ed25519"
	"encoding/hex"
	"testing"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"

	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/keeper/msgserver"
	"github.com/allora-network/allora-chain/x/emissions/keeper/queryserver"
	"github.com/allora-network/allora-chain/x/emissions/module"
	"github.com/allora-network/allora-chain/x/emissions/types"
	mintTypes "github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/suite"
)

const (
	multiPerm  = "multiple permissions account"
	randomPerm = "random permission"
)

type ChainKey struct {
	pubKey ed25519.PublicKey
	priKey ed25519.PrivateKey
}

type QueryServerTestSuite struct {
	suite.Suite

	ctx             sdk.Context
	accountKeeper   keeper.AccountKeeper
	bankKeeper      keeper.BankKeeper
	emissionsKeeper keeper.Keeper
	appModule       module.AppModule
	msgServer       types.MsgServiceServer
	queryServer     types.QueryServiceServer
	key             *storetypes.KVStoreKey
	privKeys        []secp256k1.PrivKey
	addrs           []sdk.AccAddress
	addrsStr        []string
	pubKeyHexStr    []string
}

func TestQueryServerTestSuite(t *testing.T) {
	suite.Run(t, new(QueryServerTestSuite))
}

func (s *QueryServerTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey("emissions")
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithHeaderInfo(header.Info{Time: time.Now()}) // nolint: exhaustruct
	encCfg := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, module.AppModule{})
	addressCodec := address.NewBech32Codec(params.Bech32PrefixAccAddr)

	maccPerms := map[string][]string{
		"fee_collector":                {"minter"},
		"mint":                         {"minter"},
		types.AlloraStakingAccountName: {"burner", "minter", "staking"},
		mintTypes.EcosystemModuleName:  {"burner", "minter", "staking"},
		types.AlloraRewardsAccountName: {"minter"},
		types.AlloraPendingRewardForDelegatorAccountName: {"minter"},
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

	countAddrs := 11
	var privKeys = make([]secp256k1.PrivKey, countAddrs)
	var pubKeyHexStr = make([]string, countAddrs)
	var addrs = make([]sdk.AccAddress, countAddrs)
	var addrsStr = make([]string, countAddrs)
	for i := 0; i < countAddrs; i++ {
		privKeys[i] = secp256k1.GenPrivKey()
		addrs[i] = sdk.AccAddress(privKeys[i].PubKey().Address())
		pubKeyHexStr[i] = hex.EncodeToString(privKeys[i].PubKey().Bytes())
		addrsStr[i] = addrs[i].String()
	}
	s.privKeys = privKeys
	s.addrs = addrs
	s.addrsStr = addrsStr
	s.pubKeyHexStr = pubKeyHexStr

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
	s.queryServer = queryserver.NewQueryServerImpl(s.emissionsKeeper)

	s.appModule = appModule

	// Add all tests addresses in whitelists
	for _, addr := range addrsStr {
		err := s.emissionsKeeper.AddWhitelistAdmin(ctx, addr)
		s.Require().NoError(err)
	}
}

func GeneratePrivateKeys(numKeys int) []ChainKey {
	testAddrs := make([]ChainKey, numKeys)
	for i := 0; i < numKeys; i++ {
		pk, prk, _ := ed25519.GenerateKey(nil)
		testAddrs[i] = ChainKey{
			pubKey: pk,
			priKey: prk,
		}
	}

	return testAddrs
}

func (s *QueryServerTestSuite) MintTokensToAddress(address sdk.AccAddress, amount cosmosMath.Int) {
	creatorInitialBalanceCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amount))

	err := s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, creatorInitialBalanceCoins)
	s.Require().NoError(err)
	err = s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, address, creatorInitialBalanceCoins)
	s.Require().NoError(err)
}

func (s *QueryServerTestSuite) CreateOneTopic() uint64 {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Create a topic first
	metadata := "Some metadata for the new topic"
	// Create a CreateNewTopicRequest message

	creator := s.addrs[0]

	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  creator.String(),
		Metadata:                 metadata,
		LossMethod:               "method",
		EpochLength:              10800,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		AllowNegative:            false,
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
		PNorm:                    alloraMath.NewDecFromInt64(3),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
	}

	s.MintTokensToAddress(creator, types.DefaultParams().CreateTopicFee)

	result, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on first creation")

	return result.TopicId
}

func (s *QueryServerTestSuite) TestCreateSeveralTopics() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	// Mock setup for metadata and validation steps
	metadata := "Some metadata for the new topic"
	// Create a CreateNewTopicRequest message

	creator := s.addrs[0]

	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  creator.String(),
		Metadata:                 metadata,
		LossMethod:               "mse",
		EpochLength:              10800,
		GroundTruthLag:           10800,
		WorkerSubmissionWindow:   10,
		AllowNegative:            false,
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
		PNorm:                    alloraMath.NewDecFromInt64(3),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
	}

	creatorInitialBalance := types.DefaultParams().CreateTopicFee.Mul(cosmosMath.NewInt(3))
	creatorInitialBalanceCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, creatorInitialBalance))

	err := s.bankKeeper.MintCoins(ctx, types.AlloraStakingAccountName, creatorInitialBalanceCoins)
	s.Require().NoError(err)
	err = s.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.AlloraStakingAccountName, creator, creatorInitialBalanceCoins)
	s.Require().NoError(err)

	initialTopicId, err := s.emissionsKeeper.GetNextTopicId(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(initialTopicId)

	_, err = msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on first creation")

	result, err := s.emissionsKeeper.GetNextTopicId(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(initialTopicId+1, result)

	// Create second topic
	_, err = msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on second topic")

	result, err = s.emissionsKeeper.GetNextTopicId(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(initialTopicId+2, result)
}

func (s *QueryServerTestSuite) mockTopic() types.Topic {
	return types.Topic{
		Id:                       uint64(1),
		Creator:                  s.addrs[0].String(),
		Metadata:                 "whatever",
		LossMethod:               "loss",
		EpochLastEnded:           2,
		EpochLength:              100,
		GroundTruthLag:           100,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		WorkerSubmissionWindow:   100,
		AllowNegative:            true,
		Epsilon:                  alloraMath.MustNewDecFromString("0.1"),
		InitialRegret:            alloraMath.MustNewDecFromString("0.1"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.1"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.1"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.1"),
	}
}

func (s *QueryServerTestSuite) signValueBundle(valueBundle *types.ValueBundle, privateKey secp256k1.PrivKey) []byte {
	require := s.Require()
	src := make([]byte, 0)
	src, err := valueBundle.XXX_Marshal(src, true)
	require.NoError(err, "Marshall reputer value bundle should not return an error")

	valueBundleSignature, err := privateKey.Sign(src)
	require.NoError(err, "Sign should not return an error")

	return valueBundleSignature
}
