package workflow_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/allora-network/allora-chain/app/params"
	emissions "github.com/allora-network/allora-chain/x/emissions"
	"github.com/cosmos/cosmos-sdk/client/tx"
	codectestutil "github.com/cosmos/cosmos-sdk/codec/testutil"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
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
	nextTopicIdPath := s.RpcUrl + "/emissions/state/v1/next_topic_id"
	res, err := s.HttpClient.Get(nextTopicIdPath)
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

func (s *WorkflowTestSuite) TestTopicCreate() {
	if os.Getenv("INTEGRATION") == "" {
		s.T().Skip("Skipping testing in non Integration Environment")
	}
	envAlicePk := os.Getenv("ALICE_PK")
	envAliceAddr := os.Getenv("ALICE_ADDRESS")
	s.Require().NotEmpty(envAlicePk)
	s.Require().NotEmpty(envAliceAddr)
	params.SetAddressPrefixes()
	aliceAddr, err := sdk.AccAddressFromBech32(envAliceAddr)
	s.Require().NoError(err)
	alicePKBytes, err := hex.DecodeString(envAlicePk)
	s.Require().NoError(err)
	alicePk := secp256k1.PrivKey{Key: alicePKBytes}
	computedAliceAddr := sdk.AccAddress(alicePk.PubKey().Address())
	s.Require().Equal(aliceAddr, computedAliceAddr)

	createTopicMsg := emissions.MsgCreateNewTopic{
		Creator:          envAliceAddr,
		Metadata:         "ETH prediction in 24h",
		WeightLogic:      "bafybeih6yjjf2v7qp3wm6hodvjcdlj7galu7dufirvcekzip5gd7bthq",
		WeightMethod:     "eth-price-weights-calc.wasm",
		WeightCadence:    3600,
		InferenceLogic:   "bafybeigpiwl3o73zvvl6dxdqu7zqcub5mhg65iky2xqb4rdhfmikswzqm",
		InferenceMethod:  "allora-inference-function.wasm",
		InferenceCadence: 300,
		DefaultArg:       "ETH",
	}
	cdc := codectestutil.CodecOptions{}.NewCodec()
	txConfig := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)
	txBuilder := txConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(&createTopicMsg)
	s.Require().NoError(err)

	// First round: we gather all the signer infos. We use the "set empty
	// signature" hack to do that.
	var sigsV2 []signing.SignatureV2 = make([]signing.SignatureV2, 0)
	sigsV2 = append(sigsV2, signing.SignatureV2{
		PubKey: alicePk.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: nil,
		},
		Sequence: 100, // todo get sequence from rpc FML
	})
	err = txBuilder.SetSignatures(sigsV2...)
	s.Require().NoError(err)

	// Second round: all signer infos are set, so each signer can sign.
	sigsV2 = []signing.SignatureV2{}
	signerData := xauthsigning.SignerData{
		ChainID:       "123458", // todo wtf is our chain id even lol????
		AccountNumber: 1234,
		Sequence:      100,
	}
	sigV2, err := tx.SignWithPrivKey(context.Background(),
		signing.SignMode_SIGN_MODE_DIRECT, signerData,
		txBuilder, &alicePk, txConfig, uint64(100))
	s.Require().NoError(err)
	sigsV2 = append(sigsV2, sigV2)
	err = txBuilder.SetSignatures(sigsV2...)
	s.Require().NoError(err)

	/*
		createTopicPath := s.RpcUrl // + "/emissions.state.v1.Msg/CreateNewTopic"
		//createTopicPostData := []byte(`{"body":{"messages":[{"@type":"/emissions.state.v1.MsgCreateNewTopic","creator":"alice","metadata":"ETH prediction in 24h","weight_logic":"bafybeih6yjjf2v7qp3wm6hodvjcdlj7galu7dufirvcekzip5gd7bthq","weight_method":"eth-price-weights-calc.wasm","weight_cadence":"3600","inference_logic":"bafybeigpiwl3o73zvvl6dxdqu7zqcub5mhg65iky2xqb4rdhfmikswzqm","inference_method":"allora-inference-function.wasm","inference_cadence":"300","default_arg":"ETH"}],"memo":"","timeout_height":"0","extension_options":[],"non_critical_extension_options":[]},"auth_info":{"signer_infos":[],"fee":{"amount":[],"gas_limit":"200000","payer":"","granter":""},"tip":null},"signatures":[]}`)
		createTopicPostData := []byte(`{}`)
		req, err := http.NewRequest("POST", createTopicPath, bytes.NewBuffer(createTopicPostData))
		s.Require().NoError(err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.HttpClient.Do(req)
		s.Require().NoError(err)
		defer resp.Body.Close()

		fmt.Println("response Status:", resp.Status)
		fmt.Println("response Headers:", resp.Header)
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("response Body:", string(body))
	*/
}
