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
	"github.com/allora-network/allora-chain/x/emissions/migrations/v15/oldtypes"
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
		s.Require().Equal(emissionstypes.DefaultMaxLabelsPerSubmission, gotTopic.MaxLabelsPerSubmission)
		s.Require().True(gotTopic.LabelDefaultValue.Equal(alloraMath.ZeroDec()))
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
		RequireUnity:             false,
		UnityTolerance:           alloraMath.MustNewDecFromString("0.001"),
		MaxLabelsPerSubmission:   8,
		LabelWhitelist:           []string{"bear", "bull"},
		LabelDefaultValue:        alloraMath.MustNewDecFromString("-1"),
		LabelCaseSensitive:       false,
	}
	topicStore.Set(sdk.Uint64ToBigEndian(classifTopic.Id), cdc.MustMarshal(&classifTopic))

	err := v15.MigrateTopics(s.Ctx(), store, cdc)
	s.Require().NoError(err)

	gotTopic, err := s.TopicKeeper().GetTopic(s.Ctx(), classifTopic.Id)
	s.Require().NoError(err)

	s.Require().Equal(emissionstypes.TopicType_TOPIC_TYPE_CLASSIFICATION, gotTopic.TopicType)
	s.Require().Equal(emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI, gotTopic.OutputArity)
	s.Require().False(gotTopic.RequireUnity)
	s.Require().True(gotTopic.UnityTolerance.Equal(alloraMath.MustNewDecFromString("0.001")))
	s.Require().Equal(uint64(8), gotTopic.MaxLabelsPerSubmission)
	s.Require().Equal([]string{"bear", "bull"}, gotTopic.LabelWhitelist)
	s.Require().True(gotTopic.LabelDefaultValue.Equal(classifTopic.LabelDefaultValue))
}

func (s *EmissionsV15MigrationTestSuite) TestMigrateParamsBackfillsLabelParams() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	params := emissionstypes.DefaultParams()
	params.MaxCanonicalLabelByteLength = 0
	params.MaxTopicLabelWhitelistSize = 0
	params.MaxEpochLabelRegistrySize = 0
	store.Set(emissionstypes.ParamsKey, cdc.MustMarshal(&params))

	err := v15.MigrateParams(s.Ctx(), *s.EmissionsKeeper())
	s.Require().NoError(err)

	got, err := s.ParamsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)
	s.Require().Equal(emissionstypes.DefaultParams().MaxCanonicalLabelByteLength, got.MaxCanonicalLabelByteLength)
	s.Require().Equal(emissionstypes.DefaultParams().MaxTopicLabelWhitelistSize, got.MaxTopicLabelWhitelistSize)
	s.Require().Equal(emissionstypes.DefaultParams().MaxEpochLabelRegistrySize, got.MaxEpochLabelRegistrySize)
}

func (s *EmissionsV15MigrationTestSuite) TestMigrateParamsPreservesExistingLabelParams() {
	params := emissionstypes.DefaultParams()
	params.MaxCanonicalLabelByteLength = 32
	params.MaxTopicLabelWhitelistSize = 128
	params.MaxEpochLabelRegistrySize = 512
	s.Require().NoError(s.ParamsKeeper().SetParams(s.Ctx(), params))

	err := v15.MigrateParams(s.Ctx(), *s.EmissionsKeeper())
	s.Require().NoError(err)

	got, err := s.ParamsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)
	s.Require().Equal(uint64(32), got.MaxCanonicalLabelByteLength)
	s.Require().Equal(uint64(128), got.MaxTopicLabelWhitelistSize)
	s.Require().Equal(uint64(512), got.MaxEpochLabelRegistrySize)
}

