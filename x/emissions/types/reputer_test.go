package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	alloraMath "github.com/allora-network/allora-chain/math"
)

func TestReputerValueBundlesToLossBundles(t *testing.T) {
	t.Parallel()

	const (
		topicID = uint64(1)
		block   = int64(100)
		reputer = "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy"
	)

	validValueBundle := &ValueBundle{
		TopicId: topicID,
		ReputerRequestNonce: &ReputerRequestNonce{
			ReputerNonce: &Nonce{BlockHeight: block},
		},
		Reputer:       reputer,
		CombinedValue: alloraMath.ZeroDec(),
		InfererValues: []*WorkerAttributedValue{
			{Worker: reputer, Value: alloraMath.ZeroDec()},
		},
		NaiveValue: alloraMath.ZeroDec(),
	}

	tests := []struct {
		name    string
		input   ReputerValueBundles
		wantErr error
		wantLen int
	}{
		{
			name: "nil reputer value bundle",
			input: ReputerValueBundles{
				ReputerValueBundles: []*ReputerValueBundle{nil},
			},
			wantErr: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "nil inner value bundle",
			input: ReputerValueBundles{
				ReputerValueBundles: []*ReputerValueBundle{
					{ValueBundle: nil},
				},
			},
			wantErr: sdkerrors.ErrInvalidRequest,
		},
		{
			name: "valid bundle",
			input: ReputerValueBundles{
				ReputerValueBundles: []*ReputerValueBundle{
					{ValueBundle: validValueBundle},
				},
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.input.ToLossBundles(topicID, block)
			if tt.wantErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.Len(t, got, tt.wantLen)
			require.Equal(t, validValueBundle, got[0])
		})
	}
}
