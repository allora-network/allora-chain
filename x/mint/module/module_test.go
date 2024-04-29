package mint_test

import (
	"testing"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/app/params"

	"github.com/allora-network/allora-chain/x/mint/keeper"
	mint "github.com/allora-network/allora-chain/x/mint/module"
	"github.com/allora-network/allora-chain/x/mint/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"

	emissionskeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/suite"
)

const (
	multiPerm  = "multiple permissions account"
	randomPerm = "random permission"
)

type MintModuleTestSuite struct {
	suite.Suite

	ctx             sdk.Context
	accountKeeper   types.AccountKeeper
	stakingKeeper   types.StakingKeeper
	bankKeeper      types.BankKeeper
	appModule       mint.AppModule
	emissionsKeeper emissionskeeper.Keeper
	mintKeeper      keeper.Keeper
	PKS             []cryptotypes.PubKey
}

// SetupTest setups a new test, to be run before each test case
func (s *MintModuleTestSuite) SetupTest() {
	sdk.DefaultBondDenom = params.DefaultBondDenom
	s.PKS = simtestutil.CreateTestPubKeys(4)
	key := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	encCfg := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, staking.AppModuleBasic{}, bank.AppModuleBasic{}, mint.AppModuleBasic{})
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithHeaderInfo(header.Info{Time: time.Now()})

	maccPerms := map[string][]string{
		"fee_collector":                     nil,
		"mint":                              {"minter"},
		emissions.AlloraStakingAccountName:  {"burner", "minter", "staking"},
		emissions.AlloraRequestsAccountName: {"burner", "minter", "staking"},
		"bonded_tokens_pool":                {"burner", "staking"},
		"not_bonded_tokens_pool":            {"burner", "staking"},
		multiPerm:                           {"burner", "minter", "staking"},
		randomPerm:                          {"random"},
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

	stakingKeeper := stakingkeeper.NewKeeper(
		encCfg.Codec,
		storeService,
		accountKeeper,
		bankKeeper,
		authtypes.NewModuleAddress("gov").String(),
		addresscodec.NewBech32Codec(sdk.Bech32PrefixValAddr),
		addresscodec.NewBech32Codec(sdk.Bech32PrefixConsAddr),
	)
	stakingKeeper.SetParams(ctx, stakingtypes.Params{
		UnbondingTime:     60,
		MaxValidators:     100,
		MaxEntries:        7,
		HistoricalEntries: 1000,
		BondDenom:         sdk.DefaultBondDenom,
		MinCommissionRate: cosmosMath.LegacyNewDecWithPrec(1, 2),
	})
	emissionsKeeper := emissionskeeper.NewKeeper(
		encCfg.Codec,
		addresscodec.NewBech32Codec(sdk.Bech32PrefixAccAddr),
		storeService,
		accountKeeper,
		bankKeeper,
		"fee_collector",
	)

	mintKeeper := keeper.NewKeeper(
		encCfg.Codec,
		storeService,
		stakingKeeper,
		accountKeeper,
		bankKeeper,
		emissionsKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress("gov").String(),
	)

	s.ctx = ctx
	s.accountKeeper = accountKeeper
	s.bankKeeper = bankKeeper
	s.stakingKeeper = stakingKeeper
	s.emissionsKeeper = emissionsKeeper
	s.mintKeeper = mintKeeper

	appModule := mint.NewAppModule(encCfg.Codec, s.mintKeeper, s.accountKeeper)
	defaultGenesis := appModule.DefaultGenesis(encCfg.Codec)
	appModule.InitGenesis(ctx, encCfg.Codec, defaultGenesis)
	s.appModule = appModule
}

func TestMintModuleTestSuite(t *testing.T) {
	suite.Run(t, new(MintModuleTestSuite))
}

func (s *MintModuleTestSuite) TestMintingAtMaxSupply() {
	//todo add a test here
}

