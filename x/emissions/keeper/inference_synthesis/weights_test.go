package inferencesynthesis_test

import (
	"testing"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	alloratestutil "github.com/allora-network/allora-chain/test/testutil"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"

	synth "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/module"
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

type WeightsTestSuite struct {
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

func TestWeightsTestSuite(t *testing.T) {
	suite.Run(t, new(WeightsTestSuite))
}

func (s *WeightsTestSuite) SetupTest() {
	// Setup similar to network_inference_builder_test.go
	key := storetypes.NewKVStoreKey("emissions")
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	// Set logger to show logs from the rewards module too
	logger := log.NewTestLogger(s.T()).With("module", "inference_synthesis")
	ctx := testCtx.Ctx.WithHeaderInfo(header.Info{
		Height:  1,
		Hash:    []byte("1"),
		AppHash: []byte("1"),
		ChainID: "localnet",
		Time:    time.Now(),
	}).WithLogger(logger)
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
	s.privKeys, s.pubKeyHexStr, s.addrs, s.addrsStr = alloratestutil.GenerateTestAccounts(5)

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
		authtypes.FeeCollectorName,
	)

	s.key = key
	appModule := module.NewAppModule(encCfg.Codec, s.emissionsKeeper)
	defaultGenesis := appModule.DefaultGenesis(encCfg.Codec)
	appModule.InitGenesis(ctx, encCfg.Codec, defaultGenesis)
	s.appModule = appModule

	// Add all tests addresses in whitelists
	for _, addr := range s.addrsStr {
		err := s.emissionsKeeper.AddWhitelistAdmin(ctx, addr)
		s.Require().NoError(err)
	}

	err := s.emissionsKeeper.SetTopic(s.ctx, 1, emissionstypes.Topic{
		Id:                       1,
		Creator:                  s.addrsStr[0],
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
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.01"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.01"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.01"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.01"),
	})
	s.Require().NoError(err)
}

