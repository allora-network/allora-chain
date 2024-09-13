package types

// NewGenesisState creates a new genesis state with default values.
func NewGenesisState() *GenesisState {
	return &GenesisState{
		Params:            DefaultParams(),
		CoreTeamAddresses: DefaultCoreTeamAddresses(),
	}
}

// DefaultCoreTeamAddresses returns the default core team addresses
// used for administration of params for the emissions module
// long term should be managed by the standard cosmos-sdk
// gov module
func DefaultCoreTeamAddresses() []string {
	return []string{
		"allo16270t36amc3y6wk2wqupg6gvg26x6dc2nr5xwl",
		"allo1xm0jg40dcvccqvzqwv5skxlpc7t6eku69kfz6y",
		"allo1g4y6ra95z2zewupm7p45z4ny00rs7m24rj5hn8",
		"allo10w0jcq50ufsuy9332dkz6zf4gu00xm9zhfyn3s",
		"allo1lvymnmzndmam00uvxq8hr63jq8jfrups4ymlg2",
		"allo1d7vr2dxahkcz0snk28pets9uqvyxjdlysst3z3",
		"allo19gtttc7qg50n3hjn0qxdasdudf260cx7vevk8j",
		"allo1jc2mme2fj458kg08v2z92m8f9vsqwfzt0ju9ys",
		"allo1uff55lgqpjkw2mlsx2q0p8q8z7k7p00w9s4s0f",
		"allo136eeqhawxx66sjgsfeqk9gewq0e0msyu5tjmj3",
	}
}
