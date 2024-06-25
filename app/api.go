package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"os"
	"strings"

	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type BlocklessRequest struct {
	FunctionID string `json:"function_id"`
	Method     string `json:"method"`
	TopicID    string `json:"topic,omitempty"`
	Config     Config `json:"config"`
}

type Config struct {
	Environment        []EnvVar `json:"env_vars,omitempty"`
	Stdin              *string  `json:"stdin,omitempty"`
	NodeCount          int      `json:"number_of_nodes,omitempty"`
	Timeout            int      `json:"timeout,omitempty"`
	ConsensusAlgorithm string   `json:"consensus_algorithm,omitempty"`
}

type EnvVar struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type LatestInferences struct {
	Timestamp  string          `json:"timestamp"`
	Inferences []InferenceItem `json:"inferences"`
}

type InferenceItem struct {
	Worker    string `json:"worker"`
	Inference string `json:"inference"`
}

type LossesPayload struct {
	Inferences []emissionstypes.ValueBundle `json:"inferences"`
}

func generateLossesRequest(
	ctx sdk.Context,
	inferences *emissionstypes.ValueBundle,
	functionId string,
	functionMethod string,
	topicId uint64,
	topicAllowsNegative bool,
	blockHeight emissionstypes.Nonce,
	blockHeightEval emissionstypes.Nonce,
	blocktime uint64) {

	inferencesPayloadJSON, err := json.Marshal(inferences)
	if err != nil {
		Logger(ctx).Warn(fmt.Sprintf("Error marshalling JSON: %s", err.Error()))
		return
	}

	stdin := string(inferencesPayloadJSON)
	topicIdStr := strconv.FormatUint(topicId, 10) + "/reputer"
	calcWeightsReq := BlocklessRequest{
		FunctionID: functionId,
		Method:     functionMethod,
		TopicID:    topicIdStr,
		Config: Config{
			Stdin: &stdin,
			Environment: []EnvVar{
				{
					Name:  "BLS_REQUEST_PATH",
					Value: "/api",
				},
				{
					Name:  "ALLORA_ARG_PARAMS",
					Value: strconv.FormatUint(blocktime, 10),
				},
				{
					Name:  "ALLORA_BLOCK_HEIGHT_CURRENT",
					Value: strconv.FormatInt(blockHeight.BlockHeight, 10),
				},
				{
					Name:  "ALLORA_BLOCK_HEIGHT_EVAL",
					Value: strconv.FormatInt(blockHeightEval.BlockHeight, 10),
				},
				{
					Name:  "LOSS_FUNCTION_ALLOWS_NEGATIVE",
					Value: strconv.FormatBool(topicAllowsNegative),
				},
			},
			NodeCount:          -1,     // use all nodes that reported, no minimum / max
			Timeout:            2,      // seconds to time out before rollcall complete
			ConsensusAlgorithm: "pbft", // forces worker leader write to chain through pbft
		},
	}

	payload, err := json.Marshal(calcWeightsReq)
	if err != nil {
		Logger(ctx).Warn(fmt.Sprintf("Error marshalling outer JSON: %s", err.Error()))
		return
	}
	payloadStr := string(payload)
	Logger(ctx).Debug(fmt.Sprintf("Making API call - losses, with payload: %s", payloadStr))
	err = makeApiCall(payloadStr)
	if err != nil {
		Logger(ctx).Warn("Error making API call - losses: " + err.Error())
	}
}

func generateInferencesRequest(
	ctx sdk.Context,
	functionId string,
	functionMethod string,
	param string,
	topicId uint64,
	topicAllowsNegative bool,
	nonce emissionstypes.Nonce) {

	payloadJson := BlocklessRequest{
		FunctionID: functionId,
		Method:     functionMethod,
		TopicID:    strconv.FormatUint(topicId, 10),
		Config: Config{
			Environment: []EnvVar{
				{
					Name:  "BLS_REQUEST_PATH",
					Value: "/api",
				},
				{
					Name:  "ALLORA_ARG_PARAMS",
					Value: param,
				},
				{
					Name:  "ALLORA_BLOCK_HEIGHT_CURRENT",
					Value: strconv.FormatInt(nonce.BlockHeight, 10),
				},
				{
					Name:  "LOSS_FUNCTION_ALLOWS_NEGATIVE",
					Value: strconv.FormatBool(topicAllowsNegative),
				},
			},
			NodeCount:          -1,     // use all nodes that reported, no minimum / max
			Timeout:            2,      // seconds to time out before rollcall complete
			ConsensusAlgorithm: "pbft", // forces worker leader write to chain through pbft
		},
	}
	payload, err := json.Marshal(payloadJson)
	if err != nil {
		Logger(ctx).Warn(fmt.Sprintf("Error marshalling outer JSON: %s", err.Error()))
	}
	payloadStr := string(payload)

	Logger(ctx).Debug(fmt.Sprintf("Making API call - inferences, with payload: %s", payloadStr))
	err = makeApiCall(payloadStr)
	if err != nil {
		Logger(ctx).Warn(fmt.Sprintf("Error making API call: %s", err.Error()))
	}
}

func makeApiCall(payload string) error {
	url := os.Getenv("BLOCKLESS_API_URL")
	method := "POST"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, strings.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Add("Accept", "application/json, text/plain, */*")
	req.Header.Add("Content-Type", "application/json;charset=UTF-8")

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return nil
}
