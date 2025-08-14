package cmd

import (
	"time"

	cmtcfg "github.com/cometbft/cometbft/config"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

var (
	recommendedAppSettings   = map[string]interface{}{}
	recommendedCometSettings = map[string]interface{}{
		"consensus.timeout_propose":         time.Second * 5,
		"consensus.timeout_propose_delta":   time.Millisecond * 500,
		"consensus.timeout_prevote":         time.Second * 3,
		"consensus.timeout_prevote_delta":   time.Millisecond * 500,
		"consensus.timeout_precommit":       time.Second * 3,
		"consensus.timeout_precommit_delta": time.Millisecond * 500,
		"consensus.timeout_commit":          time.Second * 5,
	}
)

func mustGetDefaultConfigs() (*serverconfig.Config, *cmtcfg.Config) {
	srvCfg := serverconfig.DefaultConfig()
	srvCfg.MinGasPrices = "10uallo"

	cmtCfg := cmtcfg.DefaultConfig()
	cmtCfg.LogLevel = "*:error,p2p:info,state:info"

	if err := overrideConfigs(srvCfg, cmtCfg); err != nil {
		panic("failed to enforce settings: " + err.Error())
	}

	return srvCfg, cmtCfg
}

func overrideViperConfig(v *viper.Viper) {
	for key, value := range recommendedAppSettings {
		v.Set(key, value)
	}

	for key, value := range recommendedCometSettings {
		v.Set(key, value)
	}
}

func overrideConfigs(srvCfg *serverconfig.Config, cmtCfg *cmtcfg.Config) error {
	if err := overrideAppConfig(srvCfg); err != nil {
		return err
	}

	return overrideCometConfig(cmtCfg)
}

func overrideCometConfig(cmtCfg *cmtcfg.Config) error {
	return overrideStructSettings(cmtCfg, recommendedCometSettings)
}

func overrideAppConfig(srvCfg *serverconfig.Config) error {
	return overrideStructSettings(srvCfg, recommendedAppSettings)
}

func overrideStructSettings(config interface{}, enforcedSettings map[string]interface{}) error {
	var cfg map[string]interface{}
	if err := mapstructure.Decode(config, cfg); err != nil {
		return err
	}

	for key, value := range enforcedSettings {
		cfg[key] = value
	}

	return mapstructure.Decode(cfg, config)
}
