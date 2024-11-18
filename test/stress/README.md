# Allora Chain Stress Testing

This package implements stress testing for the Allora blockchain, specifically focusing on testing the emissions module under load with multiple topics, workers, and reputers.

## Overview

The stress test simulates real-world usage by:
1. Creating multiple topics
2. Registering workers and reputers for each topic
3. Running continuous loops where workers submit inferences and reputers submit reputation scores

## Key Components

### Configuration
The test can be configured through environment variables:
- `SEED`: Random seed for deterministic testing (default: 1)
- `RPC_MODE`: Connection mode to RPC endpoints (single/round-robin/random)
- `RPC_URLS`: List of RPC endpoints
- `EPOCH_LENGTH`: Number of blocks per epoch (default: 12)
- `NUM_TOPICS`: Number of topics to create (default: 10)
- `WORKERS_PER_TOPIC`: Number of workers per topic (default: 5)
- `REPUTERS_PER_TOPIC`: Number of reputers per topic (default: 4)
- `CREATE_TOPICS_SAME_BLOCK`: Whether to create topics in same block (default: false)
- `STRESS_TIMEOUT_MINUTES`: Test duration in minutes (default: 30)

### Main Components
1. **Actor Management** (`actors_setup_test.go`)
   - Creates and funds actors (workers/reputers)
   - Handles actor registration for topics

2. **Topic Management** (`topics_setup_test.go`) 
   - Creates topics with specified parameters
   - Handles topic funding

3. **Simulation Loop** (`actors_loop_test.go`)
   - Runs continuous loops for workers submitting inferences
   - Runs continuous loops for reputers submitting reputation scores

4. **Data Management** (`simulation_data_test.go`)
   - Tracks simulation state
   - Manages concurrent access to shared data

## Usage

Launch a chain: 
```
bash ./test/local_testnet_l1.sh
```

Run the stress test with default values:
```bash
go test -v ./test/new_stress
```

