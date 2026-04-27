package v15_test

import (
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/stretchr/testify/suite"

	alloraMath "github.com/allora-network/allora-chain/math"
	v14 "github.com/allora-network/allora-chain/x/emissions/migrations/v14"
	v14oldtypes "github.com/allora-network/allora-chain/x/emissions/migrations/v14/oldtypes"
	v15 "github.com/allora-network/allora-chain/x/emissions/migrations/v15"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

type EmissionsV15MigrationTestSuite struct {
	testutil.TestSuite
}

func TestEmissionsV15MigrationTestSuite(t *testing.T) {
	suite.Run(t, &EmissionsV15MigrationTestSuite{
		testutil.NewTestSuite("emissions_V15Migrations"),
	})
}

func (s *EmissionsV15MigrationTestSuite) TestMigrateTopicsAddsClassificationDefaults() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	keeper := s.TopicKeeper()

	topics := []v14oldtypes.Topic{
		s.makeLegacyTopic(func(topic *v14oldtypes.Topic) {
			topic.Id = 1
			topic.Creator = s.Addrs(0).String()
			topic.Metadata = "metadata-1"
			topic.LossMethod = "mse"
			topic.EpochLength = 10800
			topic.EpochLastEnded = 0
			topic.GroundTruthLag = 10800
			topic.WorkerSubmissionWindow = 10
			topic.PNorm = alloraMath.NewDecFromInt64(3)
			topic.InitialRegret = alloraMath.MustNewDecFromString("0.0001")
			topic.AlphaRegret = alloraMath.MustNewDecFromString("0.1")
			topic.AllowNegative = false
			topic.Epsilon = alloraMath.MustNewDecFromString("0.01")
			topic.MeritSortitionAlpha = alloraMath.MustNewDecFromString("0.1")
			topic.ActiveInfererQuantile = alloraMath.MustNewDecFromString("0.05")
			topic.ActiveForecasterQuantile = alloraMath.MustNewDecFromString("0.05")
			topic.ActiveReputerQuantile = alloraMath.MustNewDecFromString("0.05")
			topic.CNorm = alloraMath.MustNewDecFromString("0.75")
		}),
		s.makeLegacyTopic(func(topic *v14oldtypes.Topic) {
			topic.Id = 2
			topic.Creator = s.Addrs(1).String()
			topic.Metadata = "metadata-2"
			topic.LossMethod = "mae"
			topic.EpochLength = 7200
			topic.EpochLastEnded = 5
			topic.GroundTruthLag = 3600
			topic.WorkerSubmissionWindow = 20
			topic.PNorm = alloraMath.NewDecFromInt64(4)
			topic.InitialRegret = alloraMath.MustNewDecFromString("0.001")
			topic.AlphaRegret = alloraMath.MustNewDecFromString("0.2")
			topic.AllowNegative = true
			topic.Epsilon = alloraMath.MustNewDecFromString("0.02")
			topic.MeritSortitionAlpha = alloraMath.MustNewDecFromString("0.2")
			topic.ActiveInfererQuantile = alloraMath.MustNewDecFromString("0.05")
			topic.ActiveForecasterQuantile = alloraMath.MustNewDecFromString("0.05")
			topic.ActiveReputerQuantile = alloraMath.MustNewDecFromString("0.05")
			topic.CNorm = alloraMath.MustNewDecFromString("0.8")
		}),
	}

	for _, topic := range topics {
		topicStore.Set(sdk.Uint64ToBigEndian(topic.Id), cdc.MustMarshal(&topic))
	}

	err := v15.MigrateTopics(s.Ctx(), store, cdc)
	s.Require().NoError(err)

	for _, topic := range topics {
		gotTopic, err := keeper.GetTopic(s.Ctx(), topic.Id)
		s.Require().NoError(err)

		s.Require().Equal(topic.Id, gotTopic.Id)
		s.Require().Equal(topic.Creator, gotTopic.Creator)
		s.Require().Equal(topic.Metadata, gotTopic.Metadata)
		s.Require().Equal(topic.LossMethod, gotTopic.LossMethod)
		s.Require().True(topic.PNorm.Equal(gotTopic.PNorm))
		s.Require().True(topic.InitialRegret.Equal(gotTopic.InitialRegret))
		s.Require().True(topic.AlphaRegret.Equal(gotTopic.AlphaRegret))
		s.Require().Equal(topic.AllowNegative, gotTopic.AllowNegative)
		s.Require().True(topic.Epsilon.Equal(gotTopic.Epsilon))
		s.Require().True(topic.MeritSortitionAlpha.Equal(gotTopic.MeritSortitionAlpha))
		s.Require().True(topic.ActiveInfererQuantile.Equal(gotTopic.ActiveInfererQuantile))
		s.Require().True(topic.ActiveForecasterQuantile.Equal(gotTopic.ActiveForecasterQuantile))
		s.Require().True(topic.ActiveReputerQuantile.Equal(gotTopic.ActiveReputerQuantile))
		s.Require().True(topic.CNorm.Equal(gotTopic.CNorm))

		s.Require().Equal(emissionstypes.TopicType_TOPIC_TYPE_REGRESSION, gotTopic.TopicType)
		s.Require().Equal(emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE, gotTopic.OutputArity)
		s.Require().False(gotTopic.RequireUnity)
		s.Require().Equal("0", gotTopic.UnityTolerance.String())
	}
}

