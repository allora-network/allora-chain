//nolint:exhaustruct,staticcheck
package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

type KeeperTestSuite struct {
	testutil.TestSuite
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, &KeeperTestSuite{
		testutil.NewTestSuite("emissions_keeper"),
	})
}

// createDefaultInfererValues generates a set of inferer values including all test addresses
// This can be used to ensure ValueBundle validation passes when InfererValues cannot be nil
func (s *KeeperTestSuite) createDefaultInfererValues() []*types.WorkerAttributedValue {
	// Create an array to hold all worker attributed values
	infererValues := make([]*types.WorkerAttributedValue, s.LenAccounts())

	// Add a WorkerAttributedValue for each address in the test suite
	for i := range s.LenAccounts() {
		infererValues[i] = &types.WorkerAttributedValue{
			Worker: s.AddrsStr(i),
			Value:  alloraMath.NewDecFromInt64(int64(i + 1)), // Using different values based on index
		}
	}

	return infererValues
}

// STATE MANAGEMENT

func (s *KeeperTestSuite) TestPruneRecordsAfterRewards() {
	// Set infereces, forecasts, and reputations for a topic
	topicId := uint64(1)
	block := types.BlockHeight(100)
	expectedInferences := types.Inferences{
		Inferences: []*types.Inference{
			{
				TopicId:     topicId,
				BlockHeight: block,
				Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(1)}, // Assuming NewDecFromInt64 exists and is appropriate
				Inferer:     s.AddrsStr(0),
			},
			{
				TopicId:     topicId,
				BlockHeight: block,
				Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(2)},
				Inferer:     s.AddrsStr(1),
			},
		},
	}
	nonce := types.Nonce{BlockHeight: block} // Assuming block type cast to int64 if needed
	err := s.WorkerKeeper().InsertActiveInferences(s.Ctx(), topicId, nonce.BlockHeight, expectedInferences)
	s.Require().NoError(err, "Inserting inferences should not fail")

	expectedForecasts := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: block,
				Forecaster:  s.AddrsStr(6),
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: s.AddrsStr(0),
						Value:   alloraMath.MustNewDecFromString("1"),
					},
					{
						Inferer: s.AddrsStr(1),
						Value:   alloraMath.MustNewDecFromString("2"),
					},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: block,
				Forecaster:  s.AddrsStr(7),
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: s.AddrsStr(0),
						Value:   alloraMath.MustNewDecFromString("3"),
					},
					{
						Inferer: s.AddrsStr(1),
						Value:   alloraMath.MustNewDecFromString("4"),
					},
				},
			},
		},
	}
	err = s.WorkerKeeper().InsertActiveForecasts(s.Ctx(), topicId, nonce.BlockHeight, expectedForecasts)
	s.Require().NoError(err)

	reputerLossBundles := []*types.LossBundle{}
	err = s.ReputerLossKeeper().InsertActiveReputerLosses(s.Ctx(), topicId, block, reputerLossBundles)
	s.Require().NoError(err, "InsertActiveReputerLosses should not return an error")

	reputerRequestNonce := &types.ReputerRequestNonce{
		ReputerNonce: &types.Nonce{BlockHeight: block},
	}
	//nolint:exhaustruct
	networkLosses := types.ValueBundle{
		Reputer:             s.AddrsStr(4),
		CombinedValue:       alloraMath.MustNewDecFromString(".0000117005278862668"),
		ReputerRequestNonce: reputerRequestNonce,
		TopicId:             topicId,
		InfererValues:       s.createDefaultInfererValues(),
		NaiveValue:          alloraMath.MustNewDecFromString("0.0"),
	}
	err = s.ReputerLossKeeper().InsertNetworkLossBundleAtBlock(s.Ctx(), topicId, block, networkLosses)
	s.Require().NoError(err, "InsertNetworkLossBundleAtBlock should not return an error")

	// Check if the records are set
	_, err = s.WorkerKeeper().GetInferencesAtBlock(s.Ctx(), topicId, block, false)
	s.Require().NoError(err, "Getting inferences should not fail")
	_, err = s.WorkerKeeper().GetForecastsAtBlock(s.Ctx(), topicId, block)
	s.Require().NoError(err, "Getting forecasts should not fail")
	lossBundles, err := s.ReputerLossKeeper().GetReputerLossBundlesAtBlock(s.Ctx(), topicId, block)
	s.Require().NoError(err, "Getting reputer loss bundles should not fail")
	s.Require().NotNil(lossBundles)
	_, err = s.ReputerLossKeeper().GetNetworkLossBundleAtBlock(s.Ctx(), topicId, block)
	s.Require().NoError(err, "Getting network loss bundle should not fail")

	// Prune records in the subsequent block
	err = s.EmissionsKeeper().PruneRecordsAfterRewards(s.Ctx(), topicId, block+1)
	s.Require().NoError(err, "Pruning records after rewards should not fail")

	// Check if the records are pruned
	inferences, err := s.WorkerKeeper().GetInferencesAtBlock(s.Ctx(), topicId, block, false)
	s.Require().NoError(err, "Getting inferences should not fail")
	s.Require().Empty(inferences.Inferences, "Must be pruned")
	forecasts, err := s.WorkerKeeper().GetForecastsAtBlock(s.Ctx(), topicId, block)
	s.Require().NoError(err, "Getting forecasts should not fail")
	s.Require().Empty(forecasts.Forecasts, "Must be pruned")
	lossbundles, err := s.ReputerLossKeeper().GetReputerLossBundlesAtBlock(s.Ctx(), topicId, block)
	s.Require().NoError(err, "Getting reputer loss bundles should not fail")
	s.Require().Empty(lossbundles, "Must be pruned")
	networkBundles, err := s.ReputerLossKeeper().GetNetworkLossBundleAtBlock(s.Ctx(), topicId, block)
	s.Require().NoError(err, "Getting network loss bundle should not fail but be empty")
	s.Require().Equal(topicId, networkBundles.TopicId, "topic id returned")
	s.Require().Empty(networkBundles.InfererValues, "inferer values is empty")
	s.Require().Empty(networkBundles.ForecasterValues, "forecaster values is empty")
	s.Require().Empty(networkBundles.OneOutInfererValues, "one out inferer values is empty")
	s.Require().Empty(networkBundles.OneOutForecasterValues, "one out forecaster values is empty")
	s.Require().Empty(networkBundles.OneInForecasterValues, "one in forecaster values is empty")
	s.Require().Empty(networkBundles.OneOutInfererForecasterValues, "one out inferer forecaster values is empty")
	s.Require().Equal("0", networkBundles.CombinedValue.String(), "Must be pruned as evidenced by empty combined value")
	s.Require().Equal("0", networkBundles.NaiveValue.String(), "Must be pruned as evidenced by empty naive value")
}

