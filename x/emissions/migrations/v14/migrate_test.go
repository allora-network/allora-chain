package v14_test

import (
	"testing"

	"cosmossdk.io/collections"
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
	s.Require().Falsef(gotTopic.RequireUnity, "Topic RequireUnity should be false, got %v", gotTopic.RequireUnity)
	s.Require().Equal("0", gotTopic.UnityTolerance.String(), "Topic UnityTolerance should be 0, got %s", gotTopic.UnityTolerance.String())
}

func (s *EmissionsV14MigrationTestSuite) TestMigrateNetworkInferences() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	oldStore := prefix.NewStore(store, emissionstypes.NetworkInferencesKey)
	newStore := prefix.NewStore(store, emissionstypes.NetworkInferenceBundleKey)

	oldBundle := emissionstypes.ValueBundle{
		TopicId: 1,
		ReputerRequestNonce: &emissionstypes.ReputerRequestNonce{
			ReputerNonce: &emissionstypes.Nonce{BlockHeight: 123},
		},
		Reputer:       s.Addrs(9).String(),
		CombinedValue: alloraMath.MustNewDecFromString("10"),
		NaiveValue:    alloraMath.MustNewDecFromString("20"),
		InfererValues: []*emissionstypes.WorkerAttributedValue{
			{
				Worker: s.Addrs(0).String(),
				Value:  alloraMath.MustNewDecFromString("1.1"),
			},
			{
				Worker: s.Addrs(1).String(),
				Value:  alloraMath.MustNewDecFromString("2.2"),
			},
		},
		ForecasterValues: []*emissionstypes.WorkerAttributedValue{
			{
				Worker: s.Addrs(2).String(),
				Value:  alloraMath.MustNewDecFromString("3.3"),
			},
		},
		OneOutInfererValues: []*emissionstypes.WithheldWorkerAttributedValue{
			{
				Worker: s.Addrs(0).String(),
				Value:  alloraMath.MustNewDecFromString("4.4"),
			},
		},
		OneOutForecasterValues: []*emissionstypes.WithheldWorkerAttributedValue{
			{
				Worker: s.Addrs(2).String(),
				Value:  alloraMath.MustNewDecFromString("5.5"),
			},
		},
		OneInForecasterValues: []*emissionstypes.WorkerAttributedValue{
			{
				Worker: s.Addrs(2).String(),
				Value:  alloraMath.MustNewDecFromString("6.6"),
			},
		},
		OneOutInfererForecasterValues: []*emissionstypes.OneOutInfererForecasterValues{
			{
				Forecaster: s.Addrs(2).String(),
				OneOutInfererValues: []*emissionstypes.WithheldWorkerAttributedValue{
					{
						Worker: s.Addrs(0).String(),
						Value:  alloraMath.MustNewDecFromString("7.7"),
					},
					{
						Worker: s.Addrs(1).String(),
						Value:  alloraMath.MustNewDecFromString("8.8"),
					},
				},
			},
		},
	}

	key := collections.Join(oldBundle.TopicId, oldBundle.ReputerRequestNonce.ReputerNonce.BlockHeight)
	kc := collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key)
	keyBz, err := collections.EncodeKeyWithPrefix(emissionstypes.NetworkInferenceBundleKey, kc, key)
	s.Require().NoError(err)

	oldStore.Set(keyBz, cdc.MustMarshal(&oldBundle))

	err = v14.MigrateNetworkInferences(s.Ctx(), store, cdc)
	s.Require().NoError(err)

	// old entry should be deleted
	s.Require().False(oldStore.Has(keyBz))

	// new entry should exist
	s.Require().True(newStore.Has(keyBz))

	var got emissionstypes.NetworkInferenceBundle
	bz := newStore.Get(keyBz)
	err = cdc.Unmarshal(bz, &got)
	s.Require().NoError(err)

	s.Require().Equal(oldBundle.TopicId, got.TopicId)
	s.Require().Equal(oldBundle.ReputerRequestNonce.ReputerNonce.BlockHeight, got.Nonce)

	s.Require().Len(got.CombinedValue, 1)
	s.Require().Equal(uint32(1), got.CombinedValue[0].LabelId)
	s.Require().Equal("y", got.CombinedValue[0].LabelName)
	s.Require().True(oldBundle.CombinedValue.Equal(got.CombinedValue[0].Value))

	s.Require().Len(got.NaiveValue, 1)
	s.Require().Equal(uint32(1), got.NaiveValue[0].LabelId)
	s.Require().Equal("y", got.NaiveValue[0].LabelName)
	s.Require().True(oldBundle.NaiveValue.Equal(got.NaiveValue[0].Value))

	s.Require().Len(got.InfererValues, len(oldBundle.InfererValues))
	for i, v := range oldBundle.InfererValues {
		s.Require().Equal(v.Worker, got.InfererValues[i].Worker)
		s.Require().Len(got.InfererValues[i].Values, 1)
		s.Require().Equal(uint32(1), got.InfererValues[i].Values[0].LabelId)
		s.Require().Equal("y", got.InfererValues[i].Values[0].LabelName)
		s.Require().True(v.Value.Equal(got.InfererValues[i].Values[0].Value))
	}

	s.Require().Len(got.ForecasterValues, len(oldBundle.ForecasterValues))
	for i, v := range oldBundle.ForecasterValues {
		s.Require().Equal(v.Worker, got.ForecasterValues[i].Worker)
		s.Require().Len(got.ForecasterValues[i].Values, 1)
		s.Require().Equal(uint32(1), got.ForecasterValues[i].Values[0].LabelId)
		s.Require().Equal("y", got.ForecasterValues[i].Values[0].LabelName)
		s.Require().True(v.Value.Equal(got.ForecasterValues[i].Values[0].Value))
	}

	s.Require().Len(got.OneOutInfererValues, len(oldBundle.OneOutInfererValues))
	for i, v := range oldBundle.OneOutInfererValues {
		s.Require().Equal(v.Worker, got.OneOutInfererValues[i].WithheldInferer)
		s.Require().Len(got.OneOutInfererValues[i].CombinedInference, 1)
		s.Require().Equal(uint32(1), got.OneOutInfererValues[i].CombinedInference[0].LabelId)
		s.Require().Equal("y", got.OneOutInfererValues[i].CombinedInference[0].LabelName)
		s.Require().True(v.Value.Equal(got.OneOutInfererValues[i].CombinedInference[0].Value))
	}

	s.Require().Len(got.OneOutForecasterValues, len(oldBundle.OneOutForecasterValues))
	for i, v := range oldBundle.OneOutForecasterValues {
		s.Require().Equal(v.Worker, got.OneOutForecasterValues[i].WithheldForecaster)
		s.Require().Len(got.OneOutForecasterValues[i].CombinedInference, 1)
		s.Require().Equal(uint32(1), got.OneOutForecasterValues[i].CombinedInference[0].LabelId)
		s.Require().Equal("y", got.OneOutForecasterValues[i].CombinedInference[0].LabelName)
		s.Require().True(v.Value.Equal(got.OneOutForecasterValues[i].CombinedInference[0].Value))
	}

	s.Require().Len(got.OneInForecasterValues, len(oldBundle.OneInForecasterValues))
	for i, v := range oldBundle.OneInForecasterValues {
		s.Require().Equal(v.Worker, got.OneInForecasterValues[i].Forecaster)
		s.Require().Len(got.OneInForecasterValues[i].CombinedInference, 1)
		s.Require().Equal(uint32(1), got.OneInForecasterValues[i].CombinedInference[0].LabelId)
		s.Require().Equal("y", got.OneInForecasterValues[i].CombinedInference[0].LabelName)
		s.Require().True(v.Value.Equal(got.OneInForecasterValues[i].CombinedInference[0].Value))
	}

	s.Require().Len(got.OneOutInfererForecasterValues, 2)

	s.Require().Equal(oldBundle.OneOutInfererForecasterValues[0].Forecaster, got.OneOutInfererForecasterValues[0].Forecaster)
	s.Require().Equal(oldBundle.OneOutInfererForecasterValues[0].OneOutInfererValues[0].Worker, got.OneOutInfererForecasterValues[0].WithheldInferer)
	s.Require().Len(got.OneOutInfererForecasterValues[0].CombinedInference, 1)
	s.Require().Equal(uint32(1), got.OneOutInfererForecasterValues[0].CombinedInference[0].LabelId)
	s.Require().Equal("y", got.OneOutInfererForecasterValues[0].CombinedInference[0].LabelName)
	s.Require().True(oldBundle.OneOutInfererForecasterValues[0].OneOutInfererValues[0].Value.Equal(got.OneOutInfererForecasterValues[0].CombinedInference[0].Value))

	s.Require().Equal(oldBundle.OneOutInfererForecasterValues[0].Forecaster, got.OneOutInfererForecasterValues[1].Forecaster)
	s.Require().Equal(oldBundle.OneOutInfererForecasterValues[0].OneOutInfererValues[1].Worker, got.OneOutInfererForecasterValues[1].WithheldInferer)
	s.Require().Len(got.OneOutInfererForecasterValues[1].CombinedInference, 1)
	s.Require().Equal(uint32(1), got.OneOutInfererForecasterValues[1].CombinedInference[0].LabelId)
	s.Require().Equal("y", got.OneOutInfererForecasterValues[1].CombinedInference[0].LabelName)
	s.Require().True(oldBundle.OneOutInfererForecasterValues[0].OneOutInfererValues[1].Value.Equal(got.OneOutInfererForecasterValues[1].CombinedInference[0].Value))
}
