package cmd

import (
	"testing"
	"time"

	cmtcfg "github.com/cometbft/cometbft/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestOverrideStructConfig(t *testing.T) {
	overrides := settings{
		{
			section: "consensus",
			key:     "timeout_propose",
			value:   time.Second * 15,
		},
		{
			section: "consensus",
			key:     "timeout_propose_delta",
			value:   time.Millisecond * 2000,
		},
	}

	cmtCfg := cmtcfg.DefaultConfig()
	require.NoError(t, overrideStructConfig(cmtCfg, overrides))
	require.Equal(t, time.Second*15, cmtCfg.Consensus.TimeoutPropose)
	require.Equal(t, time.Millisecond*2000, cmtCfg.Consensus.TimeoutProposeDelta)
}

func TestSetInViper(t *testing.T) {
	overrides := settings{
		{
			section: "consensus",
			key:     "timeout_propose",
			value:   time.Second * 15,
		},
		{
			section: "consensus",
			key:     "timeout_propose_delta",
			value:   time.Millisecond * 2000,
		},
	}

	v := viper.New()
	v.Set("consensus.timeout_propose", time.Second*2)
	v.Set("consensus.timeout_propose_delta", time.Second*6)

	overrides.setInViper(v)

	require.Equal(t, time.Second*15, v.Get("consensus.timeout_propose"))
	require.Equal(t, time.Millisecond*2000, v.Get("consensus.timeout_propose_delta"))
}
