package module_test

import (
	"time"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	state "github.com/allora-network/allora-chain/x/emissions"
	"github.com/allora-network/allora-chain/x/emissions/module"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *ModuleTestSuite) TestInactivateLowDemandTopicsRemoveTwoTopics() {
	_, err := mockCreateTopics(s, 2)
	s.Require().NoError(err, "mockCreateTopics should not throw an error")
	listTopics, err := module.InactivateLowDemandTopics(s.ctx, s.emissionsKeeper)
	s.Require().NoError(err, "InactivateLowDemandTopics should not throw an error")
	s.Require().Len(listTopics, 0, "InactivateLowDemandTopics should return 0 topics")
	s.Require().Equal([]*state.Topic{}, listTopics, "InactivateLowDemandTopics should return an empty list of topics")
}

func (s *ModuleTestSuite) TestInactivateLowDemandTopicsRemoveOneTopicLeaveOne() {
	createdTopicIds, err := mockCreateTopics(s, 2)
	s.Require().NoError(err, "mockCreateTopics should not throw an error")
	minTopicUnmetDemand, err := s.emissionsKeeper.GetMinTopicUnmetDemand(s.ctx)
	err = s.emissionsKeeper.SetTopicUnmetDemand(s.ctx, createdTopicIds[0], minTopicUnmetDemand.Add(cosmosMath.OneUint()))
	s.Require().NoError(err, "SetTopicUnmetDemand should not throw an error")
	listTopics, err := module.InactivateLowDemandTopics(s.ctx, s.emissionsKeeper)
	s.Require().NoError(err, "InactivateLowDemandTopics should not throw an error")
	s.Require().Len(listTopics, 1, "InactivateLowDemandTopics should return 0 topics")
	s.Require().Equal(createdTopicIds[0], (*listTopics[0]).Id, "InactivateLowDemandTopics should match expected")
}

func (s *ModuleTestSuite) TestIsValidAtPriceTrue() {
	price := cosmosMath.NewUint(100)
	currentTime := uint64(time.Now().UTC().Unix())
	req := state.InferenceRequest{
		Sender:               s.addrsStr[0],
		Nonce:                0,
		TopicId:              1,
		Cadence:              0,
		MaxPricePerInference: price,
		BidAmount:            price,
		LastChecked:          currentTime - 100,
		TimestampValidUntil:  currentTime + 100,
		ExtraData:            []byte("Test"),
	}
	reqId, err := req.GetRequestId()
	s.Require().NoError(err, "GetRequestId should not throw an error")
	err = s.emissionsKeeper.SetRequestDemand(s.ctx, reqId, price.Add(cosmosMath.NewUint(1)))
	s.Require().NoError(err, "SetRequestDemand should not throw an error")
	valid, err := module.IsValidAtPrice(s.ctx, s.emissionsKeeper, req, price, currentTime)
	s.Require().NoError(err, "IsValidAtPrice should not throw an error")
	s.Require().True(valid, "IsValidAtPrice should return true")
}

func (s *ModuleTestSuite) TestIsValidAtPriceFalseCadenceNotYetReady() {
	price := cosmosMath.NewUint(100)
	currentTime := uint64(time.Now().UTC().Unix())
	req := state.InferenceRequest{
		Sender:               s.addrsStr[0],
		Nonce:                0,
		TopicId:              1,
		Cadence:              currentTime + 3000,
		MaxPricePerInference: price,
		BidAmount:            price,
		LastChecked:          currentTime - 100,
		TimestampValidUntil:  currentTime + 100,
		ExtraData:            []byte("Test"),
	}
	reqId, err := req.GetRequestId()
	s.Require().NoError(err, "GetRequestId should not throw an error")
	err = s.emissionsKeeper.SetRequestDemand(s.ctx, reqId, price.Add(cosmosMath.NewUint(1)))
	s.Require().NoError(err, "SetRequestDemand should not throw an error")
	valid, err := module.IsValidAtPrice(s.ctx, s.emissionsKeeper, req, price, currentTime)
	s.Require().NoError(err, "IsValidAtPrice should not throw an error")
	s.Require().False(valid, "IsValidAtPrice should return false")
}

