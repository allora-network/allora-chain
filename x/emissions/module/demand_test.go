package module_test

import (
	"time"

	cosmosMath "cosmossdk.io/math"
	state "github.com/allora-network/allora-chain/x/emissions"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/module"
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
	err = s.emissionsKeeper.SetTopicUnmetDemand(s.ctx, createdTopicIds[0], cosmosMath.NewUint(keeper.MIN_TOPIC_DEMAND+1))
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
