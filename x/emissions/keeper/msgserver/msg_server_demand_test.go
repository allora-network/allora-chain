package msgserver_test

import (
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *MsgServerTestSuite) TestFundTopicSimple() {
	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()
	topicId := s.CreateOneTopic()
	// put some stake in the topic
	err := s.emissionsKeeper.AddStake(s.ctx, topicId, PKS[1].Address().String(), cosmosMath.NewInt(500000))
	s.Require().NoError(err)
	s.emissionsKeeper.InactivateTopic(s.ctx, topicId)
	var initialStake int64 = 1000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, senderAddr, initialStakeCoins)
	r := types.MsgFundTopic{
		Sender:  sender,
		TopicId: topicId,
		Amount:  cosmosMath.NewInt(initialStake),
	}
	params, err := s.emissionsKeeper.GetParams(s.ctx)
	s.Require().NoError(err, "GetParams should not return an error")
	topicWeightBefore, feeRevBefore, err := s.emissionsKeeper.GetCurrentTopicWeight(
		s.ctx,
		r.TopicId,
		10800,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		r.Amount,
	)
	s.Require().NoError(err)
	response, err := s.msgServer.FundTopic(s.ctx, &r)
	s.Require().NoError(err, "RequestInference should not return an error")
	s.Require().NotNil(response, "Response should not be nil")

	// Check if the topic is activated
	res, err := s.emissionsKeeper.IsTopicActive(s.ctx, r.TopicId)
	s.Require().NoError(err)
	s.Require().Equal(true, res, "TopicId is not activated")
	// check that the topic fee revenue has been updated
	topicWeightAfter, feeRevAfter, err := s.emissionsKeeper.GetCurrentTopicWeight(
		s.ctx,
		r.TopicId,
		10800,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		r.Amount,
	)
	s.Require().NoError(err)
	s.Require().True(feeRevAfter.GT(feeRevBefore), "Topic fee revenue should be greater after funding the topic")
	s.Require().True(topicWeightAfter.Gt(topicWeightBefore), "Topic weight should be greater after funding the topic")
}

func (s *MsgServerTestSuite) TestHighWeightForHighFundedTopic() {
	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()
	topicId := s.CreateOneTopic()
	topicId2 := s.CreateOneTopic()
	// put some stake in the topic
	err := s.emissionsKeeper.AddStake(s.ctx, topicId, PKS[1].Address().String(), cosmosMath.NewInt(500000))
	s.Require().NoError(err)
	s.emissionsKeeper.InactivateTopic(s.ctx, topicId)
	err = s.emissionsKeeper.AddStake(s.ctx, topicId2, PKS[1].Address().String(), cosmosMath.NewInt(500000))
	s.Require().NoError(err)
	s.emissionsKeeper.InactivateTopic(s.ctx, topicId2)
	var initialStake int64 = 1000
	var initialStake2 int64 = 10000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake+initialStake2)))
	s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, senderAddr, initialStakeCoins)
	r := types.MsgFundTopic{
		Sender:  sender,
		TopicId: topicId,
		Amount:  cosmosMath.NewInt(initialStake),
	}
	r2 := types.MsgFundTopic{
		Sender:  sender,
		TopicId: topicId2,
		Amount:  cosmosMath.NewInt(initialStake2),
	}
	params, err := s.emissionsKeeper.GetParams(s.ctx)
	s.Require().NoError(err, "GetParams should not return an error")

	response, err := s.msgServer.FundTopic(s.ctx, &r)
	s.Require().NoError(err, "RequestInference should not return an error")
	s.Require().NotNil(response, "Response should not be nil")

	response2, err := s.msgServer.FundTopic(s.ctx, &r2)
	s.Require().NoError(err, "RequestInference should not return an error")
	s.Require().NotNil(response2, "Response should not be nil")

	// Check if the topic is activated
	res, err := s.emissionsKeeper.IsTopicActive(s.ctx, r.TopicId)
	s.Require().NoError(err)
	s.Require().Equal(true, res, "TopicId is not activated")
	// check that the topic fee revenue has been updated
	topicWeight, _, err := s.emissionsKeeper.GetCurrentTopicWeight(
		s.ctx,
		r.TopicId,
		10800,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		r.Amount,
	)
	s.Require().NoError(err)

	topic2Weight, _, err := s.emissionsKeeper.GetCurrentTopicWeight(
		s.ctx,
		r2.TopicId,
		10800,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		r2.Amount,
	)
	s.Require().NoError(err)

	s.Require().Equal(topic2Weight.Gt(topicWeight), true, "Topic1 weight should be greater than Topic2 weight")
}

