package v14_test

import (
	"testing"

	"cosmossdk.io/store/prefix"
	alloraMath "github.com/allora-network/allora-chain/math"
	v14 "github.com/allora-network/allora-chain/x/emissions/migrations/v14"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/stretchr/testify/suite"
)

type EmissionsV14MigrationTestSuite struct {
	testutil.TestSuite
}

func TestEmissionsV14MigrationTestSuite(t *testing.T) {
	suite.Run(t, &EmissionsV14MigrationTestSuite{
		testutil.NewTestSuite("emissions_V14Migrations"),
	})
}

func (s *EmissionsV14MigrationTestSuite) TestMigrateTopicsUpdatesExistingTopics() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	topicMsg1 := s.MockTopicMsg()
	topicMsg2 := s.MockTopicMsg()

	s.MintTokensToAddress(s.Addrs(0), emissionstypes.DefaultParams().CreateTopicFee.MulRaw(2))

	resp1, err := s.EmissionsMsgServer().CreateNewTopic(s.Ctx(), topicMsg1)
	s.Require().NoError(err)

	resp2, err := s.EmissionsMsgServer().CreateNewTopic(s.Ctx(), topicMsg2)
	s.Require().NoError(err)

	topic1, err := s.TopicKeeper().GetTopic(s.Ctx(), resp1.TopicId)
	s.Require().NoError(err)
	topic1.ActiveInfererQuantile = alloraMath.MustNewDecFromString("0.25")
	topic1.ActiveForecasterQuantile = alloraMath.MustNewDecFromString("0.25")
	topic1.ActiveReputerQuantile = alloraMath.MustNewDecFromString("0.25")
	s.Require().NoError(s.TopicKeeper().SetTopic(s.Ctx(), resp1.TopicId, topic1))

	topic2, err := s.TopicKeeper().GetTopic(s.Ctx(), resp2.TopicId)
	s.Require().NoError(err)
	topic2.ActiveInfererQuantile = alloraMath.MustNewDecFromString("0.25")
	topic2.ActiveForecasterQuantile = alloraMath.MustNewDecFromString("0.25")
	topic2.ActiveReputerQuantile = alloraMath.MustNewDecFromString("0.25")
	s.Require().NoError(s.TopicKeeper().SetTopic(s.Ctx(), resp2.TopicId, topic2))

	err = v14.MigrateTopics(s.Ctx(), store, cdc)
	s.Require().NoError(err)

	migratedTopic1, err := s.TopicKeeper().GetTopic(s.Ctx(), resp1.TopicId)
	s.Require().NoError(err)
	s.Require().True(migratedTopic1.ActiveInfererQuantile.Equal(alloraMath.MustNewDecFromString("0.05")))
	s.Require().True(migratedTopic1.ActiveForecasterQuantile.Equal(alloraMath.MustNewDecFromString("0.05")))
	s.Require().True(migratedTopic1.ActiveReputerQuantile.Equal(alloraMath.MustNewDecFromString("0.05")))

	migratedTopic2, err := s.TopicKeeper().GetTopic(s.Ctx(), resp2.TopicId)
	s.Require().NoError(err)
	s.Require().True(migratedTopic2.ActiveInfererQuantile.Equal(alloraMath.MustNewDecFromString("0.05")))
	s.Require().True(migratedTopic2.ActiveForecasterQuantile.Equal(alloraMath.MustNewDecFromString("0.05")))
	s.Require().True(migratedTopic2.ActiveReputerQuantile.Equal(alloraMath.MustNewDecFromString("0.05")))
}

func (s *EmissionsV14MigrationTestSuite) TestMigrateTopicsFailsOnInvalidStoredTopicBytes() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	topicStore.Set([]byte("invalid-topic"), []byte("not-protobuf"))

	err := v14.MigrateTopics(s.Ctx(), store, cdc)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "failed to unmarshal topic")
}

func (s *EmissionsV14MigrationTestSuite) TestMigrateStoreUpdatesExistingTopics() {
	topicMsg := s.MockTopicMsg()

	s.MintTokensToAddress(s.Addrs(0), emissionstypes.DefaultParams().CreateTopicFee)

	resp, err := s.EmissionsMsgServer().CreateNewTopic(s.Ctx(), topicMsg)
	s.Require().NoError(err)

	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), resp.TopicId)
	s.Require().NoError(err)
	topic.ActiveInfererQuantile = alloraMath.MustNewDecFromString("0.25")
	topic.ActiveForecasterQuantile = alloraMath.MustNewDecFromString("0.25")
	topic.ActiveReputerQuantile = alloraMath.MustNewDecFromString("0.25")
	s.Require().NoError(s.TopicKeeper().SetTopic(s.Ctx(), resp.TopicId, topic))

	err = v14.MigrateStore(s.Ctx(), *s.EmissionsKeeper())
	s.Require().NoError(err)

	migratedTopic, err := s.TopicKeeper().GetTopic(s.Ctx(), resp.TopicId)
	s.Require().NoError(err)
	s.Require().True(migratedTopic.ActiveInfererQuantile.Equal(alloraMath.MustNewDecFromString("0.05")))
	s.Require().True(migratedTopic.ActiveForecasterQuantile.Equal(alloraMath.MustNewDecFromString("0.05")))
	s.Require().True(migratedTopic.ActiveReputerQuantile.Equal(alloraMath.MustNewDecFromString("0.05")))
}
