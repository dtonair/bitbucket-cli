# bitbucket-cli

A standalone Go CLI for Bitbucket Cloud pull requests, branches, and repository
info. It is the command-line port of the Pi `bitbucket` agent extension, built
so that any agent or script can drive Bitbucket without the Pi runtime.

Output is **JSON by default** (easy for agents to parse); pass `--pretty` for
human-readable text. Errors are written as a JSON envelope to stderr with a
non-zero exit code.

## Install / Build

```bash
cd ~/code/bitbucket-cli
go build -o bitbucket-cli .      # local binary
# or
go install .                     # into $GOBIN
```

Requires Go 1.24+. The only third-party dependency is `spf13/cobra`.

## Configuration

Credentials and defaults are read from environment variables only (no config
file):

```bash
export BITBUCKET_EMAIL="you@example.com"
export BITBUCKET_API_TOKEN="your-atlassian-api-token"
export BITBUCKET_DEFAULT_WORKSPACE="workspace-slug"   # optional
export BITBUCKET_DEFAULT_REPO="repository-slug"        # optional
```

The token must be an Atlassian API token with access to the target workspace.
Recommended scopes: `read:repository:bitbucket`, `read:pullrequest:bitbucket`,
and `write:pullrequest:bitbucket` to post PR comments.

If `BITBUCKET_DEFAULT_WORKSPACE`/`BITBUCKET_DEFAULT_REPO` are unset and you run
inside a git repository whose `origin` points at `bitbucket.org`, the workspace
and repo are auto-detected. Override per command with `--workspace`/`--repo`.

## Commands

| Command | Description |
| --- | --- |
| `status` | Report config validity and default repo |
| `repo get` | Repository details |
| `pr list [--state OPEN\|MERGED\|DECLINED\|SUPERSEDED] [--limit N]` | List pull requests |
| `pr get <id>` | One pull request |
| `pr comments <id> [--limit N]` | Pull request comments |
| `pr commits <id> [--limit N]` | Pull request commits |
| `pr comment <id> --body <markdown>` | Post a markdown comment (write) |
| `branch list [--query <q>] [--limit N]` | List branches |

Global flags: `--workspace`, `--repo`, `--pretty`. Default `--limit` is 20.

## Examples

```bash
# JSON (default) — for agents
bitbucket-cli pr list --state OPEN
bitbucket-cli pr get 123
bitbucket-cli branch list --query 'name ~ "feature/"'

# Human-readable
bitbucket-cli pr list --pretty
# -> #123 Fix login bug [OPEN]

# Post a comment (only run when explicitly asked to post)
bitbucket-cli pr comment 123 --body "Thanks, I will take a look."

# Override the target repo
bitbucket-cli --workspace acme --repo web pr list
```

## Output contract

- **Default**: JSON on stdout. List commands emit a JSON array of the full
  Bitbucket objects; single-entity commands emit the entity object.
- **`--pretty`**: one-line text summaries (matching the original extension).
- **Errors**: JSON `{"error":{"message":...,"status":...,"method":...,"url":...,"excerpt":...}}`
  on stderr, exit code 1. Config/usage errors carry only `message`.

## Security

Credentials are read from environment variables only and are never logged or
echoed. Do not commit tokens or `.env` files.

## Testing

```bash
go test ./...
```
