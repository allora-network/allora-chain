package emissions

import "cosmossdk.io/collections"

const ModuleName = "emissions"
const AlloraStakingAccountName = "allorastaking"
const AlloraRequestsAccountName = "allorarequests"
const AlloraRewardsAccountName = "allorarewards"

var (
	ParamsKey                     = collections.NewPrefix(0)
	TotalStakeKey                 = collections.NewPrefix(1)
	TopicStakeKey                 = collections.NewPrefix(2)
	LastRewardsUpdateKey          = collections.NewPrefix(3)
	RewardsKey                    = collections.NewPrefix(4)
	NextTopicIdKey                = collections.NewPrefix(5)
	TopicsKey                     = collections.NewPrefix(6)
	TopicWorkersKey               = collections.NewPrefix(7)
	TopicReputersKey              = collections.NewPrefix(8)
	DelegatorStakeKey             = collections.NewPrefix(9)
	BondsKey                      = collections.NewPrefix(10)
	TargetStakeKey                = collections.NewPrefix(11)
	WeightsKey                    = collections.NewPrefix(12)
	InferencesKey                 = collections.NewPrefix(13)
	WorkerNodesKey                = collections.NewPrefix(14)
	ReputerNodesKey               = collections.NewPrefix(15)
	LatestInferencesTsKey         = collections.NewPrefix(16)
	MempoolKey                    = collections.NewPrefix(17)
	RequestUnmetDemandKey         = collections.NewPrefix(18)
	TopicUnmetDemandKey           = collections.NewPrefix(19)
	AllInferencesKey              = collections.NewPrefix(20)
	StakeRemovalQueueKey          = collections.NewPrefix(21)
	AllTopicStakeSumKey           = collections.NewPrefix(22)
	AddressTopicsKey              = collections.NewPrefix(23)
	AccumulatedMetDemandKey       = collections.NewPrefix(24)
	NumInferencesInRewardEpochKey = collections.NewPrefix(25)
	WhitelistAdminsKey            = collections.NewPrefix(26)
	TopicCreationWhitelistKey     = collections.NewPrefix(27)
	WeightSettingWhitelistKey     = collections.NewPrefix(28)
	ChurnReadyTopicsKey           = collections.NewPrefix(30)
)
