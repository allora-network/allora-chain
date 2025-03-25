package inferencesynthesis_test

import (
	"fmt"
	"math/rand"
	"os"
	"runtime/pprof"
	"testing"
	"time"

	alloraMath "github.com/allora-network/allora-chain/math"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

// TestNetworkInferenceGenerationPerformance tests the performance of the entire
// network inference generation process with maximum inferers and forecasters
func (s *InferenceSynthesisTestSuite) TestNetworkInferenceGenerationPerformance() {
	// Set up test parameters
	const maxInferers = 32
	const maxForecasters = 6
	topicId := uint64(1)
	blockHeight := int64(300)
	blockHeightPreviousLosses := int64(200)

	// Generate test addresses for maximum inferers and forecasters
	require.GreaterOrEqual(s.T(), len(s.addrsStr), maxInferers+maxForecasters,
		"Test suite needs at least %d addresses for this test", maxInferers+maxForecasters)

	infererAddresses := s.addrsStr[:maxInferers]
	forecasterAddresses := s.addrsStr[maxInferers : maxInferers+maxForecasters]

	// Set up topic
	topic := s.mockTopic()
	err := s.emissionsKeeper.SetTopic(s.ctx, topicId, topic)
	require.NoError(s.T(), err)

	// Create and insert inferences
	inferences := &emissionstypes.Inferences{
		Inferences: make([]*emissionstypes.Inference, 0, maxInferers),
	}

	for _, inferer := range infererAddresses {
		value := alloraMath.NewDecFromInt64(int64(1 + rand.Intn(1000)))
		inferences.Inferences = append(inferences.Inferences, &emissionstypes.Inference{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Inferer:     inferer,
			Value:       value,
		})
	}

	err = s.emissionsKeeper.InsertActiveInferences(s.ctx, topicId, blockHeight, *inferences)
	require.NoError(s.T(), err)

	// Create and insert forecasts
	forecasts := &emissionstypes.Forecasts{
		Forecasts: make([]*emissionstypes.Forecast, 0, maxForecasters),
	}

	for _, forecaster := range forecasterAddresses {
		forecastElements := make([]*emissionstypes.ForecastElement, 0, maxInferers)
		for _, inferer := range infererAddresses {
			value := alloraMath.NewDecFromInt64(int64(1 + rand.Intn(2000)))
			forecastElements = append(forecastElements, &emissionstypes.ForecastElement{
				Inferer: inferer,
				Value:   value,
			})
		}

		forecasts.Forecasts = append(forecasts.Forecasts, &emissionstypes.Forecast{
			TopicId:          topicId,
			BlockHeight:      blockHeight,
			Forecaster:       forecaster,
			ForecastElements: forecastElements,
		})
	}

	err = s.emissionsKeeper.InsertActiveForecasts(s.ctx, topicId, blockHeight, *forecasts)
	require.NoError(s.T(), err)

	// Set up previous network losses
	combinedLoss := alloraMath.NewDecFromInt64(int64(500))
	emptyValueBundle := s.mockEmptyValueBundle(combinedLoss)
	err = s.emissionsKeeper.InsertNetworkLossBundleAtBlock(s.ctx, topicId, blockHeightPreviousLosses, emptyValueBundle)
	require.NoError(s.T(), err)

	// Set up regrets for inferers and forecasters
	for _, inferer := range infererAddresses {
		regret := alloraMath.NewDecFromInt64(int64(1 + rand.Intn(500)))
		err = s.emissionsKeeper.SetInfererNetworkRegret(s.ctx, topicId, inferer,
			emissionstypes.TimestampedValue{
				BlockHeight: blockHeight - 100,
				Value:       regret,
			})
		require.NoError(s.T(), err)

		// Set naive inferer regrets
		naiveRegret := alloraMath.NewDecFromInt64(int64(1 + rand.Intn(500)))
		err = s.emissionsKeeper.SetNaiveInfererNetworkRegret(s.ctx, topicId, inferer,
			emissionstypes.TimestampedValue{
				BlockHeight: blockHeight - 100,
				Value:       naiveRegret,
			})
		require.NoError(s.T(), err)

		// Set inferer inclusion counts
		inclusions := 10 + rand.Intn(21)
		for i := 0; i < inclusions; i++ {
			err = s.emissionsKeeper.IncrementCountInfererInclusionsInTopic(s.ctx, topicId, inferer)
			require.NoError(s.T(), err)
		}
	}

	for _, forecaster := range forecasterAddresses {
		regret := alloraMath.NewDecFromInt64(int64(1 + rand.Intn(1000)))
		err = s.emissionsKeeper.SetForecasterNetworkRegret(s.ctx, topicId, forecaster,
			emissionstypes.TimestampedValue{
				BlockHeight: blockHeight - 100,
				Value:       regret,
			})
		require.NoError(s.T(), err)

		// Set forecaster inclusion counts
		inclusions := 15 + rand.Intn(21)
		for i := 0; i < inclusions; i++ {
			err = s.emissionsKeeper.IncrementCountForecasterInclusionsInTopic(s.ctx, topicId, forecaster)
			require.NoError(s.T(), err)
		}
	}

	// Set one-out inferer regrets
	for _, inferer1 := range infererAddresses {
		for _, inferer2 := range infererAddresses {
			regret := alloraMath.NewDecFromInt64(int64(1 + rand.Intn(750)))
			err = s.emissionsKeeper.SetOneOutInfererInfererNetworkRegret(s.ctx, topicId, inferer1, inferer2,
				emissionstypes.TimestampedValue{
					BlockHeight: blockHeight - 100,
					Value:       regret,
				})
			require.NoError(s.T(), err)
		}
	}

	// Set up one-out forecaster regrets
	for _, forecaster := range forecasterAddresses {
		for _, inferer := range infererAddresses {
			regret := alloraMath.NewDecFromInt64(int64(1 + rand.Intn(750)))
			err = s.emissionsKeeper.SetOneOutForecasterInfererNetworkRegret(s.ctx, topicId, forecaster, inferer,
				emissionstypes.TimestampedValue{
					BlockHeight: blockHeight - 100,
					Value:       regret,
				})
			require.NoError(s.T(), err)

			oneInRegret := alloraMath.NewDecFromInt64(int64(1 + rand.Intn(750)))
			err = s.emissionsKeeper.SetOneInForecasterNetworkRegret(s.ctx, topicId, forecaster, inferer,
				emissionstypes.TimestampedValue{
					BlockHeight: blockHeight - 100,
					Value:       oneInRegret,
				})
			require.NoError(s.T(), err)
		}
	}

	// Set up one-out inferer forecaster regrets
	for _, inferer := range infererAddresses {
		for _, forecaster := range forecasterAddresses {
			regret := alloraMath.NewDecFromInt64(int64(1 + rand.Intn(750)))
			err = s.emissionsKeeper.SetOneOutInfererForecasterNetworkRegret(s.ctx, topicId, inferer, forecaster,
				emissionstypes.TimestampedValue{
					BlockHeight: blockHeight - 100,
					Value:       regret,
				})
			require.NoError(s.T(), err)
		}
	}

	// Set up one-out forecaster forecaster regrets
	for _, forecaster1 := range forecasterAddresses {
		for _, forecaster2 := range forecasterAddresses {
			regret := alloraMath.NewDecFromInt64(int64(1 + rand.Intn(750)))
			err = s.emissionsKeeper.SetOneOutForecasterForecasterNetworkRegret(s.ctx, topicId, forecaster1, forecaster2,
				emissionstypes.TimestampedValue{
					BlockHeight: blockHeight - 100,
					Value:       regret,
				})
			require.NoError(s.T(), err)
		}
	}

	// Set up profiling
	cpuProfilePath := "cpu_profile.prof"
	memProfilePath := "mem_profile.prof"

	// Start CPU profiling
	cpuFile, err := os.Create(cpuProfilePath)
	require.NoError(s.T(), err)
	defer cpuFile.Close()
	err = pprof.StartCPUProfile(cpuFile)
	require.NoError(s.T(), err)
	defer pprof.StopCPUProfile()

	// Measure execution time
	startTime := time.Now()
	result, err := inferencesynthesis.GetNetworkInferences(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		&blockHeight,
		false,
	)
	duration := time.Since(startTime)

	// Stop CPU profiling (already handled by defer)

	// Take memory profile after processing
	memFile, err := os.Create(memProfilePath)
	require.NoError(s.T(), err)
	defer memFile.Close()
	err = pprof.WriteHeapProfile(memFile)
	require.NoError(s.T(), err)

	// Output the results
	require.NoError(s.T(), err, "GetNetworkInferences returned an error")
	require.NotNil(s.T(), result, "GetNetworkInferences should return a result")

	// Log the timing results
	s.T().Logf("Full network inference generation with %d inferers and %d forecasters took %v",
		maxInferers, maxForecasters, duration)
	fmt.Printf("\nFull network inference generation with %d inferers and %d forecasters took %v\n",
		maxInferers, maxForecasters, duration)

	// Verify the results
	require.NotNil(s.T(), result.NetworkInferences, "NetworkInferences should not be nil")
	require.Equal(s.T(), topicId, result.NetworkInferences.TopicId, "TopicId should match")

	// Check the various result components
	require.Len(s.T(), result.NetworkInferences.InfererValues, maxInferers,
		"Should have inference values for all inferers")
	require.Len(s.T(), result.NetworkInferences.ForecasterValues, maxForecasters,
		"Should have inference values for all forecasters")
	require.Len(s.T(), result.NetworkInferences.OneOutInfererValues, maxInferers,
		"Should have one-out values for all inferers")
	require.Len(s.T(), result.NetworkInferences.OneOutForecasterValues, maxForecasters,
		"Should have one-out values for all forecasters")

	// Check that OneOutInfererForecasterValues are included
	require.Len(s.T(), result.NetworkInferences.OneOutInfererForecasterValues, maxForecasters,
		"Should have one-out inferer forecaster values for all forecasters")

	// Each forecaster should have entries for all inferers
	for _, oneOutInfererForecasterValue := range result.NetworkInferences.OneOutInfererForecasterValues {
		require.Contains(s.T(), forecasterAddresses, oneOutInfererForecasterValue.Forecaster,
			"Forecaster should be in our list")
		require.Len(s.T(), oneOutInfererForecasterValue.OneOutInfererValues, maxInferers,
			"Each forecaster should have values for all inferers")
	}

	// Assert the execution time is acceptable for block processing
	// This threshold may need to be adjusted based on testing environment and requirements
	require.Less(s.T(), duration, 100*time.Millisecond,
		"Execution time should be less than 100ms to avoid significant impact on block time")
}

// BenchmarkNetworkInferenceGeneration provides a proper Go benchmark for the entire process
func BenchmarkNetworkInferenceGeneration(b *testing.B) {
	// Set up a test suite instance to access its methods and fields
	suite := new(InferenceSynthesisTestSuite)
	suite.SetupTest()

	// Set up test parameters
	const maxInferers = 32
	const maxForecasters = 6
	topicId := uint64(1)
	blockHeight := int64(300)
	blockHeightPreviousLosses := int64(200)

	// Setup code similar to the test above
	// [Setup code omitted for brevity but would be the same as in TestNetworkInferenceGenerationPerformance]

	// Set up topic
	topic := suite.mockTopic()
	err := suite.emissionsKeeper.SetTopic(suite.ctx, topicId, topic)
	require.NoError(b, err)

	// Generate test addresses
	require.GreaterOrEqual(b, len(suite.addrsStr), maxInferers+maxForecasters,
		"Test suite needs at least %d addresses for this benchmark", maxInferers+maxForecasters)

	infererAddresses := suite.addrsStr[:maxInferers]
	forecasterAddresses := suite.addrsStr[maxInferers : maxInferers+maxForecasters]

	// Create and insert inferences
	inferences := &emissionstypes.Inferences{
		Inferences: make([]*emissionstypes.Inference, 0, maxInferers),
	}

	for i, inferer := range infererAddresses {
		value := alloraMath.NewDecFromInt64(int64(100 + i))
		inferences.Inferences = append(inferences.Inferences, &emissionstypes.Inference{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Inferer:     inferer,
			Value:       value,
		})
	}

	err = suite.emissionsKeeper.InsertActiveInferences(suite.ctx, topicId, blockHeight, *inferences)
	require.NoError(b, err)

	// Create and insert forecasts
	forecasts := &emissionstypes.Forecasts{
		Forecasts: make([]*emissionstypes.Forecast, 0, maxForecasters),
	}

	for i, forecaster := range forecasterAddresses {
		forecastElements := make([]*emissionstypes.ForecastElement, 0, maxInferers)
		for j, inferer := range infererAddresses {
			value := alloraMath.NewDecFromInt64(int64(200 + i*10 + j))
			forecastElements = append(forecastElements, &emissionstypes.ForecastElement{
				Inferer: inferer,
				Value:   value,
			})
		}

		forecasts.Forecasts = append(forecasts.Forecasts, &emissionstypes.Forecast{
			TopicId:          topicId,
			BlockHeight:      blockHeight,
			Forecaster:       forecaster,
			ForecastElements: forecastElements,
		})
	}

	err = suite.emissionsKeeper.InsertActiveForecasts(suite.ctx, topicId, blockHeight, *forecasts)
	require.NoError(b, err)

	// Set up previous network losses
	combinedLoss := alloraMath.NewDecFromInt64(int64(500))
	emptyValueBundle := suite.mockEmptyValueBundle(combinedLoss)
	err = suite.emissionsKeeper.InsertNetworkLossBundleAtBlock(suite.ctx, topicId, blockHeightPreviousLosses, emptyValueBundle)
	require.NoError(b, err)

	// Set up regrets for all entities (abbreviated setup - full setup would be the same as test)
	for _, inferer := range infererAddresses {
		regret := alloraMath.NewDecFromInt64(int64(10))
		err = suite.emissionsKeeper.SetInfererNetworkRegret(suite.ctx, topicId, inferer,
			emissionstypes.TimestampedValue{
				BlockHeight: blockHeight - 100,
				Value:       regret,
			})
		require.NoError(b, err)

		for i := 0; i < 20; i++ {
			err = suite.emissionsKeeper.IncrementCountInfererInclusionsInTopic(suite.ctx, topicId, inferer)
			require.NoError(b, err)
		}
	}

	for _, forecaster := range forecasterAddresses {
		regret := alloraMath.NewDecFromInt64(int64(20))
		err = suite.emissionsKeeper.SetForecasterNetworkRegret(suite.ctx, topicId, forecaster,
			emissionstypes.TimestampedValue{
				BlockHeight: blockHeight - 100,
				Value:       regret,
			})
		require.NoError(b, err)

		for i := 0; i < 20; i++ {
			err = suite.emissionsKeeper.IncrementCountForecasterInclusionsInTopic(suite.ctx, topicId, forecaster)
			require.NoError(b, err)
		}
	}

	// Warm up (do one call outside the benchmark)
	_, _ = inferencesynthesis.GetNetworkInferences(
		suite.ctx,
		suite.emissionsKeeper,
		topicId,
		&blockHeight,
		false,
	)

	// Reset the timer for the actual benchmark
	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		_, _ = inferencesynthesis.GetNetworkInferences(
			suite.ctx,
			suite.emissionsKeeper,
			topicId,
			&blockHeight,
			false,
		)
	}

	// Report allocations and object creation
	b.ReportAllocs()
}

