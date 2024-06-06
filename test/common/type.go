package testcommon

import "github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"

type NameAccountAndAddress struct {
	Name string
	Aa   AccountAndAddress
}

// holder for account and address
type AccountAndAddress struct {
	Acc  cosmosaccount.Account
	Addr string
}
