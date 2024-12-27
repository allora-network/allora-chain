package fuzz_test

import (
	"fmt"

	cosmossdk_io_math "cosmossdk.io/math"
	testcommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

func addToAdminWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	_ uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.AddToWhitelistAdminRequest{
			Sender:  sender.addr,
			Address: subject.addr,
		},
		&emissionstypes.AddToWhitelistAdminResponse{},
		fmt.Sprintf("adding '%s' to admin whitelist", subject),
		fmt.Sprintf("failed to add '%s' to admin whitelist", subject),
		fmt.Sprintf("added '%s' to admin whitelist", subject),
	) {
		data.addAdminWhitelist(subject)
		return true
	}
	return false
}

func removeFromAdminWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	_ uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.RemoveFromWhitelistAdminRequest{
			Sender:  sender.addr,
			Address: subject.addr,
		},
		&emissionstypes.RemoveFromWhitelistAdminResponse{},
		fmt.Sprintf("removing '%s' from admin whitelist", subject),
		fmt.Sprintf("failed to remove '%s' from admin whitelist", subject),
		fmt.Sprintf("removed '%s' from admin whitelist", subject),
	) {
		data.removeAdminWhitelist(subject)
		return true
	}
	return false
}

func addToGlobalWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	_ uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.AddToGlobalWhitelistRequest{
			Sender:  sender.addr,
			Address: subject.addr,
		},
		&emissionstypes.AddToGlobalWhitelistResponse{},
		fmt.Sprintf("adding '%s' to global whitelist", subject),
		fmt.Sprintf("failed to add '%s' to global whitelist", subject),
		fmt.Sprintf("added '%s' to global whitelist", subject),
	) {
		data.addGlobalWhitelist(subject)
		return true
	}
	return false
}

func removeFromGlobalWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	_ uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.RemoveFromGlobalWhitelistRequest{
			Sender:  sender.addr,
			Address: subject.addr,
		},
		&emissionstypes.RemoveFromGlobalWhitelistResponse{},
		fmt.Sprintf("removing '%s' from global whitelist", subject),
		fmt.Sprintf("failed to remove '%s' from global whitelist", subject),
		fmt.Sprintf("removed '%s' from global whitelist", subject),
	) {
		data.removeGlobalWhitelist(subject)
		return true
	}
	return false
}

func addToTopicCreatorWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	_ uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.AddToTopicCreatorWhitelistRequest{
			Sender:  sender.addr,
			Address: subject.addr,
		},
		&emissionstypes.AddToTopicCreatorWhitelistResponse{},
		fmt.Sprintf("adding '%s' to topic creator whitelist", subject),
		fmt.Sprintf("failed to add '%s' to topic creator whitelist", subject),
		fmt.Sprintf("added '%s' to topic creator whitelist", subject),
	) {
		data.addTopicCreatorWhitelist(subject)
		return true
	}
	return false
}

func removeFromTopicCreatorWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	_ uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.RemoveFromTopicCreatorWhitelistRequest{
			Sender:  sender.addr,
			Address: subject.addr,
		},
		&emissionstypes.RemoveFromTopicCreatorWhitelistResponse{},
		fmt.Sprintf("removing '%s' from topic creator whitelist", subject),
		fmt.Sprintf("failed to remove '%s' from topic creator whitelist", subject),
		fmt.Sprintf("removed '%s' from topic creator whitelist", subject),
	) {
		data.removeTopicCreatorWhitelist(subject)
		return true
	}
	return false
}

func enableTopicWorkerWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	_ Actor,
	_ *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.EnableTopicWorkerWhitelistRequest{
			Sender:  sender.addr,
			TopicId: topicId,
		},
		&emissionstypes.EnableTopicWorkerWhitelistResponse{},
		fmt.Sprintf("enabling topic '%d' worker whitelist", topicId),
		fmt.Sprintf("failed to enable topic '%d' worker whitelist", topicId),
		fmt.Sprintf("enabled topic '%d' worker whitelist", topicId),
	) {
		data.enableTopicWorkersWhitelist(topicId)
		return true
	}
	return false
}

func disableTopicWorkerWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	_ Actor,
	_ *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.DisableTopicWorkerWhitelistRequest{
			Sender:  sender.addr,
			TopicId: topicId,
		},
		&emissionstypes.DisableTopicWorkerWhitelistResponse{},
		fmt.Sprintf("disabling topic '%d' worker whitelist", topicId),
		fmt.Sprintf("failed to disable topic '%d' worker whitelist", topicId),
		fmt.Sprintf("disabled topic '%d' worker whitelist", topicId),
	) {
		data.disableTopicWorkersWhitelist(topicId)
		return true
	}
	return false
}

func enableTopicReputerWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	_ Actor,
	_ *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.EnableTopicReputerWhitelistRequest{
			Sender:  sender.addr,
			TopicId: topicId,
		},
		&emissionstypes.EnableTopicReputerWhitelistResponse{},
		fmt.Sprintf("enabling topic '%d' reputer whitelist", topicId),
		fmt.Sprintf("failed to enable topic '%d' reputer whitelist", topicId),
		fmt.Sprintf("enabled topic '%d' reputer whitelist", topicId),
	) {
		data.enableTopicReputersWhitelist(topicId)
		return true
	}
	return false
}

func disableTopicReputerWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	_ Actor,
	_ *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.DisableTopicReputerWhitelistRequest{
			Sender:  sender.addr,
			TopicId: topicId,
		},
		&emissionstypes.DisableTopicReputerWhitelistResponse{},
		fmt.Sprintf("disabling topic '%d' reputer whitelist", topicId),
		fmt.Sprintf("failed to disable topic '%d' reputer whitelist", topicId),
		fmt.Sprintf("disabled topic '%d' reputer whitelist", topicId),
	) {
		data.disableTopicReputersWhitelist(topicId)
		return true
	}
	return false
}

func addToTopicWorkerWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.AddToTopicWorkerWhitelistRequest{
			Sender:  sender.addr,
			Address: subject.addr,
			TopicId: topicId,
		},
		&emissionstypes.AddToTopicWorkerWhitelistResponse{},
		fmt.Sprintf("adding '%s' to topic '%d' worker whitelist", subject, topicId),
		fmt.Sprintf("failed to add '%s' to topic '%d' worker whitelist", subject, topicId),
		fmt.Sprintf("added '%s' to topic '%d' worker whitelist", subject, topicId),
	) {
		data.addTopicWorkerWhitelist(topicId, subject)
		return true
	}
	return false
}

func removeFromTopicWorkerWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.RemoveFromTopicWorkerWhitelistRequest{
			Sender:  sender.addr,
			Address: subject.addr,
			TopicId: topicId,
		},
		&emissionstypes.RemoveFromTopicWorkerWhitelistResponse{},
		fmt.Sprintf("removing '%s' from topic '%d' worker whitelist", subject, topicId),
		fmt.Sprintf("failed to remove '%s' from topic '%d' worker whitelist", subject, topicId),
		fmt.Sprintf("removed '%s' from topic '%d' worker whitelist", subject, topicId),
	) {
		data.removeTopicWorkerWhitelist(topicId, subject)
		return true
	}
	return false
}

func addToTopicReputerWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.AddToTopicReputerWhitelistRequest{
			Sender:  sender.addr,
			Address: subject.addr,
			TopicId: topicId,
		},
		&emissionstypes.AddToTopicReputerWhitelistResponse{},
		fmt.Sprintf("adding '%s' to topic '%d' reputer whitelist", subject, topicId),
		fmt.Sprintf("failed to add '%s' to topic '%d' reputer whitelist", subject, topicId),
		fmt.Sprintf("added '%s' to topic '%d' reputer whitelist", subject, topicId),
	) {
		data.addTopicReputerWhitelist(topicId, subject)
		return true
	}
	return false
}

func removeFromTopicReputerWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.RemoveFromTopicReputerWhitelistRequest{
			Sender:  sender.addr,
			Address: subject.addr,
			TopicId: topicId,
		},
		&emissionstypes.RemoveFromTopicReputerWhitelistResponse{},
		fmt.Sprintf("removing '%s' from topic '%d' reputer whitelist", subject, topicId),
		fmt.Sprintf("failed to remove '%s' from topic '%d' reputer whitelist", subject, topicId),
		fmt.Sprintf("removed '%s' from topic '%d' reputer whitelist", subject, topicId),
	) {
		data.removeTopicReputerWhitelist(topicId, subject)
		return true
	}
	return false
}

func addToGlobalWorkerWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	_ uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.AddToGlobalWorkerWhitelistRequest{
			Sender:  sender.addr,
			Address: subject.addr,
		},
		&emissionstypes.AddToGlobalWorkerWhitelistResponse{},
		fmt.Sprintf("adding '%s' to global worker whitelist", subject),
		fmt.Sprintf("failed to add '%s' to global worker whitelist", subject),
		fmt.Sprintf("added '%s' to global worker whitelist", subject),
	) {
		data.addGlobalWorkerWhitelist(subject)
		return true
	}
	return false
}

func removeFromGlobalWorkerWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	_ uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.RemoveFromGlobalWorkerWhitelistRequest{
			Sender:  sender.addr,
			Address: subject.addr,
		},
		&emissionstypes.RemoveFromGlobalWorkerWhitelistResponse{},
		fmt.Sprintf("removing '%s' from global worker whitelist", subject),
		fmt.Sprintf("failed to remove '%s' from global worker whitelist", subject),
		fmt.Sprintf("removed '%s' from global worker whitelist", subject),
	) {
		data.removeGlobalWorkerWhitelist(subject)
		return true
	}
	return false
}

func addToGlobalReputerWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	_ uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.AddToGlobalReputerWhitelistRequest{
			Sender:  sender.addr,
			Address: subject.addr,
		},
		&emissionstypes.AddToGlobalReputerWhitelistResponse{},
		fmt.Sprintf("adding '%s' to global reputer whitelist", subject),
		fmt.Sprintf("failed to add '%s' to global reputer whitelist", subject),
		fmt.Sprintf("added '%s' to global reputer whitelist", subject),
	) {
		data.addGlobalReputerWhitelist(subject)
		return true
	}
	return false
}

func removeFromGlobalReputerWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	_ uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.RemoveFromGlobalReputerWhitelistRequest{
			Sender:  sender.addr,
			Address: subject.addr,
		},
		&emissionstypes.RemoveFromGlobalReputerWhitelistResponse{},
		fmt.Sprintf("removing '%s' from global reputer whitelist", subject),
		fmt.Sprintf("failed to remove '%s' from global reputer whitelist", subject),
		fmt.Sprintf("removed '%s' from global reputer whitelist", subject),
	) {
		data.removeGlobalReputerWhitelist(subject)
		return true
	}
	return false
}

func addToGlobalAdminWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	_ uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.AddToGlobalAdminWhitelistRequest{
			Sender:  sender.addr,
			Address: subject.addr,
		},
		&emissionstypes.AddToGlobalAdminWhitelistResponse{},
		fmt.Sprintf("adding '%s' to global admin whitelist", subject),
		fmt.Sprintf("failed to add '%s' to global admin whitelist", subject),
		fmt.Sprintf("added '%s' to global admin whitelist", subject),
	) {
		data.addGlobalAdminWhitelist(subject)
		return true
	}
	return false
}

func removeFromGlobalAdminWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	_ uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.RemoveFromGlobalAdminWhitelistRequest{
			Sender:  sender.addr,
			Address: subject.addr,
		},
		&emissionstypes.RemoveFromGlobalAdminWhitelistResponse{},
		fmt.Sprintf("removing '%s' from global admin whitelist", subject),
		fmt.Sprintf("failed to remove '%s' from global admin whitelist", subject),
		fmt.Sprintf("removed '%s' from global admin whitelist", subject),
	) {
		data.removeGlobalAdminWhitelist(subject)
		return true
	}
	return false
}

func bulkAddToGlobalWorkerWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	_ uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.BulkAddToGlobalWorkerWhitelistRequest{
			Sender:    sender.addr,
			Addresses: []string{subject.addr},
		},
		&emissionstypes.BulkAddToGlobalWorkerWhitelistResponse{},
		fmt.Sprintf("bulk adding '%s' to global worker whitelist", subject),
		fmt.Sprintf("failed to bulk add '%s' to global worker whitelist", subject),
		fmt.Sprintf("bulk added '%s' to global worker whitelist", subject),
	) {
		data.addGlobalWorkerWhitelist(subject)
		return true
	}
	return false
}

func bulkRemoveFromGlobalWorkerWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	_ uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.BulkRemoveFromGlobalWorkerWhitelistRequest{
			Sender:    sender.addr,
			Addresses: []string{subject.addr},
		},
		&emissionstypes.BulkRemoveFromGlobalWorkerWhitelistResponse{},
		fmt.Sprintf("bulk removing '%s' from global worker whitelist", subject),
		fmt.Sprintf("failed to bulk remove '%s' from global worker whitelist", subject),
		fmt.Sprintf("bulk removed '%s' from global worker whitelist", subject),
	) {
		data.removeGlobalWorkerWhitelist(subject)
		return true
	}
	return false
}

func bulkAddToGlobalReputerWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	_ uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.BulkAddToGlobalReputerWhitelistRequest{
			Sender:    sender.addr,
			Addresses: []string{subject.addr},
		},
		&emissionstypes.BulkAddToGlobalReputerWhitelistResponse{},
		fmt.Sprintf("bulk adding '%s' to global reputer whitelist", subject),
		fmt.Sprintf("failed to bulk add '%s' to global reputer whitelist", subject),
		fmt.Sprintf("bulk added '%s' to global reputer whitelist", subject),
	) {
		data.addGlobalReputerWhitelist(subject)
		return true
	}
	return false
}

func bulkRemoveFromGlobalReputerWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	_ uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.BulkRemoveFromGlobalReputerWhitelistRequest{
			Sender:    sender.addr,
			Addresses: []string{subject.addr},
		},
		&emissionstypes.BulkRemoveFromGlobalReputerWhitelistResponse{},
		fmt.Sprintf("bulk removing '%s' from global reputer whitelist", subject),
		fmt.Sprintf("failed to bulk remove '%s' from global reputer whitelist", subject),
		fmt.Sprintf("bulk removed '%s' from global reputer whitelist", subject),
	) {
		data.removeGlobalReputerWhitelist(subject)
		return true
	}
	return false
}

func bulkAddToTopicWorkerWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.BulkAddToTopicWorkerWhitelistRequest{
			Sender:    sender.addr,
			Addresses: []string{subject.addr},
			TopicId:   topicId,
		},
		&emissionstypes.BulkAddToTopicWorkerWhitelistResponse{},
		fmt.Sprintf("bulk adding '%s' from topic '%d' worker whitelist", subject, topicId),
		fmt.Sprintf("failed to bulk add '%s' from topic '%d' worker whitelist", subject, topicId),
		fmt.Sprintf("bulk added '%s' from topic '%d' worker whitelist", subject, topicId),
	) {
		data.addTopicWorkerWhitelist(topicId, subject)
		return true
	}
	return false
}

func bulkRemoveFromTopicWorkerWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.BulkRemoveFromTopicWorkerWhitelistRequest{
			Sender:    sender.addr,
			Addresses: []string{subject.addr},
			TopicId:   topicId,
		},
		&emissionstypes.BulkRemoveFromTopicWorkerWhitelistResponse{},
		fmt.Sprintf("bulk removing '%s' from topic '%d' worker whitelist", subject, topicId),
		fmt.Sprintf("failed to bulk remove '%s' from topic '%d' worker whitelist", subject, topicId),
		fmt.Sprintf("bulk removed '%s' from topic '%d' worker whitelist", subject, topicId),
	) {
		data.removeTopicWorkerWhitelist(topicId, subject)
		return true
	}
	return false
}

func bulkAddToTopicReputerWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.BulkAddToTopicReputerWhitelistRequest{
			Sender:    sender.addr,
			Addresses: []string{subject.addr},
			TopicId:   topicId,
		},
		&emissionstypes.BulkAddToTopicReputerWhitelistResponse{},
		fmt.Sprintf("bulk adding '%s' to topic '%d' reputer whitelist", subject, topicId),
		fmt.Sprintf("failed to bulk add '%s' to topic '%d' reputer whitelist", subject, topicId),
		fmt.Sprintf("bulk added '%s' to topic '%d' reputer whitelist", subject, topicId),
	) {
		data.addTopicReputerWhitelist(topicId, subject)
		return true
	}
	return false
}

func bulkRemoveFromTopicReputerWhitelist(
	m *testcommon.TestConfig,
	sender Actor,
	subject Actor,
	_ *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) bool {
	if broadcastTxAndWait(m, iteration, data.failOnErr, sender,
		&emissionstypes.BulkRemoveFromTopicReputerWhitelistRequest{
			Sender:    sender.addr,
			Addresses: []string{subject.addr},
			TopicId:   topicId,
		},
		&emissionstypes.BulkRemoveFromTopicReputerWhitelistResponse{},
		fmt.Sprintf("bulk removing '%s' from topic '%d' reputer whitelist", subject, topicId),
		fmt.Sprintf("failed to bulk remove '%s' from topic '%d' reputer whitelist", subject, topicId),
		fmt.Sprintf("bulk removed '%s' from topic '%d' reputer whitelist", subject, topicId),
	) {
		data.removeTopicReputerWhitelist(topicId, subject)
		return true
	}
	return false
}
