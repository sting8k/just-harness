# Agent Instructions

Add project-specific agent instructions here.

<!-- HARNESS:BEGIN -->
## Harness

Default flow: understand the request and relevant design, implement the smallest fitting change, verify it, and report the result and any unverified gaps.

Start with `docs/HARNESS.md`. Retrieve other Harness docs only when its triggers or the task require them; do not load the full framework by default.

Use the Rust Harness CLI at `scripts/bin/harness-cli` on macOS/Linux or `scripts/bin/harness-cli.exe` on Windows for durable records and mechanical checks when they are useful.
<!-- HARNESS:END -->
