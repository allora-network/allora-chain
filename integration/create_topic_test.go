package integration_test

import "fmt"

func (s *ExternalTestSuite) TestGetParams() {
	p, err := s.n.QueryParams()
	s.Require().NoError(err)
	s.Require().NotNil(p)
	fmt.Println(p)
}
