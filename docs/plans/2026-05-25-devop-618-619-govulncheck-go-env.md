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
   `GOFLAGS=-mod=readonly`, `CGO_ENABLED=0`.
3. Pins `actions/setup-go` in `goreleaser.yml` to Go `1.23.5` so release
   artifacts are produced by the same Go toolchain as the hardened CI path
   (was `1.22.2`, drifted from `go.mod`'s `toolchain go1.23.5` directive).

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

## CGO_ENABLED=0 alignment

DEVOP-619 defaults to `CGO_ENABLED=0`. This PR sets `CGO_ENABLED=0` in all
three workflow-level env blocks (`format_and_test.yml`, `golangci-lint.yml`,
`goreleaser.yml`).

Evidence collected during review of PR #952:

- `wasmvm` / `libwasmvm` / `cosmwasm` is NOT a dependency of this repo:
  `rg -i 'wasmvm|libwasmvm|cosmwasm' go.mod go.sum` returns zero matches.
- No Go source in this repo does `import "C"` (confirmed via repo-wide search).
- Release binaries are explicitly `CGO_ENABLED=0` per-build via
  `.goreleaser.yaml` (`env: [CGO_ENABLED=0]`) with `netgo + pebbledb + ledger`
  build tags. The fact that the release build succeeds with CGO=0 + those
  tags proves none of `cockroachdb/pebble`, cosmos-ledger transitives, or
  any other dep in the graph forces CGO at link time.
- For `format_and_test.yml`, integration tests build/run the chain via
  `test/local_testnet_l1.sh`, which matches the CGO=0 release toolchain.
- For `golangci-lint.yml`, the linters themselves do not link C.

Conclusion: setting `CGO_ENABLED=0` across CI both removes a divergence
between `goreleaser.yml`'s workflow-level env and its underlying
`.goreleaser.yaml`'s per-build env, AND removes a misleading wasmvm-based
justification flagged by cubic (P3) and xmariachi on PR #952.

Out of scope for this PR: `go-hardened.yml` (added by merged PR #951) still
passes `cgo_enabled: '1'` to the reusable hardened-install caller. After
this PR lands, that pin is inconsistent with every other workflow. Tracked
as a follow-up to flip to `cgo_enabled: '0'` in a separate PR so it can be
reviewed on its own merits.

If `wasmvm` (or any other CGO-linking dep) is reintroduced in the future,
flip these workflow env blocks back to `CGO_ENABLED=1` and revisit the
`.goreleaser.yaml` per-build env at the same time. The `TODO(DEVOP-619
follow-up)` comments in each workflow point to this section.

## Reusable workflow pin

`go-vuln-scan.yml` is pinned to commit
`185ec34dc707e6779bcc06fe2a592e07f29de8ed`, which is the current HEAD of
`ci-workflows-private` PR #17 (DEVOP-616). PR #17 has not yet merged to `main`.
The SHA is addressable today and remains addressable after PR #17 merges, so the
workflow becomes valid the moment PR #17 lands. After PR #17 merges, a
follow-up should re-pin to the merge commit on `main` for auditability.
