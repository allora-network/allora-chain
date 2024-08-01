package v2

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// An example MigrateStore function that might be run in the migration
// handler.
func MigrateStore(ctx sdk.Context) error {
	ctx.Logger().Info(fmt.Sprintf(
		"###################################################" +
			"### MIGRATING STORE FROM VERSION 1 TO VERSION 2 ###" +
			"###################################################",
	))
	return nil
}
