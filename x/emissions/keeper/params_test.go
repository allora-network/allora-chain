package keeper_test

import (
	"encoding/binary"

	cosmosMath "cosmossdk.io/math"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestSetParams() {
	ctx := s.Ctx()
	k := s.ParamsKeeper()

	params := types.DefaultParams()
	// Set params
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Check params
	paramsFromKeeper, err := k.GetParams(ctx)
	s.Require().NoError(err)
	s.Require().Equal(params, paramsFromKeeper, "Params should be equal to the set params")
}

func (s *KeeperTestSuite) TestSetGetMaxTopicsPerBlock() {
	ctx := s.Ctx()
	k := s.ParamsKeeper()
	expectedValue := uint64(100)

	// Set the parameter
	params := types.DefaultParams()
	params.MaxActiveTopicsPerBlock = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MaxActiveTopicsPerBlock
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestSetGetRemoveStakeDelayWindow() {
	ctx := s.Ctx()
	k := s.ParamsKeeper()
	expectedValue := types.BlockHeight(50)

	// Set the parameter
	params := types.DefaultParams()
	params.RemoveStakeDelayWindow = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.RemoveStakeDelayWindow
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestSetGetValidatorsVsAlloraPercentReward() {
	ctx := s.Ctx()
	k := s.ParamsKeeper()
	expectedValue := alloraMath.MustNewDecFromString("0.25") // Assume a function to create LegacyDec

	// Set the parameter
	params := types.DefaultParams()
	params.ValidatorsVsAlloraPercentReward = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.ValidatorsVsAlloraPercentReward
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestGetParamsMinTopicUnmetDemand() {
	ctx := s.Ctx()
	k := s.ParamsKeeper()
	expectedValue := alloraMath.NewDecFromInt64(300)

	// Set the parameter
	params := types.DefaultParams()
	params.MinTopicWeight = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MinTopicWeight
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestGetParamsRequiredMinimumStake() {
	ctx := s.Ctx()
	k := s.ParamsKeeper()
	expectedValue, ok := cosmosMath.NewIntFromString("500")
	s.Require().True(ok)

	// Set the parameter
	params := types.DefaultParams()
	params.RequiredMinimumStake = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.RequiredMinimumStake
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestGetParamsMinEpochLength() {
	ctx := s.Ctx()
	k := s.ParamsKeeper()
	expectedValue := types.BlockHeight(720)

	// Set the parameter
	params := types.DefaultParams()
	params.MinEpochLength = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MinEpochLength
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestGetParamsEpsilon() {
	ctx := s.Ctx()
	k := s.ParamsKeeper()
	expectedValue := alloraMath.MustNewDecFromString("0.1234")

	// Set the parameter
	params := types.DefaultParams()
	params.EpsilonReputer = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.EpsilonReputer
	s.Require().True(expectedValue.Equal(actualValue))
}

func (s *KeeperTestSuite) TestGetParamsTopicCreationFee() {
	ctx := s.Ctx()
	k := s.ParamsKeeper()
	expectedValue := cosmosMath.NewInt(1000)

	// Set the parameter
	params := types.DefaultParams()
	params.CreateTopicFee = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.CreateTopicFee
	s.Require().True(expectedValue.Equal(actualValue))
}

func (s *KeeperTestSuite) TestGetParamsRegistrationFee() {
	ctx := s.Ctx()
	k := s.ParamsKeeper()
	expectedValue := cosmosMath.NewInt(500)

	// Set the parameter
	params := types.DefaultParams()
	params.RegistrationFee = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.RegistrationFee
	s.Require().True(expectedValue.Equal(actualValue))
}

func (s *KeeperTestSuite) TestGetParamsMaxSamplesToScaleScores() {
	ctx := s.Ctx()
	k := s.ParamsKeeper()
	expectedValue := uint64(1500)

	// Set the parameter
	params := types.DefaultParams()
	params.MaxSamplesToScaleScores = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MaxSamplesToScaleScores
	s.Require().Equal(expectedValue, actualValue)
}

func (s *KeeperTestSuite) TestGetParamsMaxTopInferersToReward() {
	ctx := s.Ctx()
	k := s.ParamsKeeper()
	expectedValue := uint64(50) // Example expected value

	// Set the parameter
	params := types.DefaultParams()
	params.MaxTopInferersToReward = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MaxTopInferersToReward
	s.Require().Equal(expectedValue, actualValue, "The retrieved MaxTopInferersToReward should match the expected value")
}

func (s *KeeperTestSuite) TestGetParamsMaxTopForecastersToReward() {
	ctx := s.Ctx()
	k := s.ParamsKeeper()
	expectedValue := uint64(50) // Example expected value

	// Set the parameter
	params := types.DefaultParams()
	params.MaxTopForecastersToReward = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter

	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MaxTopForecastersToReward
	s.Require().Equal(expectedValue, actualValue, "The retrieved MaxTopForecastersToReward should match the expected value")
}

func (s *KeeperTestSuite) TestGetParamsMaxTopForecasterElementToSubmit() {
	ctx := s.Ctx()
	k := s.ParamsKeeper()
	expectedValue := uint64(50) // Example expected value

	// Set the parameter
	params := types.DefaultParams()
	params.MaxElementsPerForecast = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter

	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MaxElementsPerForecast
	s.Require().Equal(expectedValue, actualValue, "The retrieved MaxElementsPerForecast should match the expected value")
}

func (s *KeeperTestSuite) TestGetMinEpochLengthRecordLimit() {
	ctx := s.Ctx()
	k := s.ParamsKeeper()
	expectedValue := int64(10)

	// Set the parameter
	params := types.DefaultParams()
	params.MinEpochLengthRecordLimit = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MinEpochLengthRecordLimit
	s.Require().Equal(expectedValue, actualValue, "The retrieved MinEpochLengthRecordLimit should be equal to the expected value")
}

func (s *KeeperTestSuite) TestGetMaxSerializedMsgLength() {
	ctx := s.Ctx()
	k := s.ParamsKeeper()
	expectedValue := int64(2048)

	// Set the parameter
	params := types.DefaultParams()
	params.MaxSerializedMsgLength = expectedValue
	err := k.SetParams(ctx, params)
	s.Require().NoError(err)

	// Get the parameter
	moduleParams, err := k.GetParams(ctx)
	s.Require().NoError(err)
	actualValue := moduleParams.MaxSerializedMsgLength
	s.Require().Equal(expectedValue, actualValue, "The retrieved MaxSerializedMsgLength should be equal to the expected value")
}

// UTILS
func (s *KeeperTestSuite) TestCalcAppropriatePaginationForUint64Cursor() {
	ctx := s.Ctx()
	k := s.ParamsKeeper()

	defaultLimit := uint64(20)
	maxLimit := uint64(50)

	params := types.DefaultParams()
	params.DefaultPageLimit = defaultLimit
	params.MaxPageLimit = maxLimit
	err := k.SetParams(ctx, params)
	s.Require().NoError(err, "Setting default and max limit parameters should not fail")

	paramsActual, err := k.GetParams(ctx)
	s.Require().NoError(err)
	s.Require().Equal(maxLimit, paramsActual.MaxPageLimit, "Max limit should be set correctly")
	s.Require().Equal(defaultLimit, paramsActual.DefaultPageLimit, "Default limit should be set correctly")

	// Test 1: Pagination request is nil
	limit, cursor, err := k.CalcAppropriatePaginationForUint64Cursor(ctx, nil)
	s.Require().NoError(err, "Should handle nil pagination request without error")
	s.Require().Equal(defaultLimit, limit, "Limit should default to the default limit")
	s.Require().Equal(uint64(0), cursor, "Cursor should be 0 when key nil")

	// Test 2: Pagination Key is empty and Limit is zero
	pagination := &types.SimpleCursorPaginationRequest{Key: []byte{}, Limit: 0}
	limit, cursor, err = k.CalcAppropriatePaginationForUint64Cursor(ctx, pagination)
	s.Require().NoError(err, "Should handle empty key and zero limit without error")
	s.Require().Equal(defaultLimit, limit, "Limit should default to the default limit")
	s.Require().Equal(uint64(0), cursor, "Cursor should be 0 when key is empty")

	// Test 3: Valid key and non-zero limit within bounds
	validKey := binary.BigEndian.AppendUint64(nil, uint64(12345)) // Convert 12345 to big-endian byte slice
	pagination = &types.SimpleCursorPaginationRequest{Key: validKey, Limit: 30}
	limit, cursor, err = k.CalcAppropriatePaginationForUint64Cursor(ctx, pagination)
	s.Require().NoError(err, "Handling valid key and valid limit should not fail")
	s.Require().Equal(uint64(30), limit, "Limit should be as specified")
	s.Require().Equal(uint64(12345), cursor, "Cursor should decode correctly from key")

	// Test 4: Limit exceeds maximum limit
	pagination = &types.SimpleCursorPaginationRequest{Key: validKey, Limit: 60}
	limit, _, err = k.CalcAppropriatePaginationForUint64Cursor(ctx, pagination)
	s.Require().NoError(err, "Handling limit exceeding maximum should not fail")
	s.Require().Equal(maxLimit, limit, "Limit should be capped at the maximum limit")
}
