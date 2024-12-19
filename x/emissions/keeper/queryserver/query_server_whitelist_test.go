package queryserver_test

import (
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *QueryServerTestSuite) TestIsWhitelistAdmin() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	// Create a test address
	testAddress := "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve"
	antitestAddress := "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh"

	err := keeper.AddWhitelistAdmin(ctx, testAddress)
	s.Require().NoError(err, "AddWhitelistAdmin should not produce an error")

	req := &types.IsWhitelistAdminRequest{
		Address: testAddress,
	}

	response, err := queryServer.IsWhitelistAdmin(ctx, req)
	s.Require().NoError(err, "IsWhitelistAdmin should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().True(response.IsAdmin, "The IsAdmin field should be true for the test address")

	req = &types.IsWhitelistAdminRequest{
		Address: antitestAddress,
	}

	response, err = queryServer.IsWhitelistAdmin(ctx, req)
	s.Require().NoError(err, "IsWhitelistAdmin should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().False(response.IsAdmin, "The IsAdmin field should be false for the anti test address")
}

func (s *QueryServerTestSuite) TestIsWhitelistedGlobalWorker() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	testAddress := "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve"
	antitestAddress := "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh"

	err := keeper.AddToGlobalWorkerWhitelist(ctx, testAddress)
	s.Require().NoError(err, "AddToGlobalWorkerWhitelist should not produce an error")

	req := &types.IsWhitelistedGlobalWorkerRequest{
		Address: testAddress,
	}

	response, err := queryServer.IsWhitelistedGlobalWorker(ctx, req)
	s.Require().NoError(err, "IsWhitelistedGlobalWorker should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().True(response.IsWhitelistedGlobalWorker, "The IsWhitelistedGlobalWorker field should be true for the test address")

	req = &types.IsWhitelistedGlobalWorkerRequest{
		Address: antitestAddress,
	}

	response, err = queryServer.IsWhitelistedGlobalWorker(ctx, req)
	s.Require().NoError(err, "IsWhitelistedGlobalWorker should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().False(response.IsWhitelistedGlobalWorker, "The IsWhitelistedGlobalWorker field should be false for the anti test address")
}

func (s *QueryServerTestSuite) TestIsWhitelistedGlobalReputer() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	testAddress := "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve"
	antitestAddress := "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh"

	err := keeper.AddToGlobalReputerWhitelist(ctx, testAddress)
	s.Require().NoError(err, "AddToGlobalReputerWhitelist should not produce an error")

	req := &types.IsWhitelistedGlobalReputerRequest{
		Address: testAddress,
	}

	response, err := queryServer.IsWhitelistedGlobalReputer(ctx, req)
	s.Require().NoError(err, "IsWhitelistedGlobalReputer should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().True(response.IsWhitelistedGlobalReputer, "The IsWhitelistedGlobalReputer field should be true for the test address")

	req = &types.IsWhitelistedGlobalReputerRequest{
		Address: antitestAddress,
	}

	response, err = queryServer.IsWhitelistedGlobalReputer(ctx, req)
	s.Require().NoError(err, "IsWhitelistedGlobalReputer should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().False(response.IsWhitelistedGlobalReputer, "The IsWhitelistedGlobalReputer field should be false for the anti test address")
}

func (s *QueryServerTestSuite) TestIsWhitelistedGlobalAdmin() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	testAddress := "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve"
	antitestAddress := "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh"

	err := keeper.AddToGlobalAdminWhitelist(ctx, testAddress)
	s.Require().NoError(err, "AddToGlobalAdminWhitelist should not produce an error")

	req := &types.IsWhitelistedGlobalAdminRequest{
		Address: testAddress,
	}

	response, err := queryServer.IsWhitelistedGlobalAdmin(ctx, req)
	s.Require().NoError(err, "IsWhitelistedGlobalAdmin should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().True(response.IsWhitelistedGlobalAdmin, "The IsWhitelistedGlobalAdmin field should be true for the test address")

	req = &types.IsWhitelistedGlobalAdminRequest{
		Address: antitestAddress,
	}

	response, err = queryServer.IsWhitelistedGlobalAdmin(ctx, req)
	s.Require().NoError(err, "IsWhitelistedGlobalAdmin should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().False(response.IsWhitelistedGlobalAdmin, "The IsWhitelistedGlobalAdmin field should be false for the anti test address")
}

func (s *QueryServerTestSuite) TestIsTopicWhitelistEnabled() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	topicId := uint64(1)

	// Initially should be disabled
	req := &types.IsTopicWorkerWhitelistEnabledRequest{
		TopicId: topicId,
	}

	response, err := queryServer.IsTopicWorkerWhitelistEnabled(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().False(response.IsTopicWorkerWhitelistEnabled)

	// Enable whitelist
	err = keeper.EnableTopicWorkerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	response, err = queryServer.IsTopicWorkerWhitelistEnabled(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().True(response.IsTopicWorkerWhitelistEnabled)
}

func (s *QueryServerTestSuite) TestIsTopicReputerWhitelistEnabled() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	topicId := uint64(1)

	// Initially should be disabled
	req := &types.IsTopicReputerWhitelistEnabledRequest{
		TopicId: topicId,
	}

	response, err := queryServer.IsTopicReputerWhitelistEnabled(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().False(response.IsTopicReputerWhitelistEnabled)

	// Enable whitelist
	err = keeper.EnableTopicReputerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	response, err = queryServer.IsTopicReputerWhitelistEnabled(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().True(response.IsTopicReputerWhitelistEnabled)
}

func (s *QueryServerTestSuite) TestIsWhitelistedTopicCreator() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	testAddr := "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve"

	req := &types.IsWhitelistedTopicCreatorRequest{
		Address: testAddr,
	}

	// Initially should not be whitelisted
	response, err := queryServer.IsWhitelistedTopicCreator(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().False(response.IsWhitelistedTopicCreator)

	// Add to whitelist
	err = keeper.AddToTopicCreatorWhitelist(ctx, testAddr)
	s.Require().NoError(err)

	response, err = queryServer.IsWhitelistedTopicCreator(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().True(response.IsWhitelistedTopicCreator)
}

func (s *QueryServerTestSuite) TestIsWhitelistedGlobalActor() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	testAddr := "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve"

	req := &types.IsWhitelistedGlobalActorRequest{
		Address: testAddr,
	}

	// Initially should not be whitelisted
	response, err := queryServer.IsWhitelistedGlobalActor(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().False(response.IsWhitelistedGlobalActor)

	// Add to whitelist
	err = keeper.AddToGlobalWhitelist(ctx, testAddr)
	s.Require().NoError(err)

	response, err = queryServer.IsWhitelistedGlobalActor(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().True(response.IsWhitelistedGlobalActor)
}

func (s *QueryServerTestSuite) TestIsWhitelistedTopicWorker() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	testAddr := "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve"
	topicId := uint64(1)

	req := &types.IsWhitelistedTopicWorkerRequest{
		TopicId: topicId,
		Address: testAddr,
	}

	// Initially should not be whitelisted
	response, err := queryServer.IsWhitelistedTopicWorker(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().False(response.IsWhitelistedTopicWorker)

	// Add to whitelist
	err = keeper.AddToTopicWorkerWhitelist(ctx, topicId, testAddr)
	s.Require().NoError(err)

	response, err = queryServer.IsWhitelistedTopicWorker(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().True(response.IsWhitelistedTopicWorker)
}

func (s *QueryServerTestSuite) TestIsWhitelistedTopicReputer() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	testAddr := "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve"
	topicId := uint64(1)

	req := &types.IsWhitelistedTopicReputerRequest{
		TopicId: topicId,
		Address: testAddr,
	}

	// Initially should not be whitelisted
	response, err := queryServer.IsWhitelistedTopicReputer(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().False(response.IsWhitelistedTopicReputer)

	// Add to whitelist
	err = keeper.AddToTopicReputerWhitelist(ctx, topicId, testAddr)
	s.Require().NoError(err)

	response, err = queryServer.IsWhitelistedTopicReputer(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().True(response.IsWhitelistedTopicReputer)
}

func (s *QueryServerTestSuite) TestCanUpdateGlobalWhitelists() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	testAddr := "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve"

	req := &types.CanUpdateAllGlobalWhitelistsRequest{
		Address: testAddr,
	}

	// Initially should not be able to update
	response, err := queryServer.CanUpdateAllGlobalWhitelists(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().False(response.CanUpdateAllGlobalWhitelists)

	// Add as admin
	err = keeper.AddWhitelistAdmin(ctx, testAddr)
	s.Require().NoError(err)

	response, err = queryServer.CanUpdateAllGlobalWhitelists(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().True(response.CanUpdateAllGlobalWhitelists)
}

func (s *QueryServerTestSuite) TestCanUpdateParams() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	testAddr := "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve"

	req := &types.CanUpdateParamsRequest{
		Address: testAddr,
	}

	// Initially should not be able to update
	response, err := queryServer.CanUpdateParams(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().False(response.CanUpdateParams)

	// Add as admin
	err = keeper.AddWhitelistAdmin(ctx, testAddr)
	s.Require().NoError(err)

	response, err = queryServer.CanUpdateParams(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().True(response.CanUpdateParams)
}

func (s *QueryServerTestSuite) TestCanUpdateTopicWhitelist() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	testAddr := "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh"

	// Create topic
	topicId := s.CreateOneTopic()

	req := &types.CanUpdateTopicWhitelistRequest{
		TopicId: topicId,
		Address: testAddr,
	}

	// Initially should not be able to update
	response, err := queryServer.CanUpdateTopicWhitelist(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().False(response.CanUpdateTopicWhitelist)

	// Add as admin
	err = keeper.AddWhitelistAdmin(ctx, testAddr)
	s.Require().NoError(err)

	response, err = queryServer.CanUpdateTopicWhitelist(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().True(response.CanUpdateTopicWhitelist)
}

func (s *QueryServerTestSuite) TestCanCreateTopic() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	testAddr := "allo14s7gd09y7mkje8547ukm0c8gjnd3hak7v3fwz6"

	req := &types.CanCreateTopicRequest{
		Address: testAddr,
	}

	// Update TopicCreatorWhitelistEnabled
	params, err := keeper.GetParams(ctx)
	s.Require().NoError(err)
	params.TopicCreatorWhitelistEnabled = false
	err = keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Initially should be able to create topic because TopicCreatorWhitelistEnabled is false
	response, err := queryServer.CanCreateTopic(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().True(response.CanCreateTopic)

	// Update TopicCreatorWhitelistEnabled
	params.TopicCreatorWhitelistEnabled = true
	err = keeper.SetParams(ctx, params)
	s.Require().NoError(err)

	// Should be unable to create topic because TopicCreatorWhitelistEnabled is true
	response, err = queryServer.CanCreateTopic(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().False(response.CanCreateTopic)

	// Add to whitelist
	err = keeper.AddToTopicCreatorWhitelist(ctx, testAddr)
	s.Require().NoError(err)

	// Should be able to create topic because TopicCreatorWhitelistEnabled is true and testAddr is whitelisted
	response, err = queryServer.CanCreateTopic(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().True(response.CanCreateTopic)
}

func (s *QueryServerTestSuite) TestCanSubmitWorkerPayload() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	testAddr := "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh"
	topicId := uint64(1)

	req := &types.CanSubmitWorkerPayloadRequest{
		TopicId: topicId,
		Address: testAddr,
	}

	// Update TopicWhitelist
	err := keeper.DisableTopicWorkerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	// Initially should be able to submit
	response, err := queryServer.CanSubmitWorkerPayload(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().True(response.CanSubmitWorkerPayload)

	// Update TopicWhitelist
	err = keeper.EnableTopicWorkerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	// Should be unable to submit after whitelist is enabled and they are not whitelisted
	response, err = queryServer.CanSubmitWorkerPayload(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().False(response.CanSubmitWorkerPayload)

	// Add to whitelist
	err = keeper.AddToTopicWorkerWhitelist(ctx, topicId, testAddr)
	s.Require().NoError(err)

	// Should be able to submit after whitelist is enabled and testAddr is whitelisted
	response, err = queryServer.CanSubmitWorkerPayload(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().True(response.CanSubmitWorkerPayload)
}

func (s *QueryServerTestSuite) TestCanSubmitReputerPayload() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	testAddr := "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh"
	topicId := uint64(1)

	req := &types.CanSubmitReputerPayloadRequest{
		TopicId: topicId,
		Address: testAddr,
	}

	// Update TopicWhitelist
	err := keeper.DisableTopicReputerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	// Initially should be able to submit
	response, err := queryServer.CanSubmitReputerPayload(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().True(response.CanSubmitReputerPayload)

	// Update TopicWhitelist
	err = keeper.EnableTopicReputerWhitelist(ctx, topicId)
	s.Require().NoError(err)

	// Should be unable to submit after whitelist is enabled and they are not whitelisted
	response, err = queryServer.CanSubmitReputerPayload(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().False(response.CanSubmitReputerPayload)

	// Add to whitelist
	err = keeper.AddToTopicReputerWhitelist(ctx, topicId, testAddr)
	s.Require().NoError(err)

	// Should be able to submit after whitelist is enabled and testAddr is whitelisted
	response, err = queryServer.CanSubmitReputerPayload(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().True(response.CanSubmitReputerPayload)
}
