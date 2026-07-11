# Project Rules

<!-- HARNESS:BEGIN -->
## Harness

Claude Code loads this file into every session, but it does not auto-load
`AGENTS.md`. The bare `@` line imports the small Harness entrypoint. Never wrap
it in backticks; that disables the import.

@AGENTS.md

Follow the entrypoint's retrieval guidance. Load Harness policy, contracts,
stories, decisions, and proof expectations only when the task requires them.
<!-- HARNESS:END -->
