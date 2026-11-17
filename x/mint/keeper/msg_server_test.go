package keeper_test

import (
	"fmt"

	alloraMath "github.com/allora-network/allora-chain/math"

	sdkmath "cosmossdk.io/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/allora-network/allora-chain/x/mint/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *MintKeeperTestSuite) TestUpdateParams() {
	params := types.DefaultParams()
	params.MintDenom = "testcoin"

	request := &types.UpdateParamsRequest{
		Sender:                    s.adminAddr,
		Params:                    params,
		BlocksPerMonth:            525590,
		RecalculateTargetEmission: false,
	}
	expectedEmissionsParams := emissionstypes.DefaultParams()
	s.emissionsKeeper.EXPECT().GetParams(s.ctx).Return(expectedEmissionsParams, nil)
	expectedEmissionsParams.BlocksPerMonth = 525590
	s.emissionsKeeper.EXPECT().SetParams(s.ctx, expectedEmissionsParams).Return(nil)
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
		Sender:                    nonAdminAddr,
		Params:                    params,
		BlocksPerMonth:            525590,
		RecalculateTargetEmission: false,
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
		Sender:                    notAnAddress,
		Params:                    defaultParams,
		BlocksPerMonth:            525590,
		RecalculateTargetEmission: false,
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
		Sender:                    s.adminAddr,
		Params:                    params,
		BlocksPerMonth:            525590,
		RecalculateTargetEmission: false,
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
		Sender:                    s.adminAddr,
		Params:                    params,
		BlocksPerMonth:            525590,
		RecalculateTargetEmission: false,
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
		Sender:                    s.adminAddr,
		Params:                    params,
		BlocksPerMonth:            525590,
		RecalculateTargetEmission: false,
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
		Sender:                    s.adminAddr,
		Params:                    params,
		BlocksPerMonth:            525590,
		RecalculateTargetEmission: false,
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
		Sender:                    s.adminAddr,
		Params:                    params,
		BlocksPerMonth:            525590,
		RecalculateTargetEmission: false,
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
		Sender:                    s.adminAddr,
		Params:                    params,
		BlocksPerMonth:            525590,
		RecalculateTargetEmission: false,
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
		Sender:                    s.adminAddr,
		Params:                    params,
		BlocksPerMonth:            525590,
		RecalculateTargetEmission: false,
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
		Sender:                    s.adminAddr,
		Params:                    params,
		BlocksPerMonth:            525590,
		RecalculateTargetEmission: false,
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
		Sender:                    s.adminAddr,
		Params:                    params,
		BlocksPerMonth:            525590,
		RecalculateTargetEmission: false,
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
		Sender:                    s.adminAddr,
		Params:                    params,
		RecalculateTargetEmission: false,
		BlocksPerMonth:            1337,
	}
	s.emissionsKeeper.EXPECT().IsWhitelistAdmin(s.ctx, s.adminAddr).Return(true, nil)
	resp, err := s.msgServer.UpdateParams(s.ctx, request)
	s.Require().Error(err)
	s.Require().Nil(resp)
}

func (s *MintKeeperTestSuite) TestUpdateParamsBlocksPerMonth() {
	params := types.DefaultParams()
	request := &types.UpdateParamsRequest{
		Sender:                    s.adminAddr,
		Params:                    params,
		BlocksPerMonth:            1337,
		RecalculateTargetEmission: false,
	}
	s.emissionsKeeper.EXPECT().IsWhitelistAdmin(s.ctx, s.adminAddr).Return(true, nil)
	expectedEmissionsParams := emissionstypes.DefaultParams()
	s.emissionsKeeper.EXPECT().GetParams(s.ctx).Return(expectedEmissionsParams, nil)
	expectedEmissionsParams.BlocksPerMonth = 1337
	s.emissionsKeeper.EXPECT().SetParams(s.ctx, expectedEmissionsParams).Return(nil)
	resp, err := s.msgServer.UpdateParams(s.ctx, request)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().Equal(&types.UpdateParamsResponse{}, resp)
}

