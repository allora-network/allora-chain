package keeper_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
)

func (s *KeeperTestSuite) TestMoveCoinsFromAlloraRewardsToEcosystemSubOneToken() {
	bk := s.EmissionsKeeper().GetBankingKeeper()

	amount := alloraMath.MustNewDecFromString("0.5")
	err := bk.MoveCoinsFromAlloraRewardsToEcosystem(s.Ctx(), amount)
	s.Require().NoError(err, "sub-1-token amounts should no-op after SdkIntTrim rounds to zero")
}

func (s *KeeperTestSuite) TestMoveCoinsFromAlloraRewardsToEcosystemZero() {
	bk := s.EmissionsKeeper().GetBankingKeeper()

	amount := alloraMath.ZeroDec()
	err := bk.MoveCoinsFromAlloraRewardsToEcosystem(s.Ctx(), amount)
	s.Require().NoError(err, "zero amount should no-op")
}

func (s *KeeperTestSuite) TestMoveCoinsFromAlloraRewardsToEcosystemNegative() {
	bk := s.EmissionsKeeper().GetBankingKeeper()

	amount := alloraMath.MustNewDecFromString("-1.0")
	err := bk.MoveCoinsFromAlloraRewardsToEcosystem(s.Ctx(), amount)
	s.Require().Error(err, "negative amount should return error")
	s.Require().Contains(err.Error(), "negative")
}
