package mint_test

import (
	"math/big"
	"testing"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	cosmosMath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"

	"github.com/allora-network/allora-chain/x/mint/keeper"
	mint "github.com/allora-network/allora-chain/x/mint/module"
	"github.com/allora-network/allora-chain/x/mint/types"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
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

	emissionskeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	emissions "github.com/allora-network/allora-chain/x/emissions/module"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

const (
	multiPerm  = "multiple permissions account"
	randomPerm = "random permission"
	thirdParty = "third_party"
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
	emissionsModule emissions.AppModule
	addrs           []sdk.AccAddress
	addrsStr        []string
	codec           codec.Codec
}

// SetupTest setups a new test, to be run before each test case
func (s *MintModuleTestSuite) SetupTest() {
	sdk.DefaultBondDenom = params.DefaultBondDenom
	countAddrs := 3
	var addrs = make([]sdk.AccAddress, countAddrs)
	var addrsStr = make([]string, countAddrs)
	for i := 0; i < countAddrs; i++ {
		privKey := secp256k1.GenPrivKey()
		addrs[i] = sdk.AccAddress(privKey.PubKey().Address())
		addrsStr[i] = addrs[i].String()
	}
	s.addrs = addrs
	s.addrsStr = addrsStr
	key := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	encCfg := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, staking.AppModuleBasic{}, bank.AppModuleBasic{}, mint.AppModuleBasic{})
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithHeaderInfo(header.Info{
		Height:  1,
		Hash:    []byte("test"),
		ChainID: "localnet",
		AppHash: []byte("test"),
		Time:    time.Now(),
	})

	maccPerms := map[string][]string{
		"fee_collector":                         nil,
		thirdParty:                              {"minter"},
		"ecosystem":                             {"burner", "minter", "staking"},
		"mint":                                  {"minter"},
		emissionstypes.AlloraRewardsAccountName: {"minter"},
		emissionstypes.AlloraPendingRewardForDelegatorAccountName: nil,
		emissionstypes.AlloraStakingAccountName:                   {"burner", "minter", "staking"},
		"bonded_tokens_pool":                                      {"burner", "staking"},
		"not_bonded_tokens_pool":                                  {"burner", "staking"},
		multiPerm:                                                 {"burner", "minter", "staking"},
		randomPerm:                                                {"random"},
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
	err := stakingKeeper.SetParams(ctx, stakingtypes.Params{
		UnbondingTime:     60,
		MaxValidators:     100,
		MaxEntries:        7,
		HistoricalEntries: 1000,
		BondDenom:         sdk.DefaultBondDenom,
		MinCommissionRate: cosmosMath.LegacyNewDecWithPrec(1, 2),
	})
	s.Require().NoError(err)
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
	)

	s.ctx = ctx
	s.accountKeeper = accountKeeper
	s.bankKeeper = bankKeeper
	s.stakingKeeper = stakingKeeper
	s.emissionsKeeper = emissionsKeeper
	s.mintKeeper = mintKeeper
	s.codec = encCfg.Codec

	emissionsModule := emissions.NewAppModule(encCfg.Codec, s.emissionsKeeper)
	emissionsDefaultGenesis := emissionsModule.DefaultGenesis(encCfg.Codec)
	emissionsModule.InitGenesis(ctx, encCfg.Codec, emissionsDefaultGenesis)
	s.emissionsModule = emissionsModule

	mintAppModule := mint.NewAppModule(encCfg.Codec, s.mintKeeper, s.accountKeeper)
	defaultGenesis := mintAppModule.DefaultGenesis(encCfg.Codec)
	mintAppModule.InitGenesis(ctx, encCfg.Codec, defaultGenesis)
	s.appModule = mintAppModule
}

func TestMintModuleTestSuite(t *testing.T) {
	suite.Run(t, new(MintModuleTestSuite))
}

