#!/bin/sh
# Retries a fetch-only command with backoff, showing its output only on failure.
max_attempts=4
log=$(mktemp)
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
