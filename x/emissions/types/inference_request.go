package types

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

// Convenience helper function that wraps a InferenceRequestInbound into an InferenceRequest
// one is passed as the msg in a msg handler, while the other is stored in the state machine
func CreateNewInferenceRequestFromListItem(sender string, item *InferenceRequestInbound) *InferenceRequest {
	newInferenceRequest := &InferenceRequest{
		Sender:               sender,
		Nonce:                item.Nonce,
		TopicId:              item.TopicId,
		Cadence:              item.Cadence,
		MaxPricePerInference: item.MaxPricePerInference,
		BidAmount:            item.BidAmount,
		BlockValidUntil:      item.BlockValidUntil,
		BlockLastChecked:     0,
		ExtraData:            item.ExtraData,
	}
	id, err := newInferenceRequest.GetRequestId()
	if err != nil {
		panic(err)
	}
	newInferenceRequest.Id = id
	return newInferenceRequest
}

func IsValidRequestId(requestId string) bool {
	if len(requestId) != 66 {
		return false
	}
	if requestId[:2] != "0x" {
		return false
	}
	for _, c := range requestId[2:] {
		// we only allow lowercase hex
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}
