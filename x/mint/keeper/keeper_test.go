package keeper_test

import (
	"errors"
	"testing"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	alloraMath "github.com/allora-network/allora-chain/math"
	alloratestutil "github.com/allora-network/allora-chain/test/testutil"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/allora-network/allora-chain/x/mint/keeper"
	mint "github.com/allora-network/allora-chain/x/mint/module"
	minttestutil "github.com/allora-network/allora-chain/x/mint/testutil"
	"github.com/allora-network/allora-chain/x/mint/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	cosmostestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
)

type MintKeeperTestSuite struct {
	suite.Suite

	mintKeeper      keeper.Keeper
	ctx             sdk.Context
	msgServer       types.MsgServiceServer
	ctrl            *gomock.Controller
	accountKeeper   *minttestutil.MockAccountKeeper
	stakingKeeper   *minttestutil.MockStakingKeeper
	bankKeeper      *minttestutil.MockBankKeeper
	emissionsKeeper *minttestutil.MockEmissionsKeeper
	adminPrivateKey secp256k1.PrivKey
	adminAddr       string
	epochGet        map[int]func(string) alloraMath.Dec
}

func TestMintKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(MintKeeperTestSuite))
}

func (s *MintKeeperTestSuite) SetupTest() {
	encCfg := moduletestutil.MakeTestEncodingConfig(mint.AppModuleBasic{})
	key := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	testCtx := cosmostestutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	s.ctx = testCtx.Ctx

	// gomock initializations
	s.ctrl = gomock.NewController(s.T())
	accountKeeper := minttestutil.NewMockAccountKeeper(s.ctrl)
	bankKeeper := minttestutil.NewMockBankKeeper(s.ctrl)
	stakingKeeper := minttestutil.NewMockStakingKeeper(s.ctrl)
	emissionsKeeper := minttestutil.NewMockEmissionsKeeper(s.ctrl)

	accountKeeper.EXPECT().GetModuleAddress(types.ModuleName).Return(sdk.AccAddress{})

	s.mintKeeper = keeper.NewKeeper(
		encCfg.Codec,
		storeService,
		stakingKeeper,
		accountKeeper,
		bankKeeper,
		emissionsKeeper,
		authtypes.FeeCollectorName,
	)
	s.stakingKeeper = stakingKeeper
	s.bankKeeper = bankKeeper
	s.emissionsKeeper = emissionsKeeper
	s.accountKeeper = accountKeeper

	// Setup a sender address
	s.adminPrivateKey = secp256k1.GenPrivKey()
	s.adminAddr = sdk.AccAddress(s.adminPrivateKey.PubKey().Address()).String()

	s.Require().Equal(testCtx.Ctx.Logger().With("module", "x/"+types.ModuleName),
		s.mintKeeper.Logger(testCtx.Ctx))

	err := s.mintKeeper.Params.Set(s.ctx, types.DefaultParams())
	s.Require().NoError(err)

	s.msgServer = keeper.NewMsgServerImpl(s.mintKeeper)
	s.epochGet = alloratestutil.GetTokenomicsSimulatorValuesGetterForEpochs()
}

func (s *MintKeeperTestSuite) TestAliasFunctions() {
	stakingTokenSupply := math.NewIntFromUint64(100000000000)
	s.stakingKeeper.EXPECT().TotalBondedTokens(s.ctx).Return(stakingTokenSupply, nil)
	tokenSupply, err := s.mintKeeper.CosmosValidatorStakedSupply(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(tokenSupply, stakingTokenSupply)

	coins := sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(1000000)))
	s.bankKeeper.EXPECT().MintCoins(s.ctx, types.ModuleName, coins).Return(nil)
	s.Require().NoError(s.mintKeeper.MintCoins(s.ctx, sdk.NewCoins()))
	s.Require().NoError(s.mintKeeper.MintCoins(s.ctx, coins))

	fees := sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(1000)))
	s.bankKeeper.EXPECT().SendCoinsFromModuleToModule(s.ctx, types.EcosystemModuleName, emissionstypes.AlloraRewardsAccountName, fees).Return(nil)
	s.emissionsKeeper.EXPECT().SetRewardCurrentBlockEmission(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.Require().NoError(s.mintKeeper.PayAlloraRewardsFromEcosystem(s.ctx, fees))
}

