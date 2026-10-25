# GiaDB

<img src="assets/gia-db-black.png" alt="GiaDB logo" width="180" align="left">

### What if your database was already sitting in GitHub?

GiaDB (GitHub Issues As Database) is a lightweight database abstraction that uses GitHub Issues for storage and the GitHub API for access and queries.

<br clear="left">

No database server. No `git clone`. No commits, branches, or local storage to manage.

Just GitHub APIs.

## Why?

Most small projects don't need another database to operate. Agent skills, personal knowledge, reusable instructions, and small collections of structured data often need something much simpler.

Yet even a small database usually means provisioning storage, managing backups, setting up authentication, and building an interface to inspect and edit the data.

**GiaDB takes a different approach: use what GitHub already gives us.**

- **No database to host.** GitHub provides the persistence; applications access the data through its APIs.
- **No separate admin UI to build.** Issues, labels, comments, and history are already available through GitHub's familiar interface.
- **Access control included.** Use GitHub's repository visibility and permission model to control access.
- **Low operational overhead.** The core design requires no dedicated database server, cache, search cluster, or background worker.
- **Human-readable by default.** Inspect, edit, discuss, and audit resources directly on GitHub.
- **Cheap to experiment with.** Start with an existing GitHub account and repository instead of provisioning a separate database service. Applicable GitHub limits and usage policies still apply.

## How it works

A repository scopes the database. **Resources are stored in GitHub Issues and accessed through GitHub APIs—not through Git, commits, or repository files.**

Every non-pull-request issue in the repository is a GiaDB resource. Labels are optional metadata. Issue and comment bodies are passed through unchanged, without GiaDB injecting or rewriting content.

| GitHub | GiaDB |
|---|---|
| Repository | Database location |
| Issue | Resource / record |
| Issue number | Resource ID |
| Title | Resource name |
| Labels | Optional metadata |
| Body | Main content |
| Comments | Sections or additional information |
| Open / closed | Active / forgotten |

GiaDB defines the semantics. Implementations use GitHub APIs to work with the data.

## What is it for?

GiaDB is designed for small, resource-oriented workloads where simplicity and human visibility matter more than raw performance.

Examples include:

- Agent skills, personas, and reusable instructions
- Personal knowledge and learning notes
- Shared context across AI tools
- Small internal registries and catalogs
- Lightweight structured data that humans should be able to inspect

Imagine giving Cursor, Claude, OpenCode, and other tools access to the same collection of skills and working preferences—without running another database just to store them.

That's the kind of problem GiaDB is meant to explore.

## API at a glance

GiaDB is defined by a language-neutral specification. The goal is for different implementations to follow the same contract.

- **Go:** [`gia-go`](https://github.com/anwerj/gia-go) — reference implementation
- **Python:** `gia-py` — planned
- **TypeScript:** `gia-ts` — planned

The following is illustrative pseudocode, not a language-specific API signature.

```text
db = GiaDB(repository, credentials)

resource = db.Create(title, body, labels?)
resource = db.Get(id)
resource = db.Update(id, fields)

page = db.List(filters, pageSize, continuationToken?)

section = db.NewSection(id, body)
page = db.ListSections(id, pageSize, continuationToken?)

db.Forget(id)
db.Reopen(id)
```

## What it isn't

GiaDB is **not a general-purpose database** and isn't intended to replace PostgreSQL, MySQL, DynamoDB, or similar systems.

GitHub API latency, rate limits, search behavior, and concurrency constraints are real. GiaDB deliberately accepts these trade-offs for workloads where they are reasonable.

No SQL compatibility, high-throughput guarantees, or ACID transactions are implied.

## Status and future scope

GiaDB starts with a small set of operations over GitHub Issues. Future extensions may expose additional native GitHub capabilities without introducing a separate data model or infrastructure.

Planned extensions include:

- **Issue relationships:** Create, list, and remove relationships between issues, supporting the relationship types exposed by GitHub.
- **Repository label discovery:** List available repository labels so clients can discover and use existing metadata.
- **Additional SDKs:** Implement the specification in more languages while keeping behavior consistent across implementations.

These are planned extensions, not part of the current core API unless explicitly defined by the specification.

GiaDB will remain a thin interface over GitHub. Future additions should preserve caller-supplied issue and comment bodies, avoid unnecessary abstractions, and rely on native GitHub capabilities wherever possible.

---

[Explore the repository](https://github.com/anwerj/gia-db) · [Open an issue](https://github.com/anwerj/gia-db/issues)