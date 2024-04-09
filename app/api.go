package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"os"
	"strings"

	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

type BlocklessRequest struct {
	FunctionID string `json:"function_id"`
	Method     string `json:"method"`
	TopicID    string `json:"topic,omitempty"`
	Config     Config `json:"config"`
}

type Config struct {
	Environment []EnvVar `json:"env_vars,omitempty"`
	Stdin       *string  `json:"stdin,omitempty"`
	NodeCount   int      `json:"number_of_nodes,omitempty"`
	Timeout     int      `json:"timeout,omitempty"`
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

type WeightInferencePayload struct {
	Inferences    []LatestInferences `json:"inferences"`
	LatestWeights map[string]string  `json:"latest_weights"`
}

func generateLosses(
	inferences []*emissionstypes.InferenceSetForScoring,
	functionId string,
	functionMethod string,
	topicId uint64,
	nonce emissionstypes.Nonce,
	blocktime uint64) {

	//
	// TODO Change this to receive and use a ValueBundle instead, not just plain inferences
	inferencesByTimestamp := []LatestInferences{}
	for _, infSet := range inferences {
		timestamp := fmt.Sprintf("%d", infSet.BlockHeight)
		inferences := []InferenceItem{}
		for _, inf := range infSet.Inferences.Inferences {
			inferences = append(inferences, InferenceItem{
				Worker:    inf.Worker,
				Inference: strconv.FormatFloat(inf.Value, 'f', -1, 64),
			})
		}
		inferencesByTimestamp = append(inferencesByTimestamp, LatestInferences{
			Timestamp:  timestamp,
			Inferences: inferences,
		})
	}

	// Combine everything into the final JSON object
	payloadObj := WeightInferencePayload{
		Inferences: inferencesByTimestamp,
	}

	payloadJSON, err := json.Marshal(payloadObj)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	params := string(payloadJSON)

	calcWeightsReq := BlocklessRequest{
		FunctionID: functionId,
		Method:     functionMethod,
		TopicID:    strconv.FormatUint(topicId, 10),
		Config: Config{
			Stdin: &params,
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
					Name:  "ALLORA_NONCE",
					Value: strconv.FormatInt(nonce.GetNonce(), 10),
				},
			},
			NodeCount: -1, // use all nodes that reported, no minimum / max
			Timeout:   2,  // seconds to time out before rollcall complete
		},
	}

	payload, err := json.Marshal(calcWeightsReq)
	if err != nil {
		fmt.Println("Error marshalling outer JSON:", err)
		return
	}
	payloadStr := string(payload)

	makeApiCall(payloadStr)
}

func generateInferences(
	functionId string,
	functionMethod string,
	param string,
	topicId uint64,
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
					Name:  "ALLORA_NONCE",
					Value: strconv.FormatInt(nonce.GetNonce(), 10),
				},
			},
			NodeCount: -1, // use all nodes that reported, no minimum / max
			Timeout:   2,  // seconds to time out before rollcall complete
		},
	}

	payload, err := json.Marshal(payloadJson)
	if err != nil {
		fmt.Println("Error marshalling outer JSON:", err)
	}
	payloadStr := string(payload)

	makeApiCall(payloadStr)
}

func makeApiCall(payload string) {
	fmt.Println("Making Api Call, Payload: ", payload)
	url := os.Getenv("BLOCKLESS_API_URL")
	method := "POST"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, strings.NewReader(payload))
	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Accept", "application/json, text/plain, */*")
	req.Header.Add("Content-Type", "application/json;charset=UTF-8")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()
}
