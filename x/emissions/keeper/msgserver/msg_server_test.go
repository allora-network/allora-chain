package msgserver_test

import (
	"testing"

	cosmosMath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/allora-network/allora-chain/app/params"
	alloratestutil "github.com/allora-network/allora-chain/test/testutil"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

var _, _, nonAdminAccounts, _ = alloratestutil.GenerateTestAccounts(4)

type MsgServerTestSuite struct {
	testutil.TestSuite
}

func TestMsgServerTestSuite(t *testing.T) {
	suite.Run(t, &MsgServerTestSuite{
		testutil.NewTestSuite("emissions_msg_server"),
	})
}

func (s *MsgServerTestSuite) TestCreateSeveralTopics() {
	creator := s.Addrs(0)

	creatorInitialBalance := types.DefaultParams().CreateTopicFee.Mul(cosmosMath.NewInt(3))
	creatorInitialBalanceCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, creatorInitialBalance))

	err := s.BankKeeper().MintCoins(s.Ctx(), types.AlloraStakingAccountName, creatorInitialBalanceCoins)
	s.Require().NoError(err)
	err = s.BankKeeper().SendCoinsFromModuleToAccount(s.Ctx(), types.AlloraStakingAccountName, creator, creatorInitialBalanceCoins)
	s.Require().NoError(err)

	initialTopicId, err := s.TopicKeeper().GetNextTopicId(s.Ctx())
	s.Require().NoError(err)
	s.Require().NotNil(initialTopicId)

	// Create first topic
	s.CreateTopic()

	result, err := s.TopicKeeper().GetNextTopicId(s.Ctx())
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(initialTopicId+1, result)

	// Create second topic
	s.CreateTopic()

	result, err = s.TopicKeeper().GetNextTopicId(s.Ctx())
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(initialTopicId+2, result)
}
