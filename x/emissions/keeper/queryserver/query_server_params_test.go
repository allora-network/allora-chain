package queryserver_test

import (
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *QueryServerTestSuite) TestParams() {
	expectedParams := types.DefaultParams()

	err := s.EmissionsKeeper().SetParams(s.Ctx(), expectedParams)
	s.Require().NoError(err, "Setting parameters should not produce an error")

	response, err := s.EmissionsQueryServer().GetParams(s.Ctx(), &types.GetParamsRequest{})

	s.Require().NoError(err, "Retrieving parameters should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(expectedParams, response.Params)
}
