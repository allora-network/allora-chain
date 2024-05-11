package mint_test

import (
	"testing"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"

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
		"third_party":                       {"minter"},
		"ecosystem":                         {"burner", "minter", "staking"},
		"allorarewards":                     nil,
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

func (s *MintModuleTestSuite) TestNoNewMintedTokensIfInferenceRequestFeesEnoughToCoverInflation() {
	feeCollectorAddress := s.accountKeeper.GetModuleAddress("fee_collector")
	alloraRewardsAddress := s.accountKeeper.GetModuleAddress(emissions.AlloraRewardsAccountName)
	ecosystemAddress := s.accountKeeper.GetModuleAddress(types.EcosystemModuleName)
	feeCollectorBalBefore := s.bankKeeper.GetBalance(s.ctx, feeCollectorAddress, sdk.DefaultBondDenom)
	alloraRewardsBalBefore := s.bankKeeper.GetBalance(s.ctx, alloraRewardsAddress, sdk.DefaultBondDenom)
	s.ctx = s.ctx.WithBlockHeight(1)
	// stake enough tokens so that the networkStaked is non zero
	err := s.emissionsKeeper.AddStake(
		s.ctx,
		0,
		sdk.AccAddress(s.PKS[0].Address()),
		cosmosMath.NewUintFromString("40000000000000000000"),
	)
	s.Require().NoError(err)

	// mint enough tokens so that the circulating supply is non zero
	// mint them to the ecosystem account to simulate paying for inference requests
	spareCoins, ok := cosmosMath.NewIntFromString("500000000000000000000000000")
	s.Require().True(ok)
	err = s.bankKeeper.MintCoins(
		s.ctx,
		types.EcosystemModuleName,
		sdk.NewCoins(
			sdk.NewCoin(
				sdk.DefaultBondDenom,
				spareCoins,
			),
		),
	)
	s.Require().NoError(err)
	ecosystemBalBefore := s.bankKeeper.GetBalance(s.ctx, ecosystemAddress, sdk.DefaultBondDenom)

	tokenSupplyBefore := s.bankKeeper.GetSupply(s.ctx, sdk.DefaultBondDenom)

	err = mint.BeginBlocker(s.ctx, s.mintKeeper)
	s.Require().NoError(err)

	feeCollectorBalAfter := s.bankKeeper.GetBalance(s.ctx, feeCollectorAddress, sdk.DefaultBondDenom)
	alloraRewardsBalAfter := s.bankKeeper.GetBalance(s.ctx, alloraRewardsAddress, sdk.DefaultBondDenom)
	ecosystemBalAfter := s.bankKeeper.GetBalance(s.ctx, ecosystemAddress, sdk.DefaultBondDenom)
	tokenSupplyAfter := s.bankKeeper.GetSupply(s.ctx, sdk.DefaultBondDenom)

	// Check that:
	// The token supply didn't change (no new tokens were minted!)
	// the ecosystem account balance went DOWN (ecosystem paid to the rewards account)
	// the fee collector account balance went UP (fee collector received the fees)
	// the allora rewards account balance went UP (allora rewards account received the rewards)
	s.Require().Equal(tokenSupplyBefore, tokenSupplyAfter)
	s.Require().True(
		ecosystemBalBefore.Amount.GT(ecosystemBalAfter.Amount),
		"Ecosystem balance should go down when minting tokens to pay for inference requests: %s > %s",
		ecosystemBalBefore.Amount.String(),
		ecosystemBalAfter.Amount.String(),
	)
	s.Require().True(
		feeCollectorBalBefore.Amount.LT(feeCollectorBalAfter.Amount),
		"Fee collector balance should go up when minting tokens to pay for inference requests: %s < %s",
		feeCollectorBalBefore.String(),
		feeCollectorBalAfter.String(),
	)
	s.Require().True(
		alloraRewardsBalBefore.Amount.LT(alloraRewardsBalAfter.Amount),
		"Allora rewards balance should go up when minting tokens to pay for inference requests: %s < %s",
		alloraRewardsBalBefore.String(),
		alloraRewardsBalAfter.String(),
	)
}

func (s *MintModuleTestSuite) TestTokensAreMintedIfInferenceRequestFeesNotEnoughToCoverInflation() {
	feeCollectorAddress := s.accountKeeper.GetModuleAddress("fee_collector")
	alloraRewardsAddress := s.accountKeeper.GetModuleAddress(emissions.AlloraRewardsAccountName)
	ecosystemAddress := s.accountKeeper.GetModuleAddress(types.EcosystemModuleName)
	feeCollectorBalBefore := s.bankKeeper.GetBalance(s.ctx, feeCollectorAddress, sdk.DefaultBondDenom)
	alloraRewardsBalBefore := s.bankKeeper.GetBalance(s.ctx, alloraRewardsAddress, sdk.DefaultBondDenom)
	ecosystemBalBefore := s.bankKeeper.GetBalance(s.ctx, ecosystemAddress, sdk.DefaultBondDenom)
	ecosystemTokensMintedBefore, err := s.mintKeeper.EcosystemTokensMinted.Get(s.ctx)
	s.Require().NoError(err)
	s.ctx = s.ctx.WithBlockHeight(1)
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
		"third_party",
		sdk.NewCoins(
			sdk.NewCoin(
				sdk.DefaultBondDenom,
				spareCoins,
			),
		),
	)
	s.Require().NoError(err)

	tokenSupplyBefore := s.bankKeeper.GetSupply(s.ctx, sdk.DefaultBondDenom)

	err = mint.BeginBlocker(s.ctx, s.mintKeeper)
	s.Require().NoError(err)

	feeCollectorBalAfter := s.bankKeeper.GetBalance(s.ctx, feeCollectorAddress, sdk.DefaultBondDenom)
	alloraRewardsBalAfter := s.bankKeeper.GetBalance(s.ctx, alloraRewardsAddress, sdk.DefaultBondDenom)
	ecosystemBalAfter := s.bankKeeper.GetBalance(s.ctx, ecosystemAddress, sdk.DefaultBondDenom)
	tokenSupplyAfter := s.bankKeeper.GetSupply(s.ctx, sdk.DefaultBondDenom)
	ecosystemTokensMintedAfter, err := s.mintKeeper.EcosystemTokensMinted.Get(s.ctx)
	s.Require().NoError(err)

	// Check that:
	// The token supply went up (new tokens were minted!)
	// the ecosystem account balance stayed the same (should have been zero and the start and zero after!)
	// ecosystem tokens minted went up (we minted tokens to pay for inference requests)
	// the fee collector account balance went UP (fee collector received the fees)
	// the allora rewards account balance went UP (allora rewards account received the rewards)
	s.Require().True(
		tokenSupplyBefore.Amount.LT(tokenSupplyAfter.Amount),
		"Token supply should go up when minting tokens as inflationary rewards: %s > %s",
		tokenSupplyBefore.Amount.String(),
		tokenSupplyAfter.Amount.String(),
	)
	s.Require().True(
		ecosystemBalBefore.Amount.Equal(ecosystemBalAfter.Amount),
		"Ecosystem bal zero before and after: before we never gave it money and after it paid out the rewards: %s > %s",
		ecosystemBalBefore.Amount.String(),
		ecosystemBalAfter.Amount.String(),
	)
	s.Require().True(
		ecosystemBalAfter.Amount.Equal(cosmosMath.ZeroInt()),
		"Ecosystem bal zero before and after: before we never gave it money and after it paid out the rewards: %s != 0",
		ecosystemBalAfter.Amount.String(),
	)
	s.Require().True(
		ecosystemTokensMintedBefore.LT(ecosystemTokensMintedAfter),
		"Ecosystem tokens minted should go up when minting tokens to pay for inference requests: %s < %s",
		ecosystemTokensMintedBefore.String(),
		ecosystemTokensMintedAfter.String(),
	)
	s.Require().True(
		feeCollectorBalBefore.Amount.LT(feeCollectorBalAfter.Amount),
		"Fee collector balance should go up when minting tokens to pay for inference requests: %s < %s",
		feeCollectorBalBefore.String(),
		feeCollectorBalAfter.String(),
	)
	s.Require().True(
		alloraRewardsBalBefore.Amount.LT(alloraRewardsBalAfter.Amount),
		"Allora rewards balance should go up when minting tokens to pay for inference requests: %s < %s",
		alloraRewardsBalBefore.String(),
		alloraRewardsBalAfter.String(),
	)
}

