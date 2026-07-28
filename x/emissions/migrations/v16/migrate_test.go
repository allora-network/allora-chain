package v16_test

import (
	"testing"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	v16 "github.com/allora-network/allora-chain/x/emissions/migrations/v16"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

type EmissionsV16MigrationTestSuite struct {
	testutil.TestSuite
}

func TestEmissionsV16MigrationTestSuite(t *testing.T) {
	suite.Run(t, &EmissionsV16MigrationTestSuite{
		testutil.NewTestSuite("emissions_V16Migrations"),
	})
}

func (s *EmissionsV16MigrationTestSuite) storeAndCodec() (storetypes.KVStore, codec.BinaryCodec) {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()
	return store, cdc
}

// writeRawTopic persists a topic directly to the topic prefix store, bypassing
// SetTopic validation. This lets tests seed a pre-migration topic whose new
// MaxTopInferersToReward field is zero (which SetTopic would now reject).
func (s *EmissionsV16MigrationTestSuite) writeRawTopic(topic emissionstypes.Topic) {
	store, cdc := s.storeAndCodec()
	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	topicStore.Set(sdk.Uint64ToBigEndian(topic.Id), cdc.MustMarshal(&topic))
}

// legacyTopic returns a fully-valid v10 topic with MaxTopInferersToReward left
// at zero, i.e. exactly how a topic persisted before this field existed decodes
// (protobuf omits the zero value).
func (s *EmissionsV16MigrationTestSuite) legacyTopic(id uint64) emissionstypes.Topic {
	topic := s.MockTopic()
	topic.Id = id
	topic.MaxTopInferersToReward = 0
	return topic
}

// backfill fills a zero field with the current global and preserves the rest.
func (s *EmissionsV16MigrationTestSuite) TestMigrateTopicsBackfillsGlobalDefault() {
	store, cdc := s.storeAndCodec()
	topic := s.legacyTopic(1)
	s.writeRawTopic(topic)

	params, err := s.EmissionsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)
	s.Require().Positive(params.MaxTopInferersToReward)

	err = v16.MigrateTopics(s.Ctx(), *s.EmissionsKeeper(), store, cdc)
	s.Require().NoError(err)

	got, err := s.TopicKeeper().GetTopic(s.Ctx(), 1)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopInferersToReward, got.MaxTopInferersToReward)

	// Pre-existing fields are untouched.
	s.Require().Equal(topic.Creator, got.Creator)
	s.Require().Equal(topic.Metadata, got.Metadata)
	s.Require().Equal(topic.LossMethod, got.LossMethod)
	s.Require().Equal(topic.EpochLength, got.EpochLength)
	s.Require().Equal(topic.GroundTruthLag, got.GroundTruthLag)
	s.Require().True(topic.PNorm.Equal(got.PNorm))
	s.Require().True(topic.CNorm.Equal(got.CNorm))
	s.Require().Equal(topic.TopicType, got.TopicType)
	s.Require().Equal(topic.OutputArity, got.OutputArity)
	s.Require().Equal(topic.MaxLabelsPerSubmission, got.MaxLabelsPerSubmission)
	s.Require().True(topic.ActiveInfererQuantile.Equal(got.ActiveInfererQuantile))
}

// the backfill value is read from live params, not a compile-time constant.
func (s *EmissionsV16MigrationTestSuite) TestMigrateTopicsUsesLiveGlobal() {
	store, cdc := s.storeAndCodec()

	params, err := s.EmissionsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)
	params.MaxTopInferersToReward = 20
	s.Require().NoError(s.EmissionsKeeper().SetParams(s.Ctx(), params))

	s.writeRawTopic(s.legacyTopic(1))

	err = v16.MigrateTopics(s.Ctx(), *s.EmissionsKeeper(), store, cdc)
	s.Require().NoError(err)

	got, err := s.TopicKeeper().GetTopic(s.Ctx(), 1)
	s.Require().NoError(err)
	s.Require().Equal(uint64(20), got.MaxTopInferersToReward)
}

// topics that already carry a value are not overwritten.
func (s *EmissionsV16MigrationTestSuite) TestMigrateTopicsPreservesExistingValue() {
	store, cdc := s.storeAndCodec()
	topic := s.legacyTopic(1)
	topic.MaxTopInferersToReward = 7
	s.writeRawTopic(topic)

	err := v16.MigrateTopics(s.Ctx(), *s.EmissionsKeeper(), store, cdc)
	s.Require().NoError(err)

	got, err := s.TopicKeeper().GetTopic(s.Ctx(), 1)
	s.Require().NoError(err)
	s.Require().Equal(uint64(7), got.MaxTopInferersToReward)
}