func (s *ModuleTestSuite) TestIsValidAtPriceFalseTimestampNoLongerValid() {
	price := cosmosMath.NewUint(100)
	currentTime := uint64(time.Now().UTC().Unix())
	req := state.InferenceRequest{
		Sender:               s.addrsStr[0],
		Nonce:                0,
		TopicId:              1,
		Cadence:              0,
		MaxPricePerInference: price,
		BidAmount:            price,
		LastChecked:          currentTime - 100,
		TimestampValidUntil:  currentTime - 100,
		ExtraData:            []byte("Test"),
	}
	reqId, err := req.GetRequestId()
	s.Require().NoError(err, "GetRequestId should not throw an error")
	err = s.emissionsKeeper.SetRequestDemand(s.ctx, reqId, price.Add(cosmosMath.NewUint(1)))
	s.Require().NoError(err, "SetRequestDemand should not throw an error")
	valid, err := module.IsValidAtPrice(s.ctx, s.emissionsKeeper, req, price, currentTime)
	s.Require().NoError(err, "IsValidAtPrice should not throw an error")
	s.Require().False(valid, "IsValidAtPrice should return false")
}

func (s *ModuleTestSuite) TestIsValidAtPriceFalseNotEnoughReqDemand() {
	price := cosmosMath.NewUint(100)
	currentTime := uint64(time.Now().UTC().Unix())
	req := state.InferenceRequest{
		Sender:               s.addrsStr[0],
		Nonce:                0,
		TopicId:              1,
		Cadence:              0,
		MaxPricePerInference: price,
		BidAmount:            price,
		LastChecked:          currentTime - 100,
		TimestampValidUntil:  currentTime + 100,
		ExtraData:            []byte("Test"),
	}
	reqId, err := req.GetRequestId()
	s.Require().NoError(err, "GetRequestId should not throw an error")
	err = s.emissionsKeeper.SetRequestDemand(s.ctx, reqId, price.Sub(cosmosMath.NewUint(1)))
	s.Require().NoError(err, "SetRequestDemand should not throw an error")
	valid, err := module.IsValidAtPrice(s.ctx, s.emissionsKeeper, req, price, currentTime)
	s.Require().NoError(err, "IsValidAtPrice should not throw an error")
	s.Require().False(valid, "IsValidAtPrice should return false")
}

func (s *ModuleTestSuite) TestIsValidAtPriceFalseMaxPriceTooHigh() {
	price := cosmosMath.NewUint(100)
	currentTime := uint64(time.Now().UTC().Unix())
	req := state.InferenceRequest{
		Sender:               s.addrsStr[0],
		Nonce:                0,
		TopicId:              1,
		Cadence:              0,
		MaxPricePerInference: price.Sub(cosmosMath.OneUint()),
		BidAmount:            price,
		LastChecked:          currentTime - 100,
		TimestampValidUntil:  currentTime + 100,
		ExtraData:            []byte("Test"),
	}
	reqId, err := req.GetRequestId()
	s.Require().NoError(err, "GetRequestId should not throw an error")
	err = s.emissionsKeeper.SetRequestDemand(s.ctx, reqId, price.Add(cosmosMath.NewUint(1)))
	s.Require().NoError(err, "SetRequestDemand should not throw an error")
	valid, err := module.IsValidAtPrice(s.ctx, s.emissionsKeeper, req, price, currentTime)
	s.Require().NoError(err, "IsValidAtPrice should not throw an error")
	s.Require().False(valid, "IsValidAtPrice should return false")
}

