package testcommon

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeDockerResult is what the fake docker does on one invocation.
type fakeDockerResult struct {
	stdout, stderr string
	exitCode       int
}

// runLocalnetLib runs snippet with localnet_lib.sh sourced, under the same shell options
// as local_testnet_l1.sh, with a docker on PATH that replays results in order and repeats
// the last one. It returns the combined output, the exit code and the docker call count.
func runLocalnetLib(t *testing.T, snippet string, results ...fakeDockerResult) (string, int, int) {
	t.Helper()
	dir := t.TempDir()
	for i, r := range results {
		for suffix, content := range map[string]string{"out": r.stdout, "err": r.stderr, "code": strconv.Itoa(r.exitCode)} {
			path := filepath.Join(dir, strconv.Itoa(i+1)+"."+suffix)
			require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
		}
	}
	calls := filepath.Join(dir, "calls")
	fakeDocker := `#!/bin/bash
echo call >> "$FAKE_DIR/calls"
n=$(wc -l < "$FAKE_DIR/calls" | tr -d ' ')
if [ "$n" -gt "$FAKE_RESULTS" ]; then n=$FAKE_RESULTS; fi
cat "$FAKE_DIR/$n.out"
cat "$FAKE_DIR/$n.err" >&2
exit "$(cat "$FAKE_DIR/$n.code")"
`
	//nolint:gosec // G306: the fake docker must be executable
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docker"), []byte(fakeDocker), 0o700))

	lib, err := filepath.Abs("../localnet_lib.sh")
	require.NoError(t, err)
	//nolint:gosec // G204: snippet is a constant in this file
	cmd := exec.Command("bash", "-c", "set -eu\n. \"$LIB\"\n"+snippet)
	cmd.Env = append(os.Environ(),
		"LIB="+lib,
		"PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FAKE_DIR="+dir,
		"FAKE_RESULTS="+strconv.Itoa(len(results)),
		"BUILD_MAX_ATTEMPTS=4",
		"BUILD_RETRY_BASE_SECONDS=0",
	)
	out, runErr := cmd.CombinedOutput()
	exitCode := 0
	if runErr != nil {
		var exitErr *exec.ExitError
		require.ErrorAs(t, runErr, &exitErr)
		exitCode = exitErr.ExitCode()
	}
	callLog, err := os.ReadFile(calls)
	if err != nil {
		require.ErrorIs(t, err, os.ErrNotExist)
	}
	return string(out), exitCode, strings.Count(string(callLog), "\n")
}

func TestBuildImageWithRetry(t *testing.T) {
	const (
		rateLimited = "ERROR: failed to solve: golang:1.22-bookworm: failed to resolve source metadata for docker.io/library/golang:1.22-bookworm: " +
			"failed to authorize: failed to fetch anonymous token: unexpected status from GET request to https://auth.docker.io/token: 429 Too Many Requests\n"
		gatewayTimeout = "ERROR: failed to solve: failed to copy: httpReadSeeker: failed open: " +
			"unexpected status code https://registry-1.docker.io/v2/library/debian/blobs/sha256:abc: 504 Gateway Timeout\n"
		compileError = "app/app.go:12:2: undefined: foo\n" +
			"ERROR: failed to solve: process \"/bin/sh -c GOPROXY=off make install\" did not complete successfully: exit code: 2\n"
		compileErrorNamingDocker = "vendor/github.com/docker/docker/client.go:429:2: undefined: foo\n" +
			"ERROR: failed to solve: process \"/bin/sh -c make install\" did not complete successfully: exit code: 2\n"
		manifestMissing = "ERROR: failed to solve: docker.io/library/golang:0.0: not found\n"
	)
	success := fakeDockerResult{stdout: "built\n"} //nolint:exhaustruct // a zero exit and no stderr

	tests := []struct {
		name          string
		results       []fakeDockerResult
		wantCalls     int
		wantFailed    bool
		wantInMessage string
	}{
		{name: "first attempt succeeds", results: []fakeDockerResult{success}, wantCalls: 1, wantFailed: false, wantInMessage: ""},
		{
			name:          "rate limit then success",
			results:       []fakeDockerResult{{stderr: rateLimited, exitCode: 1}, success}, //nolint:exhaustruct // no stdout
			wantCalls:     2,
			wantFailed:    false,
			wantInMessage: "",
		},
		{
			name:          "gateway timeout then success",
			results:       []fakeDockerResult{{stderr: gatewayTimeout, exitCode: 1}, success}, //nolint:exhaustruct // no stdout
			wantCalls:     2,
			wantFailed:    false,
			wantInMessage: "",
		},
		{
			name:          "registry never recovers",
			results:       []fakeDockerResult{{stderr: rateLimited, exitCode: 1}}, //nolint:exhaustruct // no stdout
			wantCalls:     4,
			wantFailed:    true,
			wantInMessage: "still failing after 4 attempts",
		},
		{
			name:          "compile failure is not retried",
			results:       []fakeDockerResult{{stderr: compileError, exitCode: 1}}, //nolint:exhaustruct // no stdout
			wantCalls:     1,
			wantFailed:    true,
			wantInMessage: "not retrying",
		},
		{
			name:          "compile failure naming docker is not retried",
			results:       []fakeDockerResult{{stderr: compileErrorNamingDocker, exitCode: 1}}, //nolint:exhaustruct // no stdout
			wantCalls:     1,
			wantFailed:    true,
			wantInMessage: "not retrying",
		},
		{
			name:          "missing image is not retried",
			results:       []fakeDockerResult{{stderr: manifestMissing, exitCode: 1}}, //nolint:exhaustruct // no stdout
			wantCalls:     1,
			wantFailed:    true,
			wantInMessage: "not retrying",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, exitCode, calls := runLocalnetLib(t, `build_image_with_retry -t allorad .`, tt.results...)
			require.Equal(t, tt.wantCalls, calls, out)
			require.Equal(t, tt.wantFailed, exitCode != 0, out)
			require.Contains(t, out, tt.wantInMessage)
		})
	}
}

