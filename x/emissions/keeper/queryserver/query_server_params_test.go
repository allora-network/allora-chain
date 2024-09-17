package queryserver_test

import (
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *QueryServerTestSuite) TestParams() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	queryServer := s.queryServer

	expectedParams := types.DefaultParams()

	err := keeper.SetParams(ctx, expectedParams)
	s.Require().NoError(err, "Setting parameters should not produce an error")

	response, err := queryServer.GetParams(ctx, &types.GetParamsRequest{})

	s.Require().NoError(err, "Retrieving parameters should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(expectedParams, response.Params)
}
