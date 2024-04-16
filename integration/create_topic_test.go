package integration_test

import (
	"fmt"

	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *ExternalTestSuite) TestGetParams() {
	paramsReq := &emissionstypes.QueryParamsRequest{}
	p, err := s.n.QueryClient.Params(
		s.ctx,
		paramsReq,
	)
	s.Require().NoError(err)
	s.Require().NotNil(p)
	fmt.Println(p)
}