func (s *EmissionsV15MigrationTestSuite) TestMigrateTopicsPreservesExistingClassificationFields() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)

	classifTopic := emissionstypes.Topic{
		Id:                       42,
		Creator:                  s.Addrs(0).String(),
		Metadata:                 "classification-topic",
		LossMethod:               "cross_entropy",
		EpochLastEnded:           0,
		EpochLength:              0,
		GroundTruthLag:           0,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            false,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		InitialRegret:            alloraMath.MustNewDecFromString("0.0001"),
		WorkerSubmissionWindow:   10,
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.05"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.05"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.05"),
		CNorm:                    alloraMath.MustNewDecFromString("0.75"),
		TopicType:                emissionstypes.TopicType_TOPIC_TYPE_CLASSIFICATION,
		OutputArity:              emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
		RequireUnity:             true,
		UnityTolerance:           alloraMath.MustNewDecFromString("0.001"),
		MaxLabelsPerSubmission:   8,
		LabelWhitelist:           nil,
	}
	topicStore.Set(sdk.Uint64ToBigEndian(classifTopic.Id), cdc.MustMarshal(&classifTopic))

	err := v15.MigrateTopics(s.Ctx(), store, cdc)
	s.Require().NoError(err)

	gotTopic, err := s.TopicKeeper().GetTopic(s.Ctx(), classifTopic.Id)
	s.Require().NoError(err)

	s.Require().Equal(emissionstypes.TopicType_TOPIC_TYPE_CLASSIFICATION, gotTopic.TopicType)
	s.Require().Equal(emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI, gotTopic.OutputArity)
	s.Require().True(gotTopic.RequireUnity)
	s.Require().True(gotTopic.UnityTolerance.Equal(alloraMath.MustNewDecFromString("0.001")))
}

