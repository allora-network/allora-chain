package actorutils_test

import (
	"testing"
	"time"

	"cosmossdk.io/core/header"
	clog "cosmossdk.io/log"
	alloratestutil "github.com/allora-network/allora-chain/test/testutil"

	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	"github.com/allora-network/allora-chain/x/emissions/module"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	auth "github.com/cosmos/cosmos-sdk/x/auth"
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

type ActorUtilsTestSuite struct {
	suite.Suite

	ctx             sdk.Context
	accountKeeper   keeper.AccountKeeper
	bankKeeper      keeper.BankKeeper
	emissionsKeeper keeper.Keeper
	appModule       module.AppModule
	key             *storetypes.KVStoreKey
	privKeys        []secp256k1.PrivKey
	addrs           []sdk.AccAddress
	addrsStr        []string
	pubKeyHexStr    []string
}

func (a *ActorUtilsTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey("emissions")
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(a.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithHeaderInfo(header.Info{Time: time.Now()}) // nolint: exhaustruct
	encCfg := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, module.AppModule{})
	addressCodec := address.NewBech32Codec(params.Bech32PrefixAccAddr)

	maccPerms := map[string][]string{
		"fee_collector":                         {"minter"},
		"mint":                                  {"minter"},
		emissionstypes.AlloraStakingAccountName: {"burner", "minter", "staking"},
		emissionstypes.AlloraRewardsAccountName: {"minter"},
		emissionstypes.AlloraPendingRewardForDelegatorAccountName: {"minter"},
		"bonded_tokens_pool":     {"burner", "staking"},
		"not_bonded_tokens_pool": {"burner", "staking"},
		multiPerm:                {"burner", "minter", "staking"},
		randomPerm:               {"random"},
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

	a.privKeys, a.pubKeyHexStr, a.addrs, a.addrsStr = alloratestutil.GenerateTestAccounts(50)

	bankKeeper := bankkeeper.NewBaseKeeper(
		encCfg.Codec,
		storeService,
		accountKeeper,
		map[string]bool{},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		clog.NewNopLogger(),
	)

	a.ctx = ctx
	a.accountKeeper = accountKeeper
	a.bankKeeper = bankKeeper
	a.emissionsKeeper = keeper.NewKeeper(
		encCfg.Codec,
		addressCodec,
		storeService,
		accountKeeper,
		bankKeeper,
		authtypes.FeeCollectorName,
	)
	a.key = key
	appModule := module.NewAppModule(encCfg.Codec, a.emissionsKeeper)
	defaultGenesis := appModule.DefaultGenesis(encCfg.Codec)
	appModule.InitGenesis(ctx, encCfg.Codec, defaultGenesis)
	a.appModule = appModule

	// Add all tests addresses in whitelists
	for _, addr := range a.addrsStr {
		err := a.emissionsKeeper.AddWhitelistAdmin(ctx, addr)
		a.Require().NoError(err)
	}

	err := a.emissionsKeeper.SetTopic(a.ctx, 1, emissionstypes.Topic{
		Id:                       1,
		Creator:                  a.addrsStr[49],
		Metadata:                 "metadata",
		LossMethod:               "mse",
		EpochLastEnded:           0,
		EpochLength:              100,
		GroundTruthLag:           100,
		WorkerSubmissionWindow:   100,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            false,
		InitialRegret:            alloraMath.MustNewDecFromString("0.0001"),
		Epsilon:                  alloraMath.MustNewDecFromString("0.0001"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.0001"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.0001"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.0001"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.0001"),
	})
	a.Require().NoError(err)
}

func TestModuleTestSuite(t *testing.T) {
	suite.Run(t, new(ActorUtilsTestSuite))
}

func (a *ActorUtilsTestSuite) signValueBundle(valueBundle *emissionstypes.ValueBundle, privateKey secp256k1.PrivKey) []byte {
	require := a.Require()
	src := make([]byte, 0)
	src, err := valueBundle.XXX_Marshal(src, true)
	require.NoError(err, "Marshall reputer value bundle should not return an error")

	valueBundleSignature, err := privateKey.Sign(src)
	require.NoError(err, "Sign should not return an error")

	return valueBundleSignature
}

func (a *ActorUtilsTestSuite) TestFilterUnacceptedWorkersFromReputerValueBundle() {
	workerNonce := emissionstypes.Nonce{
		BlockHeight: 1,
	}

	inferer1 := a.addrsStr[0]
	inferer2 := a.addrsStr[1]
	inferer4 := a.addrsStr[3]
	inferer5 := a.addrsStr[4]

	forecaster1 := a.addrsStr[5]
	forecaster2 := a.addrsStr[6]
	forecaster3 := a.addrsStr[7]
	forecaster4 := a.addrsStr[8]
	forecaster5 := a.addrsStr[9]

	infererLossBundle := emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{
				TopicId:     1,
				BlockHeight: 1,
				Inferer:     inferer1,
				Value:       alloraMath.NewDecFromInt64(1),
			},
			{
				TopicId:     1,
				BlockHeight: 1,
				Inferer:     inferer2,
				Value:       alloraMath.NewDecFromInt64(2),
			},
			{
				TopicId:     1,
				BlockHeight: 1,
				Inferer:     inferer4,
				Value:       alloraMath.NewDecFromInt64(3),
			},
		},
	}
	err := a.emissionsKeeper.InsertActiveInferences(a.ctx, 1, workerNonce.BlockHeight, infererLossBundle)
	a.Require().NoError(err)

	forecasterLossBundle := emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				TopicId:     1,
				BlockHeight: 1,
				Forecaster:  forecaster1,
				ForecastElements: []*emissionstypes.ForecastElement{
					{
						Inferer: inferer1,
						Value:   alloraMath.NewDecFromInt64(1),
					},
					{
						Inferer: inferer2,
						Value:   alloraMath.NewDecFromInt64(2),
					},
				},
			},
			{
				TopicId:     1,
				BlockHeight: 1,
				Forecaster:  forecaster2,
				ForecastElements: []*emissionstypes.ForecastElement{
					{
						Inferer: inferer1,
						Value:   alloraMath.NewDecFromInt64(3),
					},
				},
			},
		},
	}
	err = a.emissionsKeeper.InsertActiveForecasts(a.ctx, 1, workerNonce.BlockHeight, forecasterLossBundle)
	a.Require().NoError(err)

	// Prepare a sample ReputerValueBundle
	valueBundle := &emissionstypes.ValueBundle{
		TopicId: 1,
		ReputerRequestNonce: &emissionstypes.ReputerRequestNonce{
			ReputerNonce: &workerNonce,
		},
		Reputer:       a.addrsStr[40],
		ExtraData:     nil,
		CombinedValue: alloraMath.NewDecFromInt64(1000),
		InfererValues: []*emissionstypes.WorkerAttributedValue{
			{Worker: inferer1, Value: alloraMath.NewDecFromInt64(100)},
			{Worker: inferer2, Value: alloraMath.NewDecFromInt64(100)},
			{Worker: inferer5, Value: alloraMath.NewDecFromInt64(200)}, // Should be filtered out
		},
		ForecasterValues: []*emissionstypes.WorkerAttributedValue{
			{Worker: forecaster1, Value: alloraMath.NewDecFromInt64(300)},
			{Worker: forecaster3, Value: alloraMath.NewDecFromInt64(400)}, // Should be filtered out
		},
		NaiveValue: alloraMath.NewDecFromInt64(1000),
		OneOutInfererValues: []*emissionstypes.WithheldWorkerAttributedValue{
			{Worker: inferer1, Value: alloraMath.NewDecFromInt64(500)},
			{Worker: inferer5, Value: alloraMath.NewDecFromInt64(600)}, // Should be filtered out
		},
		OneOutForecasterValues: []*emissionstypes.WithheldWorkerAttributedValue{
			{Worker: forecaster1, Value: alloraMath.NewDecFromInt64(700)},
			{Worker: forecaster4, Value: alloraMath.NewDecFromInt64(800)}, // Should be filtered out
		},
		OneOutInfererForecasterValues: []*emissionstypes.OneOutInfererForecasterValues{
			{
				Forecaster: forecaster1,
				OneOutInfererValues: []*emissionstypes.WithheldWorkerAttributedValue{
					{Worker: inferer1, Value: alloraMath.NewDecFromInt64(900)},
					{Worker: inferer5, Value: alloraMath.NewDecFromInt64(1000)}, // Should be filtered out
				},
			},
			{
				Forecaster: forecaster3, // Should be filtered out
				OneOutInfererValues: []*emissionstypes.WithheldWorkerAttributedValue{
					{Worker: inferer1, Value: alloraMath.NewDecFromInt64(1100)},
				},
			},
		},
		OneInForecasterValues: []*emissionstypes.WorkerAttributedValue{
			{Worker: forecaster1, Value: alloraMath.NewDecFromInt64(1200)},
			{Worker: forecaster5, Value: alloraMath.NewDecFromInt64(1300)}, // Should be filtered out
		},
	}
	signature := a.signValueBundle(valueBundle, a.privKeys[40])
	reputerValueBundle := &emissionstypes.ReputerValueBundle{
		Signature:   signature,
		Pubkey:      a.pubKeyHexStr[40],
		ValueBundle: valueBundle,
	}

	acceptedBundle, err := actorutils.FilterUnacceptedWorkersFromReputerValueBundle(&a.emissionsKeeper, a.ctx, 1, emissionstypes.ReputerRequestNonce{ReputerNonce: &workerNonce}, reputerValueBundle)
	a.Require().NoError(err)

	// Validate the bundle
	a.Require().Len(acceptedBundle.ValueBundle.InfererValues, 2)
	a.Require().Len(acceptedBundle.ValueBundle.ForecasterValues, 1)
	a.Require().Len(acceptedBundle.ValueBundle.OneOutInfererValues, 1)
	a.Require().Len(acceptedBundle.ValueBundle.OneOutForecasterValues, 1)
	a.Require().Len(acceptedBundle.ValueBundle.OneOutInfererForecasterValues, 1)
	a.Require().Len(acceptedBundle.ValueBundle.OneOutInfererForecasterValues[0].OneOutInfererValues, 1)
	a.Require().Len(acceptedBundle.ValueBundle.OneInForecasterValues, 1)
}
