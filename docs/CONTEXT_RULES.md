# Context Engineering Rules

Context rules help agents decide what to read, when to read it, and when to stop. They are additive to the stable `AGENTS.md` reading list.

The goal is not to maximize context. The goal is to put the right information in the model for the current task phase and risk lane.

## Context Phases

### Intake Phase

Read to classify the request, find the affected surface, and choose a lane.

| Document or Source | Tiny | Normal | High-Risk |
| --- | --- | --- | --- |
| `AGENTS.md` | Must | Must | Must |
| `docs/FEATURE_INTAKE.md` | Must | Must | Must |
| `docs/HARNESS.md` | Must | Must | Must |
| `README.md` | Should | Must | Must |
| `scripts/bin/harness-cli query matrix` | Must | Must | Must |
| `docs/GUARDRAILS.md` | Skip | Should if project directives matter | Must |
| `docs/ARTIFACTS.md` | Skip | Should if packet shape or folder rules matter | Must |
| Relevant `docs/product/*` | Skip if unrelated | Must if work touches the work contract | Must |
| Relevant `docs/stories/*` | Skip if unrelated | Must if a packet exists | Must |
| `docs/decisions/*` | Skip | Should if durable rules are touched | Must |
| `docs/HARNESS_BACKLOG.md` | Skip | Should if harness friction is relevant | Must if changing harness behavior |

### Planning Phase

Read to decide the smallest safe approach and expected proof.

| Document or Source | Tiny | Normal | High-Risk |
| --- | --- | --- | --- |
| Current files to edit | Must | Must | Must |
| `docs/templates/story.md` | Skip | Must when creating or updating a packet | Should |
| `docs/templates/high-risk-story/*` | Skip | Skip unless risk escalates | Must |
| `docs/ARCHITECTURE.md` | Skip | Should for structural changes | Must |
| `docs/TEST_MATRIX.md` or `scripts/bin/harness-cli query matrix` | Should | Must | Must |
| Relevant decisions | Skip | Should | Must |
| `docs/GUARDRAILS.md` | Skip | Should when project directives affect the task | Must |
| `docs/ARTIFACTS.md` | Skip | Should when changing naming or folder rules | Must when changing taxonomy |
| `docs/HARNESS_BACKLOG.md` and `scripts/bin/harness-cli query backlog` | Skip | Should if friction repeats | Must if changing Harness behavior |
| `docs/HARNESS_MATURITY.md` | Skip | Should for Harness improvements | Must for maturity claims |

### Implementation Phase

Read while making the change. Keep this phase scoped to files that directly affect the selected packet.

| Document or Source | Tiny | Normal | High-Risk |
| --- | --- | --- | --- |
| Files being changed | Must | Must | Must |
| Adjacent files with the same pattern | Should | Must | Must |
| Relevant product docs | Skip if copy-only | Must if work changes behavior | Must |
| Relevant packet | Skip if no packet needed | Must | Must |
| Relevant templates | Skip | Should when adding docs | Must |
| `docs/ARCHITECTURE.md` | Skip | Should for structural changes | Must |
| Provider/API/security docs | Skip | Should if touched | Must |
| Unrelated docs and historical traces | Skip | Skip | Should only if they affect decisions |

### Validation Phase

Read to prove the change and avoid claiming unsupported completion.

| Document or Source | Tiny | Normal | High-Risk |
| --- | --- | --- | --- |
| Acceptance criteria in the packet | Should | Must | Must |
| `docs/TEST_MATRIX.md` or `scripts/bin/harness-cli query matrix` | Should | Must | Must |
| Validation section of the packet | Skip if no packet | Must | Must |
| `docs/templates/validation-report.md` | Skip | Should for notable proof | Must for high-risk proof |
| Relevant commands from README or package docs | Should | Must | Must |
| `docs/HARNESS_MATURITY.md` | Skip | Should for Harness improvements | Must for maturity claims |

### Trace Phase