func (s *MintModuleTestSuite) TestTotalStakeGoUpTargetEmissionPerUnitStakeGoDown() {
	params, err := s.mintKeeper.GetParams(s.ctx)
	s.Require().NoError(err)
	ecosystemMintSupplyRemaining, err := mint.GetEcosystemMintSupplyRemaining(s.ctx, s.mintKeeper, params)
	s.Require().NoError(err)
	// stake enough tokens so that the networkStaked is non zero
	err = s.emissionsKeeper.AddStake(
		s.ctx,
		0,
		sdk.AccAddress(s.PKS[0].Address()),
		cosmosMath.NewUintFromString("40000000000000000000"),
	)
	s.Require().NoError(err)

	// mint enough tokens so that the circulating supply is non zero
	spareCoins, ok := cosmosMath.NewIntFromString("500000000000000000000000000")
	s.Require().True(ok)
	err = s.bankKeeper.MintCoins(
		s.ctx,
		emissions.AlloraRequestsAccountName,
		sdk.NewCoins(
			sdk.NewCoin(
				params.MintDenom,
				spareCoins,
			),
		),
	)
	s.Require().NoError(err)

	_, emissionPerUnitStakedTokenBefore, err := mint.GetEmissionPerTimestep(
		s.ctx,
		s.mintKeeper,
		params,
		ecosystemMintSupplyRemaining,
	)
	s.Require().NoError(err)

	// ok now add some stake
	err = s.emissionsKeeper.AddStake(
		s.ctx,
		0,
		sdk.AccAddress(s.PKS[0].Address()),
		cosmosMath.NewUintFromString("50000000000000000000"),
	)
	s.Require().NoError(err)

	_, emissionPerUnitStakedTokenAfter, err := mint.GetEmissionPerTimestep(
		s.ctx,
		s.mintKeeper,
		params,
		ecosystemMintSupplyRemaining,
	)
	s.Require().NoError(err)

	s.Require().True(
		emissionPerUnitStakedTokenBefore.GT(emissionPerUnitStakedTokenAfter),
		"Emission per unit staked token should go down when total stake goes up all else equal: %s > %s",
		emissionPerUnitStakedTokenBefore.String(),
		emissionPerUnitStakedTokenAfter.String(),
	)

}

func (s *MintModuleTestSuite) TestEcosystemMintableRemainingGoDownTargetEmissionPerUnitStakeTokenGoDown() {
	var fEmission cosmosMath.LegacyDec = types.DefaultParams().FEmission
	networkStaked, ok := cosmosMath.NewIntFromString("1000000000000000000000") // 1000e18
	s.Require().True(ok)
	circulatingSupply, ok := cosmosMath.NewIntFromString("10000000000000000000000") // 10000e18
	s.Require().True(ok)
	maxSupply, ok := cosmosMath.NewIntFromString("1000000000000000000000000000") // 1e27
	s.Require().True(ok)
	ecosystemMintableRemainingBefore, ok := cosmosMath.NewIntFromString("367500000000000000000000000") // 1e27 * 0.3675
	s.Require().True(ok)

	e_iBefore, err := keeper.GetTargetRewardEmissionPerUnitStakedToken(
		fEmission,
		ecosystemMintableRemainingBefore,
		networkStaked,
		circulatingSupply,
		maxSupply,
	)
	s.Require().NoError(err)

	ecosystemMintableRemainingAfter, ok := cosmosMath.NewIntFromString("300000000000000000000000000") // 1e27 * 0.3
	s.Require().True(ok)
	e_iAfter, err := keeper.GetTargetRewardEmissionPerUnitStakedToken(
		fEmission,
		ecosystemMintableRemainingAfter,
		networkStaked,
		circulatingSupply,
		maxSupply,
	)
	s.Require().NoError(err)

	s.Require().True(
		e_iBefore.GT(e_iAfter),
		"Target emission per unit staked token should go down when ecosystem mintable remaining goes down all else equal: %s > %s",
	)
}
