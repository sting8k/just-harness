# Harness Guardrails

These are durable directives for the harness itself.

## How to Use This File

Write short, concrete rules here when they should shape future agent behavior. Mark each guardrail `active` or `superseded`.

## Active Guardrails

| Status | Guardrail | Why it exists |
| --- | --- | --- |
| active | Opinionated, not bureaucratic | The framework should constrain agent behavior without adding ceremony for its own sake. |
| active | Contract-first | Intake, work packet, proof, and persistence stay explicit. |
| active | Minimal core | Prefer a small set of primitives over a new artifact type for each use case. |
| active | Flat-first structure | Root + one nested group is the normal band; deeper trees are exceptional. |
| active | CLI-aware | Markdown contracts and durable CLI state must agree. |
| active | Harness-first edits | Changes in this repo adjust the framework itself, not a consumer workflow. |
| active | Escalate only when needed | Large, risky, or read-heavy work can expand into checklist, findings, tasks, and evidence inside the same packet. |

## Record Format

```md
- Date:
- Source:
- Guardrail:
- Status:
- Notes:
```