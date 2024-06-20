package v3

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// An example MigrateStore function that might be run in the migration
// handler.
func MigrateStore(ctx sdk.Context) error {
	ctx.Logger().Warn(fmt.Sprintf(
		"###################################################" +
			"### MIGRATING STORE FROM VERSION 2 TO VERSION 3 ###" +
			"###################################################",
	))
	return nil
}
