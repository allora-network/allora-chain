package v15_test

import (
	"strings"
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
			topic.GroundTruthLag = 7200
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

	err := v15.MigrateTopics(s.Ctx(), *s.EmissionsKeeper(), store, cdc)
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

	err := v15.MigrateTopics(s.Ctx(), *s.EmissionsKeeper(), store, cdc)
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

// TestMigrateTopicsPreservesExtremeButValidTopic guards against a future change
// that would clamp or re-tighten the numeric topic bounds inside the migration.
// A legacy topic sitting at the inclusive edges of the long-standing bounds
// (p_norm in [1,10], c_norm in [-100,100], alpha_regret in (0,1]) must survive
// the v15 backfill untouched: those bounds predate v15 and are enforced at topic
// create/update, so every stored topic already satisfies them and the migration
// must not reject the legitimate extremes.
func (s *EmissionsV15MigrationTestSuite) TestMigrateTopicsPreservesExtremeButValidTopic() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	keeper := s.TopicKeeper()

	topics := []v14oldtypes.Topic{
		s.makeLegacyTopic(func(topic *v14oldtypes.Topic) {
			topic.Id = 101
			topic.Creator = s.Addrs(0).String()
			topic.Metadata = "extreme-high"
			topic.LossMethod = "mse"
			topic.EpochLength = 10800
			topic.EpochLastEnded = 0
			topic.GroundTruthLag = 10800
			topic.WorkerSubmissionWindow = 10
			topic.PNorm = alloraMath.MustNewDecFromString("10")
			topic.InitialRegret = alloraMath.MustNewDecFromString("0.0001")
			topic.AlphaRegret = alloraMath.MustNewDecFromString("1")
			topic.AllowNegative = true
			topic.Epsilon = alloraMath.MustNewDecFromString("0.00001")
			topic.MeritSortitionAlpha = alloraMath.MustNewDecFromString("0.1")
			topic.ActiveInfererQuantile = alloraMath.MustNewDecFromString("0.05")
			topic.ActiveForecasterQuantile = alloraMath.MustNewDecFromString("0.05")
			topic.ActiveReputerQuantile = alloraMath.MustNewDecFromString("0.05")
			topic.CNorm = alloraMath.MustNewDecFromString("100")
		}),
		s.makeLegacyTopic(func(topic *v14oldtypes.Topic) {
			topic.Id = 102
			topic.Creator = s.Addrs(1).String()
			topic.Metadata = "extreme-low"
			topic.LossMethod = "mae"
			topic.EpochLength = 10800
			topic.EpochLastEnded = 0
			topic.GroundTruthLag = 10800
			topic.WorkerSubmissionWindow = 10
			topic.PNorm = alloraMath.MustNewDecFromString("1")
			topic.InitialRegret = alloraMath.MustNewDecFromString("0.0001")
			// Smallest practical positive value: alpha_regret must be > 0.
			topic.AlphaRegret = alloraMath.MustNewDecFromString("0.000000000001")
			topic.AllowNegative = true
			topic.Epsilon = alloraMath.MustNewDecFromString("0.00001")
			topic.MeritSortitionAlpha = alloraMath.MustNewDecFromString("0.1")
			topic.ActiveInfererQuantile = alloraMath.MustNewDecFromString("0.05")
			topic.ActiveForecasterQuantile = alloraMath.MustNewDecFromString("0.05")
			topic.ActiveReputerQuantile = alloraMath.MustNewDecFromString("0.05")
			topic.CNorm = alloraMath.MustNewDecFromString("-100")
		}),
	}

	for _, topic := range topics {
		topicStore.Set(sdk.Uint64ToBigEndian(topic.Id), cdc.MustMarshal(&topic))
	}

	err := v15.MigrateTopics(s.Ctx(), *s.EmissionsKeeper(), store, cdc)
	s.Require().NoError(err)

	for _, topic := range topics {
		gotTopic, err := keeper.GetTopic(s.Ctx(), topic.Id)
		s.Require().NoError(err)

		// Extreme numeric values survive the migration untouched.
		s.Require().True(topic.PNorm.Equal(gotTopic.PNorm))
		s.Require().True(topic.AlphaRegret.Equal(gotTopic.AlphaRegret))
		s.Require().True(topic.CNorm.Equal(gotTopic.CNorm))
		s.Require().True(topic.Epsilon.Equal(gotTopic.Epsilon))

		// Classification defaults are still backfilled.
		s.Require().Equal(emissionstypes.TopicType_TOPIC_TYPE_REGRESSION, gotTopic.TopicType)
		s.Require().Equal(emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE, gotTopic.OutputArity)
		s.Require().Equal(emissionstypes.DefaultMaxLabelsPerSubmission, gotTopic.MaxLabelsPerSubmission)
	}
}

