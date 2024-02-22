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
	valid, err := module.IsValidAtPrice(s.ctx, s.appModule, req, price, currentTime)
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
	valid, err := module.IsValidAtPrice(s.ctx, s.appModule, req, price, currentTime)
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
	valid, err := module.IsValidAtPrice(s.ctx, s.appModule, req, price, currentTime)
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
	valid, err := module.IsValidAtPrice(s.ctx, s.appModule, req, price, currentTime)
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
	valid, err := module.IsValidAtPrice(s.ctx, s.appModule, req, price, currentTime)
	s.Require().NoError(err, "IsValidAtPrice should not throw an error")
	s.Require().False(valid, "IsValidAtPrice should return false")
}
