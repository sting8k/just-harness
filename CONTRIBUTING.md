# Contributing to just-harness

The most useful contributions are evidence from real use:

- **Agent failure cases:** what you asked, what the agent got wrong, and whether a guardrail,
  contract, or the completion gate could have caught it.
- **Friction:** a step in the harness that cost time without improving the outcome.
- **Bugs** in `just-harness-cli`, with the command, the output, and your platform.

## Pull requests

1. Read `AGENTS.md` and `DECISIONS.md`.
2. Keep changes focused. Payload changes (`harness/`) ship to every user; explain why each is needed.
3. Run `go vet ./... && go test ./...`, and describe any manual smoke test.
4. If the change alters a decision, update `DECISIONS.md` in the same PR.
