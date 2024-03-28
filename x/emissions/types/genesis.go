package types

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
		"allo1srjcynn6jw5l709upwwx70gs3eyjdmkgufcqdt",
		"allo1uu2fk9gmjkmy8qme46w8hr8j2exfjnsqnweznz",
		"allo1fk0pt0fj03fxdyyq0mhj5cfq5kv9ncwmtqvy7u",
		"allo1d6hegr59ftc43xxd2n2vn5ndley699jefwdgq8",
		"allo1xw354nrfpsw6x3sf7aqnrek0wluyx58zv2uc75",
		"allo1d54qdljsc3srsy8fz6zzrx90hyzuevnccg88md",
		"allo1fumaxxyjwdv7y5uh6wxslhxmqf32jtvnr8edlf",
		"allo1memyg7exjzjdpdv98cfm6u0y0lsz4ev8mk57hc",
		"allo1shzv768qrxaextjwz0aj6nzhm3cyy4pdug8jy6",
		"allo12heywwqc75mgk6qg3n0mryw58jn6ujtp7tfvs9",
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
