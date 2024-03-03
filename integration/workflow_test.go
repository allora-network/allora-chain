package workflow_test

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type WorkflowTestSuite struct {
	suite.Suite
	RpcUrl     string
	HttpClient http.Client
}

func (s *WorkflowTestSuite) SetupTest() {
	s.RpcUrl = "http://127.0.0.1:1317"
	s.HttpClient = http.Client{Timeout: 10 * time.Second}
}

func TestWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(WorkflowTestSuite))
}

func (s *WorkflowTestSuite) TestQueryNextTopicId() {
	if os.Getenv("INTEGRATION") == "" {
		s.T().Skip("Skipping testing in non Integration Environment")
	}
	nextTopicIdPath := "/emissions/state/v1/next_topic_id"
	res, err := s.HttpClient.Get(s.RpcUrl + nextTopicIdPath)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, res.StatusCode)
	contentType := res.Header["Content-Type"]
	s.Require().Equal("application/json", contentType[0])
	type topicIdResponse struct {
		NextTopicId json.Number `json:"next_topic_id,string"`
	}
	target := &topicIdResponse{}
	defer res.Body.Close()
	s.Require().NoError(err)
	jsonDecoder := json.NewDecoder(res.Body)
	jsonDecoder.DisallowUnknownFields()
	err = jsonDecoder.Decode(target)
	s.Require().NoError(err)
	nextTopicId, err := target.NextTopicId.Int64()
	s.Require().NoError(err)
	s.Require().Greater(nextTopicId, int64(0))
}

//func (s *WorkflowTestSuite) TestTopicCreate() {
//}