func (s *KeeperTestSuite) TestResetFunctions() {
	ctx := s.Ctx()
	topicId := uint64(1)

	err := s.ReputerLossKeeper().AddActiveReputer(ctx, topicId, s.AddrsStr(0))
	s.Require().NoError(err)
	err = s.WorkerKeeper().AddActiveInferer(ctx, topicId, s.AddrsStr(1))
	s.Require().NoError(err)
	err = s.WorkerKeeper().AddActiveForecaster(ctx, topicId, s.AddrsStr(2))
	s.Require().NoError(err)

	err = s.ReputerLossKeeper().ResetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)
	activeReputers, err := s.ReputerLossKeeper().GetActiveReputersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Empty(activeReputers)

	err = s.WorkerKeeper().ResetActiveWorkersForTopic(ctx, topicId)
	s.Require().NoError(err)
	activeInferers, err := s.WorkerKeeper().GetActiveInferersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Empty(activeInferers)
	activeForecasters, err := s.WorkerKeeper().GetActiveForecastersForTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Empty(activeForecasters)

	err = s.ReputerLossKeeper().ResetReputersIndividualSubmissionsForTopic(ctx, topicId)
	s.Require().NoError(err)

	err = s.WorkerKeeper().ResetWorkersIndividualSubmissionsForTopic(ctx, topicId)
	s.Require().NoError(err)
}

func (s *KeeperTestSuite) TestGetCountInfererInclusionsInTopic() {
	// Initialize the test environment
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	require := s.Require()

	// Define test data
	topicId := uint64(1)
	inferer1 := s.AddrsStr(0)
	inferer2 := s.AddrsStr(1)
	err := k.IncrementCountInfererInclusionsInTopic(ctx, topicId, inferer1)
	require.NoError(err)
	err = k.IncrementCountInfererInclusionsInTopic(ctx, topicId, inferer1)
	require.NoError(err)
	err = k.IncrementCountInfererInclusionsInTopic(ctx, topicId, inferer2)
	require.NoError(err)

	// Assert the expected results
	count, err := k.GetCountInfererInclusionsInTopic(ctx, topicId, inferer1)
	require.NoError(err)
	require.Equal(uint64(2), count)
	count, err = k.GetCountInfererInclusionsInTopic(ctx, topicId, inferer2)
	require.NoError(err)
	require.Equal(uint64(1), count)
}

