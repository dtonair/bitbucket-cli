# AGENTS.md — bitbucket-cli

Standalone Go CLI for Bitbucket Cloud (PRs, branches, repo info). A command-line
port of the Pi `bitbucket` agent extension so any agent/script can drive Bitbucket
without the Pi runtime.

## Layout

| Path | Responsibility |
| --- | --- |
| `main.go` | Entry point → `cmd.Execute()` |
| `cmd/` | Cobra commands. One file per area: `pr.go` (read), `pr_comment.go` + `pr_write.go` (write), `branch.go`, `repo.go`, `status.go`, `config.go`, `root.go`. `common.go` holds shared helpers. |
| `bitbucket/client.go` | Thin REST 2.0 client: Basic auth, `Request`, `Paginate`, normalized `HTTPError`. |
| `config/config.go` | Config resolution: env → YAML file → git-remote auto-detect. |
| `output/` | `RenderJSON`/`RenderLines`/`WriteError` and `*Summary` text formatters. |

## Conventions

- **Output contract**: JSON on stdout by default; `--pretty` emits one-line text
  summaries (`output.*Summary`). Errors go to stderr as a JSON envelope, exit 1.
  Use `emitObject` / `emitList` (in `cmd/common.go`) so every command renders the
  same way.
- **Read vs write commands**: write commands (`pr comment`, `pr create`,
  `pr update`) must carry a `Long` description stating they are write operations to
  run only when the user explicitly asked. This is a deliberate safety marker for
  agents.
- **HTTP**: no per-command HTTP code. Call `client.Request(ctx, path,
  bitbucket.RequestOptions{Method, Body}, &out)` — it supports GET/POST/PUT with a
  JSON body and returns `*HTTPError` for non-2xx (surfaces method/url/status/excerpt).
- **Shared helpers** (`cmd/common.go`): `newClient`, `resolveRepo` (→ `/repositories/{ws}/{repo}` base path), `parseID`, `ctx`, `fail`, `emitObject`, `emitList`.
- **Body input** (`cmd/pr_write.go`): `readBody(text, file, stdin)` resolves a body
  from `--description` / `--description-file` (`-` = stdin); the two sources are
  mutually exclusive. Reuse it for any future text-body flag.

## PR update rule (important)

Bitbucket's `PUT /pullrequests/{id}` requires `title` and will **clear reviewers**
if they are omitted. `pr update` therefore does a **read-modify-write**: GET the PR,
carry over `title` (unless changed) and `reviewers`, then PUT only with the changed
fields added. Preserve this behavior for any future PR mutation.

## Testing

- `go test ./...`. Tests stub HTTP via `testTransport` (a `roundTripFunc`) injected
  by the `run(t, transport, args...)` helper in `cmd/commands_test.go`.
- `run` resets global + per-command flag state (`resetFlags`) because cobra reuses
  the `rootCmd` singleton across `Execute()` calls — flag values and `Changed`
  markers leak between tests otherwise.
- **Known gotcha**: `TestStatusMissingCredsFails` reads the developer's real
  `~/.config/bitbucket-cli.yaml`; it fails locally when that file has an `email`.
  The test should be sandboxed to a temp `BITBUCKET_CONFIG` (not yet done).

## Build

`go build -o bitbucket-cli .` or `go install .`. Deps: `spf13/cobra`,
`gopkg.in/yaml.v3` only. Release via goreleaser (`.goreleaser.yaml`).
