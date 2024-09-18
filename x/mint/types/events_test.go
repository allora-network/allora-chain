package types_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/mint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestEmitNewTokenomicsSetEvent(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	stakedTokenAmount := math.NewInt(10)
	circulatingSupply := math.NewInt(20)
	emissionsAmount := math.NewInt(30)

	types.EmitNewTokenomicsSetEvent(ctx, stakedTokenAmount, circulatingSupply, emissionsAmount)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	require.Equal(t, "mint.v3.EventTokenomicsSet", events[0].Type)
	require.Contains(t, events[0].Attributes[0].Key, "circulating_supply")
	require.Contains(t, events[0].Attributes[0].Value, circulatingSupply.String())
	require.Contains(t, events[0].Attributes[1].Key, "emissions_amount")
	require.Contains(t, events[0].Attributes[1].Value, emissionsAmount.String())
	require.Contains(t, events[0].Attributes[2].Key, "staked_token_amount")
	require.Contains(t, events[0].Attributes[2].Value, stakedTokenAmount.String())
}

func TestEmitNewEcosystemTokenMintedSetEvent(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	blockHeight := uint64(1)
	tokenAmount := math.NewInt(10)

	types.EmitNewEcosystemTokenMintSetEvent(ctx, blockHeight, tokenAmount)

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)

	require.Equal(t, "mint.v3.EventEcosystemTokenMintSet", events[0].Type)
	require.Contains(t, events[0].Attributes[0].Key, "block_height")
	require.Contains(t, events[0].Attributes[0].Value, "1")
	require.Contains(t, events[0].Attributes[1].Key, "token_amount")
	require.Contains(t, events[0].Attributes[1].Value, tokenAmount.String())
}