// TestMigrateTopicsAbortsOnInvalidTopic verifies the migration halts rather than
// silently skipping when a stored topic fails Topic.Validate after backfill.
// Topic.Validate gates against the current module params, and some of those
// bounds (here, the minimum epoch length) are not backfilled by this migration,
// so a long-lived topic created before a param was tightened can fail the gate.
// A migration must leave state fully consistent, so such a topic is a
// state-integrity failure: the migration returns an error (which deterministically
// halts the upgrade for a coordinated fix) and commits no partial state, not even
// the valid sibling topic.
func (s *EmissionsV15MigrationTestSuite) TestMigrateTopicsAbortsOnInvalidTopic() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	keeper := s.TopicKeeper()

	// Raise MinEpochLength so that a shorter-epoch topic, valid when it was
	// created, no longer satisfies Topic.Validate at migration time.
	const minEpochLength = int64(20000)
	params := emissionstypes.DefaultParams()
	params.MinEpochLength = minEpochLength
	store.Set(emissionstypes.ParamsKey, cdc.MustMarshal(&params))

	validTopic := s.makeLegacyTopic(func(topic *v14oldtypes.Topic) {
		topic.Id = 201
		topic.Creator = s.Addrs(0).String()
		topic.Metadata = "valid"
		topic.LossMethod = "mse"
		topic.EpochLength = minEpochLength + 1
		topic.GroundTruthLag = minEpochLength + 1
		topic.WorkerSubmissionWindow = 10
		topic.PNorm = alloraMath.MustNewDecFromString("3")
		topic.InitialRegret = alloraMath.MustNewDecFromString("0.0001")
		topic.AlphaRegret = alloraMath.MustNewDecFromString("0.1")
		topic.Epsilon = alloraMath.MustNewDecFromString("0.01")
		topic.MeritSortitionAlpha = alloraMath.MustNewDecFromString("0.1")
		topic.ActiveInfererQuantile = alloraMath.MustNewDecFromString("0.05")
		topic.ActiveForecasterQuantile = alloraMath.MustNewDecFromString("0.05")
		topic.ActiveReputerQuantile = alloraMath.MustNewDecFromString("0.05")
		topic.CNorm = alloraMath.MustNewDecFromString("0.75")
	})
	invalidTopic := s.makeLegacyTopic(func(topic *v14oldtypes.Topic) {
		topic.Id = 202
		topic.Creator = s.Addrs(1).String()
		topic.Metadata = "epoch-too-short"
		topic.LossMethod = "mae"
		topic.EpochLength = minEpochLength - 1 // below the tightened minimum
		topic.GroundTruthLag = minEpochLength - 1
		topic.WorkerSubmissionWindow = 10
		topic.PNorm = alloraMath.MustNewDecFromString("3")
		topic.InitialRegret = alloraMath.MustNewDecFromString("0.0001")
		topic.AlphaRegret = alloraMath.MustNewDecFromString("0.1")
		topic.Epsilon = alloraMath.MustNewDecFromString("0.01")
		topic.MeritSortitionAlpha = alloraMath.MustNewDecFromString("0.1")
		topic.ActiveInfererQuantile = alloraMath.MustNewDecFromString("0.05")
		topic.ActiveForecasterQuantile = alloraMath.MustNewDecFromString("0.05")
		topic.ActiveReputerQuantile = alloraMath.MustNewDecFromString("0.05")
		topic.CNorm = alloraMath.MustNewDecFromString("0.75")
	})

	topicStore.Set(sdk.Uint64ToBigEndian(validTopic.Id), cdc.MustMarshal(&validTopic))
	topicStore.Set(sdk.Uint64ToBigEndian(invalidTopic.Id), cdc.MustMarshal(&invalidTopic))

	// The migration must abort on the nonconforming topic, naming it and the
	// failed invariant so recovery is actionable.
	err := v15.MigrateTopics(s.Ctx(), *s.EmissionsKeeper(), store, cdc)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "topic 202")
	s.Require().Contains(err.Error(), "epoch length")

	// No partial state is committed: writes are batched after the scan, so an
	// abort mid-scan leaves every topic in its pre-v15 form, including the valid
	// sibling that was processed before the failure.
	gotValid, err := keeper.GetTopic(s.Ctx(), validTopic.Id)
	s.Require().NoError(err)
	s.Require().Equal(emissionstypes.TopicType_TOPIC_TYPE_UNSPECIFIED, gotValid.TopicType)

	gotInvalid, err := keeper.GetTopic(s.Ctx(), invalidTopic.Id)
	s.Require().NoError(err)
	s.Require().Equal(emissionstypes.TopicType_TOPIC_TYPE_UNSPECIFIED, gotInvalid.TopicType)
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

