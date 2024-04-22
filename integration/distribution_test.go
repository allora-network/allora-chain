package integration_test

import (
	"bufio"
	"errors"
	"os"
	"strings"

	"github.com/allora-network/allora-chain/app/params"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
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
			return trimmed, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", errors.New("validator address not found")
}

func CheckValidatorBalanceGoesUpOnNewBlock(m TestMetadata) {
	validatorAddr, err := GetValidatorAddressFromGenesisFile(m)
	require.NoError(m.t, err)
	validatorBalanceBefore, err := m.n.QueryDistribution.ValidatorOutstandingRewards(
		m.ctx,
		&distributiontypes.QueryValidatorOutstandingRewardsRequest{
			ValidatorAddress: validatorAddr,
		},
	)
	require.NoError(m.t, err)

	err = m.n.Client.WaitForNextBlock(m.ctx)
	require.NoError(m.t, err)

	validatorBalanceAfter, err := m.n.QueryDistribution.ValidatorOutstandingRewards(
		m.ctx,
		&distributiontypes.QueryValidatorOutstandingRewardsRequest{
			ValidatorAddress: validatorAddr,
		},
	)
	require.NoError(m.t, err)

	vba := validatorBalanceAfter.Rewards.Rewards.AmountOf(params.BaseCoinUnit)
	vbb := validatorBalanceBefore.Rewards.Rewards.AmountOf(params.BaseCoinUnit)
	require.True(
		m.t,
		vba.GT(vbb),
		"validator balance did not increase after a block %s %s",
		vba.String(),
		vbb.String(),
	)
}

// the mint module pays the ecosystem module account
// as new blocks are produced.
func CheckAlloraRewardsBalanceGoesUpOnNewBlock(m TestMetadata) {
	alloraRewardsModuleAccResponse, err := m.n.QueryAuth.ModuleAccountByName(
		m.ctx,
		&authtypes.QueryModuleAccountByNameRequest{
			Name: emissionstypes.AlloraRewardsAccountName,
		},
	)
	require.NoError(m.t, err)
	var alloraRewardsModuleAcc authtypes.ModuleAccount
	err = m.n.Cdc.Unmarshal(
		alloraRewardsModuleAccResponse.Account.Value,
		&alloraRewardsModuleAcc,
	)
	require.NoError(m.t, err)

	alloraRewardsBalanceBefore, err := m.n.QueryBank.Balance(
		m.ctx,
		&banktypes.QueryBalanceRequest{
			Address: alloraRewardsModuleAcc.Address,
			Denom:   params.BaseCoinUnit,
		},
	)
	require.NoError(m.t, err)

	err = m.n.Client.WaitForNextBlock(m.ctx)
	require.NoError(m.t, err)

	alloraRewardsBalanceAfter, err := m.n.QueryBank.Balance(
		m.ctx,
		&banktypes.QueryBalanceRequest{
			Address: alloraRewardsModuleAcc.Address,
			Denom:   params.BaseCoinUnit,
		},
	)
	require.NoError(m.t, err)

	arba := alloraRewardsBalanceAfter.Balance.Amount
	arbb := alloraRewardsBalanceBefore.Balance.Amount
	require.True(
		m.t,
		arba.GT(arbb),
		"Allora Rewards module account balance did not increase after a block %s %s",
		arba.String(),
		arbb.String(),
	)
}

// this file tests that the distribution of funds to validators
// and rewards accounts is working as expected
// basically testing the forked mint module that we use
func DistributionChecks(m TestMetadata) {
	m.t.Log("--- Check Validator Balance Goes Up When New Blocks Are Mined  ---")
	CheckValidatorBalanceGoesUpOnNewBlock(m)
	m.t.Log("--- Check Allora Rewards Module Account Balance Goes Up When New Blocks Are Mined  ---")
	CheckAlloraRewardsBalanceGoesUpOnNewBlock(m)
}
