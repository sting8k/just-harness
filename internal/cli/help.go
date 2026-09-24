package cli

import (
	"fmt"
	"io"
)

const helpTop = `just-harness-cli — shared project state for agents and people.

State lives in .harness/<kind>/<ID>.json (committed; written only by this CLI).
Prose lives in docs/. Run 'query status' at the start of every session.

Commands:
  install [path] [flags]         Install or upgrade the harness in a repo
  story add|update|verify        Work packets and the completion gate
  decision add|update            Decisions later work must inherit
  guardrail add|supersede        Durable project rules
  trace [flags]                  Record evidence of a finished piece of work
  query status|stories|decisions|guardrails|traces
  check                          Exit 1 if any record breaks an invariant
  version | help [command]

Exit codes: 0 ok · 1 runtime error or invariant refusal · 2 bad input.
Environment: HARNESS_REPO_ROOT overrides repo root detection.
`

var helpCommands = map[string]string{
	"install": `Usage: just-harness-cli install [path] [--merge|--override] [--force] [--dry-run] [--refresh-agent-shim] [--claude]

Installs the payload into path (default "."). Plans every write first; a
refused plan writes nothing. Existing AGENTS.md, docs/ or .harness/ cause a
refusal unless:
  --merge               keep existing files, add missing ones, refresh the
                        HARNESS block in AGENTS.md, replace the binary (upgrade)
  --override            back up AGENTS.md and docs/ to .harness-backup/<ts>/,
                        then install fresh (.harness/ records are never touched)
  --force               overwrite existing payload files (each backed up first)
  --dry-run             print the plan, write nothing
  --refresh-agent-shim  refresh the HARNESS block in AGENTS.md
  --claude              also create CLAUDE.md containing @AGENTS.md
`,
	"story": `Usage:
  just-harness-cli story add    --title "..." --lane tiny|normal|high_risk [--verify "cmd"] [--contract path] [--no-packet]
  just-harness-cli story update --id US-xxxx [--status S] [--title T] [--lane L] [--verify "cmd"] [--contract path] [--waive "reason"]
  just-harness-cli story verify --id US-xxxx [--timeout 10m]

status: planned|in_progress|implemented|changed|retired. Any transition is
allowed; the gate runs when a story becomes implemented (or an implemented
story changes verify or lane):
  I1  needs last_verify pass with command == verify, or --waive "reason"
  I2  high_risk without verify needs --waive; tiny/normal without verify pass
  I3  changing verify clears last_verify; 'changed' clears it too
A waiver is shown separately from a pass and dropped when the story leaves
implemented. 'story verify' runs the command from the repo root (sh -c /
cmd /C), streams its output, records last_verify (commit, dirty) and exits
with the command's exit code (124 on --timeout).
`,
	"decision": `Usage:
  just-harness-cli decision add    --title "..." [--status proposed|accepted] [--story US-xxxx]
  just-harness-cli decision update --id D-xxxx --status proposed|accepted|superseded|rejected [--by D-xxxx]

'add' also creates docs/decisions/<ID>-<slug>.md from the template.
`,
	"guardrail": `Usage:
  just-harness-cli guardrail add       --rule "..." --why "..."
  just-harness-cli guardrail supersede --id G-xxxx
`,
	"trace": `Usage: just-harness-cli trace --summary "..." --outcome completed|partial|blocked|failed
                             [--story US-xxxx] [--evidence "..."] [--unverified "..."] [--agent name]

Records what was done, the proof, and what was NOT verified. HEAD commit is added automatically.
`,
	"query": `Usage:
  just-harness-cli query status
  just-harness-cli query stories|decisions|guardrails|traces [--status S] [--md]

'status' is the session start screen: active guardrails, open stories, items
needing attention, proposed decisions, and check problems. --md renders GFM.
For traces, --status filters on outcome.
`,
	"check": `Usage: just-harness-cli check

Validates every record: schema/enums, duplicate ids, implemented stories
without a matching pass or waiver (I1/I2), missing packet/doc/contract paths,
and dangling story/superseded_by references. Exit 1 if anything is wrong.
`,
}

func printHelp(w io.Writer, args []string) error {
	if len(args) == 0 {
		fmt.Fprint(w, helpTop)
		return nil
	}
	text, ok := helpCommands[args[0]]
	if !ok {
		return inputf("no help for %q", args[0])
	}
	fmt.Fprint(w, text)
	return nil
}