func (s *MintKeeperTestSuite) TestGetEmissionInfo() {
	// Setup test data
	stakingTokenSupply := math.NewIntFromUint64(1000000000000)                       // 1 trillion
	ecosystemBalance := math.NewIntFromUint64(500000000000)                          // 500 billion
	previousBlockEmission := math.NewIntFromUint64(1000000)                          // 1 million
	blocksPerMonth := uint64(432000)                                                 // ~30 days * 24 hours * 60 minutes * 30 seconds
	monthsAlreadyUnlocked := math.NewIntFromUint64(12)                               // 12 months
	reputersPercent := math.LegacyMustNewDecFromStr("0.7")                           // 70%
	previousRewardEmissionPerUnitStakedToken := math.LegacyMustNewDecFromStr("0.02") // 2%

	// Set up the context with a specific block height
	blockHeight := int64(1000000) // 1 million blocks
	testCtx := s.ctx.WithBlockHeight(blockHeight)

	// Mock all the required keeper method calls
	s.stakingKeeper.EXPECT().TotalBondedTokens(testCtx).Return(stakingTokenSupply, nil).AnyTimes()

	// Create a proper address for the ecosystem module
	ecosystemAddr := sdk.AccAddress([]byte("ecosystem"))
	s.accountKeeper.EXPECT().GetModuleAddress(types.EcosystemModuleName).Return(ecosystemAddr).AnyTimes()
	s.bankKeeper.EXPECT().GetBalance(testCtx, ecosystemAddr, "stake").Return(sdk.NewCoin("stake", ecosystemBalance)).AnyTimes()

	// Mock emissions keeper GetParams call
	emissionsParams := emissionstypes.Params{
		BlocksPerMonth: blocksPerMonth,
	}
	s.emissionsKeeper.EXPECT().GetParams(testCtx).Return(emissionsParams, nil).AnyTimes()

	// Mock emissions keeper GetTotalStake call
	s.emissionsKeeper.EXPECT().GetTotalStake(testCtx).Return(stakingTokenSupply, nil).AnyTimes()

	// Mock emissions keeper GetPreviousPercentageRewardToStakedReputers call
	// Create a decimal that represents 0.7 (70%) when converted to LegacyDec
	var reputersPercentDec alloraMath.Dec
	var err error
	reputersPercentDec, err = alloraMath.NewDecFromString("0.7")
	s.Require().NoError(err)
	s.emissionsKeeper.EXPECT().GetPreviousPercentageRewardToStakedReputers(testCtx).Return(reputersPercentDec, nil).AnyTimes()

	// Set up required state
	err = s.mintKeeper.PreviousBlockEmission.Set(testCtx, previousBlockEmission)
	s.Require().NoError(err)

	err = s.mintKeeper.MonthsUnlocked.Set(testCtx, monthsAlreadyUnlocked)
	s.Require().NoError(err)

	err = s.mintKeeper.PreviousRewardEmissionPerUnitStakedToken.Set(testCtx, previousRewardEmissionPerUnitStakedToken)
	s.Require().NoError(err)

	// Set up EcosystemTokensMinted (required by GetEcosystemMintSupplyRemaining)
	err = s.mintKeeper.EcosystemTokensMinted.Set(testCtx, math.ZeroInt())
	s.Require().NoError(err)

	// Call the function
	params, eventInfo, err := s.mintKeeper.GetEmissionInfo(testCtx)

	// Assertions
	s.Require().NoError(err)
	s.Require().NotNil(params)
	s.Require().NotNil(eventInfo)

	// Verify params
	s.Require().Equal(types.DefaultParams().MintDenom, params.MintDenom)
	s.Require().Equal(types.DefaultParams().MaxSupply, params.MaxSupply)
	s.Require().True(params.EmissionEnabled)

	// Verify event info basic fields
	s.Require().Equal(ecosystemBalance, eventInfo.EcosystemBalance)
	s.Require().Equal(previousBlockEmission, eventInfo.PreviousBlockEmission)
	s.Require().Equal(blocksPerMonth, eventInfo.BlocksPerMonth)
	// NetworkStakedTokens is the sum of cosmos validator staked + reputers staked
	expectedNetworkStakedTokens := stakingTokenSupply.Add(stakingTokenSupply) // 1 trillion + 1 trillion = 2 trillion
	s.Require().Equal(expectedNetworkStakedTokens, eventInfo.NetworkStakedTokens)
	s.Require().Equal(monthsAlreadyUnlocked, eventInfo.MonthsAlreadyUnlocked)

	// Verify calculated fields are not zero
	s.Require().NotEqual(math.ZeroInt(), eventInfo.BlockEmission)
	s.Require().NotEqual(math.ZeroInt(), eventInfo.EmissionPerMonth)

	s.Require().NotEqual(math.LegacyZeroDec(), eventInfo.EmissionPerUnitStakedToken)
	s.Require().NotEqual(math.LegacyZeroDec(), eventInfo.TargetEmissionRatePerUnitStakedToken)

	// Verify percentages
	s.Require().Equal(reputersPercent, eventInfo.ReputersPercent)

	// Verify block height calculations
	expectedLastCalculated := (uint64(blockHeight)/blocksPerMonth)*blocksPerMonth + 1
	expectedNextCalculated := expectedLastCalculated + blocksPerMonth
	s.Require().Equal(expectedLastCalculated, eventInfo.BlockHeightTargetEILastCalculated)
	s.Require().Equal(expectedNextCalculated, eventInfo.BlockHeightTargetEINextCalculated)

	// Verify supply calculations are reasonable
	s.Require().True(eventInfo.CirculatingSupply.GT(math.ZeroInt()))
	s.Require().True(eventInfo.MaxSupply.GT(math.ZeroInt()))
	s.Require().True(eventInfo.LockedVestingTokensTotal.GTE(math.ZeroInt()))
	s.Require().True(eventInfo.EcosystemLocked.GTE(math.ZeroInt()))
}

