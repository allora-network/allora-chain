package v14_test

import (
	"testing"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

// TestMigrateTopics tests that TopicType and OutputArity is added to all existing topics
func (s *EmissionsV14MigrationTestSuite) TestMigrateTopics() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)

	keeper := s.EmissionsKeeper()

	// Create a topic first (without TopicType and OutputArity, as they would be before migration)
	oldTopic := oldtypes.Topic{
		Id:                       1,
		Creator:                  s.Addrs(0).String(),
		Metadata:                 "metadata",
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
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		CNorm:                    alloraMath.MustNewDecFromString("0.75"),
	}

	topicStore.Set(sdk.Uint64ToBigEndian(oldTopic.Id), cdc.MustMarshal(&oldTopic))

	// Run migration
	err := v14.MigrateTopics(s.Ctx(), store, cdc)
	s.Require().NoError(err)

	// Verify that the topic now have TopicType and OutputArity values set to defaults
	gotTopic, err := keeper.GetTopic(s.Ctx(), oldTopic.Id)
	s.Require().NoError(err)
	s.Require().Equal(emissionstypes.TopicType_TOPIC_TYPE_REGRESSION, gotTopic.TopicType,
		"Topic TopicType should be %s, got %s", emissionstypes.TopicType_TOPIC_TYPE_REGRESSION.String(), gotTopic.TopicType.String())
	s.Require().Equal(emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE, gotTopic.OutputArity,
		"Topic OutputArity should be %s, got %s", emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE.String(), gotTopic.OutputArity.String())
	s.Require().Equal(false, gotTopic.RequireUnity, "Topic RequireUnity should be false, got %v", gotTopic.RequireUnity)
	s.Require().Equal("0", gotTopic.UnityTolerance.String(), "Topic UnityTolerance should be 0, got %s", gotTopic.UnityTolerance.String())
}
