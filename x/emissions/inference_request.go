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
