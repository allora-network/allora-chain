###############################################################################
###                                Initialize                               ###
###############################################################################

init-allora:
	@echo "Initializing allora blockchain..."
	./network/init-allorad.sh
	./network/start-allorad.sh
	
init-axelar:
	@echo "Initializing axelar blockchain..."
	./network/init-axelar.sh

init-relayer:
	@echo "Initializing relayer..."
	./network/relayer/rly.sh
	@echo "Creating connection and ICS20 channel..."
	./network/relayer/rly-link.sh

start-relayer:
	@echo "Starting relayer..."
	./network/relayer/rly-start.sh

kill-dev:
	@echo "Killing icad and removing previous data"
	-@rm -rf ./data
	-@killall icad 2>/dev/null
