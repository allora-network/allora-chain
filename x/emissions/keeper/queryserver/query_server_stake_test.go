package queryserver_test

import (
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestGetTotalStake() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	expectedTotalStake := cosmosMath.NewUint(1000)
	err := keeper.SetTotalStake(ctx, expectedTotalStake)
	s.Require().NoError(err, "SetTotalStake should not produce an error")

	req := &types.QueryTotalStakeRequest{}
	response, err := queryServer.GetTotalStake(ctx, req)
	s.Require().NoError(err, "GetTotalStake should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(expectedTotalStake, response.Amount, "The retrieved total stake should match the expected value")
}

func (s *KeeperTestSuite) TestGetReputerStakeInTopic() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	reputerAddr := PKS[0].Address().String()

	initialStake := cosmosMath.NewUint(250)
	err := keeper.AddStake(ctx, topicId, reputerAddr, initialStake)
	s.Require().NoError(err, "AddStake should not produce an error")

	req := &types.QueryReputerStakeInTopicRequest{
		Address: reputerAddr,
		TopicId: topicId,
	}

	response, err := queryServer.GetReputerStakeInTopic(ctx, req)
	s.Require().NoError(err, "GetReputerStakeInTopic should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(initialStake, response.Amount, "The retrieved stake should match the initial stake set for the reputer in the topic")
}

func (s *KeeperTestSuite) TestGetDelegateStakeInTopicInReputer() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	delegatorAddr := PKS[0].Address().String()
	reputerAddr := PKS[1].Address().String()
	initialStakeAmount := cosmosMath.NewUint(1000)

	err := keeper.AddDelegateStake(ctx, topicId, delegatorAddr, reputerAddr, initialStakeAmount)
	s.Require().NoError(err, "AddDelegateStake should not produce an error")

	req := &types.QueryDelegateStakeInTopicInReputerRequest{
		ReputerAddress: reputerAddr,
		TopicId:        topicId,
	}

	response, err := queryServer.GetDelegateStakeInTopicInReputer(ctx, req)
	s.Require().NoError(err, "GetDelegateStakeInTopicInReputer should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(initialStakeAmount, response.Amount, "The retrieved delegate stake should match the initial stake set for the reputer in the topic")
}

func (s *KeeperTestSuite) TestGetStakeFromDelegatorInTopicInReputer() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	topicId := uint64(123)
	delegatorAddr := PKS[0].Address().String()
	reputerAddr := PKS[1].Address().String()
	stakeAmount := cosmosMath.NewUint(50)

	err := keeper.AddDelegateStake(ctx, topicId, delegatorAddr, reputerAddr, stakeAmount)
	s.Require().NoError(err, "AddDelegateStake should not produce an error")

	req := &types.QueryStakeFromDelegatorInTopicInReputerRequest{
		DelegatorAddress: delegatorAddr,
		ReputerAddress:   reputerAddr,
		TopicId:          topicId,
	}

	response, err := queryServer.GetStakeFromDelegatorInTopicInReputer(ctx, req)
	s.Require().NoError(err, "GetStakeFromDelegatorInTopicInReputer should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(stakeAmount, response.Amount, "The retrieved stake amount should match the delegated stake")
}

func (s *KeeperTestSuite) TestGetStakeFromDelegatorInTopic() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	topicId := uint64(1)
	delegatorAddr := PKS[0].Address().String()
	initialStakeAmount := cosmosMath.NewUint(500)
	additionalStakeAmount := cosmosMath.NewUint(300)

	err := keeper.AddDelegateStake(ctx, topicId, delegatorAddr, PKS[1].Address().String(), initialStakeAmount)
	s.Require().NoError(err)

	err = keeper.AddDelegateStake(ctx, topicId, delegatorAddr, PKS[1].Address().String(), additionalStakeAmount)
	s.Require().NoError(err)

	req := &types.QueryStakeFromDelegatorInTopicRequest{
		DelegatorAddress: delegatorAddr,
		TopicId:          topicId,
	}

	response, err := queryServer.GetStakeFromDelegatorInTopic(ctx, req)
	s.Require().NoError(err, "GetStakeFromDelegatorInTopic should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	expectedTotalStake := initialStakeAmount.Add(additionalStakeAmount)
	s.Require().Equal(expectedTotalStake, response.Amount, "The retrieved stake amount should match the total delegated stake")
}

func (s *KeeperTestSuite) TestGetTopicStake() {
	ctx := s.ctx
	queryServer := s.queryServer
	keeper := s.emissionsKeeper

	topicId := uint64(1)
	reputerAddr := PKS[0].Address().String()
	stakeAmount := cosmosMath.NewUint(500)

	err := keeper.AddStake(ctx, topicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)

	req := &types.QueryTopicStakeRequest{
		TopicId: topicId,
	}

	response, err := queryServer.GetTopicStake(ctx, req)
	s.Require().NoError(err, "GetTopicStake should not produce an error")
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(stakeAmount, response.Amount, "The retrieved topic stake should match the stake amount added")
}
