# Helpers for local_testnet_l1.sh. They live apart from it so the failure paths can be
# tested without Docker: localnet_lib_test.go puts a fake docker on PATH.

BUILD_MAX_ATTEMPTS="${BUILD_MAX_ATTEMPTS:-4}"
BUILD_RETRY_BASE_SECONDS="${BUILD_RETRY_BASE_SECONDS:-15}"

# is_registry_failure succeeds when a build log shows a transient error talking to
# Docker Hub. Compile errors name neither a registry host nor these statuses.
is_registry_failure() {
    grep -aE '(docker\.io|docker\.com)' "$1" |
        grep -aqE 'toomanyrequests|Too Many Requests|\b429\b|\b50[0234]\b|Gateway Time|Bad Gateway|Service Unavailable|i/o timeout|TLS handshake timeout|connection reset|unexpected EOF'
}

# build_image_with_retry runs docker build with the given arguments, repeating it with
# backoff only when the failure is a Docker Hub error. Any other failure ends the first
# attempt, since a rebuild cannot fix it.
build_image_with_retry() {
    local log status attempt=1
    log=$(mktemp)
    while true; do
        docker build "$@" 2>&1 | tee "$log"
        status=${PIPESTATUS[0]}
        if [ "$status" -eq 0 ]; then
            rm -f "$log"
            return 0
        fi
        if ! is_registry_failure "$log"; then
            rm -f "$log"
            echo "docker build failed (exit $status), not a registry error, not retrying" >&2
            return "$status"
        fi
        if [ "$attempt" -ge "$BUILD_MAX_ATTEMPTS" ]; then
            rm -f "$log"
            echo "docker build failed: Docker Hub still failing after $attempt attempts" >&2
            return "$status"
        fi
        local delay=$((attempt * attempt * BUILD_RETRY_BASE_SECONDS))
        echo "docker build hit a Docker Hub error on attempt $attempt/$BUILD_MAX_ATTEMPTS, retrying in ${delay}s" >&2
        sleep "$delay"
        attempt=$((attempt + 1))
    done
}

# capture_allorad prints the stdout of an allorad command run in the localnet image, for
# use as a value in generated config. Its arguments are a description for error messages,
# the regex the value must match, the allorad home, then the allorad arguments.
#
# It fails, naming the cause, on a non-zero exit, on a value that does not match, and on
# any stderr output: these commands are silent on stderr when the binary is healthy, and
# output there points at a real defect such as a duplicate proto registration. Lines
# starting with "WARNING: " come from the docker CLI and do not count.
#
# It reads DOCKER_IMAGE and LOCALNET_DATADIR.
capture_allorad() {
    local description="$1" value_regex="$2" home="$3"
    shift 3
    local stderr_log value status=0 diagnostics
    stderr_log=$(mktemp)
    value=$(docker run \
        -u "$(id -u):$(id -g)" \
        -v "${LOCALNET_DATADIR}:/data" \
        --entrypoint=allorad \
        -e "HOME=$home" \
        "$DOCKER_IMAGE" \
        "--home=$home" "$@" 2>"$stderr_log") || status=$?
    diagnostics=$(grep -av '^WARNING: ' "$stderr_log" || true)
    rm -f "$stderr_log"

    if [ "$status" -ne 0 ]; then
        echo "Error: allorad failed getting $description (exit $status). Its stderr:" >&2
        echo "$diagnostics" | head -n 20 >&2
        return 1
    fi
    if [ -n "$diagnostics" ]; then
        echo "Error: allorad wrote to stderr while getting $description, so the binary has a startup problem. First lines:" >&2
        echo "$diagnostics" | head -n 20 >&2
        return 1
    fi
    if ! [[ "$value" =~ $value_regex ]]; then
        echo "Error: allorad returned an invalid $description: '$value' does not match $value_regex" >&2
        return 1
    fi
    printf '%s\n' "$value"
}
