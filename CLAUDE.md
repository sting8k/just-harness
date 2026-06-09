# Project Rules

<!-- HARNESS:BEGIN -->
## Harness

Claude Code loads this file into every session, but it does not auto-load
`AGENTS.md`. The bare `@` lines below import the core Harness context at
context-load time. Never wrap them in backticks; that disables the import.

@AGENTS.md

@docs/HARNESS.md

@docs/FEATURE_INTAKE.md

@docs/CONTEXT_RULES.md

@docs/GUARDRAILS.md

@docs/ARTIFACTS.md

Also run `scripts/bin/harness-cli query matrix` before starting work.

Lane-dependent context (`README.md`, `docs/ARCHITECTURE.md`, product docs,
stories, decisions) is intentionally not imported — read it per lane, as
`docs/CONTEXT_RULES.md` prescribes.
<!-- HARNESS:END -->