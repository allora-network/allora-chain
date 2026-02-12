package v14_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/stretchr/testify/suite"

	v14 "github.com/allora-network/allora-chain/x/emissions/migrations/v14"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

type EmissionsV14MigrationTestSuite struct {
	testutil.TestSuite
}

func TestEmissionsV14MigrationTestSuite(t *testing.T) {
	suite.Run(t, &EmissionsV14MigrationTestSuite{
		testutil.NewTestSuite("emissions_V14Migrations"),
	})
}

// TestMigrateTopics tests that TopicType and OutputArity is added to all existing topics
func (s *EmissionsV14MigrationTestSuite) TestMigrateTopics() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()
	keeper := s.EmissionsKeeper()

	// Create some topics first (without TopicType and OutputArity, as they would be before migration)
	topicMsg1 := s.MockTopicMsg()
	topicMsg1.TopicType = emissionstypes.TopicType_TOPIC_TYPE_UNSPECIFIED
	topicMsg1.OutputArity = emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_UNSPECIFIED

	// Create topics using keeper
	s.MintTokensToAddress(s.Addrs(0), emissionstypes.DefaultParams().CreateTopicFee.MulRaw(3))

	resp1, err := s.EmissionsMsgServer().CreateNewTopic(s.Ctx(), topicMsg1)
	s.Require().NoError(err)
	topicId1 := resp1.TopicId

	// Run migration
	err = v14.MigrateTopics(s.Ctx(), store, cdc)
	s.Require().NoError(err)

	// Verify that all topics now have TopicType and OutputArity values set to defaults
	topic1, err := keeper.GetTopic(s.Ctx(), topicId1)
	s.Require().NoError(err)
	s.Require().True(topic1.TopicType == emissionstypes.TopicType_TOPIC_TYPE_REGRESSION,
		"Topic 1 TopicType should be %s, got %s", emissionstypes.TopicType_TOPIC_TYPE_REGRESSION.String(), topic1.TopicType.String())
}
