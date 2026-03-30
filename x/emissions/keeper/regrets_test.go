package keeper_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestSetAndGetInfererNetworkRegret() {
	ctx := s.Ctx()
	k := s.RegretsKeeper()
	topicId := uint64(1)
	worker := s.AddrsStr(1)
	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}

	// Set Inferer Network Regret
	err := k.SetInfererNetworkRegret(ctx, topicId, worker, regret)
	s.Require().NoError(err)

	// Get Inferer Network Regret
	gotRegret, _, err := k.GetInfererNetworkRegret(ctx, topicId, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
}

func (s *KeeperTestSuite) TestSetAndGetForecasterNetworkRegret() {
	ctx := s.Ctx()
	k := s.RegretsKeeper()
	topicId := uint64(1)
	worker := s.AddrsStr(3) // Assuming sdk.AccAddress is initialized with a string representing the address

	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(20)}

	// Set Forecaster Network Regret
	err := k.SetForecasterNetworkRegret(ctx, topicId, worker, regret)
	s.Require().NoError(err)

	// Get Forecaster Network Regret
	gotRegret, _, err := k.GetForecasterNetworkRegret(ctx, topicId, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
	s.Require().Equal(regret.BlockHeight, gotRegret.BlockHeight)
}

func (s *KeeperTestSuite) TestSetAndGetOneInForecasterNetworkRegret() {
	ctx := s.Ctx()
	k := s.RegretsKeeper()
	topicId := uint64(1)
	forecaster := s.AddrsStr(3)
	inferer := s.AddrsStr(1)

	regret := types.TimestampedValue{BlockHeight: 200, Value: alloraMath.NewDecFromInt64(30)}

	// Set One-In Forecaster Network Regret
	err := k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster, inferer, regret)
	s.Require().NoError(err)

	// Get One-In Forecaster Network Regret
	gotRegret, _, err := k.GetOneInForecasterNetworkRegret(ctx, topicId, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
	s.Require().Equal(regret.BlockHeight, gotRegret.BlockHeight)
}

func (s *KeeperTestSuite) TestDifferentTopicIdsYieldDifferentInfererRegrets() {
	ctx := s.Ctx()
	k := s.RegretsKeeper()
	worker := s.AddrsStr(1)

	// Topic IDs
	topicId1 := s.CreateTopic()
	topicId2 := s.CreateTopic()

	// Zero regret for initial check
	noRegret := types.TimestampedValue{BlockHeight: 0, Value: alloraMath.NewDecFromInt64(0)}

	// Initial regrets should be zero
	gotRegret1, _, err := k.GetInfererNetworkRegret(ctx, topicId1, worker)
	s.Require().NoError(err)
	s.Require().Equal(noRegret, gotRegret1, "Initial regret should be zero for Topic ID 1")

	gotRegret2, _, err := k.GetInfererNetworkRegret(ctx, topicId2, worker)
	s.Require().NoError(err)
	s.Require().Equal(noRegret, gotRegret2, "Initial regret should be zero for Topic ID 2")

	// Regrets to be set
	regret1 := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	regret2 := types.TimestampedValue{BlockHeight: 200, Value: alloraMath.NewDecFromInt64(20)}

	// Set regrets for the same worker under different topic IDs
	err = k.SetInfererNetworkRegret(ctx, topicId1, worker, regret1)
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId2, worker, regret2)
	s.Require().NoError(err)

	// Get and compare regrets after setting them
	gotRegret1, _, err = k.GetInfererNetworkRegret(ctx, topicId1, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret1, gotRegret1)
	s.Require().Equal(regret1.BlockHeight, gotRegret1.BlockHeight)

	gotRegret2, _, err = k.GetInfererNetworkRegret(ctx, topicId2, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret2, gotRegret2)
	s.Require().Equal(regret2.BlockHeight, gotRegret2.BlockHeight)

	s.Require().NotEqual(gotRegret1, gotRegret2, "Regrets from different topics should not be equal")
}

func (s *KeeperTestSuite) TestDifferentTopicIdsYieldDifferentForecasterRegrets() {
	ctx := s.Ctx()
	k := s.RegretsKeeper()
	worker := s.AddrsStr(1)

	// Topic IDs
	topicId1 := s.CreateTopic()
	topicId2 := s.CreateTopic()

	// Regrets
	noRagret := types.TimestampedValue{BlockHeight: 0, Value: alloraMath.NewDecFromInt64(0)}
	regret1 := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	regret2 := types.TimestampedValue{BlockHeight: 200, Value: alloraMath.NewDecFromInt64(20)}

	gotRegret1, _, err := k.GetForecasterNetworkRegret(ctx, topicId1, worker)
	s.Require().NoError(err)
	s.Require().Equal(noRagret, gotRegret1)

	// Set regrets for the same worker under different topic IDs
	err = k.SetForecasterNetworkRegret(ctx, topicId1, worker, regret1)
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId2, worker, regret2)
	s.Require().NoError(err)

	// Get and compare regrets
	gotRegret1, _, err = k.GetForecasterNetworkRegret(ctx, topicId1, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret1, gotRegret1)
	s.Require().Equal(regret1.BlockHeight, gotRegret1.BlockHeight)

	gotRegret2, _, err := k.GetForecasterNetworkRegret(ctx, topicId2, worker)
	s.Require().NoError(err)
	s.Require().Equal(regret2, gotRegret2)
	s.Require().Equal(regret2.BlockHeight, gotRegret2.BlockHeight)

	s.Require().NotEqual(gotRegret1, gotRegret2, "Regrets from different topics should not be equal")
}

