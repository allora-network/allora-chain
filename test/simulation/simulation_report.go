package simulation

import (
	cosmosMath "cosmossdk.io/math"
	"encoding/csv"
	alloraMath "github.com/allora-network/allora-chain/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"log"
	"os"
	"slices"
)

func WriteReport(fileName string, data []string) {
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	w := csv.NewWriter(file)
	defer w.Flush()
	if err := w.Write(data); err != nil {
		log.Fatalln("error writing record to file", err)
	}
	defer file.Close()
}
func LossesReport(
	m testCommon.TestConfig,
	topicId uint64,
	blockHeight int64,
	inferers []testCommon.AccountAndAddress,
	forecasters []testCommon.AccountAndAddress,
) {
	lossesInferers := make(map[string]alloraMath.Dec)
	lossesForecasters := make(map[string]alloraMath.Dec)

	lossData := getNetworkLossBundleAtBlock(m, topicId, blockHeight)
	for _, inferer := range inferers {
		lossesInferers[inferer.Addr] = alloraMath.ZeroDec()
		idx := slices.IndexFunc(lossData.OneOutInfererValues,
			func(value *emissionstypes.WithheldWorkerAttributedValue) bool {
				return value.Worker == inferer.Addr
			})
		if idx == -1 {
			continue
		}
		lossesInferers[inferer.Addr] = lossData.OneOutInfererValues[idx].Value
	}
	for _, forecaster := range forecasters {
		lossesInferers[forecaster.Addr] = alloraMath.ZeroDec()
		idx := slices.IndexFunc(lossData.OneOutForecasterValues,
			func(value *emissionstypes.WithheldWorkerAttributedValue) bool {
				return value.Worker == forecaster.Addr
			})
		if idx == -1 {
			continue
		}
		lossesForecasters[forecaster.Addr] = lossData.OneOutForecasterValues[idx].Value
	}

	lossesStr := make([]string, 0)
	for _, infererLoss := range lossesInferers {
		lossesStr = append(lossesStr, infererLoss.String())
	}
	for _, foreacsterLoss := range lossesForecasters {
		lossesStr = append(lossesStr, foreacsterLoss.String())
	}
	lossesStr = append(lossesStr, lossData.CombinedValue.String())

	WriteReport("losses.csv", lossesStr)
}

func RewardReport(
	m testCommon.TestConfig,
	inferers []testCommon.AccountAndAddress,
	forecasters []testCommon.AccountAndAddress,
	reputers []testCommon.AccountAndAddress,
) {
	rewardActors := make(map[string]alloraMath.Dec)
	actors := make([]testCommon.AccountAndAddress, 0)
	actors = append(actors, inferers...)
	actors = append(actors, forecasters...)
	actors = append(actors, reputers...)
	for _, actor := range actors {
		balance, err := getAccountBalance(m, m.Client.QueryBank(), actor.Addr)
		if err == nil {
			if balance.Amount.GTE(cosmosMath.NewInt(int64(InitFund))) {
				alloBalance, _ := alloraMath.NewDecFromSdkInt(balance.Amount)
				reward, err := alloBalance.Sub(alloraMath.NewDecFromInt64(int64(InitFund)))
				if err != nil {
					continue
				}
				rewardActors[actor.Addr] = reward
			}
		}
	}

	rewardsStr := make([]string, 0)
	for _, reward := range rewardActors {
		rewardsStr = append(rewardsStr, reward.String())
	}
	WriteReport("rewards.csv", rewardsStr)
}
func WorkReport(
	m testCommon.TestConfig,
	topicId uint64,
	blockHeight int64,
	inferers []testCommon.AccountAndAddress,
	forecasters []testCommon.AccountAndAddress,
	reputers []testCommon.AccountAndAddress,
) {
	LossesReport(m, topicId, blockHeight, inferers, forecasters)
	RewardReport(m, inferers, forecasters, reputers)
}
