package queryserver_test

import (
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestGetTotalStake() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	// Setup: Set an initial total stake value
	expectedTotalStake := cosmosMath.NewUint(1000)
	err := keeper.SetTotalStake(ctx, expectedTotalStake)
	s.Require().NoError(err, "SetTotalStake should not produce an error")

	// Test: Retrieve the total stake using the query server
	req := &types.QueryTotalStakeRequest{}
	response, err := queryServer.GetTotalStake(ctx, req)
	s.Require().NoError(err, "GetTotalStake should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(expectedTotalStake, response.Amount, "The retrieved total stake should match the expected value")
}