func (s *MintKeeperTestSuite) TestUpdateParamsRecalculateTargetEmission() {
	ecosystemTokensMinted := sdkmath.NewInt(1000000)
	err := s.mintKeeper.EcosystemTokensMinted.Set(s.ctx, ecosystemTokensMinted)
	s.Require().NoError(err)
	startEmission := sdkmath.LegacyNewDec(100)
	err = s.mintKeeper.PreviousRewardEmissionPerUnitStakedToken.Set(s.ctx, startEmission)
	s.Require().NoError(err)
	params := types.DefaultParams()
	request := &types.UpdateParamsRequest{
		Sender:                    s.adminAddr,
		Params:                    params,
		BlocksPerMonth:            1337,
		RecalculateTargetEmission: true,
	}

	s.emissionsKeeper.EXPECT().IsWhitelistAdmin(s.ctx, s.adminAddr).Return(true, nil)
	expectedEmissionsParams := emissionstypes.DefaultParams()
	s.emissionsKeeper.EXPECT().GetParams(s.ctx).Return(expectedEmissionsParams, nil)
	expectedEmissionsParams.BlocksPerMonth = 1337
	s.emissionsKeeper.EXPECT().SetParams(s.ctx, expectedEmissionsParams).Return(nil)
	s.accountKeeper.EXPECT().GetModuleAddress("ecosystem").Return(sdk.AccAddress{})
	s.accountKeeper.EXPECT().GetModuleAddress("ecosystem").Return(sdk.AccAddress{})
	s.bankKeeper.EXPECT().GetBalance(s.ctx, sdk.AccAddress{}, "stake").Return(sdk.Coin{Denom: "stake", Amount: sdkmath.NewInt(1000000000000000000)})
	s.bankKeeper.EXPECT().GetBalance(s.ctx, sdk.AccAddress{}, "stake").Return(sdk.Coin{Denom: "stake", Amount: sdkmath.NewInt(1000000000000000000)})
	s.emissionsKeeper.EXPECT().GetPreviousPercentageRewardToStakedReputers(s.ctx).Return(alloraMath.MustNewDecFromString("0.5"), nil)
	s.stakingKeeper.EXPECT().TotalBondedTokens(s.ctx).Return(sdkmath.NewInt(1000000000000000000), nil)
	s.emissionsKeeper.EXPECT().GetTotalStake(s.ctx).Return(sdkmath.NewInt(1000000000000000000), nil)
	s.emissionsKeeper.EXPECT().GetParams(s.ctx).Return(expectedEmissionsParams, nil)
	s.emissionsKeeper.EXPECT().GetParams(s.ctx).Return(expectedEmissionsParams, nil)

	resp, err := s.msgServer.UpdateParams(s.ctx, request)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().Equal(&types.UpdateParamsResponse{}, resp)

	updatedTargetEmission, err := s.mintKeeper.GetPreviousRewardEmissionPerUnitStakedToken(s.ctx)
	s.Require().NoError(err)
	s.Require().NotEqual(startEmission, updatedTargetEmission)
}

func (s *MintKeeperTestSuite) TestRecalculateTargetEmissionMonthsAlreadyUnlockedGrowsWithStartingBlockHeight() {
	ecosystemTokensMinted := sdkmath.NewInt(1000000)
	err := s.mintKeeper.EcosystemTokensMinted.Set(s.ctx, ecosystemTokensMinted)
	s.Require().NoError(err)
	startEmission := sdkmath.LegacyNewDec(100)
	err = s.mintKeeper.PreviousRewardEmissionPerUnitStakedToken.Set(s.ctx, startEmission)
	s.Require().NoError(err)
	request := &types.RecalculateTargetEmissionRequest{
		Sender: s.adminAddr,
	}
	s.emissionsKeeper.EXPECT().IsWhitelistAdmin(s.ctx, s.adminAddr).Return(true, nil).AnyTimes()
	expectedEmissionsParams := emissionstypes.DefaultParams()
	s.accountKeeper.EXPECT().GetModuleAddress("ecosystem").Return(sdk.AccAddress{}).AnyTimes()
	s.accountKeeper.EXPECT().GetModuleAddress("ecosystem").Return(sdk.AccAddress{}).AnyTimes()
	s.bankKeeper.EXPECT().GetBalance(s.ctx, sdk.AccAddress{}, "stake").Return(sdk.Coin{Denom: "stake", Amount: sdkmath.NewInt(1000000000000000000)}).AnyTimes()
	s.bankKeeper.EXPECT().GetBalance(s.ctx, sdk.AccAddress{}, "stake").Return(sdk.Coin{Denom: "stake", Amount: sdkmath.NewInt(1000000000000000000)}).AnyTimes()
	s.emissionsKeeper.EXPECT().GetPreviousPercentageRewardToStakedReputers(s.ctx).Return(alloraMath.MustNewDecFromString("0.5"), nil).AnyTimes()
	s.stakingKeeper.EXPECT().TotalBondedTokens(s.ctx).Return(sdkmath.NewInt(1000000000000000000), nil).AnyTimes()
	s.emissionsKeeper.EXPECT().GetTotalStake(s.ctx).Return(sdkmath.NewInt(1000000000000000000), nil).AnyTimes()
	s.emissionsKeeper.EXPECT().GetParams(s.ctx).Return(expectedEmissionsParams, nil).AnyTimes()
	s.emissionsKeeper.EXPECT().GetParams(s.ctx).Return(expectedEmissionsParams, nil).AnyTimes()

	resp, err := s.msgServer.RecalculateTargetEmission(s.ctx, request)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().Equal(&types.RecalculateTargetEmissionResponse{}, resp)

	updatedTargetEmission, err := s.mintKeeper.GetPreviousRewardEmissionPerUnitStakedToken(s.ctx)
	s.Require().NoError(err)
	s.Require().NotEqual(startEmission, updatedTargetEmission)

	monhsAlreadyUnlockedStart := s.mintKeeper.GetMonthsAlreadyUnlocked(s.ctx)
	s.Require().Equal(monhsAlreadyUnlockedStart, sdkmath.NewInt(0))

	err = s.mintKeeper.SetStartingEmissionsBlockHeight(s.ctx, 100000000) // very high
	s.Require().NoError(err)
	resp, err = s.msgServer.RecalculateTargetEmission(s.ctx, request)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Require().Equal(&types.RecalculateTargetEmissionResponse{}, resp)

	monhsAlreadyUnlockedEnd := s.mintKeeper.GetMonthsAlreadyUnlocked(s.ctx)
	s.Require().True(monhsAlreadyUnlockedEnd.GT(monhsAlreadyUnlockedStart))
}