func (s *KeeperTestSuite) TestGetCountForecasterInclusionsInTopic() {
	// Initialize the test environment
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	require := s.Require()

	// Define test data
	topicId := uint64(1)
	forecaster1 := s.AddrsStr(0)
	forecaster2 := s.AddrsStr(1)
	err := k.IncrementCountForecasterInclusionsInTopic(ctx, topicId, forecaster1)
	require.NoError(err)
	err = k.IncrementCountForecasterInclusionsInTopic(ctx, topicId, forecaster1)
	require.NoError(err)
	err = k.IncrementCountForecasterInclusionsInTopic(ctx, topicId, forecaster2)
	require.NoError(err)

	// Assert the expected results
	count, err := k.GetCountForecasterInclusionsInTopic(ctx, topicId, forecaster1)
	require.NoError(err)
	require.Equal(uint64(2), count)
	count, err = k.GetCountForecasterInclusionsInTopic(ctx, topicId, forecaster2)
	require.NoError(err)
	require.Equal(uint64(1), count)
}

func (s *KeeperTestSuite) TestUpdateNetworkInferencesOutlierMetrics() {
	// Create one topic
	topicId := s.CreateTopic()
	blockHeight := int64(1)

	// Create specific inferences for testing
	inferences := []*types.Inference{
		{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(10)},
			Inferer:     s.AddrsStr(0),
		},
		{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(11)},
			Inferer:     s.AddrsStr(1),
		},
		{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(12)},
			Inferer:     s.AddrsStr(2),
		},
	}

	inferencesWrapper := types.Inferences{Inferences: inferences}
	err := s.WorkerKeeper().InsertActiveInferences(s.Ctx(), topicId, blockHeight, inferencesWrapper)
	s.Require().NoError(err)
	// Test the update function
	err = s.WorkerKeeper().UpdateNetworkInferencesOutlierMetrics(s.Ctx(), topicId, blockHeight)
	s.Require().NoError(err)

	// Verify results
	mad, err := s.TopicKeeper().GetMadInferences(s.Ctx(), topicId)
	s.Require().NoError(err)
	median, err := s.TopicKeeper().GetLastMedianInferences(s.Ctx(), topicId)
	s.Require().NoError(err)
	// Verify expected values
	s.Require().Equal(alloraMath.NewDecFromInt64(1), mad)
	s.Require().Equal(alloraMath.NewDecFromInt64(11), median)

	// Modify a copy of the previous inferences and run it again
	inferencesWrapper.Inferences[0].Values = []alloraMath.Dec{alloraMath.NewDecFromInt64(100)}
	inferencesWrapper.Inferences[1].Values = []alloraMath.Dec{alloraMath.NewDecFromInt64(50)}
	err = s.WorkerKeeper().InsertActiveInferences(s.Ctx(), topicId, blockHeight, inferencesWrapper)
	s.Require().NoError(err)

	err = s.WorkerKeeper().UpdateNetworkInferencesOutlierMetrics(s.Ctx(), topicId, blockHeight)
	s.Require().NoError(err)

	mad, err = s.TopicKeeper().GetMadInferences(s.Ctx(), topicId)
	s.Require().NoError(err)
	median, err = s.TopicKeeper().GetLastMedianInferences(s.Ctx(), topicId)
	s.Require().NoError(err)

	s.Require().Equal(alloraMath.MustNewDecFromString("8.4"), mad)
	s.Require().Equal(alloraMath.NewDecFromInt64(50), median)
}

