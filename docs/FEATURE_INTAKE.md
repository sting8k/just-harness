# Feature Intake

Every implementation prompt enters intake before code or contract changes. A new spec also enters here before it becomes work packets or implementation work.

The human does not need to classify risk. The harness does.

## Intake Flow

```text
User prompt
    |
    v
Classify input type
    |
    v
Map context
    |
    v
Build work packet
    |
    v
Choose lane: tiny, normal, or high-risk
```

## What Intake Is For

Intake is agent warmup. It should answer:

- What kind of work is this?
- What context must be loaded?
- What should stay out of scope?
- What proof will close the work?

## Input Types

Use the input type to decide where the work should land before choosing the risk lane.

| Type | Use when | Typical artifact |
| --- | --- | --- |
| New spec | Turning a user-provided project spec into harness-ready docs | Work packets, product docs, decisions |
| Spec slice | Implementing selected behavior from an accepted spec | Work packet |
| Change request | Changing, fixing, or refining accepted behavior | Work packet or direct patch |
| New initiative | Adding a larger area that needs multiple packets | Initiative notes plus work packets |
| Maintenance request | Changing technical, operational, or dependency behavior | Work packet, validation report, or decision |
| Harness improvement | Improving how humans and agents collaborate | Direct docs update or `scripts/bin/harness-cli backlog add` |

Read-heavy or multi-repo work still uses the same intake path. Keep one work packet and add `Checklist`, `Findings`, `Tasks`, and `Evidence` sections instead of inventing a new artifact type.

Do not create or extend a monolithic spec by default after intake. Use product docs, work packets, decisions, and initiative notes as the living surface.

## Lanes

### Tiny

Use for low-risk docs, copy, names, or narrow edits.

Also use for initial project setup when the work is limited to installing declared dependencies, wiring a server entrypoint, adding a health/smoke endpoint, or opening a local development database connection without creating domain schema, CRUD behavior, auth, authorization, provider integration, or data migration. A health endpoint in a new benchmark or scaffolded project is smoke proof, not a public contract escalation by itself.

Requirements:

- Patch directly.
- Keep affected docs current.
- Run available quick checks.
- Update the harness only if friction was found.

### Normal

Use for work-packet sized behavior with bounded blast radius.

Requirements:

- Create or update one work packet file from `docs/templates/story.md`.
- Link relevant docs.
- Add or update validation expectations.
- Implement the smallest vertical slice when implementation exists.
- Record or update proof status with `scripts/bin/harness-cli story add` and `scripts/bin/harness-cli story update`.

### High-Risk

Use when the work can affect security, data, scope, contracts, or multiple roles/platforms.

Requirements:

- Create a work packet folder using `docs/templates/high-risk-story/` only when the packet is large enough to need `execplan.md`, `overview.md`, `design.md`, and `validation.md`.
- Ask for human confirmation before implementation if direction is ambiguous.
- Record a durable decision when behavior, architecture, authorization, data ownership, API shape, or validation requirements change meaningfully. Use a `docs/decisions/NNNN-*.md` file from `docs/templates/decision.md`, then add or refresh the durable row with `scripts/bin/harness-cli decision add`.
- Decision text in a trace is not a durable decision record.

## Risk Checklist

Mark one flag for each item that applies:

| Risk flag | Applies when the work touches |
| --- | --- |
| Auth | login, logout, sessions, JWT, password, refresh token |
| Authorization | roles, permissions, tenant or company scope |
| Data model | schema, migrations, uniqueness, deletion, retention |
| Audit/security | audit logs, privacy, sensitive data, access logs |
| External systems | email, payments, cloud services, provider SDKs, queues, webhooks |
| Public contracts | API shape, response envelope, client-visible behavior |
| Cross-platform | desktop/mobile/browser split, native shell behavior, deep links |
| Existing behavior | already implemented or test-covered behavior changes |
| Weak proof | unclear or missing tests around the affected area |
| Multi-domain | more than one product domain changes at once |

## Classification

```text
0-1 flags:
  tiny or normal, based on code impact

2-3 flags:
  normal with stronger validation

4+ flags:
  high-risk

Any hard gate:
  high-risk unless the human explicitly narrows scope
```

Hard gates:

- Auth.
- Authorization.
- Data loss or migration.
- Audit/security.
- External provider behavior.
- Removing or weakening validation requirements.

## Output

At the end of intake, the agent should be able to say:

```text
Type: normal
Lane: normal
Read first: docs/product/overview.md, docs/decisions/0004-sqlite-durable-layer.md
Work packet: docs/stories/US-001-short-title.md
Proof: unit and integration
Open questions: none
```
