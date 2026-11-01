package keeper_test

import (
	cosmosMath "cosmossdk.io/math"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
)

func (s *KeeperTestSuite) TestMoveCoinsFromAlloraRewardsToEcosystemReturnsTrimmedAmountMoved() {
	bk := s.EmissionsKeeper().GetBankingKeeper()
	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(100))
	ecosystemAddr := authtypes.NewModuleAddress(minttypes.EcosystemModuleName)
	before := s.BankKeeper().GetBalance(s.Ctx(), ecosystemAddr, params.DefaultBondDenom)

	moved, err := bk.MoveCoinsFromAlloraRewardsToEcosystem(s.Ctx(), alloraMath.MustNewDecFromString("12.9"))
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.NewInt(12).String(), moved.String())

	after := s.BankKeeper().GetBalance(s.Ctx(), ecosystemAddr, params.DefaultBondDenom)
	s.Require().Equal(moved.String(), after.Amount.Sub(before.Amount).String())
}

func (s *KeeperTestSuite) TestMoveCoinsFromAlloraRewardsToEcosystemSubOneToken() {
	bk := s.EmissionsKeeper().GetBankingKeeper()

	amount := alloraMath.MustNewDecFromString("0.5")
	moved, err := bk.MoveCoinsFromAlloraRewardsToEcosystem(s.Ctx(), amount)
	s.Require().NoError(err, "sub-1-token amounts should no-op after SdkIntTrim rounds to zero")
	s.Require().True(moved.IsZero())
}

func (s *KeeperTestSuite) TestMoveCoinsFromAlloraRewardsToEcosystemZero() {
	bk := s.EmissionsKeeper().GetBankingKeeper()

	amount := alloraMath.ZeroDec()
	moved, err := bk.MoveCoinsFromAlloraRewardsToEcosystem(s.Ctx(), amount)
	s.Require().NoError(err, "zero amount should no-op")
	s.Require().True(moved.IsZero())
}

func (s *KeeperTestSuite) TestMoveCoinsFromAlloraRewardsToEcosystemNegative() {
	bk := s.EmissionsKeeper().GetBankingKeeper()

	amount := alloraMath.MustNewDecFromString("-1.0")
	_, err := bk.MoveCoinsFromAlloraRewardsToEcosystem(s.Ctx(), amount)
	s.Require().Error(err, "negative amount should return error")
	s.Require().Contains(err.Error(), "negative")
}