func (s *MintKeeperTestSuite) TestGetEmissionInfo_FirstMonth() {
	// Test the case where blockHeight < blocksPerMonth (first month)
	stakingTokenSupply := math.NewIntFromUint64(1000000000000)
	ecosystemBalance := math.NewIntFromUint64(500000000000)
	previousBlockEmission := math.NewIntFromUint64(1000000)
	blocksPerMonth := uint64(432000)
	monthsAlreadyUnlocked := math.NewIntFromUint64(0) // First month

	// Set up the context with a block height less than blocksPerMonth
	blockHeight := int64(100000) // Less than blocksPerMonth
	testCtx := s.ctx.WithBlockHeight(blockHeight)

	// Mock all the required keeper method calls
	s.stakingKeeper.EXPECT().TotalBondedTokens(testCtx).Return(stakingTokenSupply, nil).AnyTimes()

	// Create a proper address for the ecosystem module
	ecosystemAddr := sdk.AccAddress([]byte("ecosystem"))
	s.accountKeeper.EXPECT().GetModuleAddress(types.EcosystemModuleName).Return(ecosystemAddr).AnyTimes()
	s.bankKeeper.EXPECT().GetBalance(testCtx, ecosystemAddr, "stake").Return(sdk.NewCoin("stake", ecosystemBalance)).AnyTimes()

	// Mock emissions keeper GetParams call
	emissionsParams := emissionstypes.Params{
		BlocksPerMonth: blocksPerMonth,
	}
	s.emissionsKeeper.EXPECT().GetParams(testCtx).Return(emissionsParams, nil).AnyTimes()

	// Mock emissions keeper GetTotalStake call
	s.emissionsKeeper.EXPECT().GetTotalStake(testCtx).Return(stakingTokenSupply, nil).AnyTimes()

	// Mock emissions keeper GetPreviousPercentageRewardToStakedReputers call
	// Create a decimal that represents 0.7 (70%) when converted to LegacyDec
	var reputersPercentDec alloraMath.Dec
	var err error
	reputersPercentDec, err = alloraMath.NewDecFromString("0.7")
	s.Require().NoError(err)
	s.emissionsKeeper.EXPECT().GetPreviousPercentageRewardToStakedReputers(testCtx).Return(reputersPercentDec, nil).AnyTimes()

	// Set up required state
	err = s.mintKeeper.PreviousBlockEmission.Set(testCtx, previousBlockEmission)
	s.Require().NoError(err)

	err = s.mintKeeper.MonthsUnlocked.Set(testCtx, monthsAlreadyUnlocked)
	s.Require().NoError(err)

	// Set up EcosystemTokensMinted (required by GetEcosystemMintSupplyRemaining)
	err = s.mintKeeper.EcosystemTokensMinted.Set(testCtx, math.ZeroInt())
	s.Require().NoError(err)

	// Call the function
	params, eventInfo, err := s.mintKeeper.GetEmissionInfo(testCtx)

	// Assertions
	s.Require().NoError(err)
	s.Require().NotNil(params)
	s.Require().NotNil(eventInfo)

	// In the first month, previousRewardEmissionPerUnitStakedToken should equal targetRewardEmissionPerUnitStakedToken
	s.Require().Equal(eventInfo.TargetRewardEmissionPerUnitStakedToken, eventInfo.PreviousRewardEmissionPerUnitStakedToken)

	// Verify block height calculations for first month
	expectedLastCalculated := uint64(1) // Should be 1 for first month
	expectedNextCalculated := blocksPerMonth + 1
	s.Require().Equal(expectedLastCalculated, eventInfo.BlockHeightTargetEILastCalculated)
	s.Require().Equal(expectedNextCalculated, eventInfo.BlockHeightTargetEINextCalculated)
}

