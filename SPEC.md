# GiaDB Specification v0.1

Status: implementation contract.

The key words MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY are normative.

The project name MUST be written exactly as `GiaDB` in prose, documentation,
and user-facing text. No alternative capitalization may be used as the project
name. Existing repository and package identifiers such as `gia-db` and
`gia-go` retain their defined forms.

## 1. Scope

GiaDB uses one configured GitHub repository as one database. GitHub is the
source of truth. An implementation MUST use GitHub application APIs and MUST
NOT require Git operations, a checkout, filesystem persistence, a cache, a
search index, a background worker, or MCP.

GiaDB targets small collections, normally hundreds to a few thousand
resources. It does not provide SQL, joins, transactions, arbitrary indexes, or
high-throughput storage.

## 2. Identity and storage

A database is identified by GitHub host, repository owner, and repository
name. A resource identifier is the positive issue number scoped to that
database. GitHub database IDs and GraphQL node IDs are adapter details and MUST
NOT replace the issue number in the public contract.

| GitHub | GiaDB |
|---|---|
| Repository | Database |
| Issue | Resource |
| Issue number | Resource ID |
| Title | Title |
| Body | Primary content |
| Labels | Optional metadata |
| Issue comment | Section |
| Open issue | Active resource |
| Closed issue | Forgotten resource |

Pull requests are not resources. Adapters using REST Issues endpoints or search
MUST reject any item with a `pull_request` member. GraphQL adapters MUST
type-check search nodes and SHOULD use `Repository.issues` for enumeration.

## 3. Resources and bodies

Every issue in the configured repository is a GiaDB resource, except a pull
request. No title, body, comment, or label identifies a resource.

The issue body and every comment body are caller-defined. SDKs MUST send and
return those bytes unchanged. An empty body is valid and MUST stay empty. An
SDK MUST NOT insert, remove, wrap, or rewrite a prefix, suffix, comment, or
other hidden content in a body.

An update that omits the body MUST leave the stored issue body unchanged and
MUST NOT send a body field. Changing labels MUST NOT change the body.

## 4. Labels

Labels are optional metadata chosen by the caller. They do not identify a
resource, assign it a type, or change its ID. No label text or prefix is
reserved. A resource MAY have no labels.

Supplying `labels` to Update replaces the complete label set. Omitting `labels`
leaves the current labels unchanged.

Label provisioning remains an administrative GitHub operation. Creation sends
the title, raw body, and any caller-supplied labels in one issue-creation
request.

## 5. Sections

The issue body is the primary content. It MAY contain text, Markdown, JSON, or
another format selected by the caller.

Comments are first-class sections. A section has its own comment ID, body,
author, and timestamps. Reading a resource MUST NOT imply that any comments
were loaded. Section reads use a separate paginated operation.

An SDK MUST NOT split, concatenate, truncate, or reinterpret section bodies.
The server is authoritative for size limits. An oversized body or section MUST
produce `validation`.

Append means create one issue comment. It is append-oriented by convention,
not immutable: edits made through GitHub remain visible.

## 6. Operations

- **Create** creates one active issue from a title, body, and optional labels.
  The body is stored exactly as supplied.
- **Get** retrieves one issue by number, regardless of active or forgotten
  state. A pull request is `not_found`.
- **Update** patches supported mutable issue fields. Omitted fields remain
  unchanged. A supplied body replaces the stored body exactly. A supplied
  label set replaces the complete label set.
- **NewSection** creates one comment.
- **ListSections** lists sections and supports pagination.
- **List** lists and searches resources. It supports pagination and filters
  for exact resource IDs, title matching, state, labels, timestamps, and
  supported sorting.
- **Forget** closes an issue without sending a body or changing labels.
- **Reopen** opens a closed issue without sending a body or changing labels.

Before Update, NewSection, ListSections, Forget, or Reopen, an SDK MUST read
the target and MUST reject a pull request with `not_found` before performing
that operation. That read checks only whether the target is a pull request. It
MUST NOT modify the body. The read and the following GitHub request are
separate and are not atomic. Forget and Reopen MUST NOT send a body.
Forgetting does not destroy historical content.

There is no destroy operation. List defaults to active resources. The public
state filters are `active`, `forgotten`, and `all`. Adapters map `active` and
`forgotten` to GitHub's open and closed states internally. Forgotten or
all-state enumeration requires an explicit state filter. Exact ID lookup is the
explicit historical read path.

## 7. Query and pagination

`List` is the only collection/query operation. Queries map to GitHub-native
capabilities: exact issue numbers, title matching, state, labels, timestamps,
supported sorting, and pagination.

Resource identity does not depend on body content. Search-backed filters use
GitHub search, after which pull requests MUST be rejected. A label or body
MUST NOT be used to decide whether an issue is a resource.

Every page MUST contain an opaque continuation token or an explicit indication
that no next page exists. Tokens are adapter-owned, unstable across query
changes, and MUST NOT be interpreted by callers. A short or empty page can
still have a continuation token after pull requests are filtered.

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

Body and comment-text matching are intentionally deferred from v0.1. SDKs MUST
NOT expose them as portable v0.1 query capabilities.

## 8. Consistency and concurrency

GiaDB does not provide transactions or atomic multi-resource operations.
GitHub issue update endpoints do not offer a portable atomic expected-version
precondition. Shared mutable fields therefore use last-write-wins behavior.
Comparing `updated_at` before a mutation is advisory and MUST NOT be described
as conflict prevention.

Independent comment creation is preferred for concurrent append workloads.
The resource returned by a successful mutation is authoritative.

A mutation transport failure that may have occurred after GitHub accepted the
request MUST produce `outcome_unknown`. It MUST NOT be represented as
`not_applied`.

## 9. Errors

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

## 10. SDK implementation boundary

REST clients MUST send `Accept: application/vnd.github+json` and pin
`X-GitHub-Api-Version: 2026-03-10`. GraphQL clients MUST inspect both HTTP
status and GraphQL errors.

A GiaDB SDK MUST provide a thin interface over GitHub APIs. It MUST implement
the resource, section, lifecycle, listing, and pagination semantics defined by
this specification. Issue and comment bodies MUST pass through unchanged.

The core SDK MUST NOT automatically retry, delay, queue, throttle, pace, or
serialize requests. It MUST NOT impose a caller concurrency limit. This applies
to both reads and mutations.

When GitHub rejects or throttles a request, the SDK MUST return a structured
error instead of waiting or retrying. The error SHOULD expose the HTTP status,
GitHub request ID, `Retry-After`, `X-RateLimit-Remaining`, and
`X-RateLimit-Reset` values when available. The caller decides whether and when
to retry.

Pagination MUST be explicit. A paginated operation returns one page and an
opaque continuation token. The SDK MUST NOT automatically load every page.

An operation MAY perform an additional GitHub read when required to reject a
pull request before mutation or before returning its sections. Such a read is
a correctness check, not caching or synchronization.

Conditional requests MAY be supported when explicitly requested by the
caller. A `304` without cached content is an explicit `not_modified` result,
not an empty successful resource.

No cache is required. If an implementation offers caching outside the core
SDK, that cache MUST be optional and disposable and MUST NOT become a source of
truth.

## 11. SDK contract

Go, Python, and TypeScript SDKs MUST implement this logical model and pass the
versioned conformance scenarios. Public models MUST NOT expose a third-party
GitHub client's types.

Authentication MAY be supplied by a caller-owned HTTP client. Convenience
token configuration is allowed, but GitHub App installation token acquisition
is not part of the database contract.

An MCP server MAY later adapt an SDK. MCP MUST NOT define core semantics.
