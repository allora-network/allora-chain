package simulation

import (
	cosmosMath "cosmossdk.io/math"
	"fmt"
	alloraMath "github.com/allora-network/allora-chain/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"log"
	"os"
	"slices"
	"strconv"
)

const LossFileName = "losses.csv"
const RewardFileName = "reward.csv"

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
	epochIndex int,
	blockHeight int64,
	inferers []testCommon.AccountAndAddress,
	forecasters []testCommon.AccountAndAddress,
) {
	lossesStr := strconv.Itoa(epochIndex) + ","
	lossData := getNetworkLossBundleAtBlock(m, topicId, blockHeight)
	for _, inferer := range inferers {
		lossInferers := alloraMath.ZeroDec()
		idx := slices.IndexFunc(lossData.OneOutInfererValues,
			func(value *emissionstypes.WithheldWorkerAttributedValue) bool {
				return value.Worker == inferer.Addr
			})
		if idx != -1 {
			lossInferers = lossData.OneOutInfererValues[idx].Value
		}
		lossesStr += lossInferers.String()
		lossesStr += ","
	}
	for _, forecaster := range forecasters {
		lossForecaster := alloraMath.ZeroDec()
		idx := slices.IndexFunc(lossData.OneOutForecasterValues,
			func(value *emissionstypes.WithheldWorkerAttributedValue) bool {
				return value.Worker == forecaster.Addr
			})
		if idx != -1 {
			lossForecaster = lossData.OneOutForecasterValues[idx].Value
		}
		lossesStr += lossForecaster.String()
		lossesStr += ","
	}

	lossesStr += lossData.CombinedValue.String()
	lossesStr += "\n"

	WriteReport(LossFileName, lossesStr)
}

func RewardReport(
	m testCommon.TestConfig,
	epochIndex int,
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

	rewardsStr := strconv.Itoa(epochIndex) + ","
	for _, reward := range rewardActors {
		rewardsStr += reward.String()
		rewardsStr += ","
	}
	rewardsStr += "\n"
	WriteReport(RewardFileName, rewardsStr)
}
func WorkReport(
	m testCommon.TestConfig,
	topicId uint64,
	epochIndex int,
	blockHeight int64,
	inferers []testCommon.AccountAndAddress,
	forecasters []testCommon.AccountAndAddress,
	reputers []testCommon.AccountAndAddress,
) {
	LossesReport(m, topicId, epochIndex, blockHeight, inferers, forecasters)
	RewardReport(m, epochIndex, inferers, forecasters, reputers)
}
func FormatReport(
	inferers []testCommon.AccountAndAddress,
	forecasters []testCommon.AccountAndAddress,
) {
	lossHeaderStr := "Epoch,"
	rewardHeaderStr := "Epoch,"
	for index, _ := range inferers {
		infererStr := fmt.Sprintf("Inferer%v,", index+1)
		lossHeaderStr += infererStr
		rewardHeaderStr += infererStr
	}
	for index, _ := range forecasters {
		infererStr := fmt.Sprintf("Forecaster%v,", index+1)
		lossHeaderStr += infererStr
		rewardHeaderStr += infererStr
	}
	lossHeaderStr += fmt.Sprintf("Network Inference\n")
	rewardHeaderStr += "\n"
	WriteReport(LossFileName, lossHeaderStr)
	WriteReport(RewardFileName, rewardHeaderStr)
}
