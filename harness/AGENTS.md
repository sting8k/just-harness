# Agent Instructions

Add project-specific instructions above or below the harness block.

<!-- HARNESS:BEGIN -->
## just-harness (placeholder — rewritten in S6)

Start every session: read `docs/HARNESS.md`, then run `.harness/bin/just-harness-cli query status`.
Write `.harness/` only through the CLI. Without the binary, read state with `cat .harness/**/*.json`.
<!-- HARNESS:END -->