// TestMigrateInferences_RegistrySeedingIsIdempotent demonstrates the only
// scenario in which the labelStore.Has guard in MigrateInferences can fire.
//
// TopicLabelRegistryKey is a new store prefix in v15, so a clean
// v0.16.0 -> v0.17.0 upgrade has no pre-existing epoch label registries, and
// MigrateInferences is the only code that seeds them. The guard therefore never
// triggers on a first run; it exists purely to keep a second pass over
// already-migrated state (genesis export/import, or re-invocation) from
// re-seeding the {"y"} registry. This test seeds once and verifies a second run
// leaves the registry byte-for-byte intact.
func (s *EmissionsV15MigrationTestSuite) TestMigrateInferences_RegistrySeedingIsIdempotent() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	legacyInference := s.makeLegacyInference(1)
	inferencesStore := prefix.NewStore(store, emissionstypes.InferencesKey)
	infKeyCodec := collections.PairKeyCodec(collections.Uint64Key, collections.StringKey)
	infKeyBytes := make([]byte, infKeyCodec.Size(collections.Join(legacyInference.TopicId, legacyInference.Inferer)))
	_, err := infKeyCodec.Encode(infKeyBytes, collections.Join(legacyInference.TopicId, legacyInference.Inferer))
	s.Require().NoError(err)
	inferencesStore.Set(infKeyBytes, cdc.MustMarshal(&legacyInference))

	// The epoch label registry store is empty before migration: the prefix is new in v15,
	// so there is never a pre-existing registry to "preserve" on a real upgrade.
	labelStore := prefix.NewStore(store, emissionstypes.TopicLabelRegistryKey)
	lblKeyCodec := collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key)
	lblKeyBytes := make([]byte, lblKeyCodec.Size(collections.Join(legacyInference.TopicId, legacyInference.BlockHeight)))
	_, err = lblKeyCodec.Encode(lblKeyBytes, collections.Join(legacyInference.TopicId, legacyInference.BlockHeight))
	s.Require().NoError(err)
	s.Require().False(labelStore.Has(lblKeyBytes), "no epoch label registry should exist before migration")

	expRegistry := emissionstypes.EpochLabelRegistry{
		TopicId: legacyInference.TopicId,
		EpochId: uint64(legacyInference.BlockHeight),
		Labels:  []*emissionstypes.TopicLabel{{Id: 1, Name: "y"}},
	}

	// First run seeds the {"y"} registry.
	s.Require().NoError(v15.MigrateInferences(s.Ctx(), store, cdc))
	s.Require().True(labelStore.Has(lblKeyBytes))
	var gotRegistry emissionstypes.EpochLabelRegistry
	cdc.MustUnmarshal(labelStore.Get(lblKeyBytes), &gotRegistry)
	s.Require().Equal(expRegistry, gotRegistry)

	// Second run is the only case where Has fires: the registry must be left intact.
	// NOTE: the inference *values* are not guarded the same way on a re-run; that
	// behavior is covered by TestMigrateStore_IdempotentSecondRun. This test isolates
	// the registry-seeding idempotency the Has guard provides.
	s.Require().NoError(v15.MigrateInferences(s.Ctx(), store, cdc))
	var gotRegistryAfter emissionstypes.EpochLabelRegistry
	cdc.MustUnmarshal(labelStore.Get(lblKeyBytes), &gotRegistryAfter)
	s.Require().Equal(expRegistry, gotRegistryAfter)
}