func (s *MintModuleTestSuite) TestTotalStakeGoUpTargetEmissionPerUnitStakeGoDown() {
	topicId := uint64(1)
	params, err := s.mintKeeper.GetParams(s.ctx)
	s.Require().NoError(err)
	ecosystemMintSupplyRemaining, err := s.mintKeeper.GetEcosystemMintSupplyRemaining(s.ctx, params)
	s.Require().NoError(err)
	// stake enough tokens so that the networkStaked is non zero
	stake, ok := cosmosMath.NewIntFromString("300000000000000000000000000")
	s.Require().True(ok)
	err = s.emissionsKeeper.GetStakingKeeper().AddReputerStake(
		s.ctx,
		topicId,
		s.addrsStr[0],
		stake,
	)
	s.Require().NoError(err)

	// mint enough tokens so that the circulating supply is non zero
	spareCoins, ok := cosmosMath.NewIntFromString("1000000000000000000000000000")
	s.Require().True(ok)
	err = s.bankKeeper.MintCoins(
		s.ctx,
		thirdParty,
		sdk.NewCoins(
			sdk.NewCoin(
				params.MintDenom,
				spareCoins,
			),
		),
	)
	s.Require().NoError(err)

	emissionsParams, err := s.emissionsKeeper.GetParamsKeeper().GetParams(s.ctx)
	s.Require().NoError(err)
	blocksPerMonth := emissionsParams.BlocksPerMonth
	monthsUnlocked := cosmosMath.NewIntFromUint64(uint64(0))

	_, emissionPerUnitStakedTokenBefore, updatedMonthsUnlocked, err := keeper.GetEmissionPerMonth(
		s.ctx,
		s.mintKeeper,
		uint64(s.ctx.BlockHeight()),
		blocksPerMonth,
		params,
		cosmosMath.ZeroInt(),
		ecosystemMintSupplyRemaining,
		cosmosMath.LegacyMustNewDecFromStr("0.25"),
		monthsUnlocked,
	)
	s.Require().NoError(err)
	s.Require().Equal(monthsUnlocked, updatedMonthsUnlocked)

	stake, ok = cosmosMath.NewIntFromString("400000000000000000000000000")
	s.Require().True(ok)
	// ok now add some stake
	err = s.emissionsKeeper.GetStakingKeeper().AddReputerStake(
		s.ctx,
		topicId,
		s.addrsStr[0],
		stake,
	)
	s.Require().NoError(err)

	_, emissionPerUnitStakedTokenAfter, updatedMonthsUnlocked, err := keeper.GetEmissionPerMonth(
		s.ctx,
		s.mintKeeper,
		uint64(s.ctx.BlockHeight()),
		blocksPerMonth,
		params,
		cosmosMath.ZeroInt(),
		ecosystemMintSupplyRemaining,
		cosmosMath.LegacyMustNewDecFromStr("0.25"),
		monthsUnlocked,
	)
	s.Require().NoError(err)
	s.Require().Equal(monthsUnlocked, updatedMonthsUnlocked)

	s.Require().True(
		emissionPerUnitStakedTokenBefore.GT(emissionPerUnitStakedTokenAfter),
		"Emission per unit staked token should go down when total stake goes up all else equal: %s > %s",
		emissionPerUnitStakedTokenBefore.String(),
		emissionPerUnitStakedTokenAfter.String(),
	)
}