func (s *ModuleTestSuite) TestGetRequestsThatMaxFeesWithOneRequest() {
	price := cosmosMath.NewUint(100)
	currentTime := uint64(time.Now().UTC().Unix())
	req := state.InferenceRequest{
		Sender:               s.addrsStr[0],
		Nonce:                0,
		TopicId:              1,
		Cadence:              0,
		MaxPricePerInference: price,
		BidAmount:            price,
		LastChecked:          currentTime - 100,
		TimestampValidUntil:  currentTime + 100,
		ExtraData:            []byte("Test"),
	}
	reqId, err := req.GetRequestId()
	s.Require().NoError(err, "GetRequestId should not throw an error")
	err = s.emissionsKeeper.SetRequestDemand(s.ctx, reqId, price.Add(cosmosMath.NewUint(1)))
	s.Require().NoError(err, "SetRequestDemand should not throw an error")

	var requestsForGivenTopic []state.InferenceRequest = []state.InferenceRequest{req}

	bestPrice, maxFees, requestsList, err := module.GetRequestsThatMaxFees(s.ctx, s.emissionsKeeper, currentTime, requestsForGivenTopic)
	s.Require().NoError(err, "GetRequestsThatMaxFees should not throw an error")
	s.Require().Equal(price, bestPrice, "GetRequestsThatMaxFees should return the expected best price")
	s.Require().Equal(price, maxFees, "GetRequestsThatMaxFees should return the expected max fees")
	s.Require().Equal(requestsForGivenTopic, requestsList, "GetRequestsThatMaxFees should return the expected requests list")
}

func (s *ModuleTestSuite) TestGetRequestsThatMaxFeesSimple() {
	currentTime := uint64(time.Now().UTC().Unix())
	req0 := state.InferenceRequest{
		Sender:               s.addrsStr[0],
		Nonce:                0,
		TopicId:              1,
		Cadence:              0,
		MaxPricePerInference: cosmosMath.NewUint(100),
		BidAmount:            cosmosMath.NewUint(100),
		LastChecked:          currentTime - 100,
		TimestampValidUntil:  currentTime + 100,
		ExtraData:            []byte("Test1"),
	}
	req1 := state.InferenceRequest{
		Sender:               s.addrsStr[1],
		Nonce:                0,
		TopicId:              1,
		Cadence:              0,
		MaxPricePerInference: cosmosMath.NewUint(200),
		BidAmount:            cosmosMath.NewUint(200),
		LastChecked:          currentTime - 100,
		TimestampValidUntil:  currentTime + 100,
		ExtraData:            []byte("Test2"),
	}
	req2 := state.InferenceRequest{
		Sender:               s.addrsStr[0],
		Nonce:                1,
		TopicId:              1,
		Cadence:              0,
		MaxPricePerInference: cosmosMath.NewUint(300),
		BidAmount:            cosmosMath.NewUint(300),
		LastChecked:          currentTime - 100,
		TimestampValidUntil:  currentTime + 100,
		ExtraData:            []byte("Test3"),
	}
	reqId0, err := req0.GetRequestId()
	s.Require().NoError(err, "GetRequestId should not throw an error")
	reqId1, err := req1.GetRequestId()
	s.Require().NoError(err, "GetRequestId should not throw an error")
	reqId2, err := req2.GetRequestId()
	s.Require().NoError(err, "GetRequestId should not throw an error")
	err = s.emissionsKeeper.SetRequestDemand(s.ctx, reqId0, cosmosMath.NewUint(100))
	s.Require().NoError(err, "SetRequestDemand should not throw an error")
	err = s.emissionsKeeper.SetRequestDemand(s.ctx, reqId1, cosmosMath.NewUint(200))
	s.Require().NoError(err, "SetRequestDemand should not throw an error")
	err = s.emissionsKeeper.SetRequestDemand(s.ctx, reqId2, cosmosMath.NewUint(300))
	s.Require().NoError(err, "SetRequestDemand should not throw an error")

	var requestsForGivenTopic []state.InferenceRequest = []state.InferenceRequest{req0, req1, req2}

	// so we have three requests, one at 100, one at 200, and one at 300
	// at price 100, 3 requests are willing to pay 100 or more tokens, so 100 * 3 = 300
	// at price 200, 2 requests are willing to pay 200 or more tokens, so 200 * 2 = 400
	// at price 300, 1 request is willing to pay 300 or more tokens, so 300 * 1 = 300
	// therefore the best price is 200, and the request that paid 200, and the request willing to pay 300
	// should be returned as the requests to be processed
	bestPrice, maxFees, requestsList, err := module.GetRequestsThatMaxFees(s.ctx, s.emissionsKeeper, currentTime, requestsForGivenTopic)
	s.Require().NoError(err, "GetRequestsThatMaxFees should not throw an error")
	s.Require().Equal(cosmosMath.NewUint(200), bestPrice, "GetRequestsThatMaxFees should return the expected best price")
	s.Require().Equal(cosmosMath.NewUint(400), maxFees, "GetRequestsThatMaxFees should return the expected max fees")
	var expectedRequests []state.InferenceRequest = []state.InferenceRequest{req1, req2}
	s.Require().Equal(expectedRequests, requestsList, "GetRequestsThatMaxFees should return the expected requests list")
}