func (s *EmissionsV15MigrationTestSuite) TestMigrateStoreFromCurrentV014State() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	key := []byte("topic-1/block-123")
	oldNetworkStore := prefix.NewStore(store, emissionstypes.NetworkInferencesKey)
	oldOutlierStore := prefix.NewStore(store, emissionstypes.OutlierResistantNetworkInferencesKey)

	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	legacyTopic := s.makeLegacyTopic(func(topic *v14oldtypes.Topic) {
		topic.Id = 1
		topic.Creator = s.Addrs(0).String()
		topic.Metadata = "legacy-topic"
		topic.LossMethod = "mse"
		topic.EpochLength = 10800
		topic.GroundTruthLag = 10800
		topic.WorkerSubmissionWindow = 10
		topic.PNorm = alloraMath.NewDecFromInt64(3)
		topic.InitialRegret = alloraMath.MustNewDecFromString("0.0001")
		topic.AlphaRegret = alloraMath.MustNewDecFromString("0.1")
		topic.Epsilon = alloraMath.MustNewDecFromString("0.01")
		topic.MeritSortitionAlpha = alloraMath.MustNewDecFromString("0.1")
		topic.ActiveInfererQuantile = alloraMath.MustNewDecFromString("0.05")
		topic.ActiveForecasterQuantile = alloraMath.MustNewDecFromString("0.05")
		topic.ActiveReputerQuantile = alloraMath.MustNewDecFromString("0.05")
		topic.CNorm = alloraMath.MustNewDecFromString("0.75")
	})
	topicStore.Set(sdk.Uint64ToBigEndian(legacyTopic.Id), cdc.MustMarshal(&legacyTopic))

	oldBundle := s.makeLegacyValueBundle(1, 123)
	oldNetworkStore.Set(key, cdc.MustMarshal(&oldBundle))
	oldOutlierStore.Set(key, cdc.MustMarshal(&oldBundle))

	err := v15.MigrateStore(s.Ctx(), *s.EmissionsKeeper())
	s.Require().NoError(err)

	gotTopic, err := s.TopicKeeper().GetTopic(s.Ctx(), legacyTopic.Id)
	s.Require().NoError(err)
	s.Require().Equal(emissionstypes.TopicType_TOPIC_TYPE_REGRESSION, gotTopic.TopicType)
	s.Require().Equal(emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE, gotTopic.OutputArity)
	s.Require().False(gotTopic.RequireUnity)
	s.Require().Equal("0", gotTopic.UnityTolerance.String())
	s.Require().True(legacyTopic.ActiveInfererQuantile.Equal(gotTopic.ActiveInfererQuantile))

	s.assertMigratedBundle(store, cdc, emissionstypes.NetworkInferencesKey, emissionstypes.NetworkInferenceBundleKey, key, oldBundle)
	s.assertMigratedBundle(store, cdc, emissionstypes.OutlierResistantNetworkInferencesKey, emissionstypes.OutlierResistantNetworkInferenceBundleKey, key, oldBundle)
}

func (s *EmissionsV15MigrationTestSuite) TestMigrateStoreFromLegacyV013StateViaV014AndV015() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	oldTopic := v14oldtypes.Topic{
		Id:                       7,
		Creator:                  s.Addrs(0).String(),
		Metadata:                 "pre-v016-topic",
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
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.25"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.35"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.45"),
		CNorm:                    alloraMath.MustNewDecFromString("0.75"),
	}
	topicStore.Set(sdk.Uint64ToBigEndian(oldTopic.Id), cdc.MustMarshal(&oldTopic))

	key := []byte("topic-7/block-321")
	oldBundle := s.makeLegacyValueBundle(oldTopic.Id, 321)
	prefix.NewStore(store, emissionstypes.NetworkInferencesKey).Set(key, cdc.MustMarshal(&oldBundle))
	prefix.NewStore(store, emissionstypes.OutlierResistantNetworkInferencesKey).Set(key, cdc.MustMarshal(&oldBundle))

	err := v14.MigrateStore(s.Ctx(), *s.EmissionsKeeper())
	s.Require().NoError(err)

	err = v15.MigrateStore(s.Ctx(), *s.EmissionsKeeper())
	s.Require().NoError(err)

	gotTopic, err := s.TopicKeeper().GetTopic(s.Ctx(), oldTopic.Id)
	s.Require().NoError(err)
	s.Require().True(gotTopic.ActiveInfererQuantile.Equal(alloraMath.MustNewDecFromString("0.05")))
	s.Require().True(gotTopic.ActiveForecasterQuantile.Equal(alloraMath.MustNewDecFromString("0.05")))
	s.Require().True(gotTopic.ActiveReputerQuantile.Equal(alloraMath.MustNewDecFromString("0.05")))
	s.Require().Equal(emissionstypes.TopicType_TOPIC_TYPE_REGRESSION, gotTopic.TopicType)
	s.Require().Equal(emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE, gotTopic.OutputArity)
	s.Require().False(gotTopic.RequireUnity)
	s.Require().Equal("0", gotTopic.UnityTolerance.String())

	s.assertMigratedBundle(store, cdc, emissionstypes.NetworkInferencesKey, emissionstypes.NetworkInferenceBundleKey, key, oldBundle)
	s.assertMigratedBundle(store, cdc, emissionstypes.OutlierResistantNetworkInferencesKey, emissionstypes.OutlierResistantNetworkInferenceBundleKey, key, oldBundle)
}