func (s *EmissionsV15MigrationTestSuite) TestMigrateStoreFromCurrentV014State() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	key := []byte("topic-1/block-123")
	oldNetworkStore := prefix.NewStore(store, emissionstypes.NetworkInferencesKey)
	oldOutlierStore := prefix.NewStore(store, emissionstypes.OutlierResistantNetworkInferencesKey)
	topicId := uint64(1)

	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	legacyTopic := s.makeLegacyTopic(func(topic *v14oldtypes.Topic) {
		topic.Id = topicId
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

	oldBundle := s.makeLegacyValueBundle(topicId, 123)
	oldNetworkStore.Set(key, cdc.MustMarshal(&oldBundle))
	oldOutlierStore.Set(key, cdc.MustMarshal(&oldBundle))

	legacyInference := s.makeLegacyInference(topicId)
	inferencesStore := prefix.NewStore(store, emissionstypes.InferencesKey)
	infKeyCodec := collections.PairKeyCodec(collections.Uint64Key, collections.StringKey)
	infKeyBytes := make([]byte, infKeyCodec.Size(collections.Join(legacyInference.TopicId, legacyInference.Inferer)))
	_, err := infKeyCodec.Encode(infKeyBytes, collections.Join(legacyInference.TopicId, legacyInference.Inferer))
	s.Require().NoError(err)
	inferencesStore.Set(infKeyBytes, cdc.MustMarshal(&legacyInference))

	blockHeight := emissionstypes.BlockHeight(100)
	legacyAllInference := s.makeLegacyInference(topicId)
	legacyAllInferences := oldtypes.Inferences{
		Inferences: []*oldtypes.Inference{&legacyAllInference},
	}
	allInferencesStore := prefix.NewStore(store, emissionstypes.AllInferencesKey)
	allInfKeyCodec := collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key)
	allInfKeyBytes := make([]byte, allInfKeyCodec.Size(collections.Join(legacyAllInference.TopicId, blockHeight)))
	_, err = allInfKeyCodec.Encode(allInfKeyBytes, collections.Join(legacyAllInference.TopicId, blockHeight))
	s.Require().NoError(err)
	allInferencesStore.Set(allInfKeyBytes, cdc.MustMarshal(&legacyAllInferences))

	params := emissionstypes.DefaultParams()
	params.MaxCanonicalLabelByteLength = 0
	params.MaxTopicLabelWhitelistSize = 0
	params.MaxEpochLabelRegistrySize = 0
	store.Set(emissionstypes.ParamsKey, cdc.MustMarshal(&params))

	err = v15.MigrateStore(s.Ctx(), *s.EmissionsKeeper())
	s.Require().NoError(err)

	gotTopic, err := s.TopicKeeper().GetTopic(s.Ctx(), legacyTopic.Id)
	s.Require().NoError(err)
	s.Require().Equal(emissionstypes.TopicType_TOPIC_TYPE_REGRESSION, gotTopic.TopicType)
	s.Require().Equal(emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE, gotTopic.OutputArity)
	s.Require().False(gotTopic.RequireUnity)
	s.Require().Equal("0", gotTopic.UnityTolerance.String())
	s.Require().Equal(emissionstypes.DefaultMaxLabelsPerSubmission, gotTopic.MaxLabelsPerSubmission)
	s.Require().True(gotTopic.LabelDefaultValue.Equal(alloraMath.ZeroDec()))
	s.Require().True(legacyTopic.ActiveInfererQuantile.Equal(gotTopic.ActiveInfererQuantile))

	gotParams, err := s.ParamsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)
	s.Require().Equal(emissionstypes.DefaultParams().MaxCanonicalLabelByteLength, gotParams.MaxCanonicalLabelByteLength)
	s.Require().Equal(emissionstypes.DefaultParams().MaxTopicLabelWhitelistSize, gotParams.MaxTopicLabelWhitelistSize)
	s.Require().Equal(emissionstypes.DefaultParams().MaxEpochLabelRegistrySize, gotParams.MaxEpochLabelRegistrySize)

	s.assertMigratedBundle(store, cdc, emissionstypes.NetworkInferencesKey, emissionstypes.NetworkInferenceBundleKey, key, oldBundle)
	s.assertMigratedBundle(store, cdc, emissionstypes.OutlierResistantNetworkInferencesKey, emissionstypes.OutlierResistantNetworkInferenceBundleKey, key, oldBundle)

	gotInference, err := s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), legacyInference.TopicId, legacyInference.Inferer)
	s.Require().NoError(err)
	s.assertInference(legacyInference, gotInference)

	gotAllInferences, err := s.WorkerKeeper().GetInferencesAtBlock(s.Ctx(), gotTopic, blockHeight, false)
	s.Require().NoError(err)
	s.Require().NotNil(gotAllInferences)
	s.Require().NotEmpty(gotAllInferences.Inferences)
	s.Require().NotNil(gotAllInferences.Inferences[0])
	s.assertInference(legacyAllInference, *gotAllInferences.Inferences[0])

	expLabelRegistry := emissionstypes.EpochLabelRegistry{
		TopicId: legacyInference.TopicId,
		EpochId: uint64(legacyInference.BlockHeight),
		Labels: []*emissionstypes.TopicLabel{
			{Id: 1, Name: "y"},
		},
	}
	gotRegistry, err := s.TopicKeeper().GetEpochLabelRegistry(s.Ctx(), legacyInference.TopicId, legacyInference.BlockHeight)
	s.Require().NoError(err)
	s.Require().Equal(expLabelRegistry, gotRegistry)
}

