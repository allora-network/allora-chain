package simulation

import (
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"log"
	"os"
	"slices"
)

func WriteReport(fileName string, data string) {
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Fatal(err)
	}
	_, err = file.WriteString(data)
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

	lossesStr := ""
	for _, infererLoss := range lossesInferers {
		lossesStr += infererLoss.String()
		lossesStr += ","
	}
	for _, foreacsterLoss := range lossesForecasters {
		lossesStr += foreacsterLoss.String()
		lossesStr += ","
	}
	lossesStr += lossData.CombinedValue.String()
	lossesStr += "\n"

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

	rewardsStr := ""
	for _, reward := range rewardActors {
		rewardsStr += reward.String()
		rewardsStr += ","
	}
	rewardsStr += "\n"
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