// TestMigrateStore_IdempotentSecondRun guards against silent value loss on a second run: the per-inference
// value moved from the old scalar Value field (field 4, now reserved) to the new
// repeated Values field (field 7). MigrateInferences and MigrateAllInferences
// rewrite their stores in place, so a second pass re-reads its own output. Without
// the already-migrated guards, decoding new-format bytes as the old type silently
// drops field 7 and rewrites the value as 0 — which still passes Validate(). This
// test runs both steps twice and asserts the migrated values are left unchanged.
func (s *EmissionsV15MigrationTestSuite) TestMigrateStore_IdempotentSecondRun() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	const topicId = uint64(1)
	blockHeight := emissionstypes.BlockHeight(100)

	// Seed a single legacy Inference (old scalar Value field) under InferencesKey.
	legacyInference := s.makeLegacyInference(topicId)
	inferencesStore := prefix.NewStore(store, emissionstypes.InferencesKey)
	infKeyCodec := collections.PairKeyCodec(collections.Uint64Key, collections.StringKey)
	infKeyBytes := make([]byte, infKeyCodec.Size(collections.Join(legacyInference.TopicId, legacyInference.Inferer)))
	_, err := infKeyCodec.Encode(infKeyBytes, collections.Join(legacyInference.TopicId, legacyInference.Inferer))
	s.Require().NoError(err)
	inferencesStore.Set(infKeyBytes, cdc.MustMarshal(&legacyInference))

	// Seed a legacy Inferences list (old scalar Value fields) under AllInferencesKey.
	legacyAllInference := s.makeLegacyInference(topicId)
	legacyAllInferences := oldtypes.Inferences{
		Inferences: []*oldtypes.Inference{&legacyAllInference},
	}
	allInferencesStore := prefix.NewStore(store, emissionstypes.AllInferencesKey)
	allInfKeyCodec := collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key)
	allInfKeyBytes := make([]byte, allInfKeyCodec.Size(collections.Join(topicId, blockHeight)))
	_, err = allInfKeyCodec.Encode(allInfKeyBytes, collections.Join(topicId, blockHeight))
	s.Require().NoError(err)
	allInferencesStore.Set(allInfKeyBytes, cdc.MustMarshal(&legacyAllInferences))

	// readMigrated decodes both stores with the new codec.
	readMigrated := func() (emissionstypes.Inference, emissionstypes.Inferences) {
		var inf emissionstypes.Inference
		cdc.MustUnmarshal(inferencesStore.Get(infKeyBytes), &inf)
		var all emissionstypes.Inferences
		cdc.MustUnmarshal(allInferencesStore.Get(allInfKeyBytes), &all)
		return inf, all
	}

	// First pass converts old -> new; the value must land in Values[0].
	s.Require().NoError(v15.MigrateInferences(s.Ctx(), store, cdc))
	s.Require().NoError(v15.MigrateAllInferences(s.Ctx(), store, cdc))

	firstInf, firstAll := readMigrated()
	s.assertInference(legacyInference, firstInf)
	s.Require().Len(firstAll.Inferences, 1)
	s.assertInference(legacyAllInference, *firstAll.Inferences[0])

	// Second pass over already-migrated bytes must be a no-op. Without the guards
	// the value would be silently rewritten to 0.
	s.Require().NoError(v15.MigrateInferences(s.Ctx(), store, cdc))
	s.Require().NoError(v15.MigrateAllInferences(s.Ctx(), store, cdc))

	secondInf, secondAll := readMigrated()
	s.assertInference(legacyInference, secondInf)
	s.Require().Len(secondAll.Inferences, 1)
	s.assertInference(legacyAllInference, *secondAll.Inferences[0])
}