func (s *ModuleTestSuite) TestGetRequestsThatMaxFeesMultipleRequestsAtSamePrice() {
	currentTime := uint64(time.Now().UTC().Unix())
	req0 := state.InferenceRequest{
		Sender:               s.addrsStr[0],
		Nonce:                0,
		TopicId:              1,
		Cadence:              0,
		MaxPricePerInference: cosmosMath.NewUint(100),
		BidAmount:            cosmosMath.NewUint(100),
		LastChecked:          currentTime - 100,
		TimestampValidUntil:  currentTime + 100,
		ExtraData:            []byte("Test1"),
	}
	req1 := state.InferenceRequest{
		Sender:               s.addrsStr[1],
		Nonce:                0,
		TopicId:              1,
		Cadence:              0,
		MaxPricePerInference: cosmosMath.NewUint(200),
		BidAmount:            cosmosMath.NewUint(200),
		LastChecked:          currentTime - 100,
		TimestampValidUntil:  currentTime + 100,
		ExtraData:            []byte("Test2"),
	}
	req2 := state.InferenceRequest{
		Sender:               s.addrsStr[0],
		Nonce:                1,
		TopicId:              1,
		Cadence:              0,
		MaxPricePerInference: cosmosMath.NewUint(200),
		BidAmount:            cosmosMath.NewUint(200),
		LastChecked:          currentTime - 100,
		TimestampValidUntil:  currentTime + 100,
		ExtraData:            []byte("Test3"),
	}
	req3 := state.InferenceRequest{
		Sender:               s.addrsStr[2],
		Nonce:                0,
		TopicId:              1,
		Cadence:              0,
		MaxPricePerInference: cosmosMath.NewUint(100),
		BidAmount:            cosmosMath.NewUint(100),
		LastChecked:          currentTime - 100,
		TimestampValidUntil:  currentTime + 100,
		ExtraData:            []byte("Test1"),
	}
	req4 := state.InferenceRequest{
		Sender:               s.addrsStr[1],
		Nonce:                8,
		TopicId:              1,
		Cadence:              0,
		MaxPricePerInference: cosmosMath.NewUint(400),
		BidAmount:            cosmosMath.NewUint(400),
		LastChecked:          currentTime - 100,
		TimestampValidUntil:  currentTime + 100,
		ExtraData:            []byte("Test2"),
	}
	req5 := state.InferenceRequest{
		Sender:               s.addrsStr[3],
		Nonce:                4,
		TopicId:              1,
		Cadence:              0,
		MaxPricePerInference: cosmosMath.NewUint(300),
		BidAmount:            cosmosMath.NewUint(300),
		LastChecked:          currentTime - 100,
		TimestampValidUntil:  currentTime + 100,
		ExtraData:            []byte("Test3"),
	}
	reqId0, err := req0.GetRequestId()
	s.Require().NoError(err, "GetRequestId should not throw an error")
	reqId1, err := req1.GetRequestId()
	s.Require().NoError(err, "GetRequestId should not throw an error")
	reqId2, err := req2.GetRequestId()
	s.Require().NoError(err, "GetRequestId should not throw an error")
	reqId3, err := req3.GetRequestId()
	s.Require().NoError(err, "GetRequestId should not throw an error")
	reqId4, err := req4.GetRequestId()
	s.Require().NoError(err, "GetRequestId should not throw an error")
	reqId5, err := req5.GetRequestId()
	s.Require().NoError(err, "GetRequestId should not throw an error")
	err = s.emissionsKeeper.SetRequestDemand(s.ctx, reqId0, cosmosMath.NewUint(100))
	s.Require().NoError(err, "SetRequestDemand should not throw an error")
	err = s.emissionsKeeper.SetRequestDemand(s.ctx, reqId1, cosmosMath.NewUint(200))
	s.Require().NoError(err, "SetRequestDemand should not throw an error")
	err = s.emissionsKeeper.SetRequestDemand(s.ctx, reqId2, cosmosMath.NewUint(200))
	s.Require().NoError(err, "SetRequestDemand should not throw an error")
	err = s.emissionsKeeper.SetRequestDemand(s.ctx, reqId3, cosmosMath.NewUint(100))
	s.Require().NoError(err, "SetRequestDemand should not throw an error")
	err = s.emissionsKeeper.SetRequestDemand(s.ctx, reqId4, cosmosMath.NewUint(400))
	s.Require().NoError(err, "SetRequestDemand should not throw an error")
	err = s.emissionsKeeper.SetRequestDemand(s.ctx, reqId5, cosmosMath.NewUint(300))
	s.Require().NoError(err, "SetRequestDemand should not throw an error")

	var requestsForGivenTopic []state.InferenceRequest = []state.InferenceRequest{req0, req1, req2, req3, req4, req5}

	// so we have five requests, [100, 200, 200, 100, 400, 300]
	// at price 100, 5 requests are willing to pay 100 or more tokens, so 100 * 5 = 500
	// at price 200, 4 requests are willing to pay 200 or more tokens, so 200 * 4 = 800
	// at price 300, 2 requests are willing to pay 300 or more tokens, so 300 * 2 = 600
	// at price 400, 1 request is willing to pay 400 or more tokens, so 400 * 1 = 400
	// therefore the best price is 200, the fees collected are 800, and the list of requests
	// to be processed should be [200, 200, 400, 300]
	bestPrice, maxFees, requestsList, err := module.GetRequestsThatMaxFees(s.ctx, s.emissionsKeeper, currentTime, requestsForGivenTopic)
	s.Require().NoError(err, "GetRequestsThatMaxFees should not throw an error")
	s.Require().Equal(cosmosMath.NewUint(200), bestPrice, "GetRequestsThatMaxFees should return the expected best price")
	s.Require().Equal(cosmosMath.NewUint(800), maxFees, "GetRequestsThatMaxFees should return the expected max fees")
	var expectedRequests []state.InferenceRequest = []state.InferenceRequest{req1, req2, req4, req5}
	s.Require().Equal(expectedRequests, requestsList, "GetRequestsThatMaxFees should return the expected requests list")
}