// running the migration twice is a no-op after the first pass.
func (s *EmissionsV16MigrationTestSuite) TestMigrateTopicsIdempotent() {
	store, cdc := s.storeAndCodec()
	s.writeRawTopic(s.legacyTopic(1))

	params, err := s.EmissionsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)

	s.Require().NoError(v16.MigrateTopics(s.Ctx(), *s.EmissionsKeeper(), store, cdc))
	s.Require().NoError(v16.MigrateTopics(s.Ctx(), *s.EmissionsKeeper(), store, cdc))

	got, err := s.TopicKeeper().GetTopic(s.Ctx(), 1)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopInferersToReward, got.MaxTopInferersToReward)
}

// an empty topic store migrates without error.
func (s *EmissionsV16MigrationTestSuite) TestMigrateTopicsEmptyStore() {
	store, cdc := s.storeAndCodec()
	s.Require().NoError(v16.MigrateTopics(s.Ctx(), *s.EmissionsKeeper(), store, cdc))
}

// a mix of zero and non-zero topics is handled per-topic.
func (s *EmissionsV16MigrationTestSuite) TestMigrateTopicsMixedSet() {
	store, cdc := s.storeAndCodec()
	params, err := s.EmissionsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)

	s.writeRawTopic(s.legacyTopic(1)) // zero -> backfilled
	preset := s.legacyTopic(2)
	preset.MaxTopInferersToReward = 5 // non-zero -> preserved
	s.writeRawTopic(preset)
	s.writeRawTopic(s.legacyTopic(3)) // zero -> backfilled

	s.Require().NoError(v16.MigrateTopics(s.Ctx(), *s.EmissionsKeeper(), store, cdc))

	got1, err := s.TopicKeeper().GetTopic(s.Ctx(), 1)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopInferersToReward, got1.MaxTopInferersToReward)

	got2, err := s.TopicKeeper().GetTopic(s.Ctx(), 2)
	s.Require().NoError(err)
	s.Require().Equal(uint64(5), got2.MaxTopInferersToReward)

	got3, err := s.TopicKeeper().GetTopic(s.Ctx(), 3)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopInferersToReward, got3.MaxTopInferersToReward)
}

// an unset floor is backfilled to the module default when the ceiling leaves
// room for it.
func (s *EmissionsV16MigrationTestSuite) TestMigrateParamsBackfillsMinTopInferersToReward() {
	store, cdc := s.storeAndCodec()

	// Write a zero floor directly, as an existing chain decodes the new field.
	params := emissionstypes.DefaultParams()
	params.MinTopInferersToReward = 0
	store.Set(emissionstypes.ParamsKey, cdc.MustMarshal(&params))

	s.Require().NoError(v16.MigrateParams(s.Ctx(), *s.EmissionsKeeper()))

	got, err := s.EmissionsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)
	s.Require().Equal(emissionstypes.DefaultMinTopInferersToReward, got.MinTopInferersToReward)
}

// a ceiling below the default floor clamps the backfill to the ceiling instead
// of failing params validation, which would halt the upgrade. MigrateTopics
// only repairs a zero ceiling, so a small non-zero one reaches MigrateParams
// intact and this is the only thing keeping the range well formed.
func (s *EmissionsV16MigrationTestSuite) TestMigrateParamsClampsMinToCeilingBelowDefault() {
	store, cdc := s.storeAndCodec()

	// Any ceiling under the default floor will do; the clamp has to pick it
	// rather than the default.
	const ceiling = uint64(3)
	s.Require().Less(ceiling, emissionstypes.DefaultMinTopInferersToReward)

	// Write directly: SetParams would reject the default floor against this
	// ceiling, which is the very situation the clamp exists to survive.
	params := emissionstypes.DefaultParams()
	params.MaxTopInferersToReward = ceiling
	params.MinTopInferersToReward = 0
	store.Set(emissionstypes.ParamsKey, cdc.MustMarshal(&params))

	s.Require().NoError(v16.MigrateParams(s.Ctx(), *s.EmissionsKeeper()))

	// The ceiling wins, not the default: min(default, ceiling). Expecting the
	// default here would be expecting the halt this clamp exists to prevent.
	got, err := s.EmissionsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)
	s.Require().Equal(ceiling, got.MinTopInferersToReward)
	s.Require().NoError(got.Validate())
}

