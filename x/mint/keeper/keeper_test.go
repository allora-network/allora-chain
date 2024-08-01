package keeper_test

import (
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

type IntegrationTestSuite struct {
	suite.Suite

	mintKeeper      keeper.Keeper
	ctx             sdk.Context
	msgServer       types.MsgServer
	ctrl            *gomock.Controller
	stakingKeeper   *minttestutil.MockStakingKeeper
	bankKeeper      *minttestutil.MockBankKeeper
	emissionsKeeper *minttestutil.MockEmissionsKeeper
	adminPrivateKey secp256k1.PrivKey
	adminAddr       string
	epochGet        map[int]func(string) alloraMath.Dec
	epoch61Get      func(string) alloraMath.Dec
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupTest() {
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

func (s *IntegrationTestSuite) TestAliasFunctions() {
	stakingTokenSupply := math.NewIntFromUint64(100000000000)
	s.stakingKeeper.EXPECT().TotalBondedTokens(s.ctx).Return(stakingTokenSupply, nil)
	tokenSupply, err := s.mintKeeper.CosmosValidatorStakedSupply(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(tokenSupply, stakingTokenSupply)

	coins := sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(1000000)))
	s.bankKeeper.EXPECT().MintCoins(s.ctx, types.ModuleName, coins).Return(nil)
	s.Require().Equal(s.mintKeeper.MintCoins(s.ctx, sdk.NewCoins()), nil)
	s.Require().Nil(s.mintKeeper.MintCoins(s.ctx, coins))

	fees := sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(1000)))
	s.bankKeeper.EXPECT().SendCoinsFromModuleToModule(s.ctx, types.EcosystemModuleName, emissionstypes.AlloraRewardsAccountName, fees).Return(nil)
	s.Require().Nil(s.mintKeeper.PayAlloraRewardsFromEcosystem(s.ctx, fees))
}