func (s *ModuleTestSuite) TestSortTopicsByReturnDescWithRandomTiebreakerSimple() {
	var unsortedList []state.Topic = []state.Topic{
		{Id: 1, Metadata: "Test1", DefaultArg: "Test1", InferenceCadence: 0, InferenceLastRan: 0, WeightCadence: 0, WeightLastRan: 0},
		{Id: 2, Metadata: "Test2", DefaultArg: "Test2", InferenceCadence: 0, InferenceLastRan: 0, WeightCadence: 0, WeightLastRan: 0},
		{Id: 3, Metadata: "Test3", DefaultArg: "Test3", InferenceCadence: 0, InferenceLastRan: 0, WeightCadence: 0, WeightLastRan: 0},
		{Id: 4, Metadata: "Test4", DefaultArg: "Test4", InferenceCadence: 0, InferenceLastRan: 0, WeightCadence: 0, WeightLastRan: 0},
		{Id: 5, Metadata: "Test5", DefaultArg: "Test5", InferenceCadence: 0, InferenceLastRan: 0, WeightCadence: 0, WeightLastRan: 0},
	}
	var weights map[uint64]module.PriceAndReturn = map[uint64]module.PriceAndReturn{
		1: {Price: cosmosMath.NewUint(100), Return: cosmosMath.NewUint(100)},
		2: {Price: cosmosMath.NewUint(300), Return: cosmosMath.NewUint(300)},
		3: {Price: cosmosMath.NewUint(700), Return: cosmosMath.NewUint(700)},
		4: {Price: cosmosMath.NewUint(400), Return: cosmosMath.NewUint(400)},
		5: {Price: cosmosMath.NewUint(200), Return: cosmosMath.NewUint(200)},
	}
	sortedList := module.SortTopicsByReturnDescWithRandomTiebreaker(unsortedList, weights, 0)

	s.Require().Equal(len(unsortedList), len(sortedList), "SortTopicsByReturnDescWithRandomTiebreaker should return the same length list")
	s.Require().Equal(uint64(3), sortedList[0].Id, "SortTopicsByReturnDescWithRandomTiebreaker should return the expected sorted list")
	s.Require().Equal(uint64(4), sortedList[1].Id, "SortTopicsByReturnDescWithRandomTiebreaker should return the expected sorted list")
	s.Require().Equal(uint64(2), sortedList[2].Id, "SortTopicsByReturnDescWithRandomTiebreaker should return the expected sorted list")
	s.Require().Equal(uint64(5), sortedList[3].Id, "SortTopicsByReturnDescWithRandomTiebreaker should return the expected sorted list")
	s.Require().Equal(uint64(1), sortedList[4].Id, "SortTopicsByReturnDescWithRandomTiebreaker should return the expected sorted list")
}