func (s *WeightsTestSuite) TestNormalizeWeights2() {
	testCases := []struct {
		name        string
		weights     synth.RegretInformedWeights
		expectError bool
		expected    map[string]alloraMath.Dec // expected normalized weights for each worker
	}{
		{
			name: "simple case - three weights",
			weights: synth.RegretInformedWeights{
				Inferers: map[string]alloraMath.Dec{
					s.addrsStr[0]: alloraMath.MustNewDecFromString("2.0"),
					s.addrsStr[1]: alloraMath.MustNewDecFromString("3.0"),
				},
				Forecasters: map[string]alloraMath.Dec{
					s.addrsStr[2]: alloraMath.MustNewDecFromString("5.0"),
				},
			},
			expectError: false,
			expected: map[string]alloraMath.Dec{
				s.addrsStr[0]: alloraMath.MustNewDecFromString("0.2"), // 2/10
				s.addrsStr[1]: alloraMath.MustNewDecFromString("0.3"), // 3/10
				s.addrsStr[2]: alloraMath.MustNewDecFromString("0.5"), // 5/10
			},
		},
		{
			name: "equal weights",
			weights: synth.RegretInformedWeights{
				Inferers: map[string]alloraMath.Dec{
					s.addrsStr[0]: alloraMath.MustNewDecFromString("1.0"),
					s.addrsStr[1]: alloraMath.MustNewDecFromString("1.0"),
				},
				Forecasters: map[string]alloraMath.Dec{
					s.addrsStr[2]: alloraMath.MustNewDecFromString("1.0"),
				},
			},
			expectError: false,
			expected: map[string]alloraMath.Dec{
				s.addrsStr[0]: alloraMath.MustNewDecFromString("0.333333333333333333"),
				s.addrsStr[1]: alloraMath.MustNewDecFromString("0.333333333333333333"),
				s.addrsStr[2]: alloraMath.MustNewDecFromString("0.333333333333333333"),
			},
		},
		{
			name: "empty maps",
			weights: synth.RegretInformedWeights{
				Inferers:    map[string]alloraMath.Dec{},
				Forecasters: map[string]alloraMath.Dec{},
			},
			expectError: true,
			expected:    nil,
		},
		{
			name: "zero weights",
			weights: synth.RegretInformedWeights{
				Inferers: map[string]alloraMath.Dec{
					s.addrsStr[0]: alloraMath.ZeroDec(),
					s.addrsStr[1]: alloraMath.ZeroDec(),
				},
				Forecasters: map[string]alloraMath.Dec{
					s.addrsStr[2]: alloraMath.ZeroDec(),
				},
			},
			expectError: true,
			expected:    nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.weights.NormalizeWeights()

			if tc.expectError {
				s.Require().Error(err)
				return
			}

			s.Require().NoError(err)

			// Verify each weight matches expected
			for addr, expectedWeight := range tc.expected {
				var actualWeight alloraMath.Dec
				if weight, ok := tc.weights.Inferers[addr]; ok {
					actualWeight = weight
				} else if weight, ok := tc.weights.Forecasters[addr]; ok {
					actualWeight = weight
				}
				ok, err := alloraMath.InDelta(expectedWeight, actualWeight, alloraMath.MustNewDecFromString("0.00000001"))
				s.Require().NoError(err)
				s.Require().True(ok,
					"Weight for %s: expected %s, got %s",
					addr, expectedWeight, actualWeight)
			}

			// Verify sum is 1.0
			sum := alloraMath.ZeroDec()
			for _, w := range tc.weights.Inferers {
				sum, err = sum.Add(w)
				s.Require().NoError(err)
			}
			for _, w := range tc.weights.Forecasters {
				sum, err = sum.Add(w)
				s.Require().NoError(err)
			}
			ok, err := alloraMath.InDelta(sum, alloraMath.OneDec(), alloraMath.MustNewDecFromString("0.00000001"))
			s.Require().NoError(err)
			s.Require().True(ok,
				"Sum of weights: expected %s, got %s",
				alloraMath.OneDec(), sum)
		})
	}
}

func (s *WeightsTestSuite) TestStoreLatestNormalizedWeights() {
	s.Run("store and retrieve normalized weights", func() {
		topicId := uint64(1)
		weights := synth.RegretInformedWeights{
			Inferers: map[string]alloraMath.Dec{
				s.addrsStr[0]: alloraMath.MustNewDecFromString("0.2"),
				s.addrsStr[1]: alloraMath.MustNewDecFromString("0.3"),
			},
			Forecasters: map[string]alloraMath.Dec{
				s.addrsStr[2]: alloraMath.MustNewDecFromString("0.5"),
			},
		}

		err := synth.StoreLatestNormalizedWeights(s.ctx, s.emissionsKeeper, topicId, weights)
		s.Require().NoError(err)

		// Verify stored weights
		for worker, expectedWeight := range weights.Inferers {
			storedWeight, err := s.emissionsKeeper.GetLatestInfererWeight(s.ctx, topicId, worker)
			s.Require().NoError(err)
			s.Require().True(expectedWeight.Equal(storedWeight))
		}
	})
}

func (s *WeightsTestSuite) TestGatherWorkerRegrets() {
	s.Run("gather regrets from workers", func() {
		inferers := []string{s.addrsStr[0], s.addrsStr[1]}
		forecasters := []string{s.addrsStr[2]}

		dec1 := alloraMath.MustNewDecFromString("0.1")
		dec2 := alloraMath.MustNewDecFromString("0.2")
		dec3 := alloraMath.MustNewDecFromString("0.3")

		infererToRegret := map[string]*alloraMath.Dec{
			s.addrsStr[0]: &dec1,
			s.addrsStr[1]: &dec2,
		}
		forecasterToRegret := map[string]*alloraMath.Dec{
			s.addrsStr[2]: &dec3,
		}

		regrets, infererRegrets, forecasterRegrets, err := synth.GatherWorkerRegrets(
			s.ctx.Logger(),
			inferers,
			forecasters,
			infererToRegret,
			forecasterToRegret,
		)
		s.Require().NoError(err)
		s.Require().Len(regrets, 3)
		s.Require().Len(infererRegrets, 2)
		s.Require().Len(forecasterRegrets, 1)
	})
}