func TestCaptureAllorad(t *testing.T) {
	const (
		address      = "allo1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu"
		protoWarning = "2026/09/24 16:45:33 proto: duplicate proto type registered: emissions.v2.Nonce\n"
		snippet      = `DOCKER_IMAGE=allorad LOCALNET_DATADIR=/tmp
value=$(capture_allorad "faucet address" '^allo1[02-9ac-hj-np-z]{38}$' /data/genesis keys show faucet -a)
printf 'captured=[%s]' "$value"`
	)

	tests := []struct {
		name       string
		result     fakeDockerResult
		wantOut    string
		wantFailed bool
	}{
		{
			name:       "clean address",
			result:     fakeDockerResult{stdout: address + "\n"}, //nolint:exhaustruct // no stderr, zero exit
			wantOut:    "captured=[" + address + "]",
			wantFailed: false,
		},
		{
			name:       "docker cli warning stays out of the value",
			result:     fakeDockerResult{stdout: address + "\n", stderr: "WARNING: Your kernel does not support swap limit capabilities\n"}, //nolint:exhaustruct // zero exit
			wantOut:    "captured=[" + address + "]",
			wantFailed: false,
		},
		{
			name:       "proto registration failure is named with a valid address on stdout",
			result:     fakeDockerResult{stdout: address + "\n", stderr: protoWarning}, //nolint:exhaustruct // zero exit
			wantOut:    "duplicate proto type registered",
			wantFailed: true,
		},
		{
			name:       "non-zero exit names the exit and stderr",
			result:     fakeDockerResult{stderr: "key not found\n", exitCode: 3}, //nolint:exhaustruct // no stdout
			wantOut:    "failed getting faucet address (exit 3)",
			wantFailed: true,
		},
		{
			name:       "value that is not an address",
			result:     fakeDockerResult{stdout: "not-an-address\n"}, //nolint:exhaustruct // no stderr, zero exit
			wantOut:    "invalid faucet address: 'not-an-address'",
			wantFailed: true,
		},
		{
			name:       "multi-line value",
			result:     fakeDockerResult{stdout: address + "\nsecond line\n"}, //nolint:exhaustruct // no stderr, zero exit
			wantOut:    "invalid faucet address",
			wantFailed: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, exitCode, calls := runLocalnetLib(t, snippet, tt.result)
			require.Equal(t, 1, calls, out)
			require.Equal(t, tt.wantFailed, exitCode != 0, out)
			require.Contains(t, out, tt.wantOut)
			if tt.wantFailed {
				require.NotContains(t, out, "captured=")
			}
		})
	}
}
