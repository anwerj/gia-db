# GIADB Specification v0.1

Status: implementation contract.

The key words MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY are normative.

## 1. Scope

GIADB uses one configured GitHub repository as one database. GitHub is the
source of truth. An implementation MUST use GitHub application APIs and MUST
NOT require Git operations, a checkout, filesystem persistence, a cache, a
search index, a background worker, or MCP.

GIADB targets small collections, normally hundreds to a few thousand
resources. It does not provide SQL, joins, transactions, arbitrary indexes, or
high-throughput storage.

## 2. Identity and storage

A database is identified by GitHub host, repository owner, and repository
name. A resource identifier is the positive issue number scoped to that
database. GitHub database IDs and GraphQL node IDs are adapter details and MUST
NOT replace the issue number in the public contract.

| GitHub | GIADB |
|---|---|
| Repository | Database |
| Issue | Resource |
| Issue number | Resource ID |
| Title | Title |
| Body | Primary content |
| Labels | Classification and metadata |
| Issue comment | Section |
| Open issue | Active resource |
| Closed issue | Forgotten resource |

Pull requests are not resources. Adapters using REST Issues endpoints or search
MUST reject any item with a `pull_request` member. GraphQL adapters MUST
type-check search nodes and SHOULD use `Repository.issues` for enumeration.

## 3. Classification labels

`gia/` is reserved. Every resource MUST contain exactly one table label:

```text
gia/table/<name>
```

`<name>` MUST match `[a-z0-9][a-z0-9._-]{0,39}`. SDKs MUST reject an empty,
uppercase, malformed, or duplicate table classification before mutation.

An issue with no table label is an ordinary issue and MUST be omitted from
GIADB enumeration and search results. An issue with more than one distinct
table label is malformed and MUST produce `invalid_resource`; it MUST NOT be
silently selected as one table.

Other reserved labels use `gia/<key>/<value>`, for example
`gia/scope/personal` and `gia/availability/always`. v0.1 assigns no singularity
or query semantics to keys other than `table`. SDKs MUST expose and preserve
all labels, including labels outside `gia/`, unless the caller explicitly
replaces them.

Labels must already exist in GitHub when required by GitHub. Label provisioning
is an administrative operation outside v0.1 resource creation. Creation MUST
send title, body, and labels in one issue-creation request.

## 4. Resources and sections

The issue body is the primary content and MAY contain text, Markdown, JSON, or
a datatype chosen by the table.

Comments are first-class sections. A section has its own comment ID, body,
author, and timestamps. Reading a resource MUST NOT imply that any comments
were loaded. Section reads use a separate paginated operation.

An SDK MUST NOT split, concatenate, truncate, or reinterpret section bodies.
The server is authoritative for size limits. An oversized body or section MUST
produce `validation`.

Append means create one issue comment. It is append-oriented by convention,
not immutable: edits made through GitHub remain visible.

## 5. Operations

- **Create** creates one active issue with title, body, and labels.
- **Get** retrieves one issue by number, regardless of open or closed state.
- **Update** patches supported mutable issue fields. Omitted fields remain
  unchanged. Replacing labels is explicit.
- **NewSection** creates one comment.
- **ListSections** lists sections and supports pagination.
- **List** lists and searches resources. It supports pagination and filters
  including exact resource IDs, title matching, state, table, labels,
  timestamps, and supported sorting.
- **Forget** closes an issue.
- **Reopen** opens a closed issue.

List defaults to active resources. Closed or all-state enumeration requires an
explicit state filter. Exact ID lookup is the explicit historical read path.
There is no destroy operation.

Mutations MUST validate the resource returned by GitHub. A response that lost
the required table label, including because of permissions, is not success.

## 6. Query and pagination

`List` is the only collection/query operation. Queries map to GitHub-native
capabilities: exact issue numbers, title matching, state, table label,
additional labels, timestamps, supported sorting, and pagination.

Every page MUST contain an opaque continuation token or an explicit indication
that no next page exists. Tokens are adapter-owned, unstable across query
changes, and MUST NOT be interpreted by callers. A short or empty page can
still have a continuation token after non-resource issues are filtered.

When `List` uses GitHub issue search for matching, it MUST add repository and
issue-only constraints. Search-backed listing is discovery, not authoritative
enumeration. Results MUST report:

- GitHub's reported total when available;
- whether GitHub marked the response incomplete;
- whether more than GitHub's retrievable 1,000-result ceiling exists; and
- the continuation token, when another retrievable page exists.

Title matching through GitHub search does not guarantee exact substring
matching or read-after-write consistency. A database scan without text filters
uses repository issue enumeration. Enumeration does not provide snapshot
isolation while GitHub changes.

## 7. Consistency and concurrency

GIADB does not provide transactions or atomic multi-resource operations.
GitHub issue update endpoints do not offer a portable atomic expected-version
precondition. Shared mutable fields therefore use last-write-wins behavior.
Comparing `updated_at` before a mutation is advisory and MUST NOT be described
as conflict prevention.

Independent comment creation is preferred for concurrent append workloads.
The resource returned by a successful mutation is authoritative.

An implementation MUST NOT blindly retry a mutation after a transport failure
that may have occurred after GitHub accepted it. Such a failure is
`outcome_unknown`. Safe reads MAY be retried.

## 8. Errors

SDKs MUST provide stable error codes with transport details available
separately:

- `invalid_argument`
- `not_found`
- `authentication`
- `permission`
- `validation`
- `invalid_resource`
- `rate_limited`
- `not_modified`
- `transport`
- `github`
- `outcome_unknown`

Rate-limit errors SHOULD include retry time, limit, remaining requests, reset
time, HTTP status, and GitHub request ID when available. Failed requests MUST
NOT be converted into empty resources or complete empty pages.

## 9. API behavior and resilience

REST clients MUST send `Accept: application/vnd.github+json` and pin
`X-GitHub-Api-Version: 2026-03-10`. GraphQL clients MUST inspect both HTTP
status and GraphQL errors.

Clients MUST use bounded concurrency. They SHOULD serialize and pace
content-generating mutations. On primary or secondary throttling they MUST
honor `Retry-After`, then `X-RateLimit-Reset`; otherwise they SHOULD use bounded
exponential backoff with jitter according to GitHub guidance.

Automatic retries MUST be context-aware and bounded. v0.1 automatically
retries only safe reads. Conditional GET with `ETag`/`If-None-Match` SHOULD be
supported. A `304` without cached content is an explicit `not_modified` result,
not an empty successful resource.

Any cache is optional and disposable and MUST NOT be required for correctness.

## 10. SDK contract

Go, Python, and TypeScript SDKs MUST implement this logical model and pass the
versioned conformance scenarios. Public models MUST NOT expose a third-party
GitHub client's types.

Authentication MAY be supplied by a caller-owned HTTP client. Convenience
token configuration is allowed, but GitHub App installation token acquisition
is not part of the database contract.

An MCP server MAY later adapt an SDK. MCP MUST NOT define core semantics.
