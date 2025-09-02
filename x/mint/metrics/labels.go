package metrics

//nolint:gosec // These are event labels, not credentials
const (
	TOKENOMICS_SET_EVENT                = "tokenomics_set_event"
	ECOSYSTEM_TOKEN_MINT_SET_EVENT      = "ecosystem_token_mint_set_event"
	REWARD_CURRENT_BLOCK_EMISSION_EVENT = "reward_current_block_emission_event"
	RECALCULATE_TARGET_EMISSION_EVENT   = "recalculate_target_emission_event"
	PARAMS_SET_EVENT                    = "params_set_event"
	EMISSION_INFO_EVENT                 = "emission_info_event"
)
