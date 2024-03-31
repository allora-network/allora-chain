package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/mint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *IntegrationTestSuite) TestUpdateParams() {
	maxSupply, ok := sdkmath.NewIntFromString("1000000000000000000000000000")
	if !ok {
		panic("invalid number")
	}
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
		{
			name: "set invalid params",
			request: &types.MsgUpdateParams{
				Authority: s.mintKeeper.GetAuthority(),
				Params: types.Params{
					MintDenom:                   sdk.DefaultBondDenom,
					BlocksPerYear:               uint64(60 * 60 * 8766 / 5),
					MaxSupply:                   sdkmath.NewIntFromUint64(0),
					FEmission:                   sdkmath.NewInt(15),
					FEmissionPrec:               2,
					OneMonthSmoothingDegree:     sdkmath.NewInt(1),
					OneMonthSmoothingDegreePrec: 1,
				},
			},
			expectErr: true,
		},
		{
			name: "set full valid params",
			request: &types.MsgUpdateParams{
				Authority: s.mintKeeper.GetAuthority(),
				Params: types.Params{
					MintDenom:                   sdk.DefaultBondDenom,
					BlocksPerYear:               uint64(60 * 60 * 8766 / 5),
					MaxSupply:                   maxSupply,
					FEmission:                   sdkmath.NewInt(15),
					FEmissionPrec:               2,
					OneMonthSmoothingDegree:     sdkmath.NewInt(1),
					OneMonthSmoothingDegreePrec: 1,
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
