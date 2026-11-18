#!/bin/sh
# SPDX-License-Identifier: AGPL-3.0

# Retries a command with backoff. Output is shown only when the command fails.
max_attempts=4
log=$(mktemp) || exit 1
trap 'rm -f "$log"' EXIT
attempt=1
until "$@" >"$log" 2>&1; do
    status=$?
    tail -n 20 "$log" >&2
    if [ "$attempt" -ge "$max_attempts" ]; then
        echo "retry-fetch: giving up after $attempt attempts (exit $status): $*" >&2
        exit "$status"
    fi
    delay=$((attempt * attempt * 5))
    echo "retry-fetch: attempt $attempt/$max_attempts failed (exit $status), retrying in ${delay}s: $*" >&2
    sleep "$delay"
    attempt=$((attempt + 1))
done