func (s *ModuleTestSuite) TestChurnRequestsGetActiveTopicsAndDemandSimple() {
	createdTopicIds, err := mockCreateTopics(s, 2)
	s.Require().NoError(err)
	timeNow := uint64(time.Now().UTC().Unix())
	var initialStake int64 = 1100
	var requestStake0 int64 = 500
	var requestStake1 int64 = 600
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.MintCoins(s.ctx, state.AlloraStakingModuleName, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, state.AlloraStakingModuleName, s.addrs[0], initialStakeCoins)
	r := state.MsgRequestInference{
		Sender: s.addrsStr[0],
		Requests: []*state.RequestInferenceListItem{
			{
				Nonce:                0,
				TopicId:              createdTopicIds[0],
				Cadence:              0,
				MaxPricePerInference: cosmosMath.NewUint(uint64(requestStake0)),
				BidAmount:            cosmosMath.NewUint(uint64(requestStake0)),
				TimestampValidUntil:  timeNow + 100,
				ExtraData:            []byte("Test"),
			},
			{
				Nonce:                1,
				TopicId:              createdTopicIds[1],
				Cadence:              0,
				MaxPricePerInference: cosmosMath.NewUint(uint64(requestStake1)),
				BidAmount:            cosmosMath.NewUint(uint64(requestStake1)),
				TimestampValidUntil:  timeNow + 400,
				ExtraData:            nil,
			},
		},
	}
	_, err = s.msgServer.RequestInference(s.ctx, &r)
	s.Require().NoError(err)

	topics, demand, err := module.ChurnRequestsGetActiveTopicsAndDemand(s.ctx, s.emissionsKeeper, timeNow+20)
	s.Require().NoError(err, "ChurnRequestsGetActiveTopicsAndDemand should not throw an error")
	s.Require().Len(topics, 2, "ChurnRequestsGetActiveTopicsAndDemand should return 2 topics")
	s.Require().Greater(demand.Uint64(), uint64(0), "ChurnRequestsGetActiveTopicsAndDemand should return greater than 0 demand")
}

