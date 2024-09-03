package actorutils_test

import (
	"fmt"
	"strconv"
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"
	alloratestutil "github.com/allora-network/allora-chain/test/testutil"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

func TestGetQuantileOfScores(t *testing.T) {
	// Note: unsorted scores. GetQuantileOfScores should sort scores within its scope
	scores := []emissionstypes.Score{
		{TopicId: 0, BlockHeight: 0, Address: "w1", Score: alloraMath.NewDecFromInt64(90)},
		{TopicId: 0, BlockHeight: 0, Address: "w4", Score: alloraMath.NewDecFromInt64(60)},
		{TopicId: 0, BlockHeight: 0, Address: "w3", Score: alloraMath.NewDecFromInt64(70)},
		{TopicId: 0, BlockHeight: 0, Address: "w5", Score: alloraMath.NewDecFromInt64(50)},
		{TopicId: 0, BlockHeight: 0, Address: "w2", Score: alloraMath.NewDecFromInt64(80)},
	}

	quantile := alloraMath.MustNewDecFromString("0.5")
	expectedResult := alloraMath.NewDecFromInt64(70)

	result, err := actorutils.GetQuantileOfScores(scores, quantile)
	require.NoError(t, err)
	require.Equal(t, expectedResult, result)

	quantile = alloraMath.MustNewDecFromString("0.2")
	expectedResult = alloraMath.NewDecFromInt64(58)

	result, err = actorutils.GetQuantileOfScores(scores, quantile)
	require.NoError(t, err)
	expectedInt, err := expectedResult.Int64()
	require.NoError(t, err)
	actualInt, err := result.Int64()
	require.NoError(t, err)
	require.Equal(t, expectedInt, actualInt)
}

func TestGetQuantileOfScoresWithLargerAltDataset(t *testing.T) {
	scoresSorted := []emissionstypes.Score{
		{Score: alloraMath.MustNewDecFromString("0.8"), Address: "w1", BlockHeight: 0, TopicId: 0},
		{Score: alloraMath.MustNewDecFromString("0.7"), Address: "w2", BlockHeight: 0, TopicId: 0},
		{Score: alloraMath.MustNewDecFromString("0.0"), Address: "w9", BlockHeight: 0, TopicId: 0},
		{Score: alloraMath.MustNewDecFromString("0.3"), Address: "w6", BlockHeight: 0, TopicId: 0},
		{Score: alloraMath.MustNewDecFromString("0.4"), Address: "w5", BlockHeight: 0, TopicId: 0},
		{Score: alloraMath.MustNewDecFromString("0.9"), Address: "w0", BlockHeight: 0, TopicId: 0},
		{Score: alloraMath.MustNewDecFromString("0.6"), Address: "w3", BlockHeight: 0, TopicId: 0},
		{Score: alloraMath.MustNewDecFromString("0.5"), Address: "w4", BlockHeight: 0, TopicId: 0},
		{Score: alloraMath.MustNewDecFromString("0.1"), Address: "w8", BlockHeight: 0, TopicId: 0},
		{Score: alloraMath.MustNewDecFromString("0.2"), Address: "w7", BlockHeight: 0, TopicId: 0},
	}
	quantile := alloraMath.MustNewDecFromString("0.2")
	expectedResult := alloraMath.MustNewDecFromString("0.18")

	result, err := actorutils.GetQuantileOfScores(scoresSorted, quantile)
	require.NoError(t, err)
	alloratestutil.InEpsilon5Dec(t, result, expectedResult)
}

func TestGetQuantileOfScoresCsv(t *testing.T) {
	for epoch := 301; epoch < 400; epoch++ {
		epochGet := alloratestutil.GetSortitionSimulatorValuesGetterForEpochs()[epoch]
		topicId := uint64(0)

		nParticipants, err := epochGet("n_participants").UInt64()
		require.NoError(t, err)
		nParticipantsDrawn, err := epochGet("n_participants_drawn").UInt64()
		require.NoError(t, err)

		// populate the data from the csv
		scoresSorted := make([]emissionstypes.Score, nParticipantsDrawn)
		for i := uint64(0); i < nParticipants; i++ {
			participantName := strconv.FormatUint(i, 10)
			active := epochGet(fmt.Sprintf("%s_active", participantName))
			if active.Equal(alloraMath.OneDec()) {
				sortPosition := epochGet(fmt.Sprintf("%s_sort_position_quality_metrics", participantName))
				sortPos, err := sortPosition.UInt64()
				require.NoError(t, err)
				qualityMetric := epochGet(fmt.Sprintf("%s_quality_metric", participantName))
				scoresSorted[sortPos] = emissionstypes.Score{
					TopicId:     topicId,
					Address:     participantName,
					BlockHeight: int64(epoch),
					Score:       qualityMetric,
				}
			}
		}
		for _, score := range scoresSorted {
			require.NotEmpty(t, score)
		}
		expected := epochGet("quality_percentile")
		percentile_to_use := epochGet("percentile")
		quantile, err := percentile_to_use.Quo(alloraMath.NewDecFromInt64(int64(100)))
		require.NoError(t, err)
		result, err := actorutils.GetQuantileOfScores(scoresSorted, quantile)
		require.NoError(t, err)
		alloratestutil.InEpsilon5Dec(t, result, expected)
	}
}