func (s *WeightsTestSuite) TestCalcStdDevPlusEpsilon() {
	testCases := []struct {
		name     string
		regrets  []alloraMath.Dec
		epsilon  alloraMath.Dec
		expected alloraMath.Dec
	}{
		{
			name: "simple case - three values",
			regrets: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("0.1"),
				alloraMath.MustNewDecFromString("0.2"),
				alloraMath.MustNewDecFromString("0.3"),
			},
			epsilon:  alloraMath.MustNewDecFromString("0.01"),
			expected: alloraMath.MustNewDecFromString("0.11"),
		},
		{
			name: "all same values",
			regrets: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("0.1"),
				alloraMath.MustNewDecFromString("0.1"),
				alloraMath.MustNewDecFromString("0.1"),
			},
			epsilon:  alloraMath.MustNewDecFromString("0.01"),
			expected: alloraMath.MustNewDecFromString("0.01"),
		},
		{
			name: "larger spread",
			regrets: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("0.0"),
				alloraMath.MustNewDecFromString("0.5"),
				alloraMath.MustNewDecFromString("1.0"),
			},
			epsilon:  alloraMath.MustNewDecFromString("0.1"),
			expected: alloraMath.MustNewDecFromString("0.6"),
		},
		{
			name: "larger epsilon",
			regrets: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("0.1"),
				alloraMath.MustNewDecFromString("0.2"),
				alloraMath.MustNewDecFromString("0.3"),
			},
			epsilon:  alloraMath.MustNewDecFromString("0.5"),
			expected: alloraMath.MustNewDecFromString("0.6"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result, err := synth.CalcStdDevPlusEpsilon(tc.regrets, tc.epsilon)
			s.Require().NoError(err)
			s.Require().True(result.Gte(tc.epsilon), "result should be greater than epsilon")
			s.Require().True(result.Equal(tc.expected),
				"expected %s but got %s", tc.expected.String(), result.String())
		})
	}
}

