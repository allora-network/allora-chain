package cmd

import (
	"fmt"

	cmtcfg "github.com/cometbft/cometbft/config"
	sm "github.com/cometbft/cometbft/state"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"
)

func EnhanceDebugCommand(debugCmd *cobra.Command) *cobra.Command {
	debugCmd.AddCommand(
		resetConsensusPrioritiesCommand(),
	)
	return debugCmd
}

func resetConsensusPrioritiesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset-consensus-priorities",
		Short: "Reset CometBFT consensus state proposer priorities",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return server.GetServerContextFromCmd(cmd).Viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			cometRPC, err := clientCtx.GetNode()
			if err != nil {
				return err
			}

			svrCtx := server.GetServerContextFromCmd(cmd)
			cfg := svrCtx.Config
			stateDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "state", Config: cfg})
			if err != nil {
				return err
			}
			stateStore := sm.NewStore(stateDB, sm.StoreOptions{
				DiscardABCIResponses: cfg.Storage.DiscardABCIResponses,
			})
			state, err := stateStore.Load()
			if err != nil {
				return err
			}

			page := 1
			perPage := 200
			vals, err := cometRPC.Validators(cmd.Context(), &state.LastBlockHeight, &page, &perPage)
			if err != nil {
				return err
			}

			valPriorities := make(map[string]int64)
			for _, val := range vals.Validators {
				valPriorities[val.Address.String()] = val.ProposerPriority
			}

			for _, val := range state.NextValidators.Validators {
				priority, exists := valPriorities[val.Address.String()]
				if !exists {
					return fmt.Errorf("validator %s does not exist in state", val.Address.String())
				}

				val.ProposerPriority = priority
			}

			state.NextValidators.IncrementProposerPriority(2)

			return stateStore.Save(state)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