func (s *MintModuleTestSuite) TestEcosystemMintableRemainingGoDownTargetEmissionPerUnitStakeTokenGoDown() {
	var fEmission = types.DefaultParams().FEmission
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
	topicId := uint64(1)
	feeCollectorAddress := s.accountKeeper.GetModuleAddress("fee_collector")
	alloraRewardsAddress := s.accountKeeper.GetModuleAddress(emissionstypes.AlloraRewardsAccountName)
	ecosystemAddress := s.accountKeeper.GetModuleAddress(types.EcosystemModuleName)
	feeCollectorBalBefore := s.bankKeeper.GetBalance(s.ctx, feeCollectorAddress, sdk.DefaultBondDenom)
	alloraRewardsBalBefore := s.bankKeeper.GetBalance(s.ctx, alloraRewardsAddress, sdk.DefaultBondDenom)
	s.ctx = s.ctx.WithBlockHeight(1)
	// stake enough tokens so that the networkStaked is non zero
	stake, ok := cosmosMath.NewIntFromString("40000000000000000000")
	s.Require().True(ok)
	err := s.emissionsKeeper.GetStakingKeeper().AddReputerStake(
		s.ctx,
		topicId,
		s.addrsStr[0],
		stake,
	)
	s.Require().NoError(err)

	// mint enough tokens so that the circulating supply is non zero
	// mint them to the ecosystem account to simulate paying for inference requests
	spareCoins, ok := cosmosMath.NewIntFromString("100000000000000000000000")
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
	s.Require().True(tokenSupplyBefore.Amount.Equal(tokenSupplyAfter.Amount),
		"Token supply should not change when minting tokens as inflationary rewards: %s == %s",
		tokenSupplyBefore.Amount.String(),
		tokenSupplyAfter.Amount.String(),
	)
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
	topicId := uint64(1)
	feeCollectorAddress := s.accountKeeper.GetModuleAddress("fee_collector")
	alloraRewardsAddress := s.accountKeeper.GetModuleAddress(emissionstypes.AlloraRewardsAccountName)
	ecosystemAddress := s.accountKeeper.GetModuleAddress(types.EcosystemModuleName)
	feeCollectorBalBefore := s.bankKeeper.GetBalance(s.ctx, feeCollectorAddress, sdk.DefaultBondDenom)
	alloraRewardsBalBefore := s.bankKeeper.GetBalance(s.ctx, alloraRewardsAddress, sdk.DefaultBondDenom)
	ecosystemBalBefore := s.bankKeeper.GetBalance(s.ctx, ecosystemAddress, sdk.DefaultBondDenom)
	ecosystemTokensMintedBefore, err := s.mintKeeper.EcosystemTokensMinted.Get(s.ctx)
	s.Require().NoError(err)
	s.ctx = s.ctx.WithBlockHeight(1)
	// stake enough tokens so that the networkStaked is non zero
	stake, ok := cosmosMath.NewIntFromString("40000000000000000000")
	s.Require().True(ok)
	err = s.emissionsKeeper.GetStakingKeeper().AddReputerStake(
		s.ctx,
		topicId,
		s.addrsStr[0],
		stake,
	)
	s.Require().NoError(err)

	// mint enough tokens so that the circulating supply is non zero
	spareCoins, ok := cosmosMath.NewIntFromString("500000000000000000000000000")
	s.Require().True(ok)
	err = s.bankKeeper.MintCoins(
		s.ctx,
		thirdParty,
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

func (s *MintModuleTestSuite) TestNotEnoughTokensToMintToCoverInflation() {
	topicId := uint64(1)
	feeCollectorAddress := s.accountKeeper.GetModuleAddress("fee_collector")
	alloraRewardsAddress := s.accountKeeper.GetModuleAddress(emissionstypes.AlloraRewardsAccountName)
	ecosystemAddress := s.accountKeeper.GetModuleAddress(types.EcosystemModuleName)
	feeCollectorBalBefore := s.bankKeeper.GetBalance(s.ctx, feeCollectorAddress, sdk.DefaultBondDenom)
	alloraRewardsBalBefore := s.bankKeeper.GetBalance(s.ctx, alloraRewardsAddress, sdk.DefaultBondDenom)
	ecosystemBalBefore := s.bankKeeper.GetBalance(s.ctx, ecosystemAddress, sdk.DefaultBondDenom)

	// set almost all ecosystem tokens as minted to reach the limit
	ecosystemTokensMintedBefore, _ := cosmosMath.NewIntFromString("359499999999999990000000000")
	s.Require().NoError(s.mintKeeper.AddEcosystemTokensMinted(s.ctx, ecosystemTokensMintedBefore))
	prevEmission := cosmosMath.NewInt(1000000000000)
	s.Require().NoError(s.mintKeeper.PreviousBlockEmission.Set(s.ctx, prevEmission))

	s.ctx = s.ctx.WithBlockHeight(2)
	// stake enough tokens so that the networkStaked is non zero
	stake, ok := cosmosMath.NewIntFromString("40000000000000000000")
	s.Require().True(ok)
	err := s.emissionsKeeper.GetStakingKeeper().AddReputerStake(
		s.ctx,
		topicId,
		s.addrsStr[0],
		stake,
	)
	s.Require().NoError(err)

	// mint enough tokens so that the circulating supply is non zero
	spareCoins, ok := cosmosMath.NewIntFromString("500000000000000000000000000")
	s.Require().True(ok)
	err = s.bankKeeper.MintCoins(
		s.ctx,
		thirdParty,
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
	// the rewards should be less than the previous emission
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
	s.Require().NoError(err)
	alloraRewards := alloraRewardsBalAfter.Amount.Sub(alloraRewardsBalBefore.Amount)
	valRewards := feeCollectorBalAfter.Amount.Sub(feeCollectorBalBefore.Amount)
	rewards := alloraRewards.Add(valRewards)
	s.Require().True(
		rewards.LT(prevEmission),
		"Rewards should be less than the previous emission: %s < %s",
		rewards.String(),
		prevEmission.String(),
	)
}

func (s *MintModuleTestSuite) TestInflationRateAsMorePeopleStakeGoesUp() {
	s.ctx = s.ctx.WithBlockHeight(1)

	topicId := uint64(1)
	// stake enough tokens so that the networkStaked is non zero
	changeInAmountStakedBefore, ok := cosmosMath.NewIntFromString("300000000000000000000000000")
	s.Require().True(ok)
	err := s.emissionsKeeper.GetStakingKeeper().AddReputerStake(
		s.ctx,
		topicId,
		s.addrsStr[0],
		changeInAmountStakedBefore,
	)
	s.Require().NoError(err)

	// mint enough tokens so that the circulating supply is non zero
	spareCoinAmount, ok := cosmosMath.NewIntFromString("1000000000000000000000000000")
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
	err = s.bankKeeper.SendCoinsFromModuleToAccount(
		s.ctx,
		"mint",
		s.addrs[2],
		spareCoins,
	)
	s.Require().NoError(err)

	tokenSupplyZero := s.bankKeeper.GetSupply(s.ctx, sdk.DefaultBondDenom)
	ecosystemTokensMintedZero, err := s.mintKeeper.EcosystemTokensMinted.Get(s.ctx)
	s.Require().NoError(err)
	// do the first inflation calculation
	err = mint.BeginBlocker(s.ctx, s.mintKeeper)
	s.Require().NoError(err)

	ecosystemTokensMintedBefore, err := s.mintKeeper.EcosystemTokensMinted.Get(s.ctx)
	s.Require().NoError(err)
	tokenSupplyBefore := s.bankKeeper.GetSupply(s.ctx, sdk.DefaultBondDenom)
	s.Require().True(
		tokenSupplyBefore.Amount.GT(tokenSupplyZero.Amount),
		"Token supply should go up when minting tokens as inflationary rewards: %s > %s",
		tokenSupplyBefore,
		tokenSupplyZero,
	)

	// now have someone come and stake,
	// then move to the blockheight where we calculate inflation again
	changeInAmounStakedAfter, ok := cosmosMath.NewIntFromString("400000000000000000000000000")
	s.Require().True(ok)
	err = s.emissionsKeeper.GetStakingKeeper().AddReputerStake(
		s.ctx,
		topicId,
		s.addrsStr[1],
		changeInAmounStakedAfter,
	)
	s.Require().NoError(err)

	emissionsParams, err := s.emissionsKeeper.GetParamsKeeper().GetParams(s.ctx)
	s.Require().NoError(err)
	blocksPerMonth := emissionsParams.BlocksPerMonth
	blocks := new(big.Int).SetUint64(blocksPerMonth)
	blocks.Add(blocks, big.NewInt(1))
	s.ctx = s.ctx.WithBlockHeight(blocks.Int64())

	err = mint.BeginBlocker(s.ctx, s.mintKeeper)
	s.Require().NoError(err)

	tokenSupplyAfter := s.bankKeeper.GetSupply(s.ctx, sdk.DefaultBondDenom)
	ecosystemTokensMintedAfter, err := s.mintKeeper.EcosystemTokensMinted.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().True(ecosystemTokensMintedAfter.GT(ecosystemTokensMintedZero))

	tokenSupplyDelta1 := tokenSupplyBefore.Amount.Sub(tokenSupplyZero.Amount)
	s.Require().True(tokenSupplyDelta1.GT(cosmosMath.ZeroInt()))
	tokenSupplyDelta2 := tokenSupplyAfter.Amount.Sub(tokenSupplyBefore.Amount)
	s.Require().True(tokenSupplyDelta2.GT(cosmosMath.ZeroInt()))

	ecosystemTokensMintedDelta1 := ecosystemTokensMintedBefore.Sub(ecosystemTokensMintedZero)
	ecosystemTokensMintedDelta2 := ecosystemTokensMintedAfter.Sub(ecosystemTokensMintedBefore)

	// Check that the amount of tokens we minted was greater than the first time
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
}

func (s *MintModuleTestSuite) TestEcosystemRefundReducesMintingInSubsequentBlock() {
	s.ctx = s.ctx.WithBlockHeight(1)
	topicId := uint64(1)
	ecosystemAddress := s.accountKeeper.GetModuleAddress(types.EcosystemModuleName)
	refundSenderAddr := s.addrs[2] // Use one of the pre-generated addresses

	// 1. Initial setup: stake, initial supply (mint to a regular account)
	stake, ok := cosmosMath.NewIntFromString("40000000000000000000") // Sufficient stake
	s.Require().True(ok)
	err := s.emissionsKeeper.GetStakingKeeper().AddReputerStake(s.ctx, topicId, s.addrsStr[0], stake)
	s.Require().NoError(err)

	initialSupply, ok := cosmosMath.NewIntFromString("500000000000000000000000000")
	s.Require().True(ok)
	err = s.bankKeeper.MintCoins(s.ctx, "mint", sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, initialSupply)))
	s.Require().NoError(err)
	err = s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, "mint", refundSenderAddr, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, initialSupply)))
	s.Require().NoError(err)

	// Ensure ecosystem starts empty
	ecosystemBalStart := s.bankKeeper.GetBalance(s.ctx, ecosystemAddress, sdk.DefaultBondDenom)
	s.Require().True(ecosystemBalStart.Amount.IsZero(), "Ecosystem account should start empty")
	ecosystemTokensMintedStart, err := s.mintKeeper.EcosystemTokensMinted.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().True(ecosystemTokensMintedStart.IsZero(), "Ecosystem tokens minted should start at zero")
	tokenSupplyStart := s.bankKeeper.GetSupply(s.ctx, sdk.DefaultBondDenom)

	// 2. Run BeginBlocker at block 1 - should mint tokens
	err = mint.BeginBlocker(s.ctx, s.mintKeeper)
	s.Require().NoError(err)

	tokenSupplyAfterBlock1 := s.bankKeeper.GetSupply(s.ctx, sdk.DefaultBondDenom)
	ecosystemTokensMintedAfterBlock1, err := s.mintKeeper.EcosystemTokensMinted.Get(s.ctx)
	s.Require().NoError(err)
	emissionBlock1, err := s.mintKeeper.PreviousBlockEmission.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().True(emissionBlock1.GT(cosmosMath.ZeroInt()), "Emission for block 1 should be positive")

	mintedInBlock1 := tokenSupplyAfterBlock1.Amount.Sub(tokenSupplyStart.Amount)
	s.Require().True(mintedInBlock1.GT(cosmosMath.ZeroInt()), "Tokens should have been minted in block 1")
	s.Require().True(mintedInBlock1.Equal(ecosystemTokensMintedAfterBlock1.Sub(ecosystemTokensMintedStart)), "Minted tokens should match increase in EcosystemTokensMinted")
	s.Require().True(mintedInBlock1.Equal(emissionBlock1), "Minted tokens in block 1 should equal the calculated emission when ecosystem is empty")

	// 3. Simulate refund to ecosystem account (less than the emission amount)
	// This simulates funds arriving in the ecosystem account before BeginBlocker runs.
	// In a real scenario, this represents collected fees or undistributed rewards
	// being returned from another module (like emissions returning funds from AlloraRewardsAccountName).
	// The latter is the case in this test.
	// For this test's purpose, sending from a regular account isolates the mint module's
	// reaction to a pre-existing balance.
	refundAmount := emissionBlock1.QuoRaw(2) // Refund half the emission amount
	s.Require().True(refundAmount.GT(cosmosMath.ZeroInt()), "Refund amount must be positive")
	refundCoins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, refundAmount))
	// Send coins from the user account to the ecosystem module account using bankKeeper
	err = s.bankKeeper.SendCoinsFromAccountToModule(s.ctx, refundSenderAddr, types.EcosystemModuleName, refundCoins)
	s.Require().NoError(err)
	ecosystemBalAfterRefund := s.bankKeeper.GetBalance(s.ctx, ecosystemAddress, sdk.DefaultBondDenom)
	s.Require().True(ecosystemBalAfterRefund.Amount.Equal(refundAmount), "Ecosystem balance should equal refund amount")

	// 4. Run BeginBlocker at block 2
	s.ctx = s.ctx.WithBlockHeight(2) // Advance block height, avoid recalculation
	err = mint.BeginBlocker(s.ctx, s.mintKeeper)
	s.Require().NoError(err)

	// 5. Check results after block 2
	tokenSupplyAfterBlock2 := s.bankKeeper.GetSupply(s.ctx, sdk.DefaultBondDenom)
	ecosystemTokensMintedAfterBlock2, err := s.mintKeeper.EcosystemTokensMinted.Get(s.ctx)
	s.Require().NoError(err)
	emissionBlock2, err := s.mintKeeper.PreviousBlockEmission.Get(s.ctx) // Should be same as block 1
	s.Require().NoError(err)
	s.Require().True(emissionBlock2.Equal(emissionBlock1), "Emission should not change between block 1 and 2")

	// Calculate changes in block 2
	mintedInBlock2 := tokenSupplyAfterBlock2.Amount.Sub(tokenSupplyAfterBlock1.Amount)
	ecosystemMintedInBlock2 := ecosystemTokensMintedAfterBlock2.Sub(ecosystemTokensMintedAfterBlock1)

	// 6. Verify less minting occurred due to refund (Now these should be equal!)
	s.Require().True(mintedInBlock2.Equal(ecosystemMintedInBlock2), "Minted supply change should equal EcosystemTokensMinted change in block 2")
	expectedMintAmountBlock2 := emissionBlock2.Sub(refundAmount)
	s.Require().True(expectedMintAmountBlock2.GT(cosmosMath.ZeroInt()), "Expected mint amount should still be positive")
	s.Require().True(
		mintedInBlock2.Equal(expectedMintAmountBlock2),
		"Actual minted tokens in block 2 (%s) should equal emission (%s) minus refund (%s)",
		mintedInBlock2.String(),
		emissionBlock2.String(),
		refundAmount.String(),
	)
	s.Require().True(
		mintedInBlock2.LT(emissionBlock2),
		"Minted tokens in block 2 (%s) should be less than the total emission (%s) because of the refund",
		mintedInBlock2.String(),
		emissionBlock2.String(),
	)

	// Check final ecosystem balance is zero (refund + minted tokens were paid out)
	ecosystemBalFinal := s.bankKeeper.GetBalance(s.ctx, ecosystemAddress, sdk.DefaultBondDenom)
	s.Require().True(ecosystemBalFinal.Amount.IsZero(), "Ecosystem account should be empty after paying emissions")
}

