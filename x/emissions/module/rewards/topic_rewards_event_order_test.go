package rewards_test

import (
	"bytes"
	"context"
	"errors"

	"cosmossdk.io/collections"
	corestore "cosmossdk.io/core/store"
	cosmosMath "cosmossdk.io/math"
	codecAddress "github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	appparams "github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// selectiveFailStoreService delegates to the emissions store and fails one
// selected write to a collection only after the test arms it.
type selectiveFailStoreService struct {
	corestore.KVStoreService
	prefix        []byte
	err           error
	enabled       bool
	failOnNth     int
	matchedWrites int
	failedWrites  int
}

func (s *selectiveFailStoreService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return &selectiveFailStore{
		KVStore: s.KVStoreService.OpenKVStore(ctx),
		service: s,
	}
}

func (s *selectiveFailStoreService) failMatchedWrite() error {
	nth := s.failOnNth
	if nth <= 0 {
		nth = 1
	}
	s.matchedWrites++
	if s.matchedWrites == nth {
		s.failedWrites++
		return s.err
	}
	return nil
}

type selectiveFailStore struct {
	corestore.KVStore
	service *selectiveFailStoreService
}

func (s *selectiveFailStore) Set(key, value []byte) error {
	if s.service.enabled && bytes.HasPrefix(key, s.service.prefix) {
		if err := s.service.failMatchedWrite(); err != nil {
			return err
		}
	}
	return s.KVStore.Set(key, value)
}

func (s *RewardsTestSuite) emissionsKeeperWithFailingWrites(
	prefix collections.Prefix,
	injectedErr error,
) (keeper.Keeper, *selectiveFailStoreService) {
	storeService := &selectiveFailStoreService{
		KVStoreService: s.StoreServiceEmissions(),
		prefix:         prefix.Bytes(),
		err:            injectedErr,
		enabled:        false,
		failOnNth:      0,
		matchedWrites:  0,
		failedWrites:   0,
	}
	emissionsKeeper := keeper.NewKeeper(
		s.EmissionsKeeper().GetBinaryCodec(),
		codecAddress.NewBech32Codec(appparams.Bech32PrefixAccAddr),
		storeService,
		s.AccountKeeper(),
		s.BankKeeper(),
		authtypes.FeeCollectorName,
	)
	return emissionsKeeper, storeService
}

// A failure after the reactivation helper has returned must not leave a successful
// reactivation event ahead of the recovery path's reactivation-error event.
func (s *RewardsTestSuite) TestGetAndUpdateActiveTopicWeightsEmitsOnlyReactivationErrorWhenCleanupFails() {
	topic := s.MockTopic()
	topic.Id = 2
	topic.EpochLength = 10
	topic.GroundTruthLag = topic.EpochLength
	topic.WorkerSubmissionWindow = topic.EpochLength

	params := types.DefaultParams()
	params.BlocksPerMonth = 864000
	params.MaxActiveTopicsPerBlock = 1
	params.MaxPageLimit = uint64(100)
	params.MinEpochLength = 1
	params.MinTopicWeight = alloraMath.MustNewDecFromString("0.0001")
	params.TopicRewardAlpha = alloraMath.MustNewDecFromString("0.5")
	params.TopicRewardStakeImportance = alloraMath.OneDec()
	params.TopicRewardFeeRevenueImportance = alloraMath.OneDec()
	s.Require().NoError(s.ParamsKeeper().SetParams(s.Ctx(), params))

	s.WithBlockHeight(0)
	s.Require().NoError(s.TopicKeeper().AddTopicFeeRevenue(s.Ctx(), topic.Id, cosmosMath.NewInt(1000)))
	s.Require().NoError(s.StakingKeeper().SetTopicStake(s.Ctx(), topic.Id, cosmosMath.NewInt(1000)))
	s.Require().NoError(s.TopicKeeper().SetTopic(s.Ctx(), topic.Id, topic))
	s.Require().NoError(s.TopicKeeper().ActivateTopic(
		s.Ctx(),
		topic.Id,
		types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED,
		alloraMath.ZeroDec(),
		alloraMath.ZeroDec(),
	))

	injectedErr := errors.New("injected store write failure")
	failingKeeper, storeService := s.emissionsKeeperWithFailingWrites(types.BlockToActiveTopicsKey, injectedErr)
	// The next epoch-end listing is written first; fail removal of the current listing.
	storeService.enabled = true
	storeService.failOnNth = 2

	block := int64(10)
	s.WithBlockHeight(block)
	eventManager := sdk.NewEventManager()
	weights, _, _, err := rewards.GetAndUpdateActiveTopicWeights(
		s.Ctx().WithEventManager(eventManager),
		failingKeeper,
		block,
	)
	s.Require().NoError(err, "the recovery inactivation should handle the failed reactivation")
	s.Require().Equal(1, storeService.failedWrites)
	s.Require().Empty(weights)

	weightEvents := typedEvents[*types.EventTopicWeightUpdated](eventManager)
	s.Require().Len(weightEvents, 1)
	statusEvents := typedEvents[*types.EventTopicStatusChanged](eventManager)
	s.Require().Len(statusEvents, 1, "a failed reactivation must not emit REACTIVATED before its recovery event")
	event := statusEvents[0]
	s.Require().Equal(topic.Id, event.TopicId)
	s.Require().False(event.IsActive)
	s.Require().Equal(types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_REACTIVATION_ERROR, event.Reason)
	s.Require().True(weightEvents[0].NewWeight.Equal(event.Weight))
	s.Require().True(params.MinTopicWeight.Equal(event.MinWeight))
	s.Require().True(event.ThresholdWeight.IsZero())

	scheduled, err := failingKeeper.GetTopicKeeper().IsTopicScheduled(s.Ctx(), topic.Id)
	s.Require().NoError(err)
	s.Require().False(scheduled)
	active, err := failingKeeper.GetTopicKeeper().IsTopicActive(s.Ctx(), topic.Id)
	s.Require().NoError(err)
	s.Require().False(active)
	activeTopicIds, err := failingKeeper.GetTopicKeeper().GetActiveTopicIds(s.Ctx())
	s.Require().NoError(err)
	s.Require().NotContains(activeTopicIds, topic.Id)
}
