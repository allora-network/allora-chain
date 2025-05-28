package v0_13_0

import (
	"testing"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/crypto/ed25519"
	sm "github.com/cometbft/cometbft/state"
	"github.com/cometbft/cometbft/types"
	"github.com/stretchr/testify/require"
)

func TestResetValidatorsProposerPriorities(t *testing.T) {
	pk1 := ed25519.GenPrivKeyFromSecret([]byte{4}).PubKey()
	pk2 := ed25519.GenPrivKeyFromSecret([]byte{3}).PubKey()
	pk3 := ed25519.GenPrivKeyFromSecret([]byte{2}).PubKey()
	pk4 := ed25519.GenPrivKeyFromSecret([]byte{1}).PubKey()
	store := sm.NewStore(dbm.NewMemDB(), sm.StoreOptions{DiscardABCIResponses: true})

	vp := int64(33333300 * 4)
	valSet := &types.ValidatorSet{
		Validators: []*types.Validator{
			{
				Address:          pk1.Address(),
				PubKey:           pk1,
				VotingPower:      33333300,
				ProposerPriority: -66666600,
			},
			{
				Address:          pk2.Address(),
				PubKey:           pk2,
				VotingPower:      33333300,
				ProposerPriority: -66666600,
			},
			{
				Address:          pk3.Address(),
				PubKey:           pk3,
				VotingPower:      33333300,
				ProposerPriority: 66666600,
			},
			{
				Address:          pk4.Address(),
				PubKey:           pk4,
				VotingPower:      33333300,
				ProposerPriority: 66666600,
			},
		},
		Proposer: &types.Validator{
			Address:          pk3.Address(),
			PubKey:           pk3,
			VotingPower:      33333300,
			ProposerPriority: 66666600,
		},
	}

	err := store.Save(sm.State{
		NextValidators: valSet,
		Validators:     valSet,
		LastValidators: valSet,
	})
	require.NoError(t, err)

	err = resetValidatorsProposerPriorities(store)
	require.NoError(t, err)

	state, err := store.Load()
	require.NoError(t, err)

	require.Equal(t, vp, state.NextValidators.TotalVotingPower())
	require.Equal(t, []*types.Validator{
		{
			Address:          pk1.Address(),
			PubKey:           pk1,
			VotingPower:      33333300,
			ProposerPriority: -99999900,
		},
		{
			Address:          pk2.Address(),
			PubKey:           pk2,
			VotingPower:      33333300,
			ProposerPriority: 33333300,
		},
		{
			Address:          pk3.Address(),
			PubKey:           pk3,
			VotingPower:      33333300,
			ProposerPriority: 33333300,
		},
		{
			Address:          pk4.Address(),
			PubKey:           pk4,
			VotingPower:      33333300,
			ProposerPriority: 33333300,
		},
	}, state.NextValidators.Validators)
	require.Equal(t, pk1.Address(), state.NextValidators.Proposer.Address)
}