func (s *WeightsTestSuite) TestCalcStdDevForWeights() {
	testCases := []struct {
		name                string
		inferers            []string
		forecasters         []string
		infererWeights      map[string]alloraMath.Dec // weights to be stored in keeper
		forecasterWeights   map[string]alloraMath.Dec // weights to be stored in keeper
		infererRegrets      map[string]*alloraMath.Dec
		forecasterRegrets   map[string]*alloraMath.Dec
		negligibleThreshold alloraMath.Dec
		epsilonTopic        alloraMath.Dec
		expectedResult      alloraMath.Dec
		expectFiltered      bool // whether we expect filtered or all regrets
	}{
		{
			name:        "all weights above threshold",
			inferers:    []string{s.addrsStr[0], s.addrsStr[1]},
			forecasters: []string{s.addrsStr[2]},
			infererWeights: map[string]alloraMath.Dec{
				s.addrsStr[0]: alloraMath.MustNewDecFromString("0.4"),
				s.addrsStr[1]: alloraMath.MustNewDecFromString("0.3"),
			},
			forecasterWeights: map[string]alloraMath.Dec{
				s.addrsStr[2]: alloraMath.MustNewDecFromString("0.3"),
			},
			infererRegrets: map[string]*alloraMath.Dec{
				s.addrsStr[0]: decPtr("0.1"),
				s.addrsStr[1]: decPtr("0.2"),
			},
			forecasterRegrets: map[string]*alloraMath.Dec{
				s.addrsStr[2]: decPtr("0.3"),
			},
			negligibleThreshold: alloraMath.MustNewDecFromString("0.1"),
			epsilonTopic:        alloraMath.MustNewDecFromString("0.01"),
			expectedResult:      alloraMath.MustNewDecFromString("0.11"),
			expectFiltered:      true,
		},
		{
			name:        "some weights below threshold",
			inferers:    []string{s.addrsStr[0], s.addrsStr[1]},
			forecasters: []string{s.addrsStr[2]},
			infererWeights: map[string]alloraMath.Dec{
				s.addrsStr[0]: alloraMath.MustNewDecFromString("0.05"), // below threshold
				s.addrsStr[1]: alloraMath.MustNewDecFromString("0.3"),
			},
			forecasterWeights: map[string]alloraMath.Dec{
				s.addrsStr[2]: alloraMath.MustNewDecFromString("0.3"),
			},
			infererRegrets: map[string]*alloraMath.Dec{
				s.addrsStr[0]: decPtr("0.1"),
				s.addrsStr[1]: decPtr("0.2"),
			},
			forecasterRegrets: map[string]*alloraMath.Dec{
				s.addrsStr[2]: decPtr("0.3"),
			},
			negligibleThreshold: alloraMath.MustNewDecFromString("0.1"),
			epsilonTopic:        alloraMath.MustNewDecFromString("0.01"),
			expectedResult:      alloraMath.MustNewDecFromString("0.08071067811865475"),
			expectFiltered:      true,
		},
		{
			name:        "less than 2 non-negligible weights",
			inferers:    []string{s.addrsStr[0], s.addrsStr[1]},
			forecasters: []string{s.addrsStr[2]},
			infererWeights: map[string]alloraMath.Dec{
				s.addrsStr[0]: alloraMath.MustNewDecFromString("0.05"), // below threshold
				s.addrsStr[1]: alloraMath.MustNewDecFromString("0.05"), // below threshold
			},
			forecasterWeights: map[string]alloraMath.Dec{
				s.addrsStr[2]: alloraMath.MustNewDecFromString("0.3"),
			},
			infererRegrets: map[string]*alloraMath.Dec{
				s.addrsStr[0]: decPtr("0.1"),
				s.addrsStr[1]: decPtr("0.2"),
			},
			forecasterRegrets: map[string]*alloraMath.Dec{
				s.addrsStr[2]: decPtr("0.3"),
			},
			negligibleThreshold: alloraMath.MustNewDecFromString("0.1"),
			epsilonTopic:        alloraMath.MustNewDecFromString("0.01"),
			expectedResult:      alloraMath.MustNewDecFromString("0.11"), // uses all regrets
			expectFiltered:      false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Store weights in keeper
			for worker, weight := range tc.infererWeights {
				err := s.emissionsKeeper.SetLatestInfererWeight(s.ctx, 1, worker, weight)
				s.Require().NoError(err)
			}
			for worker, weight := range tc.forecasterWeights {
				err := s.emissionsKeeper.SetLatestForecasterWeight(s.ctx, 1, worker, weight)
				s.Require().NoError(err)
			}

			result, err := synth.CalcRegretStdDevFilteredByWeights(synth.CalcRegretStdDevFilteredByWeightsArgs{
				Ctx:                 s.ctx,
				K:                   &s.emissionsKeeper,
				Logger:              s.ctx.Logger(),
				TopicId:             1, // topicId
				Inferers:            tc.inferers,
				Forecasters:         tc.forecasters,
				InfererToRegret:     tc.infererRegrets,
				ForecasterToRegret:  tc.forecasterRegrets,
				NegligibleThreshold: tc.negligibleThreshold,
				EpsilonTopic:        tc.epsilonTopic,
			})

			s.Require().NoError(err)
			ok, err := alloraMath.InDelta(result, tc.expectedResult, alloraMath.MustNewDecFromString("0.00000001"))
			s.Require().NoError(err)
			s.Require().True(ok,
				"Expected %s but got %s", tc.expectedResult, result)
		})
	}
}

