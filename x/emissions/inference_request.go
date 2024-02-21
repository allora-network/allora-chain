package emissions

import (
	"crypto/sha256"
	"fmt"
)

func (m *InferenceRequest) GetRequestId() (string, error) {
	inferenceRequestBytes, err := m.Marshal()
	if err != nil {
		return "", err
	}
	reqId := sha256.Sum256(inferenceRequestBytes)
	return fmt.Sprintf("0x%x", reqId), nil
}

// Convenience helper function that wraps a RequestInferenceListItem into an InferenceRequest
// one is passed as the msg in a msg handler, while the other is stored in the state machine
func CreateNewInferenceRequestFromListItem(sender string, item *RequestInferenceListItem) *InferenceRequest {
	newInferenceRequest := &InferenceRequest{
		Sender:               sender,
		Nonce:                item.Nonce,
		TopicId:              item.TopicId,
		Cadence:              item.Cadence,
		MaxPricePerInference: item.MaxPricePerInference,
		BidAmount:            item.BidAmount,
		TimestampValidUntil:  item.TimestampValidUntil,
		ExtraData:            item.ExtraData,
	}
	return newInferenceRequest
}
