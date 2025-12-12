package cmd

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/codec"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

func EnhanceDebugCommand(debugCmd *cobra.Command) *cobra.Command {
	debugCmd.AddCommand(
		queryTopicWhitelist(),
	)
	return debugCmd
}

func queryTopicWhitelist() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "topic-whitelist [reputer|worker] topicId",
		Short: "List whitelisted addresses in topic",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			var whitelistStorePrefix collections.Prefix
			switch args[0] {
			case "reputer":
				whitelistStorePrefix = emissionstypes.TopicReputerWhitelistKey
			case "worker":
				whitelistStorePrefix = emissionstypes.TopicWorkerWhitelistKey
			default:
				return fmt.Errorf("whitelist must be 'reputer' or 'worker', got: %s", args[0])
			}

			topicID, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}

			topicIDCodec := codec.NewUint64Key[uint64]()
			prefixKeySize := len(whitelistStorePrefix) + topicIDCodec.Size(topicID)
			key := make([]byte, prefixKeySize)
			copy(key, whitelistStorePrefix)

			if _, err := topicIDCodec.EncodeNonTerminal(key[len(whitelistStorePrefix):], topicID); err != nil {
				return err
			}

			resp, err := clientCtx.Client.ABCIQuery(context.Background(), "/store/emissions/subspace", key)
			if err != nil {
				return err
			}

			for _, raw := range bytes.Split(resp.Response.Value, []byte("\n6\n4")) {
				if len(raw) == 0 {
					continue
				}
				fmt.Println(string(raw[prefixKeySize:]))
			}

			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
