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
func GetValidatorAddressesFromGenesisFile(m TestMetadata) ([]string, error) {
	home := m.n.Client.Context().HomeDir
	genesisPath := home + "/config/genesis.json"

	file, err := os.Open(genesisPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var addresses []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "\"validator_address\":") {
			splitted := strings.Split(line, ":")
			require.Equal(m.t, len(splitted), 2)
			trimmed := strings.TrimSpace(splitted[1])
			trimmed = strings.Trim(trimmed, ",")
			trimmed = strings.Trim(trimmed, "\"")
			addresses = append(addresses, trimmed)

			if len(addresses) == 3 {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(addresses) < 3 {
		return nil, errors.New("not enough validator addresses found")
	}

	return addresses, nil
}

func CheckValidatorBalancesIncreaseOnNewBlock(m TestMetadata) {
	validatorAddrs, err := GetValidatorAddressesFromGenesisFile(m)
	require.NoError(m.t, err)
	require.Len(m.t, validatorAddrs, 3, "Expected exactly three validator addresses")

	balancesBefore := make(map[string]*distributiontypes.QueryValidatorOutstandingRewardsResponse)

	for _, addr := range validatorAddrs {
		response, err := m.n.QueryDistribution.ValidatorOutstandingRewards(
			m.ctx,
			&distributiontypes.QueryValidatorOutstandingRewardsRequest{
				ValidatorAddress: addr,
			},
		)
		require.NoError(m.t, err)
		balancesBefore[addr] = response
	}

	err = m.n.Client.WaitForNextBlock(m.ctx)
	require.NoError(m.t, err)

	balanceIncreased := false

	for _, addr := range validatorAddrs {
		balanceAfter, err := m.n.QueryDistribution.ValidatorOutstandingRewards(
			m.ctx,
			&distributiontypes.QueryValidatorOutstandingRewardsRequest{
				ValidatorAddress: addr,
			},
		)
		require.NoError(m.t, err)

		vba := balanceAfter.Rewards.Rewards.AmountOf(params.BaseCoinUnit)
		vbb := balancesBefore[addr].Rewards.Rewards.AmountOf(params.BaseCoinUnit)

		if vba.GT(vbb) {
			balanceIncreased = true
			break
		}
	}

	require.True(
		m.t,
		balanceIncreased,
		"None of the validator balances increased after a new block",
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
	// TODO
	// * Fund a topic
	// * Make inferences + forecasts
	// * Wait for "topic ground truth lag" number of blocks to pass
	// * Repute
	// * Wait for epoch to end
	// * Check validator balances increase
	// Bad form to have a test that depends on another test, but might be most expedient to rely
	// on the existing tests to do each of the above.
	m.t.Log("--- Check Validator Balance Goes Up When New Blocks Are Mined  ---")
	CheckValidatorBalancesIncreaseOnNewBlock(m)
	m.t.Log("--- Check Allora Rewards Module Account Balance Goes Up When New Blocks Are Mined  ---")
	CheckAlloraRewardsBalanceGoesUpOnNewBlock(m)
}
