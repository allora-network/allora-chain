package simulation

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"

	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

const LossFileName = "losses.csv"
const RewardFileName = "rewards.csv"

func getValueFromArray(
	values []*emissionstypes.WorkerAttributedValue,
	actorAddr string,
) alloraMath.Dec {
	res := alloraMath.ZeroDec()
	idx := slices.IndexFunc(values,
		func(value *emissionstypes.WorkerAttributedValue) bool {
			return value.Worker == actorAddr
		})
	if idx != -1 {
		res = values[idx].Value
	}
	return res
}
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
	reputers []testCommon.AccountAndAddress,
	reputerLosses []*emissionstypes.ReputerValueBundle,
	groundTruth []alloraMath.Dec,
) {
	lossesStr := strconv.Itoa(epochIndex) + ","
	lossData := getNetworkLossBundleAtBlock(m, topicId, blockHeight)
	for index, _ := range reputers {
		lossesStr += fmt.Sprintf("%s,", groundTruth[index].String())
		valueBundle := reputerLosses[index].ValueBundle
		for _, inferer := range inferers {
			lossesStr += getValueFromArray(valueBundle.InfererValues, inferer.Addr).String()
			lossesStr += ","
			lossesStr += getValueFromArray(convertWorkerAttributedValueType(valueBundle.OneOutInfererValues), inferer.Addr).String()
			lossesStr += ","
		}
		for _, forecaster := range forecasters {
			lossesStr += getValueFromArray(valueBundle.ForecasterValues, forecaster.Addr).String()
			lossesStr += ","
			lossesStr += getValueFromArray(convertWorkerAttributedValueType(valueBundle.OneOutForecasterValues), forecaster.Addr).String()
			lossesStr += ","
		}
		lossesStr += valueBundle.CombinedValue.String()
		lossesStr += ","
		lossesStr += valueBundle.NaiveValue.String()
		lossesStr += ","
	}
	for _, inferer := range inferers {
		lossesStr += getValueFromArray(lossData.InfererValues, inferer.Addr).String()
		lossesStr += ","
		lossesStr += getValueFromArray(convertWorkerAttributedValueType(lossData.OneOutInfererValues), inferer.Addr).String()
		lossesStr += ","
	}
	for _, forecaster := range forecasters {
		lossesStr += getValueFromArray(lossData.ForecasterValues, forecaster.Addr).String()
		lossesStr += ","
		lossesStr += getValueFromArray(convertWorkerAttributedValueType(lossData.OneOutForecasterValues), forecaster.Addr).String()
		lossesStr += ","
	}

	lossesStr += lossData.CombinedValue.String()
	lossesStr += ","
	lossesStr += lossData.NaiveValue.String()
	lossesStr += "\n"

	WriteReport(strconv.FormatUint(topicId, 10)+LossFileName, lossesStr)
}

func RewardReport(
	m testCommon.TestConfig,
	topicId uint64,
	epochIndex int,
	inferers []testCommon.AccountAndAddress,
	forecasters []testCommon.AccountAndAddress,
	reputers []testCommon.AccountAndAddress,
) {
	rewardActors := make(map[string]alloraMath.Dec)
	workers := make([]testCommon.AccountAndAddress, 0)
	workers = append(workers, inferers...)
	workers = append(workers, forecasters...)
	for _, worker := range workers {
		balance, err := getAccountBalance(m, m.Client.QueryBank(), worker.Addr)
		if err == nil {
			if balance.Amount.GTE(cosmosMath.NewInt(int64(InitFund))) {
				alloBalance, _ := alloraMath.NewDecFromSdkInt(balance.Amount)
				reward, err := alloBalance.Sub(alloraMath.NewDecFromInt64(int64(InitFund)))
				if err != nil {
					continue
				}
				rewardActors[worker.Addr] = reward
			}
		}
	}

	rewardsStr := strconv.Itoa(epochIndex) + ","
	for _, reward := range workers {
		rewardsStr += rewardActors[reward.Addr].String()
		rewardsStr += ","
	}
	for _, reputer := range reputers {
		stake, err := getReputerStake(m.Ctx, m.Client.QueryEmissions(), topicId, reputer.Addr)
		addValue := alloraMath.ZeroDec()
		if err == nil {
			if stake.Gte(alloraMath.NewDecFromInt64(int64(StakeToAdd))) {
				addValue, _ = stake.Sub(alloraMath.NewDecFromInt64(int64(StakeToAdd)))
			}
		}
		rewardsStr += addValue.String()
		rewardsStr += ","
	}
	rewardsStr += "\n"
	WriteReport(strconv.FormatUint(topicId, 10)+RewardFileName, rewardsStr)
}
func WorkReport(
	m testCommon.TestConfig,
	topicId uint64,
	epochIndex int,
	blockHeight int64,
	inferers []testCommon.AccountAndAddress,
	forecasters []testCommon.AccountAndAddress,
	reputers []testCommon.AccountAndAddress,
	reputerLosses []*emissionstypes.ReputerValueBundle,
	groundTruth []alloraMath.Dec,
) {
	LossesReport(m, topicId, epochIndex, blockHeight, inferers, forecasters, reputers, reputerLosses, groundTruth)
	RewardReport(m, topicId, epochIndex, inferers, forecasters, reputers)
}
func FormatReport(
	topicId uint64,
	inferers []testCommon.AccountAndAddress,
	forecasters []testCommon.AccountAndAddress,
	reputers []testCommon.AccountAndAddress,
) {
	lossHeaderStr := "Epoch,"
	rewardHeaderStr := "Epoch,"

	for rIndex, _ := range reputers {
		lossHeaderStr += fmt.Sprintf("Reputer %v-GroundTruth,", rIndex+1)
		for index, _ := range inferers {
			lossHeaderStr += fmt.Sprintf("Reputer%v-Inference%v,", rIndex+1, index+1)
			lossHeaderStr += fmt.Sprintf("Reputer%v-OneOutInferer%v,", rIndex+1, index+1)
		}
		for index, _ := range forecasters {
			lossHeaderStr += fmt.Sprintf("Reputer%v-Forecast%v,", rIndex+1, index+1)
			lossHeaderStr += fmt.Sprintf("Reputer%v-OneOutForecaster%v,", rIndex+1, index+1)
		}
		lossHeaderStr += fmt.Sprintf("Reputer%v-CombinedValue,", rIndex+1)
		lossHeaderStr += fmt.Sprintf("Reputer%v-NaiveValue,", rIndex+1)
	}
	for index, _ := range inferers {
		lossHeaderStr += fmt.Sprintf("Network-Inference%v,", index+1)
		lossHeaderStr += fmt.Sprintf("Network-OneOutInferer%v,", index+1)
		rewardHeaderStr += fmt.Sprintf("Inferer%d,", index+1)
	}
	for index, _ := range forecasters {
		lossHeaderStr += fmt.Sprintf("Network-Forecast%v,", index+1)
		lossHeaderStr += fmt.Sprintf("Network-OneOutForecaster%v,", index+1)
		rewardHeaderStr += fmt.Sprintf("Forecaster%d,", index+1)
	}
	for index, _ := range reputers {
		rewardHeaderStr += fmt.Sprintf("Reputer%d,", index+1)
	}
	lossHeaderStr += fmt.Sprintf("Network-CombinedValue,")
	lossHeaderStr += fmt.Sprintf("Network-NaiveValue\n")
	rewardHeaderStr += "\n"
	WriteReport(strconv.FormatUint(topicId, 10)+LossFileName, lossHeaderStr)
	WriteReport(strconv.FormatUint(topicId, 10)+RewardFileName, rewardHeaderStr)
}