func (s *EmissionsV15MigrationTestSuite) makeLegacyValueBundle(topicID uint64, blockHeight int64) emissionstypes.ValueBundle {
	//nolint:exhaustruct
	return emissionstypes.ValueBundle{
		TopicId: topicID,
		ReputerRequestNonce: &emissionstypes.ReputerRequestNonce{
			ReputerNonce: &emissionstypes.Nonce{BlockHeight: blockHeight},
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
}

func (s *EmissionsV15MigrationTestSuite) makeLegacyTopic(apply func(*v14oldtypes.Topic)) v14oldtypes.Topic {
	var topic v14oldtypes.Topic
	if apply != nil {
		apply(&topic)
	}

	return topic
}

func (s *EmissionsV15MigrationTestSuite) assertMigratedBundle(
	store storetypes.KVStore,
	cdc codec.BinaryCodec,
	oldPrefix collections.Prefix,
	newPrefix collections.Prefix,
	key []byte,
	oldBundle emissionstypes.ValueBundle,
) {
	oldStore := prefix.NewStore(store, oldPrefix)
	newStore := prefix.NewStore(store, newPrefix)

	s.Require().False(oldStore.Has(key))
	s.Require().True(newStore.Has(key))

	var got emissionstypes.NetworkInferenceBundle
	err := cdc.Unmarshal(newStore.Get(key), &got)
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
	s.Require().True(oldBundle.OneOutInfererForecasterValues[0].OneOutInfererValues[0].Value.Equal(got.OneOutInfererForecasterValues[0].CombinedInference[0].Value))
	s.Require().Equal(oldBundle.OneOutInfererForecasterValues[0].OneOutInfererValues[1].Worker, got.OneOutInfererForecasterValues[1].WithheldInferer)
	s.Require().True(oldBundle.OneOutInfererForecasterValues[0].OneOutInfererValues[1].Value.Equal(got.OneOutInfererForecasterValues[1].CombinedInference[0].Value))
}

// TestMigrateInferencesToWorkerLatestInputInferences drains the legacy
// Inferences store into the v2 WorkerLatestInputInferences store. It covers
// both topic arities: SINGLE topics fall back to Value; MULTI topics
// reverse-map Values through the seeded EpochLabelRegistry.
//
//nolint:exhaustruct
func (s *EmissionsV15MigrationTestSuite) TestMigrateInferencesToWorkerLatestInputInferences() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	legacyStore := prefix.NewStore(store, emissionstypes.InferencesKey) //nolint:staticcheck // SA1019: test seeds the legacy prefix drained by the v15 migration.
	destStore := prefix.NewStore(store, emissionstypes.WorkerLatestInputInferenceKey)
	registryStore := prefix.NewStore(store, emissionstypes.TopicLabelRegistryKey)

	singleTopic := emissionstypes.Topic{
		Id:          1,
		OutputArity: emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
	}
	multiTopic := emissionstypes.Topic{
		Id:          2,
		OutputArity: emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
	}
	topicStore.Set(sdk.Uint64ToBigEndian(singleTopic.Id), cdc.MustMarshal(&singleTopic))
	topicStore.Set(sdk.Uint64ToBigEndian(multiTopic.Id), cdc.MustMarshal(&multiTopic))

	nonceHeight := int64(123)
	multiRegistry := emissionstypes.EpochLabelRegistry{
		TopicId: multiTopic.Id,
		//nolint:gosec // non-negative
		EpochId: uint64(nonceHeight),
		Labels: []*emissionstypes.TopicLabel{
			{Id: 1, Name: "a"},
			{Id: 2, Name: "b"},
			{Id: 3, Name: "c"},
		},
	}
	regKey, err := collections.EncodeKeyWithPrefix(
		nil,
		collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key),
		collections.Join(multiTopic.Id, nonceHeight),
	)
	s.Require().NoError(err)
	registryStore.Set(regKey, cdc.MustMarshal(&multiRegistry))

	singleInferer := s.Addrs(0).String()
	multiInferer := s.Addrs(1).String()

	singleInference := emissionstypes.Inference{
		TopicId:     singleTopic.Id,
		BlockHeight: nonceHeight,
		Inferer:     singleInferer,
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("0.42")},
		ExtraData:   []byte("single-extra"),
	}
	multiInference := emissionstypes.Inference{
		TopicId:     multiTopic.Id,
		BlockHeight: nonceHeight,
		Inferer:     multiInferer,
		Values: []alloraMath.Dec{
			alloraMath.MustNewDecFromString("0.1"),
			alloraMath.MustNewDecFromString("0.2"),
			alloraMath.MustNewDecFromString("0.3"),
		},
		ExtraData: []byte("multi-extra"),
	}

	singleKey, err := collections.EncodeKeyWithPrefix(
		nil,
		collections.PairKeyCodec(collections.Uint64Key, collections.StringKey),
		collections.Join(singleTopic.Id, singleInferer),
	)
	s.Require().NoError(err)
	legacyStore.Set(singleKey, cdc.MustMarshal(&singleInference))

	multiKey, err := collections.EncodeKeyWithPrefix(
		nil,
		collections.PairKeyCodec(collections.Uint64Key, collections.StringKey),
		collections.Join(multiTopic.Id, multiInferer),
	)
	s.Require().NoError(err)
	legacyStore.Set(multiKey, cdc.MustMarshal(&multiInference))

	err = v15.MigrateInferencesToWorkerLatestInputInferences(s.Ctx(), store, cdc)
	s.Require().NoError(err)

	s.Require().False(legacyStore.Has(singleKey), "single legacy inference should be drained")
	s.Require().False(legacyStore.Has(multiKey), "multi legacy inference should be drained")
	s.Require().True(destStore.Has(singleKey))
	s.Require().True(destStore.Has(multiKey))

	var gotSingle emissionstypes.InputInference
	err = gotSingle.Unmarshal(destStore.Get(singleKey))
	s.Require().NoError(err)
	s.Require().Equal(singleTopic.Id, gotSingle.TopicId)
	s.Require().Equal(singleInferer, gotSingle.Inferer)
	s.Require().Equal(nonceHeight, gotSingle.BlockHeight)
	s.Require().Empty(gotSingle.Values)
	s.Require().True(alloraMath.MustNewDecFromString("0.42").Equal(gotSingle.Value.ToDec()))
	s.Require().Equal([]byte("single-extra"), gotSingle.ExtraData)

	var gotMulti emissionstypes.InputInference
	err = gotMulti.Unmarshal(destStore.Get(multiKey))
	s.Require().NoError(err)
	s.Require().Equal(multiTopic.Id, gotMulti.TopicId)
	s.Require().Equal(multiInferer, gotMulti.Inferer)
	s.Require().Len(gotMulti.Values, 3)
	s.Require().Equal("a", gotMulti.Values[0].Label)
	s.Require().Equal("b", gotMulti.Values[1].Label)
	s.Require().Equal("c", gotMulti.Values[2].Label)
	s.Require().True(alloraMath.MustNewDecFromString("0.1").Equal(gotMulti.Values[0].Value.ToDec()))
	s.Require().True(alloraMath.MustNewDecFromString("0.2").Equal(gotMulti.Values[1].Value.ToDec()))
	s.Require().True(alloraMath.MustNewDecFromString("0.3").Equal(gotMulti.Values[2].Value.ToDec()))
}

