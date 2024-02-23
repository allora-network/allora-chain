package emissions

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (m *InferenceRequest) GetRequestBytes() (ret []byte, err error) {
	senderAddr, err := sdk.AccAddressFromBech32(m.Sender)
	if err != nil {
		return []byte{}, err
	}
	senderBytes := senderAddr.Bytes()

	nonceBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(nonceBytes, m.Nonce)
	ret = append(senderBytes, nonceBytes...)

	topicIdBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(topicIdBytes, m.TopicId)
	ret = append(ret, topicIdBytes...)
	return ret, nil
}

// a request id is a hash of (sender, nonce, topicId)
func (m *InferenceRequest) GetRequestId() (string, error) {
	inferenceRequestBytes, err := m.GetRequestBytes()
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
		LastChecked:          0,
		ExtraData:            item.ExtraData,
	}
	return newInferenceRequest
}