func (s *MintModuleTestSuite) TestEmissionDisabled() {
	s.ctx = s.ctx.WithBlockHeight(1)

	params, err := s.mintKeeper.GetParams(s.ctx)
	s.Require().NoError(err)
	params.EmissionEnabled = false
	err = s.mintKeeper.Params.Set(s.ctx, params)
	s.Require().NoError(err)

	tokenSupplyBefore := s.bankKeeper.GetSupply(s.ctx, sdk.DefaultBondDenom)
	// call begin blocker to simulate running the mint module
	err = mint.BeginBlocker(s.ctx, s.mintKeeper)
	s.Require().NoError(err)

	tokenSupplyAfter := s.bankKeeper.GetSupply(s.ctx, sdk.DefaultBondDenom)
	s.Require().True(tokenSupplyAfter.Equal(tokenSupplyBefore),
		"Token supply should stay the same when emission is disabled: %s != %s",
		tokenSupplyAfter.String(),
		tokenSupplyBefore.String(),
	)
	ecosystemTokensMintedAfter, err := s.mintKeeper.EcosystemTokensMinted.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().True(ecosystemTokensMintedAfter.Equal(cosmosMath.ZeroInt()),
		"Ecosystem tokens minted should be zero when emission is disabled: %s != 0",
		ecosystemTokensMintedAfter.String(),
	)
}

