package queryserver

import (
	"context"
	"fmt"

	v4Migration "github.com/allora-network/allora-chain/x/emissions/migrations/v4"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (q queryServer) TriggerMigration(
	ctx context.Context,
	req *emissionstypes.TriggerMigrationRequest,
) (*emissionstypes.TriggerMigrationResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	fmt.Println("TriggerMigration")
	go func() {
		fmt.Println("TriggerMigration: starting migration")
		err := v4Migration.MigrateStore(sdkCtx, q.k)
		if err != nil {
			fmt.Println("TriggerMigration: migration error", err)
		}
	}()
	return &emissionstypes.TriggerMigrationResponse{}, nil
}
