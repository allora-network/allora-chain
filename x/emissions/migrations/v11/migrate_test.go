package v11_test

import (
	"testing"

	cosmosMath "cosmossdk.io/math"
	"github.com/stretchr/testify/suite"

	v11 "github.com/allora-network/allora-chain/x/emissions/migrations/v11"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
)

type EmissionsV11MigrationTestSuite struct {
	testutil.TestSuite
}

func TestEmissionsV11MigrationTestSuite(t *testing.T) {
	suite.Run(t, &EmissionsV11MigrationTestSuite{
		testutil.NewTestSuite("emissions_V11Migrations"),
	})
}

// Test that the migration correctly initializes the monthly rewards values.
func (s *EmissionsV11MigrationTestSuite) TestMigrateStore() {
	// Manually set some non-zero initial values to ensure the migration overwrites them
	err := s.WeightsKeeper().AddMonthlyRewards(s.Ctx(), cosmosMath.NewInt(100), cosmosMath.NewInt(200))
	s.Require().NoError(err)

	// Run migration
	err = v11.MigrateStore(s.Ctx(), s.EmissionsKeeper().GetWeightsKeeper())
	s.Require().NoError(err)

	// Verify the values are set to zero
	reputerRewards, err := s.WeightsKeeper().GetMonthlyReputerRewards(s.Ctx())
	s.Require().NoError(err)
	s.Require().True(reputerRewards.Equal(cosmosMath.ZeroInt()), "Monthly reputer rewards should be zero after migration")

	topicRewards, err := s.WeightsKeeper().GetMonthlyTopicRewards(s.Ctx())
	s.Require().NoError(err)
	s.Require().True(topicRewards.Equal(cosmosMath.ZeroInt()), "Monthly topic rewards should be zero after migration")
}
