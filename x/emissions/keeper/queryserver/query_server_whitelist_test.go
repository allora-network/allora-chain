package queryserver_test

import (
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestIsWhitelistAdmin() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	// Create a test address
	testAddress := "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve"
	antitestAddress := "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh"

	keeper.AddWhitelistAdmin(ctx, testAddress)

	req := &types.QueryIsWhitelistAdminRequest{
		Address: testAddress,
	}

	response, err := queryServer.IsWhitelistAdmin(ctx, req)
	s.Require().NoError(err, "IsWhitelistAdmin should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().True(response.IsAdmin, "The IsAdmin field should be true for the test address")

	req = &types.QueryIsWhitelistAdminRequest{
		Address: antitestAddress,
	}

	response, err = queryServer.IsWhitelistAdmin(ctx, req)
	s.Require().NoError(err, "IsWhitelistAdmin should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().False(response.IsAdmin, "The IsAdmin field should be false for the anti test address")
}
