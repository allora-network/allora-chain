package keeper

import (
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
)

func (k *Keeper) TaskHandlers() schedulertypes.TaskHandlers {
	return schedulertypes.TaskHandlers{}
}
