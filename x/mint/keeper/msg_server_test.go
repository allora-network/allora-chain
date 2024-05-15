package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/mint/types"
)

func (s *IntegrationTestSuite) TestUpdateParams() {
	defaultParams := types.DefaultParams()
	testCases := []struct {
		name      string
		request   *types.MsgUpdateParams
		expectErr bool
	}{
		{
			name: "set invalid authority (not an address)",
			request: &types.MsgUpdateParams{
				Authority: "foo",
			},
			expectErr: true,
		},
		{
			name: "set invalid authority (not defined authority)",
			request: &types.MsgUpdateParams{
				Authority: "cosmos139f7kncmglres2nf3h4hc4tade85ekfr8sulz5",
			},
			expectErr: true,
		},
		//todo do all failing testcases for each validity check against each param one by one
		{
			name: "set invalid params",
			request: &types.MsgUpdateParams{
				Authority: s.mintKeeper.GetAuthority(),
				Params: types.Params{
					MintDenom:                              defaultParams.MintDenom,
					MaxSupply:                              sdkmath.NewIntFromUint64(0),
					FEmission:                              defaultParams.FEmission,
					OneMonthSmoothingDegree:                defaultParams.OneMonthSmoothingDegree,
					EcosystemTreasuryPercentOfTotalSupply:  defaultParams.EcosystemTreasuryPercentOfTotalSupply,
					FoundationTreasuryPercentOfTotalSupply: defaultParams.FoundationTreasuryPercentOfTotalSupply,
					ParticipantsPercentOfTotalSupply:       defaultParams.ParticipantsPercentOfTotalSupply,
					InvestorsPercentOfTotalSupply:          defaultParams.InvestorsPercentOfTotalSupply,
					TeamPercentOfTotalSupply:               defaultParams.TeamPercentOfTotalSupply,
				},
			},
			expectErr: true,
		},
		{
			name: "set full valid params",
			request: &types.MsgUpdateParams{
				Authority: s.mintKeeper.GetAuthority(),
				Params: types.Params{
					MintDenom:                              defaultParams.MintDenom,
					MaxSupply:                              defaultParams.MaxSupply,
					FEmission:                              defaultParams.FEmission,
					OneMonthSmoothingDegree:                defaultParams.OneMonthSmoothingDegree,
					EcosystemTreasuryPercentOfTotalSupply:  defaultParams.EcosystemTreasuryPercentOfTotalSupply,
					FoundationTreasuryPercentOfTotalSupply: defaultParams.FoundationTreasuryPercentOfTotalSupply,
					ParticipantsPercentOfTotalSupply:       defaultParams.ParticipantsPercentOfTotalSupply,
					InvestorsPercentOfTotalSupply:          defaultParams.InvestorsPercentOfTotalSupply,
					TeamPercentOfTotalSupply:               defaultParams.TeamPercentOfTotalSupply,
				},
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			_, err := s.msgServer.UpdateParams(s.ctx, tc.request)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}