func (s *MintModuleTestSuite) TestInflationRateAsMorePeopleStakeGoesUpButPerUnitStakeGoesDown() {
	s.ctx = s.ctx.WithBlockHeight(1)

	// stake enough tokens so that the networkStaked is non zero
	changeInAmountStakedBefore := cosmosMath.NewUintFromString("40000000000000000000")
	err := s.emissionsKeeper.AddStake(
		s.ctx,
		0,
		sdk.AccAddress(s.PKS[0].Address()),
		changeInAmountStakedBefore,
	)
	s.Require().NoError(err)

	// mint enough tokens so that the circulating supply is non zero
	spareCoinAmount, ok := cosmosMath.NewIntFromString("500000000000000000000000000")
	s.Require().True(ok)
	spareCoins := sdk.NewCoins(
		sdk.NewCoin(
			sdk.DefaultBondDenom,
			spareCoinAmount,
		),
	)
	err = s.bankKeeper.MintCoins(
		s.ctx,
		"mint",
		spareCoins,
	)
	s.Require().NoError(err)
	s.bankKeeper.SendCoinsFromModuleToAccount(
		s.ctx,
		"mint",
		sdk.AccAddress(s.PKS[2].Address()),
		spareCoins,
	)

	tokenSupplyZero := s.bankKeeper.GetSupply(s.ctx, sdk.DefaultBondDenom)
	ecosystemTokensMintedZero, err := s.mintKeeper.EcosystemTokensMinted.Get(s.ctx)
	s.Require().NoError(err)
	// do the first inflation calculation
	err = mint.BeginBlocker(s.ctx, s.mintKeeper)
	s.Require().NoError(err)

	ecosystemTokensMintedBefore, err := s.mintKeeper.EcosystemTokensMinted.Get(s.ctx)
	s.Require().NoError(err)
	tokenSupplyBefore := s.bankKeeper.GetSupply(s.ctx, sdk.DefaultBondDenom)

	// now have someone come and stake,
	// then move to the blockheight where we calculate inflation again
	changeInAmounStakedAfter := cosmosMath.NewUintFromString("800000000000000000000")
	err = s.emissionsKeeper.AddStake(
		s.ctx,
		0,
		sdk.AccAddress(s.PKS[1].Address()),
		changeInAmounStakedAfter,
	)
	s.Require().NoError(err)

	mintParams, err := s.mintKeeper.GetParams(s.ctx)
	s.Require().NoError(err)
	emissionRateUpdateCadence := mintParams.BlocksPerMonth / mintParams.EmissionCalibrationsTimestepPerMonth
	s.ctx = s.ctx.WithBlockHeight(int64(emissionRateUpdateCadence + 1))

	err = mint.BeginBlocker(s.ctx, s.mintKeeper)
	s.Require().NoError(err)

	tokenSupplyAfter := s.bankKeeper.GetSupply(s.ctx, sdk.DefaultBondDenom)
	ecosystemTokensMintedAfter, err := s.mintKeeper.EcosystemTokensMinted.Get(s.ctx)
	s.Require().NoError(err)

	tokenSupplyDelta1 := tokenSupplyBefore.Amount.Sub(tokenSupplyZero.Amount)
	tokenSupplyDelta2 := tokenSupplyAfter.Amount.Sub(tokenSupplyBefore.Amount)

	tokenSupplyDelta1Dec, err := alloraMath.NewDecFromSdkInt(tokenSupplyDelta1)
	s.Require().NoError(err)
	changeInAmountStakedBeforeDec, err := alloraMath.NewDecFromSdkUint(changeInAmountStakedBefore)
	s.Require().NoError(err)
	tokenSupplyPerAmountStaked1, err := tokenSupplyDelta1Dec.Quo(changeInAmountStakedBeforeDec)
	s.Require().NoError(err)
	tokenSupplyDelta2Dec, err := alloraMath.NewDecFromSdkInt(tokenSupplyDelta2)
	s.Require().NoError(err)
	changeInAmountStakedAfterDec, err := alloraMath.NewDecFromSdkUint(changeInAmounStakedAfter)
	s.Require().NoError(err)
	tokenSupplyPerAmountStaked2, err := tokenSupplyDelta2Dec.Quo(changeInAmountStakedAfterDec)
	s.Require().NoError(err)

	ecosystemTokensMintedDelta1 := ecosystemTokensMintedBefore.Sub(ecosystemTokensMintedZero)
	ecosystemTokensMintedDelta2 := ecosystemTokensMintedAfter.Sub(ecosystemTokensMintedBefore)

	ecosystemTokenMintedDelta1Dec, err := alloraMath.NewDecFromSdkInt(ecosystemTokensMintedDelta1)
	s.Require().NoError(err)
	ecosystemTokensMintedPerAmountStaked1, err := ecosystemTokenMintedDelta1Dec.Quo(changeInAmountStakedBeforeDec)
	s.Require().NoError(err)
	ecosystemTokenMintedDelta2Dec, err := alloraMath.NewDecFromSdkInt(ecosystemTokensMintedDelta2)
	s.Require().NoError(err)
	ecosystemTokensMintedPerAmountStaked2, err := ecosystemTokenMintedDelta2Dec.Quo(changeInAmountStakedAfterDec)
	s.Require().NoError(err)

	// Check that:
	// the amount of tokens we minted was greater than the first time
	// but the amount of tokens minted PER amount of tokens staked was less
	// i.e. each additional staked token earned less minted token or caused less inflation
	s.Require().True(
		tokenSupplyDelta2.GT(tokenSupplyDelta1),
		"More stakers more inflation: %s > %s",
		tokenSupplyDelta2.String(),
		tokenSupplyDelta1.String(),
	)
	s.Require().True(
		ecosystemTokensMintedDelta2.GT(ecosystemTokensMintedDelta1),
		"Ecosystem tokens minted more stakers more inflation: %s > %s",
		ecosystemTokensMintedDelta2.String(),
		ecosystemTokensMintedDelta1.String(),
	)
	s.Require().True(
		tokenSupplyPerAmountStaked2.Lt(tokenSupplyPerAmountStaked1),
		"Token supply per amount staked should go down when more people stake: %s < %s",
		tokenSupplyPerAmountStaked2.String(),
		tokenSupplyPerAmountStaked1.String(),
	)
	s.Require().True(
		ecosystemTokensMintedPerAmountStaked2.Lt(ecosystemTokensMintedPerAmountStaked1),
		"Ecosystem tokens minted per amount staked should go down when more people stake: %s < %s",
		ecosystemTokensMintedPerAmountStaked2.String(),
		ecosystemTokensMintedPerAmountStaked1.String(),
	)
}