Read to leave useful evidence for the next agent and for benchmark scoring.

| Document or Source | Tiny | Normal | High-Risk |
| --- | --- | --- | --- |
| `docs/TRACE_SPEC.md` | Should | Must | Must |
| `scripts/bin/harness-cli query matrix` | Should | Must | Must |
| `scripts/bin/harness-cli query backlog` | Skip | Should if friction occurred | Must |
| Changed-file list from `git status --short` | Must | Must | Must |
| Validation command output | Should | Must | Must |
| Packet or progress log | Skip if no packet | Must | Must |

## Retrieval Triggers

| Trigger condition | Action |
| --- | --- |
| Task touches database schema, durable records, or migrations | Read `docs/decisions/0004-sqlite-durable-layer.md`, `scripts/schema/`, and relevant CLI code before planning. |
| Task touches CLI command behavior or installer distribution | Read `docs/decisions/0005-prebuilt-rust-harness-cli.md`, `scripts/README.md`, relevant `crates/harness-cli/*` code, CLI help output, and installer docs. |
| Task touches auth, authorization, audit/security, data loss, or external providers | Treat as high-risk, read the high-risk packet template, and check prior decisions before implementation. |
| Task changes policy, source hierarchy, risk classification, validation requirements, guardrails, or artifact taxonomy | Read `docs/HARNESS.md`, `docs/FEATURE_INTAKE.md`, `docs/CONTEXT_RULES.md`, `docs/GUARDRAILS.md`, `docs/ARTIFACTS.md`, and relevant decisions; pause if direction is ambiguous. |
| Task discovers repeated confusion, stale docs, or missing proof | Read `docs/HARNESS_BACKLOG.md`, record `harness_friction`, and add a backlog item when the fix is out of scope. |
| Task makes a maturity, observability, trace quality, or benchmark claim | Read `docs/HARNESS_COMPONENTS.md`, `docs/HARNESS_MATURITY.md`, and `docs/TRACE_SPEC.md`. |
| Task is normal or high-risk and spans multiple iterations | Create or update a packet or progress file under `docs/stories/` and keep it current. |
| Final response is being prepared | Re-read the validation evidence, `git status --short`, and `docs/TRACE_SPEC.md` before recording the final trace. |

## Token Budget Guidance

| Lane | Target Context Budget | Read Shape | Reasoning |
| --- | --- | --- | --- |
| Tiny | About 2K tokens of Harness context | `AGENTS.md`, `docs/FEATURE_INTAKE.md`, matrix query, and the exact file being changed. | Tiny work should not spend more context on policy than on the edit. |
| Normal | About 5K tokens of Harness context | Intake docs, relevant work docs, architecture when structural, validation expectations, and trace spec at the end. | Normal work needs enough context to preserve contracts and record proof without reading every historical file. |
| High-risk | About 10K tokens of Harness context | Full intake, architecture, relevant decisions, high-risk templates, work docs, validation docs, trace spec, and component or maturity docs when Harness behavior changes. | High-risk work needs source hierarchy, prior decisions, and proof expectations in context before implementation. |

Budget rules:

- Prefer targeted searches over bulk reading.
- Read the smallest section that answers the current phase question.
- Escalate context when a retrieval trigger fires.
- Do not keep reading unrelated history after the lane, affected files, and validation path are clear.

## Additive Behavior

These rules do not replace `AGENTS.md`. Agents should still read the stable entrypoint documents listed there before work. This document explains what to retrieve after that initial context, based on lane, phase, and trigger.

## Review Checklist

Before implementation:

- Lane is chosen from `docs/FEATURE_INTAKE.md`.
- Relevant work docs are identified.
- Any high-risk trigger has been handled.

Before final response:

- Validation evidence has been read.
- `docs/TRACE_SPEC.md` has been read for normal or high-risk tasks.
- The final trace includes files read, files changed, outcome, proof status, and `harness_friction` as a concrete issue or `none` when repo-changing work was done.
