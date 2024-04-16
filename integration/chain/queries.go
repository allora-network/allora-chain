package chain_test

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

func (n *NodeConfig) QueryGRPCGateway(path string, parameters ...string) ([]byte, error) {
	if len(parameters)%2 != 0 {
		return nil, fmt.Errorf("invalid number of parameters, must follow the format of key + value")
	}

	// add the URL for the given validator ID, and prepend to to path.
	hostPort := n.GetHostPort()
	endpoint := fmt.Sprintf("http://%s", hostPort)
	fullQueryPath := fmt.Sprintf("%s/%s", endpoint, path)

	var resp *http.Response
	require.Eventually(n.t, func() bool {
		req, err := http.NewRequest("GET", fullQueryPath, nil)
		if err != nil {
			return false
		}

		if len(parameters) > 0 {
			q := req.URL.Query()
			for i := 0; i < len(parameters); i += 2 {
				q.Add(parameters[i], parameters[i+1])
			}
			req.URL.RawQuery = q.Encode()
		}

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			n.t.Logf("error while executing HTTP request: %s", err.Error())
			return false
		}

		return resp.StatusCode != http.StatusServiceUnavailable
	}, time.Minute, 10*time.Millisecond, "failed to execute HTTP request")

	defer resp.Body.Close()

	bz, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bz))
	}
	return bz, nil
}

// QueryParams Gets the emissions module params
func (n *NodeConfig) QueryParams() (types.Params, error) {
	path := "/emissions/v1/params"

	bz, err := n.QueryGRPCGateway(path)
	if err != nil {
		return types.Params{}, err
	}

	var response types.QueryParamsResponse
	err = n.cdc.UnmarshalJSON(bz, &response)
	require.NoError(n.t, err)
	return response.Params, nil
}
