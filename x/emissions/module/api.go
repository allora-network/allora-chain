package module

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"cosmossdk.io/math"

	state "github.com/upshot-tech/protocol-state-machine-module"
)

type InferenceItem struct {
	Worker    string `json:"worker"`
	Inference string `json:"inference"`
}

type WeightInferencePayload struct {
	Inferences    map[string][]InferenceItem `json:"inferences"`
	LatestWeights map[string]string         `json:"latest_weights"`
}

func generateWeights(weights map[string]map[string]*math.Uint, inferences []*state.InferenceSetForScoring, functionId string, functionMethod string, topicId uint64) {
	inferencesByTimestamp := make(map[string][]InferenceItem)
	for _, infSet := range inferences {
		timestamp := fmt.Sprintf("%d", infSet.Timestamp)
		for _, inf := range infSet.Inferences.Inferences {
			inferencesByTimestamp[timestamp] = append(inferencesByTimestamp[timestamp], InferenceItem{
				Worker:    inf.Worker,
				Inference: inf.Value.String(),
			})
		}
	}

	latestWeights := make(map[string]string)
	for _, workerWeights := range weights {
		for worker, weight := range workerWeights {
            latestWeights[worker] = weight.String()
		}
	}

	// Combine everything into the final JSON object
	payloadObj := WeightInferencePayload{
		Inferences:    inferencesByTimestamp,
		LatestWeights: latestWeights,
	}

	payloadJSON, err := json.Marshal(payloadObj)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	params := string(payloadJSON)
	payload := fmt.Sprintf(`{
		"function_id": "%s",
		"method": "%s",
		"config": {
			"stdin": %s,
			"env_vars": [
				{
					"name": "TOPIC_ID",
					"value": %v
				}
			],
			"number_of_nodes": 1
		}
	}`, functionId, functionMethod, params, topicId)

	makeApiCall(payload)
}

func generateInferences(functionId string, functionMethod string, param string, topicId uint64) {
	payload := fmt.Sprintf(`{
		"function_id": "%s",
		"method": "%s",
		"config": {
			"env_vars": [
				{
					"name": "BLS_REQUEST_PATH",
					"value": "/api"
				},
				{
					"name": "UPSHOT_ARG_PARAMS",
					"value": %s
				},
				{
					"name": "TOPIC_ID",
					"value": %v
				}
			],
			"number_of_nodes": 1
		}
	}`, functionId, functionMethod, param, topicId)

	makeApiCall(payload)
}

func makeApiCall(payload string) {
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
