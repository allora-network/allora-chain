package testing

import (
	"encoding/json"

	"github.com/allora-network/allora-chain/x/ibc/gmp"
)

func (s *IBCTestSuite) TestGMPMessageFrom_Success() {
	s.IBCTransferProviderToAllora(s.providerAddr, s.alloraAddr, nativeDenom, ibcTransferAmount, "")

	generalMsg := gmp.Message{
		SourceChain:   "axelar",
		SourceAddress: "allora",
		Payload:       []byte("Hello Allora, I am Axelar"),
		Type:          gmp.TypeGeneralMessage,
	}
	generalMsgJson, _ := json.Marshal(generalMsg)
	s.IBCTransferProviderToAllora(
		s.providerAddr,
		s.alloraAddr,
		nativeDenom,
		ibcTransferAmount,
		string(generalMsgJson),
	)

	generalMsgWithToken := gmp.Message{
		SourceChain:   "axelar",
		SourceAddress: "allora",
		Payload:       []byte("Hello Allora, I am Axelar"),
		Type:          gmp.TypeGeneralMessageWithToken,
	}
	generalMsgWithTokenJson, _ := json.Marshal(generalMsgWithToken)
	s.IBCTransferProviderToAllora(
		s.providerAddr,
		s.alloraAddr,
		nativeDenom,
		ibcTransferAmount,
		string(generalMsgWithTokenJson),
	)
}

func (s *IBCTestSuite) TestGMPMessageTo_Success() {
	generalMsg := gmp.Message{
		SourceChain:   "allora",
		SourceAddress: "axelar",
		Payload:       []byte("Hello Axelar, I am Allora"),
		Type:          gmp.TypeGeneralMessage,
	}
	generalMsgJson, _ := json.Marshal(generalMsg)
	s.IBCTransferAlloraToProvider(
		s.alloraAddr,
		s.providerAddr,
		nativeDenom,
		ibcTransferAmount,
		string(generalMsgJson),
	)
}