// TestNetworkInferenceGenerationScaling measures how performance scales with different
// numbers of inferers and forecasters
func (s *InferenceSynthesisTestSuite) TestNetworkInferenceGenerationScaling() {
	// Define parameter combinations to test
	infererCounts := []int{8, 16, 24, 32}
	forecasterCounts := []int{2, 4, 6}

	// Table to store results
	fmt.Println("\nPerformance scaling for Network Inference Generation:")
	fmt.Println("Inferers | Forecasters | Duration (ms)")
	fmt.Println("---------|-------------|-------------")

	topicId := uint64(1)
	blockHeight := int64(300)
	blockHeightPreviousLosses := int64(200)

	// Ensure we have enough addresses
	require.GreaterOrEqual(s.T(), len(s.addrsStr), 38, "Need at least 38 addresses for all test combinations")

	// Test each combination
	for _, infererCount := range infererCounts {
		for _, forecasterCount := range forecasterCounts {
			// Skip combinations that would exceed available addresses
			if infererCount+forecasterCount > len(s.addrsStr) {
				continue
			}

			// Create a clean environment for each test
			s.SetupTest()

			// Set up topic
			topic := s.mockTopic()
			err := s.emissionsKeeper.SetTopic(s.ctx, topicId, topic)
			require.NoError(s.T(), err)

			// Generate addresses for this combination
			infererAddresses := s.addrsStr[:infererCount]
			forecasterAddresses := s.addrsStr[infererCount : infererCount+forecasterCount]

			// Create and insert inferences
			inferences := &emissionstypes.Inferences{
				Inferences: make([]*emissionstypes.Inference, 0, infererCount),
			}

			for i, inferer := range infererAddresses {
				value := alloraMath.NewDecFromInt64(int64(100 + i))
				inferences.Inferences = append(inferences.Inferences, &emissionstypes.Inference{
					TopicId:     topicId,
					BlockHeight: blockHeight,
					Inferer:     inferer,
					Value:       value,
				})
			}

			err = s.emissionsKeeper.InsertActiveInferences(s.ctx, topicId, blockHeight, *inferences)
			require.NoError(s.T(), err)

			// Create and insert forecasts
			forecasts := &emissionstypes.Forecasts{
				Forecasts: make([]*emissionstypes.Forecast, 0, forecasterCount),
			}

			for i, forecaster := range forecasterAddresses {
				forecastElements := make([]*emissionstypes.ForecastElement, 0, infererCount)
				for j, inferer := range infererAddresses {
					value := alloraMath.NewDecFromInt64(int64(200 + i*10 + j))
					forecastElements = append(forecastElements, &emissionstypes.ForecastElement{
						Inferer: inferer,
						Value:   value,
					})
				}

				forecasts.Forecasts = append(forecasts.Forecasts, &emissionstypes.Forecast{
					TopicId:          topicId,
					BlockHeight:      blockHeight,
					Forecaster:       forecaster,
					ForecastElements: forecastElements,
				})
			}

			err = s.emissionsKeeper.InsertActiveForecasts(s.ctx, topicId, blockHeight, *forecasts)
			require.NoError(s.T(), err)

			// Set up previous network losses
			combinedLoss := alloraMath.NewDecFromInt64(int64(500))
			emptyValueBundle := s.mockEmptyValueBundle(combinedLoss)
			err = s.emissionsKeeper.InsertNetworkLossBundleAtBlock(s.ctx, topicId, blockHeightPreviousLosses, emptyValueBundle)
			require.NoError(s.T(), err)

			// Set up minimal regrets for all entities
			for _, inferer := range infererAddresses {
				regret := alloraMath.NewDecFromInt64(int64(10))
				err = s.emissionsKeeper.SetInfererNetworkRegret(s.ctx, topicId, inferer,
					emissionstypes.TimestampedValue{
						BlockHeight: blockHeight - 100,
						Value:       regret,
					})
				require.NoError(s.T(), err)

				// Set naive inferer regrets
				err = s.emissionsKeeper.SetNaiveInfererNetworkRegret(s.ctx, topicId, inferer,
					emissionstypes.TimestampedValue{
						BlockHeight: blockHeight - 100,
						Value:       regret,
					})
				require.NoError(s.T(), err)

				// Set inferer inclusion counts
				for i := 0; i < 20; i++ {
					err = s.emissionsKeeper.IncrementCountInfererInclusionsInTopic(s.ctx, topicId, inferer)
					require.NoError(s.T(), err)
				}
			}

			for _, forecaster := range forecasterAddresses {
				regret := alloraMath.NewDecFromInt64(int64(20))
				err = s.emissionsKeeper.SetForecasterNetworkRegret(s.ctx, topicId, forecaster,
					emissionstypes.TimestampedValue{
						BlockHeight: blockHeight - 100,
						Value:       regret,
					})
				require.NoError(s.T(), err)

				// Set forecaster inclusion counts
				for i := 0; i < 20; i++ {
					err = s.emissionsKeeper.IncrementCountForecasterInclusionsInTopic(s.ctx, topicId, forecaster)
					require.NoError(s.T(), err)
				}
			}

			// Measure execution time (average of 2 runs)
			var totalDuration time.Duration
			runs := 2

			for i := 0; i < runs; i++ {
				startTime := time.Now()
				_, err := inferencesynthesis.GetNetworkInferences(
					s.ctx,
					s.emissionsKeeper,
					topicId,
					&blockHeight,
					false,
				)
				duration := time.Since(startTime)
				totalDuration += duration
				require.NoError(s.T(), err)
			}

			avgDuration := totalDuration / time.Duration(runs)
			fmt.Printf("%8d | %11d | %11.2f\n", infererCount, forecasterCount, float64(avgDuration.Milliseconds()))
		}
	}
}
