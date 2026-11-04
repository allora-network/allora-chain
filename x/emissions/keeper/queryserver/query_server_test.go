package queryserver_test

import (
	"testing"

	cosmosMath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

type QueryServerTestSuite struct {
	testutil.TestSuite
}

func TestQueryServerTestSuite(t *testing.T) {
	suite.Run(t, &QueryServerTestSuite{
		testutil.NewTestSuite("emissions_query_server"),
	})
}

func (s *QueryServerTestSuite) TestCreateSeveralTopics() {
	ctx, msgServer := s.Ctx(), s.EmissionsMsgServer()
	require := s.Require()
	// Mock setup for metadata and validation steps
	metadata := "Some metadata for the new topic"
	// Create a CreateNewTopicRequest message

	creator := s.Addrs(0)

	newTopicMsg := s.MockTopicMsg()
	newTopicMsg.Metadata = metadata

	creatorInitialBalance := types.DefaultParams().CreateTopicFee.Mul(cosmosMath.NewInt(3))
	creatorInitialBalanceCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, creatorInitialBalance))

	err := s.BankKeeper().MintCoins(ctx, types.AlloraStakingAccountName, creatorInitialBalanceCoins)
	s.Require().NoError(err)
	err = s.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.AlloraStakingAccountName, creator, creatorInitialBalanceCoins)
	s.Require().NoError(err)

	initialTopicId, err := s.TopicKeeper().GetNextTopicId(s.Ctx())
	s.Require().NoError(err)
	s.Require().NotNil(initialTopicId)

	_, err = msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on first creation")

	result, err := s.TopicKeeper().GetNextTopicId(s.Ctx())
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(initialTopicId+1, result)

	// Create second topic
	_, err = msgServer.CreateNewTopic(ctx, newTopicMsg)
	require.NoError(err, "CreateTopic fails on second topic")

	result, err = s.TopicKeeper().GetNextTopicId(s.Ctx())
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(initialTopicId+2, result)
}