// Helper function to create Dec pointer
func decPtr(s string) *alloraMath.Dec {
	dec := alloraMath.MustNewDecFromString(s)
	return &dec
}

func (s *WeightsTestSuite) TestCalcWeightsGivenWorkers() {
	testCases := []struct {
		name          string
		args          synth.CalcWeightsGivenWorkersArgs
		expectedError bool
		checkResult   func(result synth.RegretInformedWeights)
	}{
		{
			name: "basic calculation with single inferer and forecaster",
			args: synth.CalcWeightsGivenWorkersArgs{
				Logger:      s.ctx.Logger(),
				Inferers:    []string{s.addrsStr[0]},
				Forecasters: []string{s.addrsStr[1]},
				InfererToRegret: map[string]*alloraMath.Dec{
					s.addrsStr[0]: decPtr("1.0"),
				},
				ForecasterToRegret: map[string]*alloraMath.Dec{
					s.addrsStr[1]: decPtr("2.0"),
				},
				EpsilonTopic:      alloraMath.MustNewDecFromString("0.01"),
				PNorm:             alloraMath.MustNewDecFromString("3.0"),
				CNorm:             alloraMath.MustNewDecFromString("0.75"),
				StdDevPlusEpsilon: alloraMath.MustNewDecFromString("1.0"),
			},
			expectedError: false,
			checkResult: func(result synth.RegretInformedWeights) {
				s.Require().Equal(1, len(result.Inferers))
				s.Require().Equal(1, len(result.Forecasters))
				s.Require().True(result.Inferers[s.addrsStr[0]].Lt(result.Forecasters[s.addrsStr[1]]))
			},
		},
		{
			name: "basic calculation with negative inferer and positive forecaster",
			args: synth.CalcWeightsGivenWorkersArgs{
				Logger:      s.ctx.Logger(),
				Inferers:    []string{s.addrsStr[0]},
				Forecasters: []string{s.addrsStr[1]},
				InfererToRegret: map[string]*alloraMath.Dec{
					s.addrsStr[0]: decPtr("-1.0"),
				},
				ForecasterToRegret: map[string]*alloraMath.Dec{
					s.addrsStr[1]: decPtr("2.0"),
				},
				EpsilonTopic:      alloraMath.MustNewDecFromString("0.01"),
				PNorm:             alloraMath.MustNewDecFromString("3.0"),
				CNorm:             alloraMath.MustNewDecFromString("0.75"),
				StdDevPlusEpsilon: alloraMath.MustNewDecFromString("1.0"),
			},
			expectedError: false,
			checkResult: func(result synth.RegretInformedWeights) {
				s.T().Logf("Single worker test results:")
				s.Require().Equal(1, len(result.Inferers))
				s.Require().Equal(1, len(result.Forecasters))
				s.Require().True(result.Inferers[s.addrsStr[0]].Lt(result.Forecasters[s.addrsStr[1]]))
			},
		},
		{
			name: "basic calculation with positive inferer and negative forecaster",
			args: synth.CalcWeightsGivenWorkersArgs{
				Logger:      s.ctx.Logger(),
				Inferers:    []string{s.addrsStr[0]},
				Forecasters: []string{s.addrsStr[1]},
				InfererToRegret: map[string]*alloraMath.Dec{
					s.addrsStr[0]: decPtr("1.0"),
				},
				ForecasterToRegret: map[string]*alloraMath.Dec{
					s.addrsStr[1]: decPtr("-2.0"),
				},
				EpsilonTopic:      alloraMath.MustNewDecFromString("0.01"),
				PNorm:             alloraMath.MustNewDecFromString("3.0"),
				CNorm:             alloraMath.MustNewDecFromString("0.75"),
				StdDevPlusEpsilon: alloraMath.MustNewDecFromString("1.0"),
			},
			expectedError: false,
			checkResult: func(result synth.RegretInformedWeights) {
				s.Require().Equal(1, len(result.Inferers))
				s.Require().Equal(1, len(result.Forecasters))
				s.Require().True(result.Inferers[s.addrsStr[0]].Gt(result.Forecasters[s.addrsStr[1]]))
			},
		},
		{
			name: "calculation with multiple workers and mixed positive and negative regrets",
			args: synth.CalcWeightsGivenWorkersArgs{
				Logger:      s.ctx.Logger(),
				Inferers:    []string{s.addrsStr[0], s.addrsStr[1]},
				Forecasters: []string{s.addrsStr[2], s.addrsStr[3]},
				InfererToRegret: map[string]*alloraMath.Dec{
					s.addrsStr[0]: decPtr("-1.0"),
					s.addrsStr[1]: decPtr("2.0"),
				},
				ForecasterToRegret: map[string]*alloraMath.Dec{
					s.addrsStr[2]: decPtr("1.5"),
					s.addrsStr[3]: decPtr("-0.5"),
				},
				EpsilonTopic:      alloraMath.MustNewDecFromString("0.01"),
				PNorm:             alloraMath.MustNewDecFromString("3.0"),
				CNorm:             alloraMath.MustNewDecFromString("0.75"),
				StdDevPlusEpsilon: alloraMath.MustNewDecFromString("1.0"),
			},
			expectedError: false,
			checkResult: func(result synth.RegretInformedWeights) {
				s.Require().Equal(2, len(result.Inferers))
				s.Require().Equal(2, len(result.Forecasters))

				// Check that worker with higher regret has a higher weight
				s.Require().True(result.Inferers[s.addrsStr[0]].Lt(result.Inferers[s.addrsStr[1]]))
				s.Require().True(result.Forecasters[s.addrsStr[2]].Gt(result.Forecasters[s.addrsStr[3]]))
				// compare mixed
				s.Require().True(result.Forecasters[s.addrsStr[3]].Gt(result.Inferers[s.addrsStr[0]]))
				s.Require().True(result.Forecasters[s.addrsStr[2]].Lt(result.Inferers[s.addrsStr[1]]))

			},
		},
		{
			name: "empty workers should error",
			args: synth.CalcWeightsGivenWorkersArgs{
				Logger:             s.ctx.Logger(),
				Inferers:           []string{},
				Forecasters:        []string{},
				InfererToRegret:    map[string]*alloraMath.Dec{},
				ForecasterToRegret: map[string]*alloraMath.Dec{},
				EpsilonTopic:       alloraMath.MustNewDecFromString("0.01"),
				PNorm:              alloraMath.MustNewDecFromString("3.0"),
				CNorm:              alloraMath.MustNewDecFromString("0.75"),
				StdDevPlusEpsilon:  alloraMath.MustNewDecFromString("1.0"),
			},
			expectedError: true,
		},
		{
			name: "missing regret values should error",
			args: synth.CalcWeightsGivenWorkersArgs{
				Logger:             s.ctx.Logger(),
				Inferers:           []string{s.addrsStr[0]},
				Forecasters:        []string{s.addrsStr[1]},
				InfererToRegret:    map[string]*alloraMath.Dec{},
				ForecasterToRegret: map[string]*alloraMath.Dec{},
				EpsilonTopic:       alloraMath.MustNewDecFromString("0.01"),
				PNorm:              alloraMath.MustNewDecFromString("3.0"),
				CNorm:              alloraMath.MustNewDecFromString("0.75"),
				StdDevPlusEpsilon:  alloraMath.MustNewDecFromString("1.0"),
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result, err := synth.CalcWeightsGivenWorkers(tc.args)

			if tc.expectedError {
				s.Require().Error(err)
				return
			}

			s.Require().NoError(err)
			s.Require().NotNil(result)

			if tc.checkResult != nil {
				tc.checkResult(result)
			}
		})
	}
}
