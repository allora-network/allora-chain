package integration_test

import (
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *ExternalTestSuite) TestGetParams() {
	paramsReq := &emissionstypes.QueryParamsRequest{}
	p, err := s.n.QueryClient.Params(
		s.ctx,
		paramsReq,
	)
	s.Require().NoError(err)
	s.Require().NotNil(p)
}

func (s *ExternalTestSuite) TestCreateTopic() (topicId uint64) {
	topicIdStart, err := s.n.QueryClient.GetNextTopicId(
		s.ctx,
		&emissionstypes.QueryNextTopicIdRequest{},
	)
	s.Require().NoError(err)
	s.Require().Greater(topicIdStart.NextTopicId, uint64(0))
	aliceAddr, err := s.n.AliceAcc.Address(params.HumanCoinUnit)
	s.Require().NoError(err)
	createTopicRequest := &emissionstypes.MsgCreateNewTopic{
		Creator:          aliceAddr,
		Metadata:         "ETH 24h Prediction",
		LossLogic:        "bafybeiazhgps7ywkhouwj6m6a7bkq36w3g734kx4b5iqql4n52zf3jjdxa",
		LossMethod:       "loss-calculation-eth.wasm",
		InferenceLogic:   "bafybeigpiwl3o73zvvl6dxdqu7zqcub5mhg65jiky2xqb4rdhfmikswzqm",
		InferenceMethod:  "allora-inference-function.wasm",
		EpochLength:      10800,
		GroundTruthLag:   60,
		DefaultArg:       "ETH",
		Pnorm:            2,
		AlphaRegret:      alloraMath.MustNewDecFromString("3.14"),
		PrewardReputer:   alloraMath.MustNewDecFromString("6.2"),
		PrewardInference: alloraMath.MustNewDecFromString("7.3"),
		PrewardForecast:  alloraMath.MustNewDecFromString("8.4"),
		FTolerance:       alloraMath.MustNewDecFromString("5.5"),
	}
	txResp, err := s.n.Client.BroadcastTx(s.ctx, s.n.AliceAcc, createTopicRequest)
	s.Require().NoError(err)
	createTopicResponse := &emissionstypes.MsgCreateNewTopicResponse{}
	err = txResp.Decode(createTopicResponse)
	s.Require().NoError(err)
	s.Require().Equal(topicIdStart.NextTopicId, createTopicResponse.TopicId)
	topicIdEnd, err := s.n.QueryClient.GetNextTopicId(
		s.ctx,
		&emissionstypes.QueryNextTopicIdRequest{},
	)
	s.Require().NoError(err)
	s.Require().Equal(topicIdEnd.NextTopicId, createTopicResponse.TopicId+1)
	return createTopicResponse.TopicId
}
