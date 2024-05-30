package integration_test

import (
	"bufio"
	"errors"
	"os"
	"strings"

	"github.com/allora-network/allora-chain/app/params"
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/stretchr/testify/require"
)

// Get the validator address that is stored in the genesis file
// didn't know of a better way to get a validator address to check with
func GetValidatorAddressesFromGenesisFile(m testCommon.TestConfig) ([]string, error) {
	home := m.AlloraHomeDir
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
			require.Equal(m.T, len(splitted), 2)
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

func CheckValidatorBalancesIncreaseOnNewBlock(m testCommon.TestConfig) {
	validatorAddrs, err := GetValidatorAddressesFromGenesisFile(m)
	require.NoError(m.T, err)
	require.Len(m.T, validatorAddrs, 3, "Expected exactly three validator addresses")

	balancesBefore := make(map[string]*distributiontypes.QueryValidatorOutstandingRewardsResponse)

	for _, addr := range validatorAddrs {
		response, err := m.Client.QueryDistribution().ValidatorOutstandingRewards(
			m.Ctx,
			&distributiontypes.QueryValidatorOutstandingRewardsRequest{
				ValidatorAddress: addr,
			},
		)
		require.NoError(m.T, err)
		balancesBefore[addr] = response
	}

	err = m.Client.WaitForNextBlock(m.Ctx)
	require.NoError(m.T, err)

	balanceIncreased := false

	for _, addr := range validatorAddrs {
		balanceAfter, err := m.Client.QueryDistribution().ValidatorOutstandingRewards(
			m.Ctx,
			&distributiontypes.QueryValidatorOutstandingRewardsRequest{
				ValidatorAddress: addr,
			},
		)
		require.NoError(m.T, err)

		vba := balanceAfter.Rewards.Rewards.AmountOf(params.BaseCoinUnit)
		vbb := balancesBefore[addr].Rewards.Rewards.AmountOf(params.BaseCoinUnit)

		if vba.GT(vbb) {
			balanceIncreased = true
			break
		}
	}

	require.True(
		m.T,
		balanceIncreased,
		"None of the validator balances increased after a new block",
	)
}

// the mint module pays the ecosystem module account
// as new blocks are produced.
func CheckAlloraRewardsBalanceGoesUpOnNewBlock(m testCommon.TestConfig) {
	alloraRewardsModuleAccResponse, err := m.Client.QueryAuth().ModuleAccountByName(
		m.Ctx,
		&authtypes.QueryModuleAccountByNameRequest{
			Name: emissionstypes.AlloraRewardsAccountName,
		},
	)
	require.NoError(m.T, err)
	var alloraRewardsModuleAcc authtypes.ModuleAccount
	err = m.Cdc.Unmarshal(
		alloraRewardsModuleAccResponse.Account.Value,
		&alloraRewardsModuleAcc,
	)
	require.NoError(m.T, err)

	alloraRewardsBalanceBefore, err := m.Client.QueryBank().Balance(
		m.Ctx,
		&banktypes.QueryBalanceRequest{
			Address: alloraRewardsModuleAcc.Address,
			Denom:   params.BaseCoinUnit,
		},
	)
	require.NoError(m.T, err)

	err = m.Client.WaitForNextBlock(m.Ctx)
	require.NoError(m.T, err)

	alloraRewardsBalanceAfter, err := m.Client.QueryBank().Balance(
		m.Ctx,
		&banktypes.QueryBalanceRequest{
			Address: alloraRewardsModuleAcc.Address,
			Denom:   params.BaseCoinUnit,
		},
	)
	require.NoError(m.T, err)

	arba := alloraRewardsBalanceAfter.Balance.Amount
	arbb := alloraRewardsBalanceBefore.Balance.Amount
	require.True(
		m.T,
		arba.GT(arbb),
		"Allora Rewards module account balance did not increase after a block %s %s",
		arba.String(),
		arbb.String(),
	)
}

// this file tests that the distribution of funds to validators
// and rewards accounts is working as expected
// basically testing the forked mint module that we use
func DistributionChecks(m testCommon.TestConfig) {
	// TODO
	// * Fund a topic
	// * Make inferences + forecasts
	// * Wait for "topic ground truth lag" number of blocks to pass
	// * Repute
	// * Wait for epoch to end
	// * Check validator balances increase
	// Bad form to have a test that depends on another test, but might be most expedient to rely
	// on the existing tests to do each of the above.
	m.T.Log("--- Check Validator Balance Goes Up When New Blocks Are Mined  ---")
	CheckValidatorBalancesIncreaseOnNewBlock(m)
	m.T.Log(
		"--- Check Allora Rewards Module Account Balance Goes Up When New Blocks Are Mined  ---",
	)
	CheckAlloraRewardsBalanceGoesUpOnNewBlock(m)
}
