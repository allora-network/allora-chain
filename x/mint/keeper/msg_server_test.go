package keeper_test

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/mint/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *MintKeeperTestSuite) TestUpdateParams() {
	params := types.DefaultParams()
	params.MintDenom = "testcoin"

	request := &types.UpdateParamsRequest{
		Sender: s.adminAddr,
		Params: params,
	}
	s.emissionsKeeper.EXPECT().IsWhitelistAdmin(s.ctx, s.adminAddr).Return(true, nil)
	resp, err := s.msgServer.UpdateParams(s.ctx, request)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().Equal(&types.UpdateParamsResponse{}, resp)
}

func (s *MintKeeperTestSuite) TestUpdateParamsInvalidSigner() {
	// Setup a non-whitelisted sender address
	nonAdminPrivateKey := secp256k1.GenPrivKey()
	nonAdminAddr := sdk.AccAddress(nonAdminPrivateKey.PubKey().Address()).String()

	params := types.DefaultParams()
	request := &types.UpdateParamsRequest{
		Sender: nonAdminAddr,
		Params: params,
	}

	s.emissionsKeeper.EXPECT().IsWhitelistAdmin(s.ctx, nonAdminAddr).Return(false, nil)
	resp, err := s.msgServer.UpdateParams(s.ctx, request)
	s.Require().ErrorIs(
		err,
		types.ErrUnauthorized,
	)
	s.Require().Nil(resp)
}

func (s *MintKeeperTestSuite) TestUpdateParamsNonAddressSigner() {
	defaultParams := types.DefaultParams()

	notAnAddress := "not an address lol"
	request := &types.UpdateParamsRequest{
		Sender: notAnAddress,
		Params: defaultParams,
	}
	s.emissionsKeeper.EXPECT().IsWhitelistAdmin(s.ctx, notAnAddress).Return(false, fmt.Errorf("error key encode:"))
	resp, err := s.msgServer.UpdateParams(s.ctx, request)
	s.Require().Error(err)
	s.Require().Nil(resp)
}

func (s *MintKeeperTestSuite) TestUpdateParamsInvalidParamsMintDenom() {
	params := types.DefaultParams()
	params.MintDenom = ""
	request := &types.UpdateParamsRequest{
		Sender: s.adminAddr,
		Params: params,
	}
	s.emissionsKeeper.EXPECT().IsWhitelistAdmin(s.ctx, s.adminAddr).Return(true, nil)
	resp, err := s.msgServer.UpdateParams(s.ctx, request)
	s.Require().Error(err)
	s.Require().Nil(resp)
}

func (s *MintKeeperTestSuite) TestUpdateParamsInvalidParamsMaxSupply() {
	params := types.DefaultParams()
	params.MaxSupply = sdkmath.NewIntFromUint64(0)
	request := &types.UpdateParamsRequest{
		Sender: s.adminAddr,
		Params: params,
	}
	s.emissionsKeeper.EXPECT().IsWhitelistAdmin(s.ctx, s.adminAddr).Return(true, nil)
	resp, err := s.msgServer.UpdateParams(s.ctx, request)
	s.Require().Error(err)
	s.Require().Nil(resp)
}

func (s *MintKeeperTestSuite) TestUpdateParamsInvalidParamsFEmission() {
	params := types.DefaultParams()
	params.FEmission = sdkmath.LegacyNewDec(205)
	request := &types.UpdateParamsRequest{
		Sender: s.adminAddr,
		Params: params,
	}
	s.emissionsKeeper.EXPECT().IsWhitelistAdmin(s.ctx, s.adminAddr).Return(true, nil)
	resp, err := s.msgServer.UpdateParams(s.ctx, request)
	s.Require().Error(err)
	s.Require().Nil(resp)
}

func (s *MintKeeperTestSuite) TestUpdateParamsInvalidParamsOneMonthSmoothingDegree() {
	params := types.DefaultParams()
	params.OneMonthSmoothingDegree = sdkmath.LegacyNewDec(15)
	request := &types.UpdateParamsRequest{
		Sender: s.adminAddr,
		Params: params,
	}
	s.emissionsKeeper.EXPECT().IsWhitelistAdmin(s.ctx, s.adminAddr).Return(true, nil)
	resp, err := s.msgServer.UpdateParams(s.ctx, request)
	s.Require().Error(err)
	s.Require().Nil(resp)
}

