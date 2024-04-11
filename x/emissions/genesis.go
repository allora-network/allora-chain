package emissions

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewGenesisState creates a new genesis state with default values.
func NewGenesisState() *GenesisState {
	return &GenesisState{
		Params:            DefaultParams(),
		CoreTeamAddresses: DefaultCoreTeamAddresses(),
	}
}

func DefaultCoreTeamAddresses() []string {
	return []string{
		"allo1mgpsfkr3eysxuznumvljfh882xdwy77de5za24",
		"allo18xlecttrwhp20lesmdc4cyghtzge8n4dautq6f",
		"allo1sv8smn9l4aypxdd747kefky69u743wf8ggkeha",
		"allo1ymtvmxzurfyd6mtvsynzw058kd2xsjnucfcfgq",
		"allo1al6r86ag9t6u2lw7e7zhcjvwe38zzmqrelg7h6",
		"allo178x3rw8c3wc6ye86ksjrdxpqtczz4qergwmfhd",
		"allo1ca6vtdw27yhysyn5zf8atf77rkst9kkw9a5kyt",
		"allo1acsdeqcqawa0vycnxkfd02v8fj0xm2eukqes22",
		"allo16zfquhlcanqr32dpr593mgd989agc827nh2yhw",
		"allo1334sr00z2qlet64h0vv7j0luqvrqahdzjpmn3r",
	}
}

// Validate performs basic genesis state validation returning an error upon any
func (gs *GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	// Ensure that the core team addresses are valid
	for _, addr := range gs.CoreTeamAddresses {
		_, err := sdk.AccAddressFromBech32(addr)
		if err != nil {
			return err
		}
	}

	return nil
}
