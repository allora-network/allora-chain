#!/bin/bash
#
# Compares the proposer priorities that CometBFT RPC endpoints report for the
# validator set at one height, and groups the endpoints by what they report.
#
# Why priorities can differ between nodes
#
#   Every node keeps a proposer priority for each validator and uses it to decide
#   whose turn it is to propose a block. A node that state syncs does not get these
#   priorities from the snapshot; the snapshot holds application state only. It
#   takes them from the `/validators` responses of the RPC servers listed under
#   `[statesync] rpc_servers`. The light client checks those validator sets against
#   the block headers, but the validator set hash covers only public keys and voting
#   power, so wrong priorities pass the check and become the node's consensus state.
#   Restoring a copied data directory copies CometBFT's state.db and its priorities
#   the same way.
#
#   A node with wrong priorities disagrees about who the proposer is. It rejects the
#   real proposer's block and may propose out of turn, so rounds time out and blocks
#   slow down. CometBFT never corrects the priorities; the node stays wrong until it
#   syncs again from a healthy source, and every node that state syncs through it
#   inherits the same wrong priorities.
#
# Usage
#
#   scripts/check_proposer_priorities.sh [-H height] RPC_URL [RPC_URL...]
#
#   The height defaults to 20 blocks below the first endpoint's latest height. A
#   load-balanced endpoint can be listed several times to sample several backends;
#   the moniker printed for it comes from a separate /status request and may name a
#   different backend than the one that answered /validators.
#
#   Requires curl and jq. Exits 0 when every endpoint agrees, 1 when they disagree,
#   and 2 on bad arguments or missing tools.
#
# Reading the result
#
#   Each endpoint is assigned to a group; endpoints in the same group report
#   identical priorities. When there is more than one group, the script lists the
#   validators whose priorities differ. The group that matches the nodes of most of
#   the voting power is the healthy one; the other groups are wrong.
#
# Fixing a network where nodes disagree
#
#   1. Run this against every node that can be reached: validators, sentries, peer
#      and archive nodes, and every endpoint operators use as a state sync RPC
#      server.
#   2. Sync every node outside the healthy group again from a healthy source. This
#      includes non-validator nodes. A wrong RPC or peer node left running passes its
#      priorities to the next node that state syncs through it.
#   3. Before a freshly synced validator signs, run this with its RPC and a node from
#      the healthy group, and confirm they agree.
#
# Evidence to collect before fixing
#
#   From a node outside the healthy group, save its `[statesync]` configuration
#   (which RPC servers it synced from), the height it synced at, and the output of
#   `/validators` at heights around that height. This shows whether it inherited the
#   wrong priorities from a provider or whether something else produced them.

set -euo pipefail

usage() {
	echo "usage: $0 [-H height] RPC_URL [RPC_URL...]" >&2
}

height=""
while getopts "H:h" opt; do
	case "$opt" in
	H) height="$OPTARG" ;;
	h)
		usage
		exit 0
		;;
	*)
		usage
		exit 2
		;;
	esac
done
shift $((OPTIND - 1))

if [ $# -lt 1 ]; then
	usage
	exit 2
fi
for tool in curl jq; do
	if ! command -v "$tool" >/dev/null; then
		echo "$tool is required" >&2
		exit 2
	fi
done

rpc_get() {
	curl -fsS --max-time 20 "$1"
}

if [ -z "$height" ]; then
	latest=$(rpc_get "$1/status" | jq -r '.result.sync_info.latest_block_height')
	height=$((latest - 20))
fi
echo "height $height"

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# Writes "address priority" lines for the whole validator set, sorted by address.
fetch_priorities() {
	local rpc=$1 out=$2 page=1 total=-1 count=0 res
	: >"$out.unsorted"
	while [ "$count" != "$total" ]; do
		if [ "$page" -gt 100 ]; then
			return 1
		fi
		res=$(rpc_get "$rpc/validators?height=$height&page=$page&per_page=100") || return 1
		total=$(jq -r '.result.total' <<<"$res")
		jq -r '.result.validators[] | "\(.address) \(.proposer_priority)"' <<<"$res" >>"$out.unsorted"
		count=$(wc -l <"$out.unsorted" | tr -d ' ')
		page=$((page + 1))
	done
	sort "$out.unsorted" >"$out"
}

groups=()
n=0
for rpc in "$@"; do
	n=$((n + 1))
	file="$work/$n"
	moniker=$(rpc_get "$rpc/status" 2>/dev/null | jq -r '.result.node_info.moniker' 2>/dev/null) || moniker="unknown"
	if ! fetch_priorities "$rpc" "$file"; then
		echo "$rpc: request failed"
		continue
	fi
	group=""
	for ((i = 0; i < ${#groups[@]}; i++)); do
		if cmp -s "$file" "${groups[$i]}"; then
			group=$i
			break
		fi
	done
	if [ -z "$group" ]; then
		groups+=("$file")
		group=$((${#groups[@]} - 1))
	fi
	echo "group $((group + 1))  $rpc  moniker=$moniker  validators=$(wc -l <"$file" | tr -d ' ')"
done

if [ ${#groups[@]} -le 1 ]; then
	echo "all endpoints agree"
	exit 0
fi

echo "endpoints disagree: ${#groups[@]} groups"
for ((i = 1; i < ${#groups[@]}; i++)); do
	echo "validators that differ between group 1 and group $((i + 1)) (address, group 1, group $((i + 1))):"
	join -a1 -a2 -e missing -o 0,1.2,2.2 "${groups[0]}" "${groups[$i]}" | awk '$2 != $3'
done
exit 1
