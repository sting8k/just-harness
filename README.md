# just-harness

> Stub — rewritten in S6 (see SPEC-v1-go-engine §0, §5.8).

One Go binary, `just-harness-cli`, that installs a small agent harness into a
repo and keeps shared project state in committed JSON records under `.harness/`.

```sh
go build ./cmd/just-harness-cli && ./just-harness-cli install /path/to/repo
```