// Helper methods for monthly reset tests
func (s *MintModuleTestSuite) WithBlockHeight(height int64) {
	s.ctx = s.ctx.WithBlockHeight(height)
}

func (s *MintModuleTestSuite) MintTokensToModule(moduleName string, amount cosmosMath.Int) {
	coins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amount))
	err := s.bankKeeper.MintCoins(s.ctx, moduleName, coins)
	s.Require().NoError(err)
}

func (s *MintModuleTestSuite) BeginBlock() {
	err := mint.BeginBlocker(s.ctx, s.mintKeeper)
	s.Require().NoError(err)
}

func (s *MintModuleTestSuite) EndBlock() {
	err := s.emissionsModule.EndBlock(s.ctx)
	s.Require().NoError(err)
}

// TestMonthlyPercentageRewardCalculation tests that monthly rewards reset happens correctly
// in the mint BeginBlocker when blockCountSinceTGE % blocksPerMonth == 1.
// This test verifies the chain behavior: rewards accumulate over a month, then reset happens
// at the start of the next month via BeginBlocker.
func (s *MintModuleTestSuite) TestMonthlyPercentageRewardCalculation() {
	// Disable emissions for this test - we're only testing monthly reset, not emission calculations
	// The monthly reset happens before the emission check, so it will still run
	mintParams, err := s.mintKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	mintParams.EmissionEnabled = false
	err = s.mintKeeper.Params.Set(s.ctx, mintParams)
	s.Require().NoError(err)

	// 1. Setup Params
	params, err := s.emissionsKeeper.GetParamsKeeper().GetParams(s.ctx)
	s.Require().NoError(err)
	blocksPerMonth := int64(10)
	params.BlocksPerMonth = uint64(blocksPerMonth)
	err = s.emissionsKeeper.GetParamsKeeper().SetParams(s.ctx, params)
	s.Require().NoError(err)

	// 2. Fund Rewards Module (required for EndBlocker checks)
	initialRewardAmount := cosmosMath.NewInt(1000000)
	s.MintTokensToModule(emissionstypes.AlloraRewardsAccountName, initialRewardAmount)

	// 3. Define simulated reward increments per block
	reputerIncrement := cosmosMath.NewInt(10)
	topicIncrement := cosmosMath.NewInt(50)

	// 4. Simulate Block Progression and Reward Accumulation
	// Monthly rewards reset happens in mint BeginBlocker when blockCountSinceTGE % blocksPerMonth == 1
	// Since startingEmissionsBlockHeight defaults to 0, reset happens at blocks 1, 11, 21, etc.
	for i := int64(1); i <= blocksPerMonth; i++ {
		s.WithBlockHeight(i)

		// Run BeginBlocker - monthly reset happens here when i % blocksPerMonth == 1
		s.BeginBlock()

		// Run EndBlocker (simulates standard block processing)
		s.EndBlock()

		// Manually add rewards to simulate accumulation during the month
		// Skip adding rewards at block 1 since that's when the first reset happens
		if i > 1 && i%blocksPerMonth != 1 {
			err = s.emissionsKeeper.GetWeightsKeeper().AddMonthlyRewards(s.ctx, reputerIncrement, topicIncrement)
			s.Require().NoError(err, "Failed to add rewards at block %d", i)
		}
	}

	// 5. Calculate Expected Totals and Percentage *before* the reset block
	// Rewards accumulate from block 2 to block 10 (9 blocks), then reset happens at block 11
	totalReputerRewards := reputerIncrement.MulRaw(blocksPerMonth - 1)
	totalTopicRewards := topicIncrement.MulRaw(blocksPerMonth - 1)

	expectedPercentage := alloraMath.ZeroDec()
	if !totalTopicRewards.IsZero() {
		reputersDec, err := alloraMath.NewDecFromSdkInt(totalReputerRewards)
		s.Require().NoError(err)
		topicDec, err := alloraMath.NewDecFromSdkInt(totalTopicRewards)
		s.Require().NoError(err)
		expectedPercentage, err = reputersDec.Quo(topicDec)
		s.Require().NoError(err)
	}

	// Sanity check the accumulated values before the reset block
	reputerRewardsBeforeFinal, err := s.emissionsKeeper.GetWeightsKeeper().GetMonthlyReputerRewards(s.ctx)
	s.Require().NoError(err)
	s.Require().True(totalReputerRewards.Equal(reputerRewardsBeforeFinal), "Mismatch in accumulated reputer rewards before reset")
	topicRewardsBeforeFinal, err := s.emissionsKeeper.GetWeightsKeeper().GetMonthlyTopicRewards(s.ctx)
	s.Require().NoError(err)
	s.Require().True(totalTopicRewards.Equal(topicRewardsBeforeFinal), "Mismatch in accumulated topic rewards before reset")

	// 6. Trigger BeginBlocker at the start of the next month (block blocksPerMonth + 1)
	// This is when the monthly reset happens (blockCountSinceTGE % blocksPerMonth == 1)
	s.WithBlockHeight(blocksPerMonth + 1)
	s.BeginBlock()

	// 7. Verify State After BeginBlocker (monthly reset happens here)
	actualPercentageAfter, err := s.emissionsKeeper.GetScoresKeeper().GetPreviousPercentageRewardToStakedReputers(s.ctx)
	s.Require().NoError(err)
	s.T().Logf("Expected percentage %s, got %s", expectedPercentage.String(), actualPercentageAfter.String())
	s.Require().True(expectedPercentage.Equal(actualPercentageAfter), "Expected percentage %s, got %s", expectedPercentage.String(), actualPercentageAfter.String())

	// Verify counters reset by BeginBlocker
	reputerRewardsAfter, err := s.emissionsKeeper.GetWeightsKeeper().GetMonthlyReputerRewards(s.ctx)
	s.Require().NoError(err)
	s.Require().True(reputerRewardsAfter.IsZero(), "Monthly reputer rewards not reset by BeginBlocker")

	topicRewardsAfter, err := s.emissionsKeeper.GetWeightsKeeper().GetMonthlyTopicRewards(s.ctx)
	s.Require().NoError(err)
	s.Require().True(topicRewardsAfter.IsZero(), "Monthly topic rewards not reset by BeginBlocker")
}

