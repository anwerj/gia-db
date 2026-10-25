# Contributing to GiaDB

Read [`CONSTITUTION.md`](CONSTITUTION.md) before changing this repository. [`AGENTS.md`](AGENTS.md) is the short contributor checklist. `db/SPEC.md` is the specification; this file does not change it.

## Checks

Install the contract validator dependencies and run it from the repository root:

```sh
python3 -m pip install -r db/tools/requirements.txt
python3 db/tools/validate_contract.py
```

Go checks run from `go/`:

```sh
cd go
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
| `db/**` | Contract validator, then Go formatting, `go vet`, and `go test -race ./...` |
| `go/**` | Go formatting, `go vet`, and `go test -race ./...` |
| `.github/workflows/**` | Contract validator and the Go checks |
| Root docs only (`README.md`, `AGENTS.md`, `CONSTITUTION.md`, `CONTRIBUTING.md`, `RELEASING.md`) | Review the docs against `db/SPEC.md`. Component builds are not required |

Python, MCP, and TypeScript are not in this repository yet. When they are added, a `py/` change must also run the MCP checks, and a `db/` change must run conformance for every implemented SDK.

A behavior change starts in `db/SPEC.md` and `db/contract/`. Do not weaken conformance fixtures so an implementation passes.
