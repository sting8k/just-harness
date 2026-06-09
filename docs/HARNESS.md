# Harness

The project goal is to provide a reusable operating harness that helps humans and agents turn intent into safe, validated work.

This repository is the framework surface. Changing these docs changes the framework itself.

## Operating Model

```text
User input
  -> Intake / Warmup
      -> Classify
      -> Map context
      -> Build work packet
  -> Execute
  -> Verify
  -> Persist learning
  -> Next intent
```

Intake warms the agent up. Classify chooses the kind of work. Map context chooses what to read. Build work packet turns the request into a bounded contract.

## Contract Layers

- **Intake contract**: what this is, how risky it is, and how much context to load.
- **Work contract**: goal, scope, non-goals, constraints, affected surfaces, and proof.
- **Proof contract**: what must pass before the work can close.
- **Persistence contract**: what must survive as guardrail, decision, trace, backlog item, or harness delta.

## Output Types

Every task may produce one or both of these durable outcome types:

1. **Work delta**: docs, code, tests, findings, tasks, checklists, or evidence that move the selected work forward.
2. **Harness delta**: policy docs, templates, validation expectations, backlog items, guardrails, or decisions that improve the framework.

## Harness v0 Scope

Harness v0 includes:

- Agent entrypoint.
- Empty product/work documentation structure.
- Intake/warmup and risk lanes.
- Story templates.
- Decision templates.
- Validation templates.
- Test matrix.
- Harness growth backlog.
- Durable layer: SQLite database and CLI for operational records.

Harness v0 deliberately excludes:

- A project-specific `SPEC.md`.
- Pre-sliced product domains.
- A locked application stack.
- App source scaffolding.
- Package scripts.
- Test runner config.
- CI workflows.

Those arrive only when a selected work packet needs them.

## Durable Layer

Policy docs describe how to work. The durable layer stores what happened.

Operational data — intake classifications, work packet status, decision outcomes, backlog items, and execution traces — lives in a SQLite database (`harness.db`) managed by the Rust Harness CLI at `scripts/bin/harness-cli`. Agents and humans should use that binary for Harness work. The database is local to each project instance and `.gitignore`d. The schema is version-controlled under `scripts/schema/`.

This separation keeps policy docs stable and human-readable while giving agents a structured, queryable record of operational state. It also prepares the harness for future observability and automated evolution without adding more markdown files.

Initialize the database if it does not exist:

```bash
scripts/bin/harness-cli init
```

Common commands:

```bash
scripts/bin/harness-cli intake  --type <type> --summary <text> --lane <lane> --context <paths> --packet <id>
scripts/bin/harness-cli story   add --id <id> --title <text> --lane <lane>
scripts/bin/harness-cli story   update --id <id> --status <status>
scripts/bin/harness-cli story   update --id <id> --unit 1 --integration 1 --e2e 0 --platform 0
scripts/bin/harness-cli story   verify <id>
scripts/bin/harness-cli decision add --id <id> --title <text> --doc docs/decisions/<file>.md
scripts/bin/harness-cli guardrail add --guardrail "<rule>" --why "<reason>"
scripts/bin/harness-cli guardrail list --active
scripts/bin/harness-cli trace   --summary <text> --outcome <outcome>
scripts/bin/harness-cli score-trace
scripts/bin/harness-cli query   matrix
scripts/bin/harness-cli query   matrix --numeric
scripts/bin/harness-cli query   guardrails
scripts/bin/harness-cli query   backlog
scripts/bin/harness-cli query   stats
scripts/bin/harness-cli --version
```

## Source Hierarchy

```text
User input or supplied spec
  input material for the first buildout or for future changes

docs/product/*
  current work contract derived from accepted input

docs/stories/*
  work packets and historical evidence

docs/decisions/*
  why the contract changed

docs/GUARDRAILS.md
  durable project directives

docs/ARTIFACTS.md
  naming and folder rules

scripts/bin/harness-cli query matrix
  behavior-to-proof control panel backed by the durable layer
```

