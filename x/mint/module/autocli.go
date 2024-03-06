package mint

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	mintv1beta1 "github.com/allora-network/allora-chain/x/mint/api/v1beta1"
)

func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: mintv1beta1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Query the current minting parameters",
				},
				{
					RpcMethod: "Inflation",
					Use:       "inflation",
					Short:     "Query the current minting inflation value",
				},
				{
					RpcMethod: "AnnualProvisions",
					Use:       "annual-provisions",
					Short:     "Query the current minting annual provisions value",
				},
			},
		},
	}
}