func (s *MintKeeperTestSuite) TestGetEmissionInfo_ErrorCases() {
	// Test error case: failed to get staked tokens
	ecosystemAddr := sdk.AccAddress([]byte("ecosystem"))
	s.accountKeeper.EXPECT().GetModuleAddress(types.EcosystemModuleName).Return(ecosystemAddr).AnyTimes()
	s.stakingKeeper.EXPECT().TotalBondedTokens(s.ctx).Return(math.ZeroInt(), errors.New("staking error")).AnyTimes()
	s.bankKeeper.EXPECT().GetBalance(s.ctx, ecosystemAddr, "stake").Return(sdk.NewCoin("stake", math.NewIntFromUint64(1000000))).AnyTimes()

	// Mock emissions keeper calls
	emissionsParams := emissionstypes.Params{
		BlocksPerMonth: 432000,
	}
	s.emissionsKeeper.EXPECT().GetParams(s.ctx).Return(emissionsParams, nil).AnyTimes()
	s.emissionsKeeper.EXPECT().GetTotalStake(s.ctx).Return(math.ZeroInt(), nil).AnyTimes()

	var reputersPercentDec alloraMath.Dec
	reputersPercentDec, err := alloraMath.NewDecFromString("0.7")
	s.Require().NoError(err)
	s.emissionsKeeper.EXPECT().GetPreviousPercentageRewardToStakedReputers(s.ctx).Return(reputersPercentDec, nil).AnyTimes()

	// Set up minimal required state to get past the initial checks
	err = s.mintKeeper.PreviousBlockEmission.Set(s.ctx, math.ZeroInt())
	s.Require().NoError(err)
	err = s.mintKeeper.MonthsUnlocked.Set(s.ctx, math.ZeroInt())
	s.Require().NoError(err)
	err = s.mintKeeper.EcosystemTokensMinted.Set(s.ctx, math.ZeroInt())
	s.Require().NoError(err)

	_, _, err = s.mintKeeper.GetEmissionInfo(s.ctx)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "failed to get number of staked tokens")
}
