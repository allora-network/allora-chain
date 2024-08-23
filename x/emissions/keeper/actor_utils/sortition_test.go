package actorutils_test

import (
	"fmt"
	"strconv"
	"testing"

	storetypes "cosmossdk.io/store/types"
	alloraMath "github.com/allora-network/allora-chain/math"
	alloratestutil "github.com/allora-network/allora-chain/test/testutil"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// test helper function reduce code duplication
func setUpSomeWorkers(t *testing.T) (
	testCtx sdk.Context,
	worker1Addr sdk.AccAddress,
	worker2Addr sdk.AccAddress,
	worker3Addr sdk.AccAddress,
	worker4Addr sdk.AccAddress,
	worker5Addr sdk.AccAddress,
) {
	worker1PrivateKey := secp256k1.GenPrivKey()
	worker2PrivateKey := secp256k1.GenPrivKey()
	worker3PrivateKey := secp256k1.GenPrivKey()
	worker4PrivateKey := secp256k1.GenPrivKey()
	worker5PrivateKey := secp256k1.GenPrivKey()

	worker1Addr = sdk.AccAddress(worker1PrivateKey.PubKey().Address())
	worker2Addr = sdk.AccAddress(worker2PrivateKey.PubKey().Address())
	worker3Addr = sdk.AccAddress(worker3PrivateKey.PubKey().Address())
	worker4Addr = sdk.AccAddress(worker4PrivateKey.PubKey().Address())
	worker5Addr = sdk.AccAddress(worker5PrivateKey.PubKey().Address())

	key := storetypes.NewKVStoreKey("test")
	testCtx = testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test")).Ctx

	return testCtx, worker1Addr, worker2Addr, worker3Addr, worker4Addr, worker5Addr
}

// basic sanity test
func TestFindTopNByScoreDesc(t *testing.T) {
	topicId := uint64(0)
	testCtx,
		worker1Addr,
		worker2Addr,
		worker3Addr,
		worker4Addr,
		worker5Addr := setUpSomeWorkers(t)

	reputerScoreEmas := []emissionstypes.Score{
		{TopicId: topicId, BlockHeight: 1, Address: worker1Addr.String(), Score: alloraMath.NewDecFromInt64(90)},
		{TopicId: topicId, BlockHeight: 1, Address: worker2Addr.String(), Score: alloraMath.NewDecFromInt64(40)},
		{TopicId: topicId, BlockHeight: 1, Address: worker3Addr.String(), Score: alloraMath.NewDecFromInt64(80)},
		{TopicId: topicId, BlockHeight: 1, Address: worker4Addr.String(), Score: alloraMath.NewDecFromInt64(20)},
		{TopicId: topicId, BlockHeight: 1, Address: worker5Addr.String(), Score: alloraMath.NewDecFromInt64(100)},
	}

	topActors, _, topActorsBool := actorutils.FindTopNByScoreDesc(testCtx, 3, reputerScoreEmas, 1)
	require.Equal(t, worker5Addr.String(), topActors[0].Address)
	require.Equal(t, worker1Addr.String(), topActors[1].Address)
	require.Equal(t, worker3Addr.String(), topActors[2].Address)

	_, isTop := topActorsBool[worker1Addr.String()]
	require.Equal(t, isTop, true)
	_, isTop = topActorsBool[worker2Addr.String()]
	require.Equal(t, isTop, false)
	_, isTop = topActorsBool[worker3Addr.String()]
	require.Equal(t, isTop, true)
	_, isTop = topActorsBool[worker4Addr.String()]
	require.Equal(t, isTop, false)
	_, isTop = topActorsBool[worker5Addr.String()]
	require.Equal(t, isTop, true)
}

// helper function that handles the expected types of the top actors
func requireIsTop(t *testing.T, expectedTop alloraMath.Dec, isTop bool) {
	t.Helper()
	require.True(
		t, expectedTop.Equal(alloraMath.OneDec()) || expectedTop.Equal(alloraMath.ZeroDec()),
		"expectedTop must be 0 or 1, got %s", expectedTop.String(),
	)
	expectedTopBool := expectedTop.Equal(alloraMath.OneDec())
	require.Equal(t, expectedTopBool, isTop)
}

func TestFindTopNByScoreDescCsv(t *testing.T) {
	for epoch := 301; epoch < 400; epoch++ {
		epochGet := alloratestutil.GetSortitionSimulatorValuesGetterForEpochs()[epoch]
		key := storetypes.NewKVStoreKey("test")
		testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test")).Ctx
		topicId := uint64(0)

		nParticipants, err := epochGet("n_participants").UInt64()
		reputerScoreEmas := make([]emissionstypes.Score, 0)
		for i := 0; uint64(i) < nParticipants; i++ {
			participantName := strconv.Itoa(i)
			reputerScoreEmas = append(reputerScoreEmas, emissionstypes.Score{
				TopicId:     topicId,
				BlockHeight: int64(epoch),
				Address:     participantName,
				Score:       epochGet(fmt.Sprintf("%s_prev_quality_ema", participantName)),
			})
		}

		nParticipantsDrawn, err := epochGet("n_participants_drawn").UInt64()
		require.NoError(t, err)

		_, _, topActorsBool := actorutils.FindTopNByScoreDesc(
			testCtx,
			nParticipantsDrawn,
			reputerScoreEmas,
			1,
		)

		for i := 0; uint64(i) < nParticipants; i++ {
			participantName := strconv.Itoa(i)
			expectedTop := epochGet(fmt.Sprintf("%s_active", participantName))
			_, isTop := topActorsBool[participantName]
			requireIsTop(t, expectedTop, isTop)
		}
	}
}

// Not convinced we should be not throwing errors in FindTopNbyScoreDesc
// but for now instead of throwing errors, we find top N including empty scores
// and just file empty scores at the end of the list
func TestFindTopNByScoreDescWithNils(t *testing.T) {
	topicId := uint64(0)
	testCtx, _, _, worker3Addr, _, worker5Addr := setUpSomeWorkers(t)

	reputerScoreEmas := []emissionstypes.Score{
		{TopicId: topicId, BlockHeight: 1, Address: worker3Addr.String(), Score: alloraMath.NewDecFromInt64(80)},
		{TopicId: topicId, BlockHeight: 1, Address: worker5Addr.String(), Score: alloraMath.NewDecFromInt64(100)},
		{}, //nolint:exhaustruct
	}

	// Actors with nil scores sent to the end
	topActors, allActorsSorted, actorIsTop := actorutils.FindTopNByScoreDesc(testCtx, 3, reputerScoreEmas, 1)
	require.Equal(t, 3, len(topActors))
	require.Equal(t, worker5Addr.String(), topActors[0].Address)
	require.Equal(t, struct{}{}, actorIsTop[worker5Addr.String()])
	require.Equal(t, worker3Addr.String(), topActors[1].Address)
	require.Equal(t, struct{}{}, actorIsTop[worker3Addr.String()])
	require.Equal(t, "", topActors[2].Address)
	require.Equal(t, 3, len(allActorsSorted))
}

func TestGetQuantileOfScores(t *testing.T) {
	scoresSorted := []emissionstypes.Score{
		{TopicId: 0, BlockHeight: 0, Address: "w1", Score: alloraMath.NewDecFromInt64(90)},
		{TopicId: 0, BlockHeight: 0, Address: "w2", Score: alloraMath.NewDecFromInt64(80)},
		{TopicId: 0, BlockHeight: 0, Address: "w3", Score: alloraMath.NewDecFromInt64(70)},
		{TopicId: 0, BlockHeight: 0, Address: "w4", Score: alloraMath.NewDecFromInt64(60)},
		{TopicId: 0, BlockHeight: 0, Address: "w5", Score: alloraMath.NewDecFromInt64(50)},
	}

	quantile := alloraMath.MustNewDecFromString("0.5")
	expectedResult := alloraMath.NewDecFromInt64(70)

	result, err := actorutils.GetQuantileOfScores(scoresSorted, quantile)
	require.NoError(t, err)
	require.Equal(t, expectedResult, result)

	quantile = alloraMath.MustNewDecFromString("0.2")
	expectedResult = alloraMath.NewDecFromInt64(58)

	result, err = actorutils.GetQuantileOfScores(scoresSorted, quantile)
	require.NoError(t, err)
	expectedInt, err := expectedResult.Int64()
	require.NoError(t, err)
	actualInt, err := result.Int64()
	require.NoError(t, err)
	require.Equal(t, expectedInt, actualInt)
}