/*

// test more than one inference in the message
func (s *KeeperTestSuite) TestRequestInferenceBatchSimple() {
	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()
	topicId := s.CreateOneTopic()
	blockNow := s.ctx.BlockHeight()
	var initialStake int64 = 1000
	var requestStake int64 = 500
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, senderAddr, initialStakeCoins)
	requests := []*types.InferenceRequestInbound{
		{
			Nonce:                0,
			TopicId:              topicId,
			Cadence:              0,
			MaxPricePerInference: cosmosMath.NewUint(uint64(requestStake)),
			BidAmount:            cosmosMath.NewUint(uint64(requestStake)),
			BlockValidUntil:      blockNow + 100,
			ExtraData:            []byte("Test"),
		},
		{
			Nonce:                1,
			TopicId:              topicId,
			Cadence:              0,
			MaxPricePerInference: cosmosMath.NewUint(uint64(requestStake)),
			BidAmount:            cosmosMath.NewUint(uint64(requestStake)),
			BlockValidUntil:      blockNow + 400,
			ExtraData:            nil,
		},
	}
	r := []*types.MsgRequestInference{
		{
			Sender:  sender,
			Request: requests[0],
		},
		{
			Sender:  sender,
			Request: requests[1],
		},
	}
	response, err := s.msgServer.RequestInference(s.ctx, r[0])
	s.Require().NoError(err, "RequestInference should not return an error")
	s.Require().NotNil(response.RequestId, "RequestInference should contain the id of the new request")

	// Check updated stake for delegator
	r0 := types.CreateNewInferenceRequestFromListItem(r[0].Sender, r[0].Request)
	requestId, err := r0.GetRequestId()
	s.Require().NoError(err)
	storedRequest, err := s.emissionsKeeper.GetMempoolInferenceRequestById(s.ctx, requestId)
	s.Require().NoError(err)
	s.Require().Equal(r0.Sender, storedRequest.Sender, "Stored request sender should match the request")
	s.Require().Equal(r0.Nonce, storedRequest.Nonce, "Stored request nonce should match the request")
	s.Require().Equal(r0.TopicId, storedRequest.TopicId, "Stored request topic id should match the request")
	s.Require().Equal(r0.Cadence, storedRequest.Cadence, "Stored request cadence should match the request")
	s.Require().Equal(r0.MaxPricePerInference, storedRequest.MaxPricePerInference, "Stored request max price per inference should match the request")
	s.Require().Equal(r0.BidAmount, storedRequest.BidAmount, "Stored request bid amount should match the request")
	s.Require().GreaterOrEqual(storedRequest.BlockLastChecked, blockNow, "LastChecked should be greater than timeNow")
	s.Require().Equal(r0.BlockValidUntil, storedRequest.BlockValidUntil, "Stored request timestamp valid until should match the request")
	s.Require().Equal(r0.ExtraData, storedRequest.ExtraData, "Stored request extra data should match the request")

	response, err = s.msgServer.RequestInference(s.ctx, r[1])
	s.Require().NoError(err, "RequestInference should not return an error")
	s.Require().NotNil(response.RequestId, "RequestInference should contain the id of the new request")
	r1 := types.CreateNewInferenceRequestFromListItem(r[1].Sender, r[1].Request)
	requestId, err = r1.GetRequestId()
	s.Require().NoError(err)
	storedRequest, err = s.emissionsKeeper.GetMempoolInferenceRequestById(s.ctx, requestId)
	s.Require().NoError(err)
	s.Require().Equal(r1.Sender, storedRequest.Sender, "Stored request sender should match the request")
	s.Require().Equal(r1.Nonce, storedRequest.Nonce, "Stored request nonce should match the request")
	s.Require().Equal(r1.TopicId, storedRequest.TopicId, "Stored request topic id should match the request")
	s.Require().Equal(r1.Cadence, storedRequest.Cadence, "Stored request cadence should match the request")
	s.Require().Equal(r1.MaxPricePerInference, storedRequest.MaxPricePerInference, "Stored request max price per inference should match the request")
	s.Require().Equal(r1.BidAmount, storedRequest.BidAmount, "Stored request bid amount should match the request")
	s.Require().GreaterOrEqual(storedRequest.BlockLastChecked, blockNow, "LastChecked should be greater than timeNow")
	s.Require().Equal(r1.BlockValidUntil, storedRequest.BlockValidUntil, "Stored request timestamp valid until should match the request")
	s.Require().Equal(r1.ExtraData, storedRequest.ExtraData, "Stored request extra data should match the request")
}

func (s *KeeperTestSuite) TestRequestInferenceInvalidTopicDoesNotExist() {
	blockNow := s.ctx.BlockHeight()
	senderAddr := sdk.AccAddress(PKS[0].Address()).String()
	r := types.MsgRequestInference{
		Sender: senderAddr,
		Request: &types.InferenceRequestInbound{
			Nonce:                0,
			TopicId:              0,
			Cadence:              0,
			MaxPricePerInference: cosmosMath.NewUint(100),
			BidAmount:            cosmosMath.NewUint(100),
			BlockValidUntil:      blockNow + 10,
			ExtraData:            []byte("Test"),
		},
	}
	_, err := s.msgServer.RequestInference(s.ctx, &r)
	s.Require().ErrorIs(err, types.ErrInvalidTopicId, "RequestInference should return an error when the topic does not exist")
}

func (s *KeeperTestSuite) TestRequestInferenceInvalidBidAmountNotEnoughForPriceSet() {
	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()
	topicId := s.CreateOneTopic()
	blockNow := s.ctx.BlockHeight()
	var initialStake int64 = 1000
	r := types.MsgRequestInference{
		Sender: sender,
		Request: &types.InferenceRequestInbound{
			Nonce:                0,
			TopicId:              topicId,
			Cadence:              0,
			MaxPricePerInference: cosmosMath.NewUint(uint64(initialStake + 20)),
			BidAmount:            cosmosMath.NewUint(uint64(initialStake)),
			BlockValidUntil:      blockNow + 10,
			ExtraData:            []byte("Test"),
		},
	}
	_, err := s.msgServer.RequestInference(s.ctx, &r)
	s.Require().ErrorIs(err, types.ErrInferenceRequestBidAmountLessThanPrice, "RequestInference should return an error when the bid amount is less than the price")
}

func (s *KeeperTestSuite) TestRequestInferenceInvalidSendSameRequestTwice() {
	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()
	topicId := s.CreateOneTopic()
	blockNow := s.ctx.BlockHeight()
	var initialStake int64 = 1000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, senderAddr, initialStakeCoins)
	r := types.MsgRequestInference{
		Sender: sender,
		Request: &types.InferenceRequestInbound{
			Nonce:                0,
			TopicId:              topicId,
			Cadence:              0,
			MaxPricePerInference: cosmosMath.NewUint(uint64(initialStake)),
			BidAmount:            cosmosMath.NewUint(uint64(initialStake)),
			BlockValidUntil:      blockNow + 100,
			ExtraData:            []byte("Test"),
		},
	}
	s.msgServer.RequestInference(s.ctx, &r)

	_, err := s.msgServer.RequestInference(s.ctx, &r)
	s.Require().ErrorIs(err, types.ErrInferenceRequestAlreadyInMempool, "RequestInference should return an error when the request already exists")
}

func (s *KeeperTestSuite) TestRequestInferenceInvalidRequestInThePast() {
	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()
	topicId := s.CreateOneTopic()
	blockNow := s.ctx.BlockHeight()
	var initialStake int64 = 1000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, senderAddr, initialStakeCoins)
	r := types.MsgRequestInference{
		Sender: sender,
		Request: &types.InferenceRequestInbound{
			Nonce:                0,
			TopicId:              topicId,
			Cadence:              0,
			MaxPricePerInference: cosmosMath.NewUint(uint64(initialStake)),
			BidAmount:            cosmosMath.NewUint(uint64(initialStake)),
			BlockValidUntil:      blockNow - 100,
			ExtraData:            []byte("Test"),
		},
	}
	_, err := s.msgServer.RequestInference(s.ctx, &r)
	s.Require().ErrorIs(err, types.ErrInferenceRequestBlockValidUntilInPast, "RequestInference should return an error when the request timestamp is in the past")
}

func (s *KeeperTestSuite) TestRequestInferenceInvalidRequestTooFarInFuture() {
	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()
	topicId := s.CreateOneTopic()
	var initialStake int64 = 1000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, senderAddr, initialStakeCoins)
	r := types.MsgRequestInference{
		Sender: sender,
		Request: &types.InferenceRequestInbound{
			Nonce:                0,
			TopicId:              topicId,
			Cadence:              0,
			MaxPricePerInference: cosmosMath.NewUint(uint64(initialStake)),
			BidAmount:            cosmosMath.NewUint(uint64(initialStake)),
			BlockValidUntil:      math.MaxInt64,
			ExtraData:            []byte("Test"),
		},
	}
	_, err := s.msgServer.RequestInference(s.ctx, &r)
	s.Require().ErrorIs(err, types.ErrInferenceRequestBlockValidUntilTooFarInFuture, "RequestInference should return an error when the request timestamp is too far in the future")
}

func (s *KeeperTestSuite) TestRequestInferenceInvalidRequestCadenceHappensAfterNoLongerValid() {
	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()
	topicId := s.CreateOneTopic()
	blockNow := s.ctx.BlockHeight()
	var initialStake int64 = 1000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, senderAddr, initialStakeCoins)
	r := types.MsgRequestInference{
		Sender: sender,
		Request: &types.InferenceRequestInbound{
			Nonce:                0,
			TopicId:              topicId,
			Cadence:              1000,
			MaxPricePerInference: cosmosMath.NewUint(uint64(initialStake)),
			BidAmount:            cosmosMath.NewUint(uint64(initialStake)),
			BlockValidUntil:      blockNow + 10,
			ExtraData:            []byte("Test"),
		},
	}
	_, err := s.msgServer.RequestInference(s.ctx, &r)
	s.Require().ErrorIs(err, types.ErrInferenceRequestWillNeverBeScheduled, "RequestInference should return an error when the request cadence happens after the request is no longer valid")
}

/*
func (s *KeeperTestSuite) TestRequestInferenceInvalidRequestCadenceTooFast() {
	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()
	s.CreateOneTopic()
	blockNow := s.ctx.BlockHeight()
	var initialStake int64 = 1000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, senderAddr, initialStakeCoins)
	r := types.MsgRequestInference{
		Sender: sender,
		Request: &types.InferenceRequestInbound{
			Nonce:                0,
			TopicId:              0,
			Cadence:              1,
			MaxPricePerInference: cosmosMath.NewUint(uint64(initialStake)),
			BidAmount:            cosmosMath.NewUint(uint64(initialStake)),
			BlockValidUntil:      blockNow + 100,
			ExtraData:            []byte("Test"),
		},
	}
	_, err := s.msgServer.RequestInference(s.ctx, &r)
	s.Require().ErrorIs(err, types.ErrInferenceRequestCadenceTooFast, "RequestInference should return an error when the request cadence is too fast")
}
*/
/*

func (s *KeeperTestSuite) TestRequestInferenceInvalidRequestCadenceTooSlow() {
	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()
	topicId := s.CreateOneTopic()
	blockNow := s.ctx.BlockHeight()
	var initialStake int64 = 1000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, senderAddr, initialStakeCoins)
	r := types.MsgRequestInference{
		Sender: sender,
		Request: &types.InferenceRequestInbound{
			Nonce:                0,
			TopicId:              topicId,
			Cadence:              math.MaxInt64,
			MaxPricePerInference: cosmosMath.NewUint(uint64(initialStake)),
			BidAmount:            cosmosMath.NewUint(uint64(initialStake)),
			BlockValidUntil:      blockNow + 10,
			ExtraData:            []byte("Test"),
		},
	}
	_, err := s.msgServer.RequestInference(s.ctx, &r)
	s.Require().ErrorIs(err, types.ErrInferenceRequestCadenceTooSlow, "RequestInference should return an error when the request cadence is too slow")
}

func (s *KeeperTestSuite) TestRequestInferenceInvalidBidAmountLessThanGlobalMinimum() {
	senderAddr := sdk.AccAddress(PKS[0].Address())
	sender := senderAddr.String()
	topicId := s.CreateOneTopic()
	blockNow := s.ctx.BlockHeight()
	var initialStake int64 = 1000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, initialStakeCoins)
	s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, senderAddr, initialStakeCoins)
	r := types.MsgRequestInference{
		Sender: sender,
		Request: &types.InferenceRequestInbound{
			Nonce:                0,
			TopicId:              topicId,
			Cadence:              0,
			MaxPricePerInference: cosmosMath.ZeroInt(),
			BidAmount:            cosmosMath.ZeroInt(),
			BlockValidUntil:      blockNow + 10,
			ExtraData:            []byte("Test"),
		},
	}
	_, err := s.msgServer.RequestInference(s.ctx, &r)
	s.Require().ErrorIs(err, types.ErrFundAmountTooLow, "RequestInference should return an error when the bid amount is below global minimum threshold")
}
*/
