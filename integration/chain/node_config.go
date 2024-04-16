package chain_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
)

type NodeConfig struct {
	t    *testing.T
	host string
	port string
	cdc  codec.Codec
}

func NewNodeConfig(t *testing.T, host, port string, cdc codec.Codec) NodeConfig {
	return NodeConfig{
		t:    t,
		host: host,
		port: port,
		cdc:  cdc,
	}
}

func (n *NodeConfig) GetHostPort() string {
	return n.host + ":" + n.port
}
