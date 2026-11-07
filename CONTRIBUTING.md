# Contributing to GiaDB

Read [`CONSTITUTION.md`](CONSTITUTION.md) before changing this repository. [`AGENTS.md`](AGENTS.md) is the short contributor checklist. `db/SPEC.md` is the specification; this file does not change it.

## Checks

Install the contract validator dependencies and run it from the repository root:

```sh
python3 -m pip install -r db/tools/requirements.txt
python3 db/tools/validate_contract.py
```

Go SDK checks run from `go/`:

```sh
cd go
test -z "$(gofmt -l .)"
go vet ./...
go test -race ./...
```

MCP checks run from `mcp/`:

```sh
cd mcp
test -z "$(gofmt -l .)"
go vet ./...
go test -race ./...
```

Ordinary tests use fixtures. They do not need a GitHub token or network access to GitHub.

Live tests are opt-in and are not part of pull-request CI. From `go/`:

```sh
GIADB_TEST_TOKEN=... GIADB_TEST_OWNER=... GIADB_TEST_REPOSITORY=... \
  go test -tags=live -run TestLiveList
```

## What a change must run

| Changed paths | Required work |
|---|---|
| `db/**` | Contract validator, then the Go SDK checks |
| `go/**` | Go SDK checks and MCP checks |
| `mcp/**` | MCP checks |
| `.github/workflows/**` | Contract validator, Go SDK checks, and MCP checks |
| Root docs only (`README.md`, `AGENTS.md`, `CONSTITUTION.md`, `CONTRIBUTING.md`, `RELEASING.md`) | Review the docs against `db/SPEC.md`. Component builds are not required |

Python and TypeScript are not in this repository yet. A `db/` change must run conformance for every implemented SDK. The MCP adapter is checked with the Go SDK because it depends on that module.

A behavior change starts in `db/SPEC.md` and `db/contract/`. Do not weaken conformance fixtures so an implementation passes.