// TestMigrateInferencesToWorkerLatestInputInferences_DropsMultiWithMissingRegistry
// asserts that a MULTI inference whose EpochLabelRegistry is missing or
// shorter than the values slice is dropped (not crashed on, and the legacy
// store is still drained).
//
//nolint:exhaustruct
func (s *EmissionsV15MigrationTestSuite) TestMigrateInferencesToWorkerLatestInputInferences_DropsMultiWithMissingRegistry() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	legacyStore := prefix.NewStore(store, emissionstypes.InferencesKey) //nolint:staticcheck // SA1019: test seeds the legacy prefix drained by the v15 migration.
	destStore := prefix.NewStore(store, emissionstypes.WorkerLatestInputInferenceKey)

	multiTopic := emissionstypes.Topic{
		Id:          3,
		OutputArity: emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI,
	}
	topicStore.Set(sdk.Uint64ToBigEndian(multiTopic.Id), cdc.MustMarshal(&multiTopic))

	inferer := s.Addrs(2).String()
	inf := emissionstypes.Inference{
		TopicId:     multiTopic.Id,
		BlockHeight: 42,
		Inferer:     inferer,
		Values: []alloraMath.Dec{
			alloraMath.MustNewDecFromString("0.1"),
			alloraMath.MustNewDecFromString("0.2"),
		},
	}

	key, err := collections.EncodeKeyWithPrefix(
		nil,
		collections.PairKeyCodec(collections.Uint64Key, collections.StringKey),
		collections.Join(multiTopic.Id, inferer),
	)
	s.Require().NoError(err)
	legacyStore.Set(key, cdc.MustMarshal(&inf))

	err = v15.MigrateInferencesToWorkerLatestInputInferences(s.Ctx(), store, cdc)
	s.Require().NoError(err)

	s.Require().False(legacyStore.Has(key))
	s.Require().False(destStore.Has(key))
}

