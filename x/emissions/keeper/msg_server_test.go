package keeper_test

import (
	"fmt"
	"math"
	"time"

	cosmosMath "cosmossdk.io/math"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	state "github.com/upshot-tech/protocol-state-machine-module"
)

var (
	PKS     = simtestutil.CreateTestPubKeys(3)
	Addr    = sdk.AccAddress(PKS[0].Address())
	ValAddr = sdk.ValAddress(Addr)
)

func (s *KeeperTestSuite) TestMsgSetWeights() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	reputerAddr := sdk.AccAddress(PKS[0].Address()).String()
	workerAddr := sdk.AccAddress(PKS[1].Address()).String()

	// Create a MsgSetWeights message
	weightMsg := &state.MsgSetWeights{
		Weights: []*state.Weight{
			{
				TopicId: 1,
				Reputer: reputerAddr,
				Worker:  workerAddr,
				Weight:  cosmosMath.NewUint(100),
			},
		},
	}

	_, err := msgServer.SetWeights(ctx, weightMsg)
	require.NoError(err, "SetWeights should not return an error")
}

func (s *KeeperTestSuite) TestMsgSetInferences() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for addresses
	workerAddr := sdk.AccAddress(PKS[1].Address()).String()

	// Create a MsgSetInferences message
	inferencesMsg := &state.MsgSetInferences{
		Inferences: []*state.Inference{
			{
				TopicId:   1,
				Worker:    workerAddr,
				Value:     cosmosMath.NewUint(12),
				ExtraData: []byte("test"),
			},
		},
	}

	_, err := msgServer.SetInferences(ctx, inferencesMsg)
	require.NoError(err, "SetInferences should not return an error")
}

func (s *KeeperTestSuite) TestMsgSetLatestTimestampsInference() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for topic
	topicId := uint64(1)
	inferenceTs := uint64(time.Now().UTC().Unix())

	// Create a MsgSetInferences message
	inferencesMsg := &state.MsgSetLatestInferencesTimestamp{
		TopicId:            topicId,
		InferenceTimestamp: inferenceTs,
	}

	_, err := msgServer.SetLatestInferencesTimestamp(ctx, inferencesMsg)
	require.NoError(err, "SetLatestTimestampInferences should not return an error")

	result, err := s.upshotKeeper.GetLatestInferenceTimestamp(s.ctx, topicId)
	fmt.Printf("The timestamp value is %d.\n", result)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(result, inferenceTs)
}

func (s *KeeperTestSuite) TestProcessInferencesAndQuery() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for inferences
	inferences := []*state.Inference{
		{TopicId: 1, Worker: "worker1", Value: cosmosMath.NewUint(2200)},
		{TopicId: 1, Worker: "worker2", Value: cosmosMath.NewUint(2100)},
		{TopicId: 2, Worker: "worker2", Value: cosmosMath.NewUint(12)},
	}

	// Call the ProcessInferences function to test writes
	processInferencesMsg := &state.MsgProcessInferences{
		Inferences: inferences,
	}
	_, err := msgServer.ProcessInferences(ctx, processInferencesMsg)
	require.NoError(err, "Processing Inferences should not fail")

	/*
	 * Inferences over threshold should be returned
	 */
	// Ensure low ts for topic 1
	inferencesMsg := &state.MsgSetLatestInferencesTimestamp{
		TopicId:            uint64(1),
		InferenceTimestamp: 1500000000,
	}
	_, err = msgServer.SetLatestInferencesTimestamp(ctx, inferencesMsg)

	allInferences, err := s.upshotKeeper.GetLatestInferencesFromTopic(ctx, uint64(1))
	require.Equal(len(allInferences), 1)
	for _, inference := range allInferences {
		require.Equal(len(inference.Inferences.Inferences), 2)
	}
	require.NoError(err, "Inferences over ts threshold should be returned")

	/*
	 * Inferences under threshold should not be returned
	 */
	// Ensure highest ts for topic 1
	inferencesMsg = &state.MsgSetLatestInferencesTimestamp{
		TopicId:            uint64(1),
		InferenceTimestamp: math.MaxUint64,
	}
	_, err = msgServer.SetLatestInferencesTimestamp(ctx, inferencesMsg)

	allInferences, err = s.upshotKeeper.GetLatestInferencesFromTopic(ctx, uint64(1))
	require.Equal(len(allInferences), 0)
	require.NoError(err, "Inferences under ts threshold should not be returned")

}

func (s *KeeperTestSuite) TestCreateSeveralTopics() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()

	// Mock setup for metadata and validation steps
	metadata := "Some metadata for the new topic"
	validationSteps := []string{"step1", "step2"}
	// Create a MsgSetInferences message
	newTopicMsg := &state.MsgCreateNewTopic{
		Metadata:         metadata,
		WeightLogic:      "logic",
		WeightCadence:    10800,
		InferenceCadence: 60,
		Active:           true,
		ValidationSteps:  validationSteps,
	}

	_, err := msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on first creation")

	result, err := s.upshotKeeper.GetNumTopics(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(result, uint64(1), "Topic count after first topic is not 1.")

	// Create second topic
	_, err = msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on second topic")

	result, err = s.upshotKeeper.GetNumTopics(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(result, uint64(2), "Topic count after second topic insertion is not 2")
}
