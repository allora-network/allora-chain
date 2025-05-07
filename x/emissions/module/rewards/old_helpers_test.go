package rewards_test

import (
	cosmosMath "cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *RewardsTestSuite) FundAccount(amount int64, accAddress sdk.AccAddress) {
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(amount)))
	err := s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, initialStakeCoins)
	s.Require().NoError(err)
	err = s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, accAddress, initialStakeCoins)
	s.Require().NoError(err)
}

func (s *RewardsTestSuite) signValueBundle(
	reputerValueBundle *types.ValueBundle,
	privateKey secp256k1.PrivKey,
) []byte {
	require := s.Require()
	src := make([]byte, 0)
	src, err := reputerValueBundle.XXX_Marshal(src, true)
	require.NoError(err, "Marshall reputer value bundle should not return an error")
	valueBundleSignature, err := privateKey.Sign(src)
	require.NoError(err, "Sign should not return an error")
	return valueBundleSignature
}

func (s *RewardsTestSuite) MintTokensToAddress(address sdk.AccAddress, amount cosmosMath.Int) {
	creatorInitialBalanceCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amount))

	err := s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, creatorInitialBalanceCoins)
	s.Require().NoError(err)
	err = s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, address, creatorInitialBalanceCoins)
	s.Require().NoError(err)
}

func (s *RewardsTestSuite) MintTokensToModule(moduleName string, amount cosmosMath.Int) {
	creatorInitialBalanceCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, amount))
	err := s.bankKeeper.MintCoins(s.ctx, moduleName, creatorInitialBalanceCoins)
	s.Require().NoError(err)
}

func (s *RewardsTestSuite) RegisterAllWorkersOfPayload(topicId types.TopicId, payload *types.InputWorkerDataBundle) {
	worker := payload.InferenceForecastsBundle.Inference.Inferer
	// Define sample OffchainNode information for a worker
	workerInfo := types.OffchainNode{
		Owner:       s.addrsStr[4],
		NodeAddress: worker,
	}
	err := s.emissionsKeeper.InsertWorker(s.ctx, topicId, worker, workerInfo)
	s.Require().NoError(err)
	for _, element := range payload.InferenceForecastsBundle.Forecast.ForecastElements {
		worker = element.Inferer
		workerInfo = types.OffchainNode{
			Owner:       worker,
			NodeAddress: worker,
		}
		err := s.emissionsKeeper.InsertWorker(s.ctx, topicId, worker, workerInfo)
		s.Require().NoError(err)
	}
}

func (s *RewardsTestSuite) RegisterAllReputersOfPayload(topicId types.TopicId, payload *types.InputReputerValueBundle) {
	reputer := payload.ValueBundle.Reputer
	// Define sample OffchainNode information for a worker
	reputerInfo := types.OffchainNode{
		Owner:       reputer,
		NodeAddress: reputer,
	}
	err := s.emissionsKeeper.InsertReputer(s.ctx, topicId, reputer, reputerInfo)
	s.Require().NoError(err)
}

func (s *RewardsTestSuite) setUpTopic(
	blockHeight int64,
	workerIndexes []int,
	reputerIndexes []int,
	stake cosmosMath.Int,
	alphaRegret alloraMath.Dec,
) uint64 {
	return s.setUpTopicWithEpochLength(blockHeight, workerIndexes, reputerIndexes, stake, alphaRegret, 10800)
}