func (s *KeeperTestSuite) TestFilterOutlierResistantInferences() {
	topicId := s.CreateTopic()

	// Ensure param is set to 11
	params := types.DefaultParams()
	params.InferenceOutlierDetectionThreshold = alloraMath.MustNewDecFromString("11")

	// Set the maximum number of unfulfilled worker nonces via the SetParams method
	err := s.EmissionsKeeper().SetParams(s.Ctx(), params)
	s.Require().NoError(err, "Error retrieving nonces after addition")

	testCases := []struct {
		name          string
		setupMetrics  func()
		inferences    types.Inferences
		expectedCount int
		expectedVals  []alloraMath.Dec
	}{
		{
			name: "filter with median=10, mad=1",
			setupMetrics: func() {
				err := s.TopicKeeper().SetLastMedianInferences(s.Ctx(), topicId, alloraMath.NewDecFromInt64(10))
				s.Require().NoError(err)
				err = s.TopicKeeper().SetMadInferences(s.Ctx(), topicId, alloraMath.NewDecFromInt64(1))
				s.Require().NoError(err)
			},
			inferences: types.Inferences{
				Inferences: []*types.Inference{
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(9)}},  // within bounds (|-1| < 11*1)
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(11)}}, // within bounds (|1| < 11*1)
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(25)}}, // outlier (|15| > 11*1)
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(-5)}}, // outlier (|-15| > 11*1)
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(10)}}, // within bounds (|0| < 11*1)
				},
			},
			expectedCount: 3,
			expectedVals: []alloraMath.Dec{
				alloraMath.NewDecFromInt64(9),
				alloraMath.NewDecFromInt64(11),
				alloraMath.NewDecFromInt64(10),
			},
		},
		{
			name: "filter with median=100, mad=10",
			setupMetrics: func() {
				err := s.TopicKeeper().SetLastMedianInferences(s.Ctx(), topicId, alloraMath.NewDecFromInt64(100))
				s.Require().NoError(err)
				err = s.TopicKeeper().SetMadInferences(s.Ctx(), topicId, alloraMath.NewDecFromInt64(10))
				s.Require().NoError(err)
			},
			inferences: types.Inferences{
				Inferences: []*types.Inference{
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(80)}},  // within bounds (|-20| < 11*10)
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(120)}}, // within bounds (|20| < 11*10)
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(250)}}, // outlier (|150| > 11*10)
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(-50)}}, // outlier (|-150| > 11*10)
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(100)}}, // within bounds (|0| < 11*10)
				},
			},
			expectedCount: 3,
			expectedVals: []alloraMath.Dec{
				alloraMath.NewDecFromInt64(80),
				alloraMath.NewDecFromInt64(120),
				alloraMath.NewDecFromInt64(100),
			},
		},
		{
			name: "zero mad - should return all inferences",
			setupMetrics: func() {
				err := s.TopicKeeper().SetLastMedianInferences(s.Ctx(), topicId, alloraMath.NewDecFromInt64(10))
				s.Require().NoError(err)
				err = s.TopicKeeper().SetMadInferences(s.Ctx(), topicId, alloraMath.ZeroDec())
				s.Require().NoError(err)
			},
			inferences: types.Inferences{
				Inferences: []*types.Inference{
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(10)}},
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(11)}},
				},
			},
			expectedCount: 2,
			expectedVals: []alloraMath.Dec{
				alloraMath.NewDecFromInt64(10),
				alloraMath.NewDecFromInt64(11),
			},
		},
		{
			name: "zero last_median - should return all inferences",
			setupMetrics: func() {
				err := s.TopicKeeper().SetLastMedianInferences(s.Ctx(), topicId, alloraMath.ZeroDec())
				s.Require().NoError(err)
				err = s.TopicKeeper().SetMadInferences(s.Ctx(), topicId, alloraMath.NewDecFromInt64(1))
				s.Require().NoError(err)
			},
			inferences: types.Inferences{
				Inferences: []*types.Inference{
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(5)}},  // within bounds (|5| < 11*1)
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(-5)}}, // within bounds (|-5| < 11*1)
					{Values: []alloraMath.Dec{alloraMath.NewDecFromInt64(15)}}, // outlier (|15| > 11*1)
				},
			},
			expectedCount: 3,
			expectedVals: []alloraMath.Dec{
				alloraMath.NewDecFromInt64(5),
				alloraMath.NewDecFromInt64(-5),
				alloraMath.NewDecFromInt64(15),
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.setupMetrics()

			filtered, err := s.TopicKeeper().FilterOutlierResistantInferences(s.Ctx(), topicId, tc.inferences)
			s.Require().NoError(err)

			s.Require().Len(filtered.Inferences, tc.expectedCount)

			for i, inf := range filtered.Inferences {
				s.Require().Equal(tc.expectedVals[i], inf.Values[0])
			}
		})
	}
}