// TestMigrateParams_BackfillsDefault verifies that zero-valued
// MaxLabelsPerSubmission on a pre-v15 Params object is backfilled to
// DefaultMaxLabelsPerSubmission.
//
//nolint:exhaustruct
func (s *EmissionsV15MigrationTestSuite) TestMigrateParams_BackfillsDefault() {
	params, err := s.EmissionsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)
	params.MaxLabelsPerSubmission = 0

	// We bypass the keeper's Validate path (which already requires >=1) by
	// writing the raw params proto directly at the collections.Item key.
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	paramsBz, err := params.Marshal()
	s.Require().NoError(err)
	store.Set(emissionstypes.ParamsKey, paramsBz)

	err = v15.MigrateParams(s.Ctx(), *s.EmissionsKeeper())
	s.Require().NoError(err)

	got, err := s.EmissionsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)
	s.Require().Equal(emissionstypes.DefaultMaxLabelsPerSubmission, got.MaxLabelsPerSubmission)
}

// TestMigrateParams_LeavesNonZeroAlone ensures MigrateParams does not
// overwrite an already-populated MaxLabelsPerSubmission.
//
//nolint:exhaustruct
func (s *EmissionsV15MigrationTestSuite) TestMigrateParams_LeavesNonZeroAlone() {
	params, err := s.EmissionsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)
	const custom = uint64(7)
	params.MaxLabelsPerSubmission = custom
	err = s.ParamsKeeper().SetParams(s.Ctx(), params)
	s.Require().NoError(err)

	err = v15.MigrateParams(s.Ctx(), *s.EmissionsKeeper())
	s.Require().NoError(err)

	got, err := s.EmissionsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)
	s.Require().Equal(custom, got.MaxLabelsPerSubmission)
}
