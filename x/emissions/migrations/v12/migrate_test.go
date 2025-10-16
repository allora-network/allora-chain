package v12_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	v12 "github.com/allora-network/allora-chain/x/emissions/migrations/v12"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
)

type EmissionsV12MigrationTestSuite struct {
	testutil.TestSuite
}

func TestEmissionsV12MigrationTestSuite(t *testing.T) {
	suite.Run(t, &EmissionsV12MigrationTestSuite{
		testutil.NewTestSuite("emissions_V12Migrations"),
	})
}

// Test that the migration correctly performs the necessary changes.
func (s *EmissionsV12MigrationTestSuite) TestMigrateStore() {

	// Run migration
	err := v12.MigrateStore(s.Ctx(), *s.EmissionsKeeper())
	s.Require().NoError(err)

}