Before implementation, work docs describe intent. After implementation, the work contract plus executable tests become the living contract.

## Work Packet Rule

Default unit of work is one work packet. Keep it as a single markdown file first. Split into a folder only when the packet becomes large, repeated, or needs sibling subfiles.

Use the same packet shape for product changes, audits, inventories, spikes, migrations, and harness improvements. Read-heavy packets can add `Checklist`, `Findings`, `Tasks`, and `Evidence` sections instead of introducing new artifact types.

## Spec Lifecycle

Harness v0 starts without a tracked project spec. When the human provides a specification, treat it as input material, not as a permanent operating manual. Use it to populate work packets, product docs, decisions, and validation expectations during the first buildout.

After the specification has been decomposed, do not keep extending it as the living plan. Ongoing work should update the smaller docs, packets, durable proof records, and decisions.

## Growth Rule

The harness grows from friction.

When an agent is confused, repeats manual reasoning, needs a new validation command, discovers a missing rule, or sees a recurring failure pattern, it must either improve the harness directly or record the friction:

```bash
scripts/bin/harness-cli backlog add --title "<short name>" --pain "<what was hard>"
```

Use the backlog outcome loop for improvements expected to change agent behavior or validation results.

The `harness_friction` field on traces also captures per-task friction so patterns can be queried later.

Backlog risk uses the same lane vocabulary as intake and stories: `tiny`, `normal`, or `high-risk`. Use `--risk tiny` for low-risk follow-up items; `low` is not a valid lane.

## Task Loop

Use this loop for repo-changing work: implementation, docs edits, harness updates, validation changes, or any task that should leave durable evidence. For read-only questions, status checks, or trivial commands, skip durable records and say why.

1. Classify the request with `docs/FEATURE_INTAKE.md`.
2. Record a fresh classification with `scripts/bin/harness-cli intake`; prefer `--context` for context-map paths and `--packet` when linking a work packet. Do not reuse a previous task's intake as current evidence.
3. Create or update a story when the work changes behavior, acceptance criteria, multiple files, or multiple steps. Tiny direct patches can skip a story when the final trace explains why.
4. Locate the affected docs and packet files.
5. Check proof status with `scripts/bin/harness-cli query matrix`.
6. Work only inside the selected lane: tiny, normal, or high-risk.
7. Verify before claiming behavior works. If proof is missing, too expensive, or failing, report the behavior as unverified, skipped, partial, or failed instead of completed.
8. Before finishing, ask whether work truth, validation expectations, guardrails, architecture rules, repeated failure patterns, or next-agent instructions changed; record new durable guardrails with `scripts/bin/harness-cli guardrail add`.
9. Record a fresh trace with `scripts/bin/harness-cli trace`, using `docs/TRACE_SPEC.md` for the expected trace tier and field depth. Include proof and `harness_friction`; use `none` only after checking for friction.
10. Review the trace score printed by `scripts/bin/harness-cli trace`; use `scripts/bin/harness-cli score-trace --id <id>` only when re-checking a specific historical trace.
11. If harness friction was found, either fix it directly or record it with `scripts/bin/harness-cli backlog add`.

## Story Verification

Stories may carry a mechanical proof command:

```bash
scripts/bin/harness-cli story add --id US-012 --title "Story verification" --lane normal --verify "cargo test --workspace"
scripts/bin/harness-cli story update --id US-012 --verify "cargo test --workspace"
scripts/bin/harness-cli story verify US-012
```

`story verify` runs the command from the repository root, records `last_verified_at` and `last_verified_result`, and exits 0 on pass or 1 on fail. When `trace --story <id>` links to a story whose verification command has never passed, the trace still records but prints an advisory warning before close.

`story verify` accepts only the work packet id. Configure the command with `story add --verify` or `story update --verify`. Record proof booleans with `story update`, using numeric values: `1` means yes and `0` means no. The Rust CLI rejects text values such as `yes` and `no`.

Use `scripts/bin/harness-cli query matrix --numeric` when copying proof values into story updates.