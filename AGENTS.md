# Agent Instructions (just-harness dev repo)

This repo builds `just-harness-cli`. It is not itself an installed harness.

- `harness/` is the payload embedded into the binary and installed into other repos. Everything in
  it ships to users; keep it generic and small. Harness-internal rationale goes in `DECISIONS.md`.
- Code layout: `cmd/just-harness-cli` (entry), `internal/domain` (pure types and invariants),
  `internal/store` (records, IDs, prose), `internal/cli` (commands), `internal/install` (plan-first installer).
- Invariants that must not regress: records written only by the CLI, with deterministic serialization
  and atomic writes; the completion gate rule is shared by the CLI and `check`; a refused or dry-run
  `install` writes nothing; `install` never overwrites existing records.
- Go 1.24+, standard library only. Do not add dependencies without a decision in `DECISIONS.md`.
- Proof: `go vet ./... && go test ./...`. For install or CLI behavior, also smoke-test a built binary
  in a temporary git repo.
- Changing payload wording changes agent behavior. Prefer neutral, proportional wording; see D1 and D7.
