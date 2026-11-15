# gia-mcp

`gia-mcp` is the GiaDB MCP adapter. It exposes the current GiaDB operations as MCP tools and delegates every operation to the Go SDK, `github.com/anwerj/gia-db/go`. It does not call GitHub itself and it does not define GiaDB semantics.

This adapter version is `0.0.1` in `gia/version.go`. It adapts specification 0.1. The first release is prepared by the `mcp/v0.0.1` workflow.

## Requirements

Go 1.26 or newer. Build it from this directory:

```sh
go build -o gia-mcp .
```

The module uses a local `replace` directive for the Go SDK. A published release must require a published SDK version instead of that path.

For the `mcp/v0.0.1` release, publish `go/v0.0.1` first, then replace the local
SDK dependency with `github.com/anwerj/gia-db/go v0.0.1` before creating the MCP
tag.

## Configuration

`gia-mcp` needs the repository that holds the database and a GitHub token.

| Setting | Environment |
|---|---|
| Owner | `GIADB_OWNER` |
| Repository | `GIADB_REPOSITORY` |
| Token | `GIADB_TOKEN`, or `GITHUB_TOKEN` when that is unset |

Owner, repository, and token are read only from the environment. They are not process arguments. A blank value counts as unset, and the process names each missing variable.

## Transports

Both transports serve the same registered tools.

```sh
gia-mcp
gia-mcp stdio
gia-mcp http --listen 127.0.0.1:8080
```

The default command and the `stdio` subcommand start stdio. MCP messages use
stdout. Diagnostics, including startup and MCP library errors, use stderr. The
process stops on stdin EOF, SIGINT, or SIGTERM.

HTTP uses Streamable HTTP at `/mcp`. The default listen address is `127.0.0.1:8080`, so the endpoint is `http://127.0.0.1:8080/mcp`. DNS-rebinding protection for loopback connections stays enabled. SIGINT and SIGTERM shut the server down.

A non-loopback address, including `0.0.0.0` and an empty host, is refused unless `--allow-non-loopback` is set. That flag exposes the process, and the GitHub token it holds, on the network. The HTTP endpoint does not add its own authentication. Bind loopback unless that exposure is intentional.

Browser CORS is disabled unless `--allow-origin` names an origin. Repeat the flag for more than one origin.

The process stays in the foreground. It does not install itself as a background service.

## Tools

| Tool | SDK operation |
|---|---|
| `create_resource` | Create |
| `get_resource` | Get |
| `update_resource` | Update |
| `list_resources` | List |
| `new_section` | NewSection |
| `list_sections` | ListSections |
| `forget_resource` | Forget |
| `reopen_resource` | Reopen |

`id` on resource tools is the GitHub issue number. `resource_id` on section tools is that same issue number. A section's returned `id` is the comment id. Issue and comment bodies are passed through unchanged. List results keep the SDK continuation token, exhausted flag, reported total, incomplete flag, and truncated flag.

`list_resources` supports `title_match` for title-only search and `text_match`
for title, body, and comment search through GitHub Search.

## Tests

From `mcp/`:

```sh
test -z "$(gofmt -l .)"
go vet ./...
go test -race ./...
```

Tests mock GitHub with fixtures and do not use a live token.
