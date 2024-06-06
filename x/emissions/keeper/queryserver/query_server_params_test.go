package queryserver_test

import "github.com/allora-network/allora-chain/x/emissions/types"

func (s *KeeperTestSuite) TestParams() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	queryServer := s.queryServer

	expectedParams := types.Params{
		Version: "1.0",
	}

	err := keeper.SetParams(ctx, expectedParams)
	s.Require().NoError(err, "Setting parameters should not produce an error")

	response, err := queryServer.Params(ctx, &types.QueryParamsRequest{})

	s.Require().NoError(err, "Retrieving parameters should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(expectedParams.Version, response.Params.Version)
}
