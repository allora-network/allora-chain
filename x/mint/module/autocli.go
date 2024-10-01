package mint

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	mintv1beta1 "github.com/allora-network/allora-chain/x/mint/api/mint/v1beta1"
	mintv2 "github.com/allora-network/allora-chain/x/mint/api/mint/v2"
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
					RpcMethod: "EmissionInfo",
					Use:       "emission-info",
					Short:     "Get a bunch of debugging info about the inflation rate",
				},
			},
			SubCommands:          nil,
			EnhanceCustomCommand: false,
			Short:                "Querying commands for the mint module",
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service: mintv2.MsgService_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Use:       "update-params [sender] [params]",
					Short:     "Update params of the network",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
						{ProtoField: "params"},
					},
				},
				{
					RpcMethod: "RecalculateTargetEmission",
					Use:       "recalculate-target-emission [sender]",
					Short:     "Recalculate target emission of the network",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "sender"},
					},
				},
			},
			SubCommands:          nil,
			EnhanceCustomCommand: false,
			Short:                "Transaction commands for the mint module",
		},
	}
}
