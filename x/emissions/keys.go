package emissions

import "cosmossdk.io/collections"

const ModuleName = "emissions"
const AlloraStakingModuleName = "allorastaking"
const AlloraRequestsModuleName = "allorarequests"

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
	ForecastsKey                  = collections.NewPrefix(14)
	WorkerNodesKey                = collections.NewPrefix(15)
	ReputerNodesKey               = collections.NewPrefix(16)
	LatestInferencesTsKey         = collections.NewPrefix(17)
	MempoolKey                    = collections.NewPrefix(18)
	RequestUnmetDemandKey         = collections.NewPrefix(19)
	TopicUnmetDemandKey           = collections.NewPrefix(20)
	AllInferencesKey              = collections.NewPrefix(21)
	AllForecastsKey               = collections.NewPrefix(22)
	StakeRemovalQueueKey          = collections.NewPrefix(23)
	AllTopicStakeSumKey           = collections.NewPrefix(24)
	AddressTopicsKey              = collections.NewPrefix(25)
	AccumulatedMetDemandKey       = collections.NewPrefix(26)
	NumInferencesInRewardEpochKey = collections.NewPrefix(27)
	WhitelistAdminsKey            = collections.NewPrefix(28)
	TopicCreationWhitelistKey     = collections.NewPrefix(29)
	WeightSettingWhitelistKey     = collections.NewPrefix(30)
	ChurnReadyTopicsKey           = collections.NewPrefix(31)
	FoundationWhitelistKey        = collections.NewPrefix(32)
)
