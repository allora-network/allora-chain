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
	h := sha256.New()
	h.Write(inferenceRequestBytes)
	reqId := h.Sum(nil)
	return fmt.Sprintf("0x%x", reqId), nil
}
