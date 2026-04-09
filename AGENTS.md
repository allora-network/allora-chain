# AGENTS.md

Short, high-signal guidance for working in the `allora-chain` repo. Keep context small and only load extra docs when needed.

## Quick Rules (Apply To Most Changes)
- Determinism: no `time.Now()`/`rand`, no map-order dependence, no floats in consensus; use `alloraMath.Dec`/`sdk.Int`/`sdk.Dec` and stable iteration.
- State access: keepers are the only state access point; no IO, goroutines, or non-determinism inside keepers.
- Validation: `ValidateBasic` + keeper stateful checks; use `sdkerrors/errorsmod` for typed errors.
- Events: emit events for all state transitions with consistent type/attribute keys.
- Protos: never change field numbers; add new optional fields; use `Request/Response` for list queries; new versions for breaking changes.
- State layout: define keys/prefixes in `types/keys.go`; keep prefix types stable; add indexes for query-heavy data.
- Params/genesis: validate every param change + authority checks; genesis is deterministic with round-trip tests.

## Go Conventions
- Always `gofmt`/`goimports`; keep `golangci-lint` clean. Always after producing code that may be pushed.
- Keep `nolint` statements where needed, adding a comment on the rationale if none is there.
- Prefer table-driven tests and explicit dependency injection for easy mocking.
- Use `sdk.Context.Logger()` for chain logs; avoid noisy logs in consensus paths.
- Only apply Allora Go Service patterns (brynbellomy errors, sqlc, viper, Gin, zerolog) if editing non-chain services.

## Changes That Require Extra Steps
- Proto/state layout changes: add module migration + tests; bump `ConsensusVersion` and wire upgrades if needed.
- New module/store keys: update app wiring for store upgrades.
- Generated files (`*.pb.go`, `*.pulsar.go`): never edit by hand; run codegen.

## Minimal Context Strategy
Start in the module you touch: `x/<module>/keeper`, `x/<module>/types`, `x/<module>/module`.
Read extra references only when needed:
- Protos/state/migrations/upgrades: `.agents/skills/allora-chain-style-guide/references/{protos,state,migrations,upgrades}.md`
- Determinism/events/errors/params/genesis: `.agents/skills/allora-chain-style-guide/references/{determinism,events_and_errors,params,genesis}.md`
- Tests/integration/fuzzer: `.agents/skills/allora-chain-style-guide/references/{tests,integration_tests,fuzzer}.md`
- Go services (only if relevant): `.agents/skills/allora-go-style-guide/references/*.md`

## Repo Layout (Short)
- `app/`: app wiring, keepers, upgrades
- `cmd/`: binaries
- `x/`: modules (keeper, types, module, migrations, testutil)
- `test/`, `testutil/`: integration + helpers

## Git Commit Policy
**All commits MUST be signed.** This repository requires verified (signed) commits. Before making any commit:
1. **Verify the signing key is available**: Run `ssh-add -l` and confirm the configured signing key is listed. If the agent has no identities or the key is missing, **stop immediately** and tell the user to load the key (e.g. `ssh-add ~/.ssh/id_ed25519` but this may vary) before proceeding. Do NOT bypass signing.
2. **Never disable signing**: Do not use `-c commit.gpgsign=false`, `--no-gpg-sign`, or any other mechanism to skip commit signing. Unsigned commits will show as "Unverified" on GitHub and are not acceptable.
3. **If signing fails or hangs**: Do not retry with signing disabled. Stop and inform the user that the SSH signing key is unavailable and needs to be loaded into the agent.
4. **Verify after committing**: After each commit, run `git log -1 --format='%G?'` to confirm the commit was signed. A result of `G` means good signature. If the signature is missing or bad, stop and inform the user before pushing.
5. **Never use `--no-verify`**: Do not skip pre-commit hooks unless the user explicitly asks for it.

### Pre-flight Check
Before the first commit in any session, run these checks. If any fail, **stop and inform the user** before doing any work — but ask whether they still want you to proceed with non-committing tasks (e.g. code changes, exploration, reviews) while they resolve the issue.

### Git Remote Protocol
**Never change the git remote URL.** The remote must stay as it is unless otherwise requested. Do not switch HTTPS/SSH, do not run `git remote set-url`, and do not run `gh auth setup-git`, unless explicitly stated, requested and only execute if approved by the user. 