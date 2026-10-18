<img src="assets/gia-db-black.png" alt="GiaDB logo" width="180">

# GiaDB

### What if your database was already sitting in GitHub?

**GiaDB — GitHub Issues As Database** — is a specification for building lightweight databases using GitHub Issues and the GitHub API.

No database server. No `git clone`. No commits, branches, or local storage to manage.

Just GitHub APIs.

## Why?

Most small projects don't need another database to operate. Agent skills, personal knowledge, reusable instructions, and small collections of structured data often need something much simpler.

Yet even a small database usually means provisioning storage, managing backups, setting up authentication, and building an interface to inspect and edit the data.

**GiaDB takes a different approach: use what GitHub already gives us.**

- **No database to host.** GitHub provides the persistence; your application talks to its APIs.
- **No separate admin UI to build.** Issues, labels, comments, search, and history are already available through GitHub's familiar interface.
- **Access control included.** Use GitHub's repository visibility and permission model to control who can access the data.
- **Low operational overhead.** No dedicated database server, cache, search cluster, or background worker is required by the core design.
- **Human-readable by default.** Inspect, edit, discuss, and audit resources directly on GitHub.
- **Cheap to experiment with.** Use an existing GitHub account and repository instead of paying for a separate database service.

## How it works

The repository is simply where the database lives. **The database itself is built on GitHub's Issues feature—not on Git, commits, or repository files.**

| GitHub | GiaDB |
|---|---|
| Repository | Database location |
| Issue | Resource / record |
| Issue number | Resource ID |
| Labels | Resource type and metadata |
| Title | Resource name |
| Body | Main content |
| Comments | Sections or additional information |
| Open / closed | Active / forgotten |

GiaDB defines the semantics. Implementations use GitHub APIs to work with the data.

## What is it for?

GiaDB is designed for small, resource-oriented workloads where simplicity and human visibility matter more than raw performance.

Think:

- Agent skills, personas, and reusable instructions
- Personal knowledge and learning notes
- Shared context across AI tools
- Small internal registries and catalogs
- Lightweight, structured data that humans should be able to inspect

Imagine giving Cursor, Claude, OpenCode, and other tools access to the same collection of skills and working preferences—without running another database just to store them.

That's the kind of problem GiaDB is meant to explore.

## What it isn't

GiaDB is **not a general-purpose database** and isn't intended to replace PostgreSQL, MySQL, DynamoDB, or similar systems.

GitHub API latency, rate limits, search behavior, and concurrency constraints are real. GiaDB deliberately accepts these trade-offs for workloads where they are reasonable.

No SQL compatibility, high-throughput guarantees, or ACID transactions are implied.

## Specification first

GiaDB is designed as a language-neutral specification. The goal is for different implementations to follow the same contract, with SDKs planned for:

- Go — `gia-go`
- Python — `gia-py`
- TypeScript — `gia-ts`

An MCP server can be built on top of an SDK, but **GiaDB itself is not an MCP server**.

## Status

GiaDB is an early-stage project. The specification and implementation contracts are still being developed.

The goal is simple:

**Make small databases so lightweight that the database infrastructure becomes almost an afterthought.**

[Explore the repository](https://github.com/anwerj/gia-db) · [Open an issue](https://github.com/anwerj/gia-db/issues)