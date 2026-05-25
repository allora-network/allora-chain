# DEVOP-618 + DEVOP-619 — Scheduled govulncheck + Pinned Go env in CI

Date: 2026-05-25
Tickets: [DEVOP-618](https://linear.app/allora/issue/DEVOP-618), [DEVOP-619](https://linear.app/allora/issue/DEVOP-619)

## What this PR does

1. Adds `.github/workflows/go-vuln-scan.yml`, a thin scheduled invocation of the
   reusable `allora-network/ci-workflows-private/.github/workflows/go-vuln-scan.yml`
   workflow. Runs weekly (`cron: '17 5 * * 1'`) and on `workflow_dispatch`, with
   `fail_on_findings: true` so a newly-disclosed vuln in our deps surfaces loudly
   between PR runs.
2. Adds a top-level `env:` block to the three workflows that invoke `go` at the
   runner level: `format_and_test.yml`, `golangci-lint.yml`, `goreleaser.yml`.
   Block: `GOPROXY=https://proxy.golang.org,direct`, `GOSUMDB=sum.golang.org`,
   `GOFLAGS=-mod=readonly`, `CGO_ENABLED=1`.

## Workflows audited

- `format_and_test.yml` — calls `setup-go` + `go test`. **env block added**.
- `golangci-lint.yml` — calls `setup-go` + `go run` for custom linters. **env block added**.
- `goreleaser.yml` — calls `setup-go` + goreleaser (which shells out to `go`). **env block added**.
- `go-hardened.yml` — calls the reusable hardened-install workflow which sets these
  env vars internally. **Skipped (already covered)**.
- `buf-ci.yaml` — only runs `bufbuild/buf-action`; no `go` at runner level. **Skipped**.
- `build_push_docker_hub.yml`, `build_push_upgrader_docker_hub.yml` — only invoke
  `docker build`; `go` runs inside the container, not the runner. **Skipped**.

Audit for forbidden env (`rg 'GONOSUMCHECK|GOFLAGS:.*-insecure|GOSUMDB:.*off' .github/workflows/`)
returned no hits.

## CGO_ENABLED=1 carve-out

DEVOP-619 defaults to `CGO_ENABLED=0`. allora-chain's existing `go-hardened.yml`
passes `cgo_enabled: '1'` to the reusable hardened-install workflow, so this
PR mirrors that convention for consistency across the Shai-Hulud env block.

Notes:

- `wasmvm` / `libwasmvm` is NOT a dependency of this repo (confirmed against
  `go.mod` and `go.sum`); the comment that originally cited it has been
  removed. Real CGO-linking deps in the graph are `cockroachdb/pebble` and
  cosmos-ledger transitives (under the `pebbledb`/`ledger` build tags).
- Release binaries override this to `CGO_ENABLED=0` per-build via
  `.goreleaser.yaml` (`env: [CGO_ENABLED=0]`), so the goreleaser workflow's
  workflow-level env is effectively unused for the actual `go build` step.
- For `format_and_test.yml`, integration tests build/run the chain via
  `test/local_testnet_l1.sh` against the CGO-enabled build path.
- For `golangci-lint.yml`, the linters themselves do not link C; the env is
  carried for uniformity with the rest of the Shai-Hulud block.

If the future audit finds that `CGO_ENABLED=0` works for the test/lint path
without behavioral change, flipping it is straightforward. Until then, "1"
keeps a single env block reusable.

## Reusable workflow pin

`go-vuln-scan.yml` is pinned to commit
`185ec34dc707e6779bcc06fe2a592e07f29de8ed`, which is the current HEAD of
`ci-workflows-private` PR #17 (DEVOP-616). PR #17 has not yet merged to `main`.
The SHA is addressable today and remains addressable after PR #17 merges, so the
workflow becomes valid the moment PR #17 lands. After PR #17 merges, a
follow-up should re-pin to the merge commit on `main` for auditability.