func (s *MintKeeperTestSuite) TestUpdateInvestorsTeamVestingDelay() {
	govAddr := s.mintKeeper.GetAuthority()
	request := &types.UpdateInvestorsTeamVestingDelayRequest{
		Authority: govAddr,
		Months:    6,
	}

	emissionsParams := emissionstypes.DefaultParams()
	s.emissionsKeeper.EXPECT().GetParams(s.ctx).Return(emissionsParams, nil)

	resp, err := s.msgServer.UpdateInvestorsTeamVestingDelay(s.ctx, request)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	params, err := s.mintKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(uint64(6), params.InvestorsTeamVestingDelayMonths)

	// event was emitted
	events := s.ctx.EventManager().Events()
	var found bool
	for _, ev := range events {
		if ev.Type == "mint.v3.EventParamsSet" {
			found = true
			break
		}
	}
	s.Require().True(found, "expected EventParamsSet event")
}

func (s *MintKeeperTestSuite) TestUpdateInvestorsTeamVestingDelayUnauthorized() {
	// a whitelist admin is NOT the authority
	nonGov := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	request := &types.UpdateInvestorsTeamVestingDelayRequest{
		Authority: nonGov,
		Months:    6,
	}

	resp, err := s.msgServer.UpdateInvestorsTeamVestingDelay(s.ctx, request)
	s.Require().Error(err)
	s.Require().Nil(resp)

	// even the whitelist admin addr is rejected
	request2 := &types.UpdateInvestorsTeamVestingDelayRequest{
		Authority: s.adminAddr,
		Months:    6,
	}
	resp2, err2 := s.msgServer.UpdateInvestorsTeamVestingDelay(s.ctx, request2)
	s.Require().Error(err2)
	s.Require().Nil(resp2)
}

func (s *MintKeeperTestSuite) TestUpdateInvestorsTeamVestingDelayAboveBound() {
	govAddr := s.mintKeeper.GetAuthority()
	request := &types.UpdateInvestorsTeamVestingDelayRequest{
		Authority: govAddr,
		Months:    types.MaxInvestorsTeamVestingDelayMonths + 1,
	}

	resp, err := s.msgServer.UpdateInvestorsTeamVestingDelay(s.ctx, request)
	s.Require().Error(err)
	s.Require().Nil(resp)
}

func (s *MintKeeperTestSuite) TestUpdateParamsRejectsDelayChange() {
	// set a non-zero delay first
	params, err := s.mintKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	params.InvestorsTeamVestingDelayMonths = 6
	s.Require().NoError(s.mintKeeper.Params.Set(s.ctx, params))

	// try to change it via UpdateParams
	newParams := params
	newParams.InvestorsTeamVestingDelayMonths = 12
	request := &types.UpdateParamsRequest{
		Sender:         s.adminAddr,
		Params:         newParams,
		BlocksPerMonth: 525590,
	}
	s.emissionsKeeper.EXPECT().IsWhitelistAdmin(s.ctx, s.adminAddr).Return(true, nil)
	resp, err := s.msgServer.UpdateParams(s.ctx, request)
	s.Require().Error(err)
	s.Require().Nil(resp)
	s.Require().Contains(err.Error(), "MsgUpdateInvestorsTeamVestingDelay")
}

func (s *MintKeeperTestSuite) TestUpdateParamsAllowsUnchangedDelay() {
	// set a non-zero delay first
	params, err := s.mintKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	params.InvestorsTeamVestingDelayMonths = 6
	s.Require().NoError(s.mintKeeper.Params.Set(s.ctx, params))

	// same delay in the request should succeed
	newParams := params
	request := &types.UpdateParamsRequest{
		Sender:         s.adminAddr,
		Params:         newParams,
		BlocksPerMonth: 525590,
	}
	expectedEmissionsParams := emissionstypes.DefaultParams()
	s.emissionsKeeper.EXPECT().IsWhitelistAdmin(s.ctx, s.adminAddr).Return(true, nil)
	s.emissionsKeeper.EXPECT().GetParams(s.ctx).Return(expectedEmissionsParams, nil)
	expectedEmissionsParams.BlocksPerMonth = 525590
	s.emissionsKeeper.EXPECT().SetParams(s.ctx, expectedEmissionsParams).Return(nil)
	resp, err := s.msgServer.UpdateParams(s.ctx, request)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
}
