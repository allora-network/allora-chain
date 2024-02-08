package module

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"os"
	"strings"

	"cosmossdk.io/math"
	state "github.com/allora-network/allora-chain/x/emissions"
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

func generateWeights(weights map[string]map[string]*math.Uint, inferences []*state.InferenceSetForScoring, functionId string, functionMethod string, topicId uint64) {
	inferencesByTimestamp := []LatestInferences{}

	for _, infSet := range inferences {
		timestamp := fmt.Sprintf("%d", infSet.Timestamp)
		inferences := []InferenceItem{}
		for _, inf := range infSet.Inferences.Inferences {
			inferences = append(inferences, InferenceItem{
				Worker:    inf.Worker,
				Inference: inf.Value.String(),
			})
		}
		inferencesByTimestamp = append(inferencesByTimestamp, LatestInferences{
			Timestamp:  timestamp,
			Inferences: inferences,
		})
	}

	// Format the weights map into a map of strings
	var latestWeights map[string]string = make(map[string]string)
	if len(weights) == 0 {
		for _, inference := range inferences {
			for _, inf := range inference.Inferences.Inferences {
				latestWeights[inf.Worker] = "0"
			}
		}
	} else {
		for _, workerWeights := range weights {
			for worker, weight := range workerWeights {
				latestWeights[worker] = weight.String()
			}
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

	calcWeightsReq := BlocklessRequest{
		FunctionID: functionId,
		Method:     functionMethod,
		Config: Config{
			Stdin: &params,
			Environment: []EnvVar{
				{
					Name:  "TOPIC_ID",
					Value: fmt.Sprintf("%v", topicId),
				},
			},
			NodeCount: 1,
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

func generateInferences(functionId string, functionMethod string, param string, topicId uint64) {

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
					Name:  "UPSHOT_ARG_PARAMS",
					Value: param,
				},
				{
					Name:  "TOPIC_ID",
					Value: fmt.Sprintf("%v", topicId),
				},
			},
			NodeCount: -1, // use all nodes that reported, no minimum / max
			Timeout:   2,  // seconds to time out before rollcall complete
		},
	}

	payload, err := json.Marshal(payloadJson)
	if err != nil {
		fmt.Println("Error marshalling outer JSON:", err)
		return
	}
	payloadStr := string(payload)

	makeApiCall(payloadStr)
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
