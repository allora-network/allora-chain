package v16_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	v16 "github.com/allora-network/allora-chain/x/emissions/migrations/v16"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

type EmissionsV16MigrationTestSuite struct {
	testutil.TestSuite
}

func TestEmissionsV16MigrationTestSuite(t *testing.T) {
	suite.Run(t, &EmissionsV16MigrationTestSuite{
		testutil.NewTestSuite("emissions_V16Migrations"),
	})
}

func (s *EmissionsV16MigrationTestSuite) TestMigrateStoreConvertsTimingAndEnrolls() {
	ctx := s.Ctx()
	topicID := s.CreateTopic(
		testutil.WithEpochLength(30),
		testutil.WithGroundTruthLag(30),
		testutil.WithWorkerSubmissionWindow(10),
	)
	block := ctx.BlockHeight() + 10_000
	s.Require().NoError(s.TopicKeeper().SetTopicToNextPossibleChurningBlock(ctx, topicID, block))
	s.Require().NoError(s.TopicKeeper().SetActiveTopics(ctx, topicID))

	s.Require().NoError(v16.MigrateStore(ctx, *s.EmissionsKeeper()))

	topic, err := s.TopicKeeper().GetTopic(ctx, topicID)
	s.Require().NoError(err)
	s.Require().Equal(int64(30)*6, topic.EpochLength)

	_, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicID)
	s.Require().NoError(err)
	s.Require().True(found)

	params, err := s.ParamsKeeper().GetParams(ctx)
	s.Require().NoError(err)
	s.Require().Equal(types.DefaultParams().MinEpochLength*6, params.MinEpochLength)
}