// TestMonthlyPercentageRewardCalculation_ZeroTopicRewards tests monthly reset when there are no topic rewards.
// This verifies that the percentage calculation handles zero topic rewards correctly.
func (s *MintModuleTestSuite) TestMonthlyPercentageRewardCalculation_ZeroTopicRewards() {
	// Disable emissions for this test - we're only testing monthly reset, not emission calculations
	// The monthly reset happens before the emission check, so it will still run
	mintParams, err := s.mintKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	mintParams.EmissionEnabled = false
	err = s.mintKeeper.Params.Set(s.ctx, mintParams)
	s.Require().NoError(err)

	// 1. Setup Params
	params, err := s.emissionsKeeper.GetParamsKeeper().GetParams(s.ctx)
	s.Require().NoError(err)
	blocksPerMonth := int64(10)
	params.BlocksPerMonth = uint64(blocksPerMonth)
	err = s.emissionsKeeper.GetParamsKeeper().SetParams(s.ctx, params)
	s.Require().NoError(err)

	// 2. Fund Rewards Module (required for EndBlocker checks)
	initialRewardAmount := cosmosMath.NewInt(1000000)
	s.MintTokensToModule(emissionstypes.AlloraRewardsAccountName, initialRewardAmount)

	// 3. Ensure No Topics or Reward Activity
	// No topics created, no workers/reputers registered, no data submitted.
	// This ensures EmitRewards calculates zero rewards throughout the month.

	// 4. Simulate Block Progression up to the Monthly Boundary
	for i := int64(1); i <= blocksPerMonth; i++ {
		s.WithBlockHeight(i)

		// Run BeginBlocker - monthly reset happens here when i % blocksPerMonth == 1
		s.BeginBlock()

		// Run EndBlocker (simulates standard block processing)
		s.EndBlock()

		// Sanity check: Verify topic rewards remain zero during the month
		topicRewards, err := s.emissionsKeeper.GetWeightsKeeper().GetMonthlyTopicRewards(s.ctx)
		s.Require().NoError(err)
		s.Require().True(topicRewards.IsZero(), "Topic rewards became non-zero before month end at block %d", i)
	}

	// 5. Trigger BeginBlocker at the start of the next month (block blocksPerMonth + 1)
	s.WithBlockHeight(blocksPerMonth + 1)
	s.BeginBlock()

	// 6. Verify State After BeginBlocker
	// Verify percentage is zero due to zero topic rewards
	expectedPercentage := alloraMath.ZeroDec()
	actualPercentageAfter, err := s.emissionsKeeper.GetScoresKeeper().GetPreviousPercentageRewardToStakedReputers(s.ctx)
	s.Require().NoError(err)
	s.Require().True(expectedPercentage.Equal(actualPercentageAfter), "Expected percentage %s, got %s", expectedPercentage.String(), actualPercentageAfter.String())

	// Verify counters reset by BeginBlocker
	reputerRewardsAfter, err := s.emissionsKeeper.GetWeightsKeeper().GetMonthlyReputerRewards(s.ctx)
	s.Require().NoError(err)
	s.Require().True(reputerRewardsAfter.IsZero(), "Monthly reputer rewards not reset by BeginBlocker")

	topicRewardsAfter, err := s.emissionsKeeper.GetWeightsKeeper().GetMonthlyTopicRewards(s.ctx)
	s.Require().NoError(err)
	s.Require().True(topicRewardsAfter.IsZero(), "Monthly topic rewards not reset by BeginBlocker")
}
