# Decisions (dev repo, not shipped)

Decisions about just-harness itself. Projects that install the harness do not receive this file.

## D1 — History before v1 (superseded)

v0.1–v0.3 shipped a Rust CLI with a local, gitignored SQLite database, installed by Bash and
PowerShell scripts, plus about 40 policy, template, and decision files. Benchmarks
(`harness-benchmark`) showed three things. Directive, artifact-heavy wording roughly doubled
task time with no functional gain. Agents wrote markdown records but skipped the matching DB rows.
The harness's own decisions leaked into consumer repos as misleading context.
v1 replaces all of it. The old history remains in git.

## D2 — Purpose and design test

The harness exists to make project knowledge shareable, let a new agent or session grasp progress,
and keep agents aligned with the user's scope through contracts, guardrails, and a completion gate.
A component that serves none of these is removed.

## D3 — Committed JSON records are the single source of truth

State lives in `.harness/<kind>/<ID>.json`, one record per file. Records are committed,
serialized deterministically, written atomically, and written only by the CLI. Prose lives in
markdown and carries no state.
Rejected: a local DB (progress not shared), a committed SQLite file (binary merges lose data and
diffs are unreadable), and status headers in markdown (easy to bypass by hand).

## D4 — One Go binary, standard library only

`just-harness-cli` embeds the payload and installs itself. No SQLite and no CGO. Releases target
linux/darwin on amd64/arm64 plus windows/amd64. Installation is a README one-liner; there is no
bootstrap script.

## D5 — Short random IDs

IDs look like `US-k3f9`, `D-…`, `G-…`, and `T-…` (4 characters of lowercase Crockford base32).
Parallel branches never allocate the same sequence number, and ordering uses `created_at`.

## D6 — Completion gate and waivers

A story becomes `implemented` only with a passing `last_verify` whose command equals the current
verify command. A `high_risk` story without a verify command needs a waiver. Waivers are explicit,
shown separately from passes, and dropped whenever the gate re-runs. The gate proves that a command
passed, not that the command tests the right thing. `check` re-applies the same rule to catch
hand edits and merges.

## D7 — Proportional policy

Most work creates no records. A story is created when tracking, handoff, or coordination needs it,
and always for `high_risk` work. Decisions, guardrails, and traces are created only when their
completion-contract clause applies. Payload wording stays neutral rather than directive (see D1).

## D8 — User over guardrail, but never silently

Guardrails bind agents. If the user's request conflicts with one, the agent names the conflict
and asks before proceeding. The user has the final say.

## D9 — Dropped from v0.3

Intake records, backlog, trace scoring tiers, self-reported proof booleans, arbitrary SQL,
`--json`, brownfield import, HARNESS_MATURITY/COMPONENTS, ARCHITECTURE/GLOSSARY/TEST_MATRIX docs,
and multi-file high-risk templates.
