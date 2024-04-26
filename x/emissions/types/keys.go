package types

import "cosmossdk.io/collections"

const (
	ModuleName                                = "emissions"
	StoreKey                                  = "emissions"
	AlloraStakingAccountName                  = "allorastaking"
	AlloraRequestsAccountName                 = "allorarequests"
	AlloraRewardsAccountName                  = "allorarewards"
	AlloraPendingRewardForDelegatorAccoutName = "allorapendingrewards"
)

const (
	oneE18 = "1000000000000000000"
)

var (
	ParamsKey                          = collections.NewPrefix(0)
	TotalStakeKey                      = collections.NewPrefix(1)
	TopicStakeKey                      = collections.NewPrefix(2)
	LastRewardsUpdateKey               = collections.NewPrefix(3)
	RewardsKey                         = collections.NewPrefix(4)
	NextTopicIdKey                     = collections.NewPrefix(5)
	TopicsKey                          = collections.NewPrefix(6)
	TopicWorkersKey                    = collections.NewPrefix(7)
	TopicReputersKey                   = collections.NewPrefix(8)
	DelegatorStakeKey                  = collections.NewPrefix(9)
	DelegateStakePlacementKey          = collections.NewPrefix(10)
	TargetStakeKey                     = collections.NewPrefix(11)
	InferencesKey                      = collections.NewPrefix(12)
	ForecastsKey                       = collections.NewPrefix(13)
	WorkerNodesKey                     = collections.NewPrefix(14)
	ReputerNodesKey                    = collections.NewPrefix(15)
	LatestInferencesTsKey              = collections.NewPrefix(16)
	ActiveTopicsKey                    = collections.NewPrefix(17)
	RequestUnmetDemandKey              = collections.NewPrefix(18)
	TopicUnmetDemandKey                = collections.NewPrefix(19)
	AllInferencesKey                   = collections.NewPrefix(20)
	AllForecastsKey                    = collections.NewPrefix(21)
	AllLossBundlesKey                  = collections.NewPrefix(22)
	StakeRemovalKey                    = collections.NewPrefix(23)
	StakeByReputerAndTopicId           = collections.NewPrefix(24)
	DelegateStakeRemovalKey            = collections.NewPrefix(25)
	AllTopicStakeSumKey                = collections.NewPrefix(26)
	AddressTopicsKey                   = collections.NewPrefix(27)
	WhitelistAdminsKey                 = collections.NewPrefix(28)
	TopicCreationWhitelistKey          = collections.NewPrefix(29)
	ReputerWhitelistKey                = collections.NewPrefix(30)
	ChurnReadyTopicsKey                = collections.NewPrefix(31)
	NetworkLossBundlesKey              = collections.NewPrefix(32)
	NetworkRegretsKey                  = collections.NewPrefix(33)
	StakeByReputerAndTopicIdKey        = collections.NewPrefix(34)
	ReputerScoresKey                   = collections.NewPrefix(35)
	InferenceScoresKey                 = collections.NewPrefix(36)
	ForecastScoresKey                  = collections.NewPrefix(37)
	ReputerListeningCoefficientKey     = collections.NewPrefix(38)
	InfererNetworkRegretsKey           = collections.NewPrefix(39)
	ForecasterNetworkRegretsKey        = collections.NewPrefix(40)
	OneInForecasterNetworkRegretsKey   = collections.NewPrefix(41)
	UnfulfilledWorkerNoncesKey         = collections.NewPrefix(42)
	UnfulfilledReputerNoncesKey        = collections.NewPrefix(43)
	FeeRevenueEpochKey                 = collections.NewPrefix(44)
	TopicFeeRevenueKey                 = collections.NewPrefix(45)
	PreviousTopicWeightKey             = collections.NewPrefix(46)
	PreviousReputerRewardFractionKey   = collections.NewPrefix(47)
	PreviousInferenceRewardFractionKey = collections.NewPrefix(48)
	PreviousForecastRewardFractionKey  = collections.NewPrefix(49)
	LatestInfererScoresByWorkerKey     = collections.NewPrefix(50)
	LatestForecasterScoresByWorkerKey  = collections.NewPrefix(51)
	LatestReputerScoresByReputerKey    = collections.NewPrefix(52)
	TopicRewardNonceKey                = collections.NewPrefix(53)
	RequestsKey                        = collections.NewPrefix(54)
	TopicRequestsKey                   = collections.NewPrefix(55)
	NumRequestsPerTopicKey             = collections.NewPrefix(56)
	DelegateRewardPerShare             = collections.NewPrefix(57)
)
