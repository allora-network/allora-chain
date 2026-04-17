package v14_test

import (
	"testing"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/stretchr/testify/suite"

	alloraMath "github.com/allora-network/allora-chain/math"
	v14 "github.com/allora-network/allora-chain/x/emissions/migrations/v14"
	"github.com/allora-network/allora-chain/x/emissions/migrations/v14/oldtypes"
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

func (s *EmissionsV14MigrationTestSuite) TestMigrateTopics() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	keeper := s.TopicKeeper()

	oldTopics := []oldtypes.Topic{
		{
			Id:                       1,
			Creator:                  s.Addrs(0).String(),
			Metadata:                 "metadata-1",
			LossMethod:               "mse",
			EpochLength:              10800,
			EpochLastEnded:           0,
			GroundTruthLag:           10800,
			WorkerSubmissionWindow:   10,
			PNorm:                    alloraMath.NewDecFromInt64(3),
			InitialRegret:            alloraMath.MustNewDecFromString("0.0001"),
			AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
			AllowNegative:            false,
			Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
			MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
			ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
			ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.25"),
			ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.3"),
			CNorm:                    alloraMath.MustNewDecFromString("0.75"),
		},
		{
			Id:                       2,
			Creator:                  s.Addrs(1).String(),
			Metadata:                 "metadata-2",
			LossMethod:               "mae",
			EpochLength:              7200,
			EpochLastEnded:           5,
			GroundTruthLag:           3600,
			WorkerSubmissionWindow:   20,
			PNorm:                    alloraMath.NewDecFromInt64(4),
			InitialRegret:            alloraMath.MustNewDecFromString("0.001"),
			AlphaRegret:              alloraMath.MustNewDecFromString("0.2"),
			AllowNegative:            true,
			Epsilon:                  alloraMath.MustNewDecFromString("0.02"),
			MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.2"),
			ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.4"),
			ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.5"),
			ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.6"),
			CNorm:                    alloraMath.MustNewDecFromString("0.8"),
		},
	}

	for _, oldTopic := range oldTopics {
		topicStore.Set(sdk.Uint64ToBigEndian(oldTopic.Id), cdc.MustMarshal(&oldTopic))
	}

	err := v14.MigrateTopics(s.Ctx(), store, cdc)
	s.Require().NoError(err)

	for _, oldTopic := range oldTopics {
		gotTopic, err := keeper.GetTopic(s.Ctx(), oldTopic.Id)
		s.Require().NoError(err)

		// existing fields preserved
		s.Require().Equal(oldTopic.Id, gotTopic.Id)
		s.Require().Equal(oldTopic.Creator, gotTopic.Creator)
		s.Require().Equal(oldTopic.Metadata, gotTopic.Metadata)
		s.Require().Equal(oldTopic.LossMethod, gotTopic.LossMethod)
		s.Require().True(oldTopic.PNorm.Equal(gotTopic.PNorm))
		s.Require().True(oldTopic.InitialRegret.Equal(gotTopic.InitialRegret))
		s.Require().True(oldTopic.AlphaRegret.Equal(gotTopic.AlphaRegret))
		s.Require().Equal(oldTopic.AllowNegative, gotTopic.AllowNegative)
		s.Require().True(oldTopic.Epsilon.Equal(gotTopic.Epsilon))
		s.Require().True(oldTopic.MeritSortitionAlpha.Equal(gotTopic.MeritSortitionAlpha))
		s.Require().True(oldTopic.CNorm.Equal(gotTopic.CNorm))

		// migrated quantiles
		s.Require().True(gotTopic.ActiveInfererQuantile.Equal(alloraMath.MustNewDecFromString("0.05")))
		s.Require().True(gotTopic.ActiveForecasterQuantile.Equal(alloraMath.MustNewDecFromString("0.05")))
		s.Require().True(gotTopic.ActiveReputerQuantile.Equal(alloraMath.MustNewDecFromString("0.05")))

	}
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