func (s *EmissionsV16MigrationTestSuite) TestMigrateParamsPreservesExistingMinTopInferersToReward() {
	params := emissionstypes.DefaultParams()
	params.MinTopInferersToReward = 7
	s.Require().NoError(s.EmissionsKeeper().SetParams(s.Ctx(), params))

	s.Require().NoError(v16.MigrateParams(s.Ctx(), *s.EmissionsKeeper()))

	got, err := s.EmissionsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)
	s.Require().Equal(uint64(7), got.MinTopInferersToReward)
}

func (s *EmissionsV16MigrationTestSuite) TestMigrateTopicsRepairsZeroGlobal() {
	store, cdc := s.storeAndCodec()

	// Write a zero global directly, bypassing SetParams validation which now
	// rejects zero.
	badParams := emissionstypes.DefaultParams()
	badParams.MaxTopInferersToReward = 0
	store.Set(emissionstypes.ParamsKey, cdc.MustMarshal(&badParams))

	s.writeRawTopic(s.legacyTopic(1))

	err := v16.MigrateTopics(s.Ctx(), *s.EmissionsKeeper(), store, cdc)
	s.Require().NoError(err)

	defaultCap := emissionstypes.DefaultParams().MaxTopInferersToReward
	repaired, err := s.EmissionsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)
	s.Require().Equal(defaultCap, repaired.MaxTopInferersToReward)

	got, err := s.TopicKeeper().GetTopic(s.Ctx(), 1)
	s.Require().NoError(err)
	s.Require().Equal(defaultCap, got.MaxTopInferersToReward)
}

// a pre-existing topic that is invalid for an unrelated reason (ground truth
// lag lower than epoch length, nothing to do with max_top_inferers_to_reward)
// is still backfilled: the migration does not run full topic validation, so
// unrelated field invalidity cannot halt the upgrade.
func (s *EmissionsV16MigrationTestSuite) TestMigrateTopicsBackfillsDespiteUnrelatedInvalidTopic() {
	store, cdc := s.storeAndCodec()
	topic := s.legacyTopic(1)
	// Ground truth lag lower than epoch length violates Topic.Validate
	// independently of the max_top_inferers_to_reward backfill.
	topic.GroundTruthLag = topic.EpochLength - 1
	s.writeRawTopic(topic)

	params, err := s.EmissionsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)

	err = v16.MigrateTopics(s.Ctx(), *s.EmissionsKeeper(), store, cdc)
	s.Require().NoError(err)

	got, err := s.TopicKeeper().GetTopic(s.Ctx(), 1)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopInferersToReward, got.MaxTopInferersToReward)
}

// the top-level MigrateStore entry point backfills a legacy topic end-to-end.
func (s *EmissionsV16MigrationTestSuite) TestMigrateStoreBackfillsTopic() {
	s.writeRawTopic(s.legacyTopic(1))

	params, err := s.EmissionsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)

	s.Require().NoError(v16.MigrateStore(s.Ctx(), *s.EmissionsKeeper()))

	got, err := s.TopicKeeper().GetTopic(s.Ctx(), 1)
	s.Require().NoError(err)
	s.Require().Equal(params.MaxTopInferersToReward, got.MaxTopInferersToReward)
}

// a degenerate zero ceiling alongside the unset floor must not fail the upgrade:
// the ceiling is repaired before the floor is persisted, so params validation
// never sees floor > ceiling.
func (s *EmissionsV16MigrationTestSuite) TestMigrateStoreWithZeroCeilingAndFloor() {
	store, cdc := s.storeAndCodec()

	badParams := emissionstypes.DefaultParams()
	badParams.MaxTopInferersToReward = 0
	badParams.MinTopInferersToReward = 0
	store.Set(emissionstypes.ParamsKey, cdc.MustMarshal(&badParams))

	s.writeRawTopic(s.legacyTopic(1))

	s.Require().NoError(v16.MigrateStore(s.Ctx(), *s.EmissionsKeeper()))

	repaired, err := s.EmissionsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)
	s.Require().Equal(emissionstypes.DefaultParams().MaxTopInferersToReward, repaired.MaxTopInferersToReward)
	s.Require().Equal(emissionstypes.DefaultMinTopInferersToReward, repaired.MinTopInferersToReward)
}