// seedLegacyAllInferences writes a legacy archive entry (oldtypes.Inferences)
// at (topicId, block) and returns the encoded label-registry key for that epoch.
func (s *EmissionsV15MigrationTestSuite) seedLegacyAllInferences(
	store storetypes.KVStore,
	cdc codec.BinaryCodec,
	topicId emissionstypes.TopicId,
	block emissionstypes.BlockHeight,
) []byte {
	legacy := s.makeLegacyInference(topicId)
	legacy.BlockHeight = block
	entry := oldtypes.Inferences{Inferences: []*oldtypes.Inference{&legacy}}

	allInfKeyCodec := collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key)
	allInfKeyBytes := make([]byte, allInfKeyCodec.Size(collections.Join(topicId, block)))
	_, err := allInfKeyCodec.Encode(allInfKeyBytes, collections.Join(topicId, block))
	s.Require().NoError(err)
	prefix.NewStore(store, emissionstypes.AllInferencesKey).Set(allInfKeyBytes, cdc.MustMarshal(&entry))

	lblKeyBytes := make([]byte, allInfKeyCodec.Size(collections.Join(topicId, block)))
	_, err = allInfKeyCodec.Encode(lblKeyBytes, collections.Join(topicId, block))
	s.Require().NoError(err)
	return lblKeyBytes
}

// TestMigrateAllInferences_SeedsHistoricalRegistries verifies that the
// per-(topic, block) historical archive gets a {"y"} registry seeded for
// every archived epoch, symmetrically with the latest-inference store. Without
// it, denormalizing a historical inference through the registry would fail with
// a values/labels length mismatch for pre-upgrade blocks.
func (s *EmissionsV15MigrationTestSuite) TestMigrateAllInferences_SeedsHistoricalRegistries() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	type epoch struct {
		topicId emissionstypes.TopicId
		block   emissionstypes.BlockHeight
	}
	epochs := []epoch{
		{topicId: 1, block: 100},
		{topicId: 1, block: 200},
		{topicId: 2, block: 150},
	}
	for _, e := range epochs {
		s.seedLegacyAllInferences(store, cdc, e.topicId, e.block)
	}

	s.Require().NoError(v15.MigrateAllInferences(s.Ctx(), store, cdc))

	for _, e := range epochs {
		gotRegistry, err := s.TopicKeeper().GetEpochLabelRegistry(s.Ctx(), e.topicId, e.block)
		s.Require().NoError(err)
		s.Require().Equal(
			emissionstypes.EpochLabelRegistry{
				TopicId: e.topicId,
				EpochId: uint64(e.block),
				Labels: []*emissionstypes.TopicLabel{
					{Id: emissionstypes.SingleArityCanonicalLabelID, Name: emissionstypes.SingleArityCanonicalLabel},
				},
			},
			gotRegistry,
		)

		// The migrated inference must be denormalizable through the seeded registry:
		// a 1-element values vector against a 1-label registry, no length mismatch.
		//nolint:exhaustruct // GetInferencesAtBlock only reads topic.Id for the store key
		gotInferences, err := s.WorkerKeeper().GetInferencesAtBlock(
			s.Ctx(), emissionstypes.Topic{Id: e.topicId}, e.block, false)
		s.Require().NoError(err)
		s.Require().Len(gotInferences.Inferences, 1)
		s.Require().Len(gotInferences.Inferences[0].Values, 1)
		labeled, err := emissionstypes.ConvertInferenceValuesToLabeledValues(
			gotInferences.Inferences[0].Values, &gotRegistry)
		s.Require().NoError(err)
		s.Require().Len(labeled, 1)
		s.Require().Equal(emissionstypes.SingleArityCanonicalLabel, labeled[0].LabelName)
	}
}

