package integration_test

import (
	"bufio"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/allora-network/allora-chain/app/params"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/stretchr/testify/require"
)

// Get the validator address that is stored in the genesis file
// didn't know of a better way to get a validator address to check with
func GetValidatorAddressFromGenesisFile(m TestMetadata) (string, error) {
	home := m.n.Client.Context().HomeDir
	genesisPath := home + "/config/genesis.json"

	file, err := os.Open(genesisPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "\"validator_address\":") {
			splitted := strings.Split(line, ":")
			require.Equal(m.t, len(splitted), 2)
			trimmed := strings.TrimSpace(splitted[1])
			trimmed = strings.Trim(trimmed, ",")
			trimmed = strings.Trim(trimmed, "\"")
			m.t.Log(trimmed)
			return trimmed, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", errors.New("validator address not found")
}

// this file tests that the distribution of funds to validators
// and rewards accounts is working as expected
// basically testing the forked mint module that we use
func DistributionChecks(m TestMetadata) {
	// alloraRewardsModuleAccResponse, err := m.n.QueryAuth.ModuleAccountByName(
	// 	m.ctx,
	// 	&authtypes.QueryModuleAccountByNameRequest{
	// 		Name: "allorarewards",
	// 	},
	// )
	// require.NoError(m.t, err)
	// var alloraRewardsAccount authtypes.ModuleAccount
	// err = m.n.Cdc.Unmarshal(
	// 	alloraRewardsModuleAccResponse.Account.Value,
	// 	&alloraRewardsAccount,
	// )
	// require.NoError(m.t, err)
	// alloraRewardsBalanceStart, err := m.n.QueryBank.Balance(
	// 	m.ctx,
	// 	&banktypes.QueryBalanceRequest{
	// 		Address: alloraRewardsAccount.Address,
	// 		Denom:   params.HumanCoinUnit,
	// 	},
	// )
	// require.NoError(m.t, err)

	validatorAddr, err := GetValidatorAddressFromGenesisFile(m)
	require.NoError(m.t, err)
	validatorBalanceBefore, err := m.n.QueryDistribution.ValidatorOutstandingRewards(
		m.ctx,
		&distributiontypes.QueryValidatorOutstandingRewardsRequest{
			ValidatorAddress: validatorAddr,
		},
	)
	require.NoError(m.t, err)

	WaitNumBlocks(m, 1, 10*time.Second)

	validatorBalanceAfter, err := m.n.QueryDistribution.ValidatorOutstandingRewards(
		m.ctx,
		&distributiontypes.QueryValidatorOutstandingRewardsRequest{
			ValidatorAddress: validatorAddr,
		},
	)

	require.True(
		m.t,
		validatorBalanceAfter.Rewards.Rewards.AmountOf(params.HumanCoinUnit).GT(
			validatorBalanceBefore.Rewards.Rewards.AmountOf(params.HumanCoinUnit),
		),
		"validator balance did not increase after a block %s %s",
		validatorBalanceAfter.Rewards.Rewards.String(),
		validatorBalanceBefore.Rewards.Rewards.String(),
	)

}
