package cmd

import (
	"errors"
	"time"

	cmtcfg "github.com/cometbft/cometbft/config"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

var (
	recommendedAppSettings   settings
	recommendedCometSettings = settings{
		{
			section: "consensus",
			key:     "timeout_propose",
			value:   time.Second * 5,
		}, {
			section: "consensus",
			key:     "timeout_propose_delta",
			value:   time.Millisecond * 500,
		}, {
			section: "consensus",
			key:     "timeout_prevote",
			value:   time.Second * 3,
		}, {
			section: "consensus",
			key:     "timeout_prevote_delta",
			value:   time.Millisecond * 500,
		}, {
			section: "consensus",
			key:     "timeout_precommit",
			value:   time.Second * 3,
		}, {
			section: "consensus",
			key:     "timeout_precommit_delta",
			value:   time.Millisecond * 500,
		}, {
			section: "consensus",
			key:     "timeout_commit",
			value:   time.Second * 5,
		},
	}
)

type settings []setting

func (s settings) setInMap(cfg map[string]interface{}) error {
	for _, setting := range s {
		if err := setting.setInMap(cfg); err != nil {
			return err
		}
	}
	return nil
}

func (s settings) setInViper(v *viper.Viper) {
	for _, setting := range s {
		setting.setInViper(v)
	}
}

type setting struct {
	section string
	key     string
	value   interface{}
}

func (s setting) setInMap(cfg map[string]interface{}) error {
	if _, ok := cfg[s.section]; !ok {
		cfg[s.section] = make(map[string]interface{})
	}
	cfgSection, ok := cfg[s.section].(map[string]interface{})
	if !ok {
		return errors.New("section '" + s.section + "' is not a map[string]interface{}")
	}

	cfgSection[s.key] = s.value
	return nil
}

func (s setting) setInViper(v *viper.Viper) {
	v.Set(s.section+"."+s.key, s.value)
}

func mustGetDefaultConfigs() (*serverconfig.Config, *cmtcfg.Config) {
	srvCfg := serverconfig.DefaultConfig()

	cmtCfg := cmtcfg.DefaultConfig()
	cmtCfg.LogLevel = "*:error,p2p:info,state:info"

	if err := overrideConfigs(srvCfg, cmtCfg); err != nil {
		panic("failed to enforce settings: " + err.Error())
	}

	return srvCfg, cmtCfg
}

func overrideViperConfig(v *viper.Viper) {
	recommendedAppSettings.setInViper(v)
	recommendedCometSettings.setInViper(v)
}

func overrideConfigs(srvCfg *serverconfig.Config, cmtCfg *cmtcfg.Config) error {
	if err := overrideAppConfig(srvCfg); err != nil {
		return err
	}

	return overrideCometConfig(cmtCfg)
}

func overrideCometConfig(cmtCfg *cmtcfg.Config) error {
	return overrideStructConfig(cmtCfg, recommendedCometSettings)
}

func overrideAppConfig(srvCfg *serverconfig.Config) error {
	return overrideStructConfig(srvCfg, recommendedAppSettings)
}

func overrideStructConfig(config interface{}, enforcedSettings settings) error {
	var cfg map[string]interface{}
	if err := mapstructure.Decode(config, &cfg); err != nil {
		return err
	}

	if err := enforcedSettings.setInMap(cfg); err != nil {
		return err
	}

	return mapstructure.Decode(cfg, config)
}
