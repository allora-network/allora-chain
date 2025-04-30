package v11_test

import (
	"testing"

	cosmosMath "cosmossdk.io/math"
	codecAddress "github.com/cosmos/cosmos-sdk/codec/address"

	"cosmossdk.io/core/store"
	"github.com/allora-network/allora-chain/app/params"

	"github.com/allora-network/allora-chain/x/emissions/keeper"

	v11 "github.com/allora-network/allora-chain/x/emissions/migrations/v11"
	emissions "github.com/allora-network/allora-chain/x/emissions/module"
	emissionstestutil "github.com/allora-network/allora-chain/x/emissions/testutil"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	storetypes "cosmossdk.io/store/types"
	cosmostestutil "github.com/cosmos/cosmos-sdk/testutil"
)

type EmissionsV10MigrationTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	ctx             sdk.Context
	storeService    store.KVStoreService
	emissionsKeeper *keeper.Keeper
}

func TestEmissionsV10MigrationTestSuite(t *testing.T) {
	suite.Run(t, new(EmissionsV10MigrationTestSuite))
}

func (s *EmissionsV10MigrationTestSuite) SetupTest() {
	encCfg := moduletestutil.MakeTestEncodingConfig(emissions.AppModule{})
	key := storetypes.NewKVStoreKey(emissionstypes.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	s.storeService = storeService
	testCtx := cosmostestutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	s.ctx = testCtx.Ctx

	// gomock initializations
	s.ctrl = gomock.NewController(s.T())
	accountKeeper := emissionstestutil.NewMockAccountKeeper(s.ctrl)
	bankKeeper := emissionstestutil.NewMockBankKeeper(s.ctrl)
	emissionsKeeper := keeper.NewKeeper(
		encCfg.Codec,
		codecAddress.NewBech32Codec(params.Bech32PrefixAccAddr),
		storeService,
		accountKeeper,
		bankKeeper,
		authtypes.FeeCollectorName)

	s.emissionsKeeper = &emissionsKeeper
}

// Test that the migration correctly initializes the monthly rewards values.
func (s *EmissionsV10MigrationTestSuite) TestMigrateStore() {
	// Manually set some non-zero initial values to ensure the migration overwrites them
	err := s.emissionsKeeper.AddMonthlyRewards(s.ctx, cosmosMath.NewInt(100), cosmosMath.NewInt(200))
	s.Require().NoError(err)

	// Run migration
	err = v11.MigrateStore(s.ctx, *s.emissionsKeeper)
	s.Require().NoError(err)

	// Verify the values are set to zero
	reputerRewards, err := s.emissionsKeeper.GetMonthlyReputerRewards(s.ctx)
	s.Require().NoError(err)
	s.Require().True(reputerRewards.Equal(cosmosMath.ZeroInt()), "Monthly reputer rewards should be zero after migration")

	topicRewards, err := s.emissionsKeeper.GetMonthlyTopicRewards(s.ctx)
	s.Require().NoError(err)
	s.Require().True(topicRewards.Equal(cosmosMath.ZeroInt()), "Monthly topic rewards should be zero after migration")
}
