#!/bin/bash
# Reports each localnet validator's height, then saves its logs under the directory
# given as $1. It always exits 0, so diagnostics never change a job's result.
set +e

log_dir="${1:?usage: collect_validator_diagnostics.sh LOG_DIR}"

# Prints "unreachable" when the node gives no JSON and "absent" when the
# field is missing, which is how an RPC error response looks.
rpc_field() {
  local value
  value=$(curl -fsS --connect-timeout 2 --max-time 5 "http://localhost:$1/$2" | jq -r "$3 // \"absent\"")
  echo "${value:-unreachable}"
}

for i in 0 1 2; do
  port=$((26657 + i))
  echo "validator$i port=$port" \
    "status.latest_block_height=$(rpc_field "$port" status .result.sync_info.latest_block_height)" \
    "abci_info.last_block_height=$(rpc_field "$port" abci_info .result.response.last_block_height)"
done
docker ps -a

mkdir -p "$log_dir"
for node in validator0 validator1 validator2; do
  echo "=== $node ==="
  if ! docker inspect "$node" > /dev/null; then
    echo "no logs captured"
    continue
  fi
  log_file="$log_dir/$node.log"
  docker logs -t "$node" > "$log_file" 2>&1 || echo "docker logs failed, $log_file may be partial"
  # -a keeps grep printing matched lines when a log contains binary bytes.
  grep -aE 'Invariants broken|panic|CONSENSUS FAILURE' "$log_file"
  case $? in
    0) ;;
    1) echo "no lines match Invariants broken, panic, or CONSENSUS FAILURE" ;;
    *) echo "could not scan $log_file" ;;
  esac
done
exit 0