// TestMigrateAllInferences_RegistrySeedingIsIdempotent verifies the labelStore.Has
// guard: a (topic, block) whose registry was already seeded (e.g. by MigrateInferences,
// which runs first in MigrateStore) is left byte-for-byte intact, while the remaining
// archived epochs are still seeded.
func (s *EmissionsV15MigrationTestSuite) TestMigrateAllInferences_RegistrySeedingIsIdempotent() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	preSeeded := s.seedLegacyAllInferences(store, cdc, 1, 100)
	s.seedLegacyAllInferences(store, cdc, 1, 200)

	// Simulate MigrateInferences having already seeded the registry for (1, 100)
	// with a distinct sentinel so we can prove it is not overwritten.
	labelStore := prefix.NewStore(store, emissionstypes.TopicLabelRegistryKey)
	sentinel := emissionstypes.EpochLabelRegistry{
		TopicId: 1,
		EpochId: 100,
		Labels: []*emissionstypes.TopicLabel{
			{Id: emissionstypes.SingleArityCanonicalLabelID, Name: emissionstypes.SingleArityCanonicalLabel},
			{Id: 2, Name: "sentinel"},
		},
	}
	labelStore.Set(preSeeded, cdc.MustMarshal(&sentinel))

	s.Require().NoError(v15.MigrateAllInferences(s.Ctx(), store, cdc))

	// Pre-seeded registry untouched (guard fired).
	gotPreSeeded, err := s.TopicKeeper().GetEpochLabelRegistry(s.Ctx(), 1, 100)
	s.Require().NoError(err)
	s.Require().Equal(sentinel, gotPreSeeded)

	// The other epoch was still seeded with the canonical {"y"} registry.
	gotOther, err := s.TopicKeeper().GetEpochLabelRegistry(s.Ctx(), 1, 200)
	s.Require().NoError(err)
	s.Require().Equal(
		emissionstypes.EpochLabelRegistry{
			TopicId: 1,
			EpochId: 200,
			Labels: []*emissionstypes.TopicLabel{
				{Id: emissionstypes.SingleArityCanonicalLabelID, Name: emissionstypes.SingleArityCanonicalLabel},
			},
		},
		gotOther,
	)
}

func (s *EmissionsV15MigrationTestSuite) TestMigrateNetworkInferences_AbortsOnNilReputerNonce() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	key := []byte("topic-1/block-123")
	oldNetworkStore := prefix.NewStore(store, emissionstypes.NetworkInferencesKey)

	invalidBundle := s.makeLegacyValueBundle(1, 123)
	invalidBundle.ReputerRequestNonce = nil
	oldNetworkStore.Set(key, cdc.MustMarshal(&invalidBundle))

	err := v15.MigrateNetworkInferences(s.Ctx(), store, cdc)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "value bundle reputer request nonce block height must be greater than 0")
	s.Require().True(oldNetworkStore.Has(key), "source bundle must remain when migration aborts")
	s.Require().False(prefix.NewStore(store, emissionstypes.NetworkInferenceBundleKey).Has(key))
}

func (s *EmissionsV15MigrationTestSuite) TestMigrateNetworkInferences_AbortsOnEmptyInfererValues() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	key := []byte("topic-1/block-456")
	oldNetworkStore := prefix.NewStore(store, emissionstypes.NetworkInferencesKey)

	invalidBundle := s.makeLegacyValueBundle(1, 456)
	invalidBundle.InfererValues = nil
	oldNetworkStore.Set(key, cdc.MustMarshal(&invalidBundle))

	err := v15.MigrateNetworkInferences(s.Ctx(), store, cdc)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "value bundle inferer values cannot be nil")
	s.Require().True(oldNetworkStore.Has(key), "source bundle must remain when migration aborts")
	s.Require().False(prefix.NewStore(store, emissionstypes.NetworkInferenceBundleKey).Has(key))
}

func (s *EmissionsV15MigrationTestSuite) TestMigrateStore_AbortsWhenLegacyBundleWouldBeInvalid() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	key := []byte("topic-1/block-789")
	oldNetworkStore := prefix.NewStore(store, emissionstypes.NetworkInferencesKey)

	validBundle := s.makeLegacyValueBundle(1, 789)
	oldNetworkStore.Set(key, cdc.MustMarshal(&validBundle))

	invalidBundle := s.makeLegacyValueBundle(1, 790)
	invalidBundle.ReputerRequestNonce = nil
	oldNetworkStore.Set([]byte("topic-1/block-790"), cdc.MustMarshal(&invalidBundle))

	err := v15.MigrateStore(s.Ctx(), *s.EmissionsKeeper())
	s.Require().Error(err)
	s.Require().True(
		strings.Contains(err.Error(), "value bundle reputer request nonce block height must be greater than 0") ||
			strings.Contains(err.Error(), "failed to validate network inferences"),
		"expected bundle validation failure, got: %v", err,
	)
	s.Require().True(oldNetworkStore.Has(key), "valid source bundle must remain when migration aborts")
	s.Require().False(prefix.NewStore(store, emissionstypes.NetworkInferenceBundleKey).Has(key))
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