func (s *EmissionsV15MigrationTestSuite) TestMigrateStoreFromLegacyV013StateViaV014AndV015() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	topicId := uint64(7)
	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	oldTopic := v14oldtypes.Topic{
		Id:                       topicId,
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

	legacyInference := s.makeLegacyInference(topicId)
	inferencesStore := prefix.NewStore(store, emissionstypes.InferencesKey)
	infKeyCodec := collections.PairKeyCodec(collections.Uint64Key, collections.StringKey)
	infKeyBytes := make([]byte, infKeyCodec.Size(collections.Join(legacyInference.TopicId, legacyInference.Inferer)))
	_, err := infKeyCodec.Encode(infKeyBytes, collections.Join(legacyInference.TopicId, legacyInference.Inferer))
	s.Require().NoError(err)
	inferencesStore.Set(infKeyBytes, cdc.MustMarshal(&legacyInference))

	blockHeight := emissionstypes.BlockHeight(100)
	legacyAllInference := s.makeLegacyInference(topicId)
	legacyAllInferences := oldtypes.Inferences{
		Inferences: []*oldtypes.Inference{&legacyAllInference},
	}
	allInferencesStore := prefix.NewStore(store, emissionstypes.AllInferencesKey)
	allInfKeyCodec := collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key)
	allInfKeyBytes := make([]byte, allInfKeyCodec.Size(collections.Join(legacyAllInference.TopicId, blockHeight)))
	_, err = allInfKeyCodec.Encode(allInfKeyBytes, collections.Join(legacyAllInference.TopicId, blockHeight))
	s.Require().NoError(err)
	allInferencesStore.Set(allInfKeyBytes, cdc.MustMarshal(&legacyAllInferences))

	err = v14.MigrateStore(s.Ctx(), *s.EmissionsKeeper())
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
	s.Require().Equal(emissionstypes.DefaultMaxLabelsPerSubmission, gotTopic.MaxLabelsPerSubmission)
	s.Require().True(gotTopic.LabelDefaultValue.Equal(alloraMath.ZeroDec()))

	s.assertMigratedBundle(store, cdc, emissionstypes.NetworkInferencesKey, emissionstypes.NetworkInferenceBundleKey, key, oldBundle)
	s.assertMigratedBundle(store, cdc, emissionstypes.OutlierResistantNetworkInferencesKey, emissionstypes.OutlierResistantNetworkInferenceBundleKey, key, oldBundle)

	gotInference, err := s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), legacyInference.TopicId, legacyInference.Inferer)
	s.Require().NoError(err)
	s.assertInference(legacyInference, gotInference)

	gotAllInferences, err := s.WorkerKeeper().GetInferencesAtBlock(s.Ctx(), gotTopic, blockHeight, false)
	s.Require().NoError(err)
	s.Require().NotNil(gotAllInferences)
	s.Require().NotEmpty(gotAllInferences.Inferences)
	s.Require().NotNil(gotAllInferences.Inferences[0])
	s.assertInference(legacyAllInference, *gotAllInferences.Inferences[0])

	expLabelRegistry := emissionstypes.EpochLabelRegistry{
		TopicId: legacyInference.TopicId,
		EpochId: uint64(legacyInference.BlockHeight),
		Labels: []*emissionstypes.TopicLabel{
			{Id: 1, Name: "y"},
		},
	}
	gotRegistry, err := s.TopicKeeper().GetEpochLabelRegistry(s.Ctx(), legacyInference.TopicId, legacyInference.BlockHeight)
	s.Require().NoError(err)
	s.Require().Equal(expLabelRegistry, gotRegistry)
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

func (s *EmissionsV15MigrationTestSuite) makeLegacyInference(topicId emissionstypes.TopicId) oldtypes.Inference {
	oldInference := oldtypes.Inference{
		TopicId:     topicId,
		BlockHeight: 100,
		Inferer:     s.AddrsStr(0),
		Value:       alloraMath.MustNewDecFromString("100"),
		ExtraData:   []byte("extra_data"),
		Proof:       "proof",
	}
	return oldInference
}

func (s *EmissionsV15MigrationTestSuite) assertInference(want oldtypes.Inference, got emissionstypes.Inference) {
	s.Require().Equal(want.TopicId, got.TopicId)
	s.Require().Equal(want.BlockHeight, got.BlockHeight)
	s.Require().Equal(want.Inferer, got.Inferer)
	s.Require().Len(got.Values, 1)
	s.Require().True(want.Value.Equal(got.Values[0]))
	s.Require().Equal(want.ExtraData, got.ExtraData)
	s.Require().Equal(want.Proof, got.Proof)
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