func (s *ModuleTestSuite) TestDemandFlowEndBlock() {
	createdTopicIds, err := mockCreateTopics(s, 2)
	s.Require().NoError(err)
	timeNow := uint64(time.Now().UTC().Unix())
	var initialStake int64 = 1100
	var requestStake0 int64 = 500
	var requestStake1 int64 = 600
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.MintCoins(s.ctx, state.AlloraStakingModuleName, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, state.AlloraStakingModuleName, s.addrs[0], initialStakeCoins)
	r := state.MsgRequestInference{
		Sender: s.addrsStr[0],
		Requests: []*state.RequestInferenceListItem{
			{
				Nonce:                0,
				TopicId:              createdTopicIds[0],
				Cadence:              0,
				MaxPricePerInference: cosmosMath.NewUint(uint64(requestStake0)),
				BidAmount:            cosmosMath.NewUint(uint64(requestStake0)),
				TimestampValidUntil:  timeNow + 100,
				ExtraData:            []byte("Test"),
			},
			{
				Nonce:                1,
				TopicId:              createdTopicIds[1],
				Cadence:              0,
				MaxPricePerInference: cosmosMath.NewUint(uint64(requestStake1)),
				BidAmount:            cosmosMath.NewUint(uint64(requestStake1)),
				TimestampValidUntil:  timeNow + 400,
				ExtraData:            nil,
			},
		},
	}
	_, err = s.msgServer.RequestInference(s.ctx, &r)
	s.Require().NoError(err)
	reputers, err := mockSomeReputers(s, createdTopicIds[0])
	s.NoError(err)
	workers, err := mockSomeWorkers(s, createdTopicIds[0])
	s.NoError(err)
	err = mockSetWeights(s, createdTopicIds[0], reputers, workers, getConstWeights())
	s.NoError(err, "Error setting weights")
	requestsModuleAccAddr := s.accountKeeper.GetModuleAddress(state.AlloraRequestsModuleName)
	requestsModuleBalanceBefore := s.bankKeeper.GetBalance(s.ctx, requestsModuleAccAddr, params.DefaultBondDenom)
	s.Require().Equal(
		initialStakeCoins.AmountOf(params.DefaultBondDenom),
		requestsModuleBalanceBefore.Amount,
		"Initial balance of requests module should be equal to expected after requests are stored in the state machine")

	lastInferenceRanTopic0Before, err := s.emissionsKeeper.GetTopicInferenceLastRan(s.ctx, createdTopicIds[0])
	s.Require().NoError(err)
	lastInferenceRanTopic1Before, err := s.emissionsKeeper.GetTopicInferenceLastRan(s.ctx, createdTopicIds[1])
	s.Require().NoError(err)

	epochLength, err := s.emissionsKeeper.GetEpochLength(s.ctx)
	s.Require().NoError(err)
	s.ctx = s.ctx.WithBlockHeight(epochLength + 1)

	// make a messaging channel that can pass between threads
	done := make(chan bool)
	go func() {
		// we just made a new multi threaded context that the compiler is aware of
		err = s.appModule.EndBlock(s.ctx)
		s.NoError(err, "EndBlock error")
		// send that letter in the main to whoever is listening to this channel
		done <- true
	}()
	// this thread has halted waiting for someone to send me a love letter
	<-done

	lastInferenceRanTopic0After, err := s.emissionsKeeper.GetTopicInferenceLastRan(s.ctx, createdTopicIds[0])
	s.Require().NoError(err)
	lastInferenceRanTopic1After, err := s.emissionsKeeper.GetTopicInferenceLastRan(s.ctx, createdTopicIds[1])
	s.Require().NoError(err)

	s.Require().Greater(lastInferenceRanTopic0After, lastInferenceRanTopic0Before, "Inference last ran should be greater after EndBlock")
	s.Require().Greater(lastInferenceRanTopic1After, lastInferenceRanTopic1Before, "Inference last ran should be greater after EndBlock")

	requestsModuleBalanceAfter := s.bankKeeper.GetBalance(s.ctx, requestsModuleAccAddr, params.DefaultBondDenom)
	s.Require().Equal(cosmosMath.ZeroInt(), requestsModuleBalanceAfter.Amount, "Balance should be zero after inferences are processed")
}