func (s *KeeperTestSuite) TestDifferentTopicIdsYieldDifferentOneInForecasterNetworkRegrets() {
	ctx := s.Ctx()
	topicId1 := s.CreateTopic() // Topic 1
	topicId2 := s.CreateTopic() // Topic 2
	k := s.RegretsKeeper()
	forecaster := s.AddrsStr(3)
	inferer := s.AddrsStr(1)

	// Zero regret for initial checks
	noRegret := types.TimestampedValue{BlockHeight: 0, Value: alloraMath.NewDecFromInt64(0)}

	// Initial regrets should be zero
	gotRegret1, _, err := k.GetOneInForecasterNetworkRegret(ctx, topicId1, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(noRegret, gotRegret1, "Initial regret should be zero for Topic ID 1")

	gotRegret2, _, err := k.GetOneInForecasterNetworkRegret(ctx, topicId2, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(noRegret, gotRegret2, "Initial regret should be zero for Topic ID 2")

	// Regrets to be set
	regret1 := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}
	regret2 := types.TimestampedValue{BlockHeight: 200, Value: alloraMath.NewDecFromInt64(20)}

	// Set regrets for the same forecaster-inferer pair under different topic IDs
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId1, forecaster, inferer, regret1)
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId2, forecaster, inferer, regret2)
	s.Require().NoError(err)

	// Get and compare regrets after setting them
	gotRegret1, _, err = k.GetOneInForecasterNetworkRegret(ctx, topicId1, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(regret1, gotRegret1)
	s.Require().Equal(regret1.BlockHeight, gotRegret1.BlockHeight)

	gotRegret2, _, err = k.GetOneInForecasterNetworkRegret(ctx, topicId2, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(regret2, gotRegret2)
	s.Require().Equal(regret2.BlockHeight, gotRegret2.BlockHeight)

	s.Require().NotEqual(gotRegret1, gotRegret2, "Regrets from different topics should not be equal")
}

func (s *KeeperTestSuite) TestSetAndGetNaiveInfererNetworkRegret() {
	ctx := s.Ctx()
	k := s.RegretsKeeper()
	topicId := uint64(1)
	inferer := s.AddrsStr(1)

	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(10)}

	err := k.SetNaiveInfererNetworkRegret(ctx, topicId, inferer, regret)
	s.Require().NoError(err)

	gotRegret, _, err := k.GetNaiveInfererNetworkRegret(ctx, topicId, inferer)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
}

func (s *KeeperTestSuite) TestSetAndGetLatestOneOutInfererInfererNetworkRegret() {
	ctx := s.Ctx()
	k := s.RegretsKeeper()
	topicId := uint64(1)
	inferer1 := s.AddrsStr(1)
	inferer2 := s.AddrsStr(2)

	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(15)}

	err := k.SetOneOutInfererInfererNetworkRegret(ctx, topicId, inferer1, inferer2, regret)
	s.Require().NoError(err)

	gotRegret, _, err := k.GetOneOutInfererInfererNetworkRegret(ctx, topicId, inferer1, inferer2)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
}

func (s *KeeperTestSuite) TestSetAndGetLatestOneOutInfererForecasterNetworkRegret() {
	ctx := s.Ctx()
	k := s.RegretsKeeper()
	topicId := uint64(1)
	inferer := s.AddrsStr(1)
	forecaster := s.AddrsStr(3)

	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(20)}

	err := k.SetOneOutInfererForecasterNetworkRegret(ctx, topicId, inferer, forecaster, regret)
	s.Require().NoError(err)

	gotRegret, _, err := k.GetOneOutInfererForecasterNetworkRegret(ctx, topicId, inferer, forecaster)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
}

func (s *KeeperTestSuite) TestSetAndGetLatestOneOutForecasterInfererNetworkRegret() {
	ctx := s.Ctx()
	k := s.RegretsKeeper()
	topicId := uint64(1)
	forecaster := s.AddrsStr(3)
	inferer := s.AddrsStr(1)

	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(25)}

	err := k.SetOneOutForecasterInfererNetworkRegret(ctx, topicId, forecaster, inferer, regret)
	s.Require().NoError(err)

	gotRegret, _, err := k.GetOneOutForecasterInfererNetworkRegret(ctx, topicId, forecaster, inferer)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
}

func (s *KeeperTestSuite) TestSetAndGetLatestOneOutForecasterForecasterNetworkRegret() {
	ctx := s.Ctx()
	k := s.RegretsKeeper()
	topicId := uint64(1)
	forecaster1 := s.AddrsStr(3)
	forecaster2 := s.AddrsStr(4)

	regret := types.TimestampedValue{BlockHeight: 100, Value: alloraMath.NewDecFromInt64(30)}

	err := k.SetOneOutForecasterForecasterNetworkRegret(ctx, topicId, forecaster1, forecaster2, regret)
	s.Require().NoError(err)

	gotRegret, _, err := k.GetOneOutForecasterForecasterNetworkRegret(ctx, topicId, forecaster1, forecaster2)
	s.Require().NoError(err)
	s.Require().Equal(regret, gotRegret)
}
