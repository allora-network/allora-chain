package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func TestEpochLabelRegistryLabelNamesByID(t *testing.T) {
	label := func(id uint32, name string) *types.TopicLabel { return &types.TopicLabel{Id: id, Name: name} }
	registry := func(labels ...*types.TopicLabel) types.EpochLabelRegistry {
		return types.EpochLabelRegistry{TopicId: 1, EpochId: 10, Labels: labels}
	}
	tests := []struct {
		name    string
		reg     types.EpochLabelRegistry
		want    []string
		wantErr bool
	}{
		{name: "in id order", reg: registry(label(1, "a"), label(2, "b"), label(3, "c")), want: []string{"a", "b", "c"}, wantErr: false},
		{name: "out of id order", reg: registry(label(2, "b"), label(1, "a")), want: []string{"a", "b"}, wantErr: false},
		{name: "empty", reg: registry(), want: []string{}, wantErr: false},
		{name: "id zero", reg: registry(label(0, "a")), want: nil, wantErr: true},
		{name: "id past the end", reg: registry(label(1, "a"), label(3, "c")), want: nil, wantErr: true},
		{name: "repeated id", reg: registry(label(1, "a"), label(1, "b")), want: nil, wantErr: true},
		{name: "repeated id after an empty label", reg: registry(label(1, ""), label(1, "b")), want: nil, wantErr: true},
		{name: "nil label", reg: registry(nil), want: nil, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.reg.LabelNamesByID()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// The frozen-registry event carries the labels in compact-id order, and its
// size is their count.
func TestEmitNewEpochLabelRegistryFrozenEventCarriesTheLabels(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())

	types.EmitNewEpochLabelRegistryFrozenEvent(ctx, 7, 940, []string{"up", "flat", "down"})

	events := ctx.EventManager().Events()
	require.Len(t, events, 1)
	require.Equal(t, "emissions.v10.EventEpochLabelRegistryFrozen", events[0].Type)
	labels, ok := events[0].GetAttribute("labels")
	require.True(t, ok)
	require.Equal(t, `["up","flat","down"]`, labels.GetValue())
	size, ok := events[0].GetAttribute("registry_size")
	require.True(t, ok)
	require.Equal(t, `"3"`, size.GetValue())
}

// A network inference bundle's label names follow the registry its values were
// labeled with, in compact-id order, so they match the frozen-registry event's
// labels for the same epoch.
//
//nolint:exhaustruct // the bundle has many fields that play no part in label order
func TestNetworkInferenceBundleLabelNamesFollowTheRegistry(t *testing.T) {
	registry := types.EpochLabelRegistry{TopicId: 1, EpochId: 10, Labels: []*types.TopicLabel{
		{Id: 1, Name: "up"}, {Id: 2, Name: "flat"}, {Id: 3, Name: "down"},
	}}
	values := []alloraMath.Dec{alloraMath.NewDecFromInt64(1), alloraMath.NewDecFromInt64(2), alloraMath.NewDecFromInt64(3)}
	combined, err := types.ConvertInferenceValuesToLabeledValues(values, &registry)
	require.NoError(t, err)

	event, ok := types.NewNetworkInferencesEventBase(types.NetworkInferenceBundle{TopicId: 1, Nonce: 10, CombinedValue: combined}).(*types.EventNetworkInferenceBundle)
	require.True(t, ok)

	frozenLabels, err := registry.LabelNamesByID()
	require.NoError(t, err)
	require.Equal(t, frozenLabels, event.LabelNames)
}
