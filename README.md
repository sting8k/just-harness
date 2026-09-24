# just-harness

Shared project state for coding agents and people, in one binary.

`just-harness-cli` installs a small harness into any repository. After that, every agent session
starts from the same picture of the project: what is in progress, what is done and proven, and
which rules and decisions bind future work.

- **Shared:** state is JSON committed under `.harness/`, one file per record. It travels with git,
  merges across branches, and reads plainly in pull request diffs.
- **Progress:** `just-harness-cli query status` shows a new session the open work, the waived or
  stale proof, and the active guardrails.
- **Aligned:** guardrails, owning contracts, and a completion gate keep agents inside the user's
  scope. A story cannot be marked implemented until its verify command passes, unless the agent
  records an explicit waiver.

## Install

Download the binary for your platform and run it in the target repository.

```sh
# macOS (Apple silicon). Other targets: darwin-amd64, linux-amd64, linux-arm64
curl -fsSLo just-harness-cli https://github.com/sting8k/just-harness/releases/latest/download/just-harness-cli-darwin-arm64 \
  && chmod +x just-harness-cli && ./just-harness-cli install .
```

```powershell
# Windows
irm https://github.com/sting8k/just-harness/releases/latest/download/just-harness-cli-windows-amd64.exe -OutFile just-harness-cli.exe; .\just-harness-cli.exe install .
```

The installer copies itself to `.harness/bin/` (gitignored), so the downloaded file can be deleted.
Each release asset has a `.sha256` file next to it.

| Flag | Effect |
| --- | --- |
| `--dry-run` | Print the plan and write nothing. |
| `--merge` | Install into a repo that already has `AGENTS.md` or `docs/`: keep existing files, refresh the harness block, and upgrade the binary. Run it again later to upgrade. |
| `--override` | Move existing `AGENTS.md` and `docs/` to `.harness-backup/<timestamp>/` first. |
| `--force` | Overwrite conflicting payload files, backing them up first. |
| `--claude` | Also create `CLAUDE.md` that imports `AGENTS.md`. |

The installer does not modify existing records in `.harness/<kind>/`.

## What gets installed

```text
AGENTS.md                  harness block (between HARNESS:BEGIN/END markers)
docs/HARNESS.md            the operating policy agents follow
docs/templates/            story and decision templates
.harness/                  committed state, written only by the CLI
.gitignore, .gitattributes .harness/bin/, .harness-backup/ ignored; LF for .harness/
```

## Using it

```sh
H=.harness/bin/just-harness-cli
$H query status                                                  # start of every session
$H story add --title "Rate limit login" --lane high_risk --verify "go test ./auth/..."
$H story verify --id US-k3f9                                     # run the proof
$H story update --id US-k3f9 --status implemented                # refused until verify passes
$H decision add --title "Sessions use signed cookies"            # creates docs/decisions/D-….md
$H guardrail add --rule "No new runtime dependencies" --why "Single static binary"
$H check                                                         # validate all records (CI-friendly)
```

Run `just-harness-cli help <command>` for details. Commit `.harness/` together with the code it describes.

## Build from source

```sh
go test ./...
go build -o just-harness-cli ./cmd/just-harness-cli
```

Go 1.24+, standard library only, `CGO_ENABLED=0`.
The shipped payload lives in `harness/` and is embedded at build time.

## License

MIT. See [LICENSE](LICENSE).