func (s *ModuleTestSuite) TestDemandFlowEndBlockConsumesSubscriptionLeavesDust() {
	createdTopicIds, err := mockCreateTopics(s, 2)
	s.Require().NoError(err)
	timeNow := uint64(time.Now().UTC().Unix())
	var initialStake int64 = 500
	var requestStake0 int64 = 500
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.MintCoins(s.ctx, state.AlloraStakingModuleName, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, state.AlloraStakingModuleName, s.addrs[0], initialStakeCoins)
	r := state.MsgRequestInference{
		Sender: s.addrsStr[0],
		Requests: []*state.RequestInferenceListItem{
			{
				Nonce:                0,
				TopicId:              createdTopicIds[0],
				Cadence:              61,
				MaxPricePerInference: cosmosMath.NewUint(uint64(requestStake0)),
				BidAmount:            cosmosMath.NewUint(uint64(requestStake0)),
				TimestampValidUntil:  timeNow + 100,
				ExtraData:            []byte("Test"),
			},
		},
	}
	_, err = s.msgServer.RequestInference(s.ctx, &r)
	s.Require().NoError(err)
	reputers, err := mockSomeReputers(s, createdTopicIds[0])
	s.NoError(err)
	workers, err := mockSomeWorkers(s, createdTopicIds[0])
	s.NoError(err)
	err = mockSetWeights(s, createdTopicIds[0], reputers, workers, getConstWeights())
	s.NoError(err, "Error setting weights")
	requestsModuleAccAddr := s.accountKeeper.GetModuleAddress(state.AlloraRequestsModuleName)
	requestsModuleBalanceBefore := s.bankKeeper.GetBalance(s.ctx, requestsModuleAccAddr, params.DefaultBondDenom)
	s.Require().Equal(
		initialStakeCoins.AmountOf(params.DefaultBondDenom),
		requestsModuleBalanceBefore.Amount,
		"Initial balance of requests module should be equal to expected after requests are stored in the state machine")

	mempool, err := s.emissionsKeeper.GetMempool(s.ctx)
	s.Require().NoError(err)
	s.Require().Len(mempool, 1, "Mempool should have exactly 1 request")

	epochLength, err := s.emissionsKeeper.GetEpochLength(s.ctx)
	s.Require().NoError(err)
	s.ctx = s.ctx.WithBlockHeight(epochLength + 1)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second * 61))

	// make a messaging channel that can pass between threads
	done := make(chan bool)
	go func() {
		// we just made a new multi threaded context that the compiler is aware of
		err = s.appModule.EndBlock(s.ctx)
		s.NoError(err, "EndBlock error")
		// send that letter in the main to whoever is listening to this channel
		done <- true
	}()
	// this thread has halted waiting for someone to send me a love letter
	<-done

	mempool, err = s.emissionsKeeper.GetMempool(s.ctx)
	s.Require().NoError(err)
	s.Require().Len(mempool, 0, "Mempool should be empty after EndBlock")
}
