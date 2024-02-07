package module_test

import (
	"context"
	"fmt"

	cosmosMath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	state "github.com/upshot-tech/protocol-state-machine-module"
	"github.com/upshot-tech/protocol-state-machine-module/module"
)

func (s *ModuleTestSuite) TestGetParticipantEmissionsForTopicNoError() {
	topicId, err := s.mockTopic(s.ctx)
	s.NoError(err, "Error creating topic")
	_, err = s.mockTopic(s.ctx)
	s.NoError(err, "Error creating topic 2")
	s.Equal(uint64(1), topicId, "Topic ID should start at 1")
	err = s.mockSomeReputers(topicId)
	s.NoError(err, "Error creating reputers")
	err = s.mockSomeWorkers(topicId)
	s.NoError(err, "Error creating workers")
	topicEmissions := cosmosMath.NewUint(2000)
	cumulativeEmissions := cosmosMath.NewUint(5000)
	totalStake := cosmosMath.NewUint(10000)
	_, _, err = module.GetParticipantEmissionsForTopic(
		s.ctx,
		s.appModule,
		topicId,
		&topicEmissions,
		&cumulativeEmissions,
		&totalStake,
	)
	//s.NoError(err, "Cumulative emissions for zero blocks should just be 0")
}

func (s *ModuleTestSuite) TestEmitRewardsSimple() {
	// mock mint the coins to reputers
	// mock mint the coins to workers
	// have the reputers upload some weights
	// increment the block number
	// call endblock on the module somehow
}

// mock mint coins to participants
func (s *ModuleTestSuite) mockMintRewardCoins(amount []cosmosMath.Int, target []sdk.AccAddress) error {
	if len(amount) != len(target) {
		return fmt.Errorf("amount and target must be the same length")
	}
	for i, addr := range target {
		coins := sdk.NewCoins(sdk.NewCoin("upt", amount[i]))
		s.bankKeeper.MintCoins(s.ctx, s.appModule.Name(), coins)
		s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, s.appModule.Name(), addr, coins)
	}
	return nil
}

// give some reputers coins, have them stake those coins
func (s *ModuleTestSuite) mockSomeReputers(topicId uint64) error {
	reputerAddrs := []sdk.AccAddress{
		sdk.AccAddress([]byte("reputer1_______________")),
		sdk.AccAddress([]byte("reputer2_______________")),
	}
	reputerAmounts := []cosmosMath.Int{
		cosmosMath.NewInt(1337),
		cosmosMath.NewInt(6969),
	}
	err := s.mockMintRewardCoins(
		reputerAmounts,
		reputerAddrs,
	)
	if err != nil {
		return err
	}
	_, err = s.msgServer.RegisterReputer(s.ctx, &state.MsgRegisterReputer{
		Creator:      reputerAddrs[0].String(),
		LibP2PKey:    "libp2pkeyReputer1",
		MultiAddress: "multiaddressReputer1",
		TopicId:      topicId,
		InitialStake: cosmosMath.NewUintFromBigInt(reputerAmounts[0].BigInt()),
	})
	if err != nil {
		return err
	}
	_, err = s.msgServer.RegisterReputer(s.ctx, &state.MsgRegisterReputer{
		Creator:      reputerAddrs[1].String(),
		LibP2PKey:    "libp2pkeyReputer2",
		MultiAddress: "multiaddressReputer2",
		TopicId:      topicId,
		InitialStake: cosmosMath.NewUintFromBigInt(reputerAmounts[1].BigInt()),
	})
	if err != nil {
		return err
	}
	return nil
}

// give some workers coins, have them stake those coins
func (s *ModuleTestSuite) mockSomeWorkers(topicId uint64) error {
	// copy reputer code
	// then get total stake from keeper and put it into function above
	return nil
}

// create a topic
func (s *ModuleTestSuite) mockTopic(ctx context.Context) (uint64, error) {
	topicMessage := state.MsgCreateNewTopic{
		Creator:          "",
		Metadata:         "",
		WeightLogic:      "",
		WeightMethod:     "",
		WeightCadence:    0,
		InferenceLogic:   "",
		InferenceMethod:  "",
		InferenceCadence: 0,
		Active:           true,
	}
	response, err := s.msgServer.CreateNewTopic(ctx, &topicMessage)
	if err != nil {
		return 0, err
	}
	return response.TopicId, nil
}
