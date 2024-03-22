package msgserver_test

import (
	"math"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestProcessInferencesAndQuery() {
	ctx, msgServer := s.ctx, s.msgServer
	require := s.Require()
	s.CreateOneTopic()

	// Mock setup for inferences
	inferences := []*types.Inference{
		{TopicId: 0, Worker: sdk.AccAddress(PKS[0].Address()).String(), Value: cosmosMath.NewUint(2200)},
		{TopicId: 0, Worker: sdk.AccAddress(PKS[1].Address()).String(), Value: cosmosMath.NewUint(2100)},
		{TopicId: 2, Worker: sdk.AccAddress(PKS[2].Address()).String(), Value: cosmosMath.NewUint(12)},
	}

	// Call the ProcessInferences function to test writes
	processInferencesMsg := &types.MsgProcessInferences{
		Inferences: inferences,
	}
	_, err := msgServer.ProcessInferences(ctx, processInferencesMsg)
	require.NoError(err, "Processing Inferences should not fail")

	/*
	 * Inferences over threshold should be returned
	 */
	// Ensure low ts for topic 1
	var topicId = uint64(0)
	var inferenceTimestamp = uint64(1500000000)

	// _, err = msgServer.SetLatestInferencesTimestamp(ctx, inferencesMsg)
	err = s.emissionsKeeper.UpdateTopicEpochLastEnded(ctx, topicId, inferenceTimestamp)
	require.NoError(err, "Setting latest inference timestamp should not fail")

	allInferences, err := s.emissionsKeeper.GetLatestInferencesFromTopic(ctx, uint64(0))
	require.Equal(len(allInferences), 1)
	for _, inference := range allInferences {
		require.Equal(len(inference.Inferences.Inferences), 2)
	}
	require.NoError(err, "Inferences over ts threshold should be returned")

	/*
	 * Inferences under threshold should not be returned
	 */
	inferenceTimestamp = math.MaxUint64

	err = s.emissionsKeeper.UpdateTopicEpochLastEnded(ctx, topicId, inferenceTimestamp)
	require.NoError(err)

	allInferences, err = s.emissionsKeeper.GetLatestInferencesFromTopic(ctx, uint64(1))

	require.Equal(len(allInferences), 0)
	require.NoError(err, "Inferences under ts threshold should not be returned")
}