// Creates topic
// Registers workers and reputers from addresses in addrsStr
// Mints stake to reputers
// Funds topic
// Returns topicId
func (s *RewardsTestSuite) setUpTopicWithEpochLength(
	blockHeight int64,
	workerIndexes []int,
	reputerIndexes []int,
	stake cosmosMath.Int,
	alphaRegret alloraMath.Dec,
	epochLength int64,
) uint64 {
	require := s.Require()
	s.ctx = s.ctx.WithBlockHeight(blockHeight)

	// Create topic
	newTopicMsg := &types.CreateNewTopicRequest{
		Creator:                  s.addrsStr[reputerIndexes[0]],
		Metadata:                 "test",
		LossMethod:               "mse",
		EpochLength:              epochLength,
		AllowNegative:            false,
		GroundTruthLag:           epochLength,
		WorkerSubmissionWindow:   min(10, epochLength-2),
		AlphaRegret:              alphaRegret,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    true,
		EnableReputerWhitelist:   true,
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	require.NoError(err)

	// Get Topic Id
	topicId := res.TopicId

	for _, index := range workerIndexes {
		workerRegMsg := &types.RegisterRequest{
			Sender:    s.addrsStr[index],
			TopicId:   topicId,
			IsReputer: false,
			Owner:     s.addrsStr[index],
		}
		_, err := s.msgServer.Register(s.ctx, workerRegMsg)
		require.NoError(err)
	}

	for _, index := range reputerIndexes {
		reputerRegMsg := &types.RegisterRequest{
			Sender:    s.addrsStr[index],
			TopicId:   topicId,
			IsReputer: true,
			Owner:     s.addrsStr[index],
		}
		_, err := s.msgServer.Register(s.ctx, reputerRegMsg)
		require.NoError(err)
	}
	for _, index := range reputerIndexes {
		s.MintTokensToAddress(s.addrs[index], stake)
		_, err := s.msgServer.AddStake(s.ctx, &types.AddStakeRequest{
			Sender:  s.addrsStr[index],
			Amount:  stake,
			TopicId: topicId,
		})
		require.NoError(err)
	}

	var initialStake int64 = 1000
	s.MintTokensToAddress(s.addrs[reputerIndexes[0]], cosmosMath.NewInt(initialStake))
	fundTopicMessage := types.FundTopicRequest{
		Sender:  s.addrsStr[reputerIndexes[0]],
		TopicId: topicId,
		Amount:  cosmosMath.NewInt(initialStake),
	}
	_, err = s.msgServer.FundTopic(s.ctx, &fundTopicMessage)
	require.NoError(err)

	return topicId
}

func (s *RewardsTestSuite) SetParamsForTest() {
	// Setup a sender address
	adminPrivateKey := secp256k1.GenPrivKey()
	adminAddr := sdk.AccAddress(adminPrivateKey.PubKey().Address())
	err := s.emissionsKeeper.AddWhitelistAdmin(s.ctx, adminAddr.String())
	s.Require().NoError(err)

	newParams := &types.OptionalParams{
		MaxTopInferersToReward:  []uint64{24},
		MinEpochLength:          []int64{1},
		RegistrationFee:         []cosmosMath.Int{cosmosMath.NewInt(6)},
		MaxActiveTopicsPerBlock: []uint64{2},
		BlocksPerMonth:          []uint64{864000},
		// Exaggerated TopicRewardAlpha to compensate the effect of latest topic reward alpha vs
		// the dripping effect and separate epochs running multiple topics.
		TopicRewardAlpha: []alloraMath.Dec{alloraMath.MustNewDecFromString("0.999375")},
		// the following fields are not set
		Version:                             nil,
		MaxSerializedMsgLength:              nil,
		MinTopicWeight:                      nil,
		RequiredMinimumStake:                nil,
		RemoveStakeDelayWindow:              nil,
		BetaEntropy:                         nil,
		LearningRate:                        nil,
		MaxGradientThreshold:                nil,
		MinStakeFraction:                    nil,
		MaxUnfulfilledWorkerRequests:        nil,
		MaxUnfulfilledReputerRequests:       nil,
		TopicRewardStakeImportance:          nil,
		TopicRewardFeeRevenueImportance:     nil,
		TaskRewardAlpha:                     nil,
		ValidatorsVsAlloraPercentReward:     nil,
		MaxSamplesToScaleScores:             nil,
		MaxTopForecastersToReward:           nil,
		MaxTopReputersToReward:              nil,
		CreateTopicFee:                      nil,
		GradientDescentMaxIters:             nil,
		DefaultPageLimit:                    nil,
		MaxPageLimit:                        nil,
		MinEpochLengthRecordLimit:           nil,
		PRewardInference:                    nil,
		PRewardForecast:                     nil,
		PRewardReputer:                      nil,
		CRewardInference:                    nil,
		CRewardForecast:                     nil,
		CNorm:                               nil,
		EpsilonReputer:                      nil,
		HalfMaxProcessStakeRemovalsEndBlock: nil,
		DataSendingFee:                      nil,
		EpsilonSafeDiv:                      nil,
		MaxElementsPerForecast:              nil,
		MaxStringLength:                     nil,
		InitialRegretQuantile:               nil,
		PNormSafeDiv:                        nil,
		GlobalWhitelistEnabled:              nil,
		TopicCreatorWhitelistEnabled:        nil,
		MinExperiencedWorkerRegrets:         nil,
		InferenceOutlierDetectionThreshold:  nil,
		InferenceOutlierDetectionAlpha:      nil,
		LambdaInitialScore:                  nil,
		GlobalWorkerWhitelistEnabled:        nil,
		GlobalReputerWhitelistEnabled:       nil,
		GlobalAdminWhitelistAppended:        nil,
		MaxWhitelistInputArrayLength:        nil,
		MinWeightThresholdForStdnorm:        nil,
	}

	updateMsg := &types.UpdateParamsRequest{
		Sender: adminAddr.String(),
		Params: newParams,
	}

	response, err := s.msgServer.UpdateParams(s.ctx, updateMsg)
	s.Require().NoError(err)
	s.Require().NotNil(response)
}

// Generates slice of consecutive numbers
func returnIndexes(start, count int) []int {
	res := make([]int, count)
	for ind := start; ind < start+count; ind++ {
		res[ind-start] = ind
	}
	return res
}

func decPtr(s string) *alloraMath.Dec {
	d := alloraMath.MustNewDecFromString(s)
	return &d
}

// Helper function to compare rewards
func (s *RewardsTestSuite) compareRewards(actual, expected map[uint64]*alloraMath.Dec) bool {
	if len(actual) != len(expected) {
		return false
	}
	for topicID, expectedReward := range expected {
		actualReward, exists := actual[topicID]
		if !exists {
			return false
		}
		inDelta, err := alloraMath.InDelta(*actualReward, *expectedReward, alloraMath.MustNewDecFromString("0.0001"))
		s.Require().NoError(err)
		s.Require().True(inDelta)
		if !inDelta {
			s.T().Logf("Actual   %v not found", actualReward)
			s.T().Logf("Expected %v not found", expectedReward)
			return false
		}
	}
	return true
}

func getIndexesFromValues(values []TestWorkerValue) []int {
	indexes := make([]int, 0)
	for _, value := range values {
		indexes = append(indexes, value.Index)
	}
	return indexes
}

func getWorkerValuesFromIndexes(indexes []int, value ...string) []TestWorkerValue {
	values := make([]TestWorkerValue, 0)
	for i, index := range indexes {
		values = append(values, TestWorkerValue{Index: index, Value: value[i%len(value)]})
	}
	return values
}

func (s *RewardsTestSuite) getReputerValuesFromIndexes(reputerIndexes, workerIndexes []int, value ...string) []map[string]string {
	values := make([]map[string]string, len(reputerIndexes))
	for _, repIdx := range reputerIndexes {
		for i, wrkIdx := range workerIndexes {
			values[repIdx][s.addrsStr[workerIndexes[wrkIdx]]] = value[i%len(value)]
		}
	}
	return values
}

func areTaskRewardsEqualIgnoringTopicId(s *RewardsTestSuite, A []types.TaskReward, B []types.TaskReward) bool {
	if len(A) != len(B) {
		s.Fail("Lengths are different")
	}

	for _, taskRewardA := range A {
		found := false
		for _, taskRewardB := range B {
			if taskRewardA.Address == taskRewardB.Address && taskRewardA.Type == taskRewardB.Type {
				if found {
					s.Fail("Worker %v found twice", taskRewardA.Address)
				}
				found = true
				inDelta, err := alloraMath.InDelta(
					taskRewardA.Reward,
					taskRewardB.Reward,
					alloraMath.MustNewDecFromString("0.00001"),
				)
				if err != nil {
					s.Fail("Error finding out if taskRewardA in delta with taskRewardB",
						taskRewardA.Reward.String(),
						taskRewardB.Reward.String(),
					)
				}
				if !inDelta {
					return false
				}
			}
		}
		if !found {
			s.T().Logf("Worker %v not found", taskRewardA.Address)
			return false
		}
	}

	return true
}