func (s *MintKeeperTestSuite) TestUpdateParamsInvalidParamsEcosystemTreasuryPercentOfTotalSupply() {
	params := types.DefaultParams()
	params.EcosystemTreasuryPercentOfTotalSupply = sdkmath.LegacyNewDec(101)
	request := &types.UpdateParamsRequest{
		Sender: s.adminAddr,
		Params: params,
	}
	s.emissionsKeeper.EXPECT().IsWhitelistAdmin(s.ctx, s.adminAddr).Return(true, nil)
	resp, err := s.msgServer.UpdateParams(s.ctx, request)
	s.Require().Error(err)
	s.Require().Nil(resp)
}

func (s *MintKeeperTestSuite) TestUpdateParamsInvalidParamsFoundationTreasuryPercentOfTotalSupply() {
	params := types.DefaultParams()
	params.FoundationTreasuryPercentOfTotalSupply = sdkmath.LegacyNewDec(101)
	request := &types.UpdateParamsRequest{
		Sender: s.adminAddr,
		Params: params,
	}
	s.emissionsKeeper.EXPECT().IsWhitelistAdmin(s.ctx, s.adminAddr).Return(true, nil)
	resp, err := s.msgServer.UpdateParams(s.ctx, request)
	s.Require().Error(err)
	s.Require().Nil(resp)
}

func (s *MintKeeperTestSuite) TestUpdateParamsInvalidParamsParticipantsPercentOfTotalSupply() {
	params := types.DefaultParams()
	params.ParticipantsPercentOfTotalSupply = sdkmath.LegacyNewDec(101)
	request := &types.UpdateParamsRequest{
		Sender: s.adminAddr,
		Params: params,
	}
	s.emissionsKeeper.EXPECT().IsWhitelistAdmin(s.ctx, s.adminAddr).Return(true, nil)
	resp, err := s.msgServer.UpdateParams(s.ctx, request)
	s.Require().Error(err)
	s.Require().Nil(resp)
}

func (s *MintKeeperTestSuite) TestUpdateParamsInvalidParamsInvestorsPercentOfTotalSupply() {
	params := types.DefaultParams()
	params.ParticipantsPercentOfTotalSupply = sdkmath.LegacyNewDec(101)
	request := &types.UpdateParamsRequest{
		Sender: s.adminAddr,
		Params: params,
	}
	s.emissionsKeeper.EXPECT().IsWhitelistAdmin(s.ctx, s.adminAddr).Return(true, nil)
	resp, err := s.msgServer.UpdateParams(s.ctx, request)
	s.Require().Error(err)
	s.Require().Nil(resp)
}
func (s *MintKeeperTestSuite) TestUpdateParamsInvalidParamsTeamPercentOfTotalSupply() {
	params := types.DefaultParams()
	params.TeamPercentOfTotalSupply = sdkmath.LegacyNewDec(101)
	request := &types.UpdateParamsRequest{
		Sender: s.adminAddr,
		Params: params,
	}
	s.emissionsKeeper.EXPECT().IsWhitelistAdmin(s.ctx, s.adminAddr).Return(true, nil)
	resp, err := s.msgServer.UpdateParams(s.ctx, request)
	s.Require().Error(err)
	s.Require().Nil(resp)
}
func (s *MintKeeperTestSuite) TestUpdateParamsInvalidParamsMaximumMonthlyPercentageYield() {
	params := types.DefaultParams()
	params.MaximumMonthlyPercentageYield = sdkmath.LegacyNewDec(101)
	request := &types.UpdateParamsRequest{
		Sender: s.adminAddr,
		Params: params,
	}
	s.emissionsKeeper.EXPECT().IsWhitelistAdmin(s.ctx, s.adminAddr).Return(true, nil)
	resp, err := s.msgServer.UpdateParams(s.ctx, request)
	s.Require().Error(err)
	s.Require().Nil(resp)
}
