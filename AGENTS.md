# GIADB implementation rules

This file applies to the specification, conformance contract, and SDK work
derived from this repository.

## MUST

- Treat `SPEC.md` and the versioned contract pack as normative.
- Treat GitHub as the only source of truth.
- Identify resources by repository-scoped issue number.
- require exactly one valid `gia/table/<name>` label on every resource.
- Preserve labels that a mutation does not explicitly replace.
- Exclude pull requests and ordinary issues from resource results.
- Keep body and comment/section reads distinct.
- Mark every paged result as continued or exhausted.
- Keep `List` as the only resource collection/query operation.
- Expose truncation and incomplete results when `List` uses GitHub search.
- Return structured, stable error codes.
- Honor GitHub throttling headers and bound retries and concurrency.
- Validate GitHub's returned representation after each mutation.
- Keep public SDK contracts independent of MCP and GitHub client libraries.
- Add or update conformance scenarios when semantics change.

## SHOULD

- Prefer GitHub-native REST and GraphQL capabilities.
- Prefer append-oriented comments for concurrent additions.
- Pace content-generating mutations.
- Retry safe reads only.
- Support caller-provided HTTP clients and test endpoints.
- Keep optional caches disposable.
- Pin API versions and SDK dependencies.

## MUST NOT

- Use Git operations, clones, worktrees, or repository files as database
  persistence.
- Add a required local database, cache, index, worker, or sync engine.
- Treat a pull request as a resource.
- Pick one of multiple table labels silently.
- Concatenate, split, or truncate comments automatically.
- Promise search consistency, transactions, or atomic optimistic updates.
- Blindly retry creates, comments, or other ambiguous mutations.
- Hide GitHub validation, permissions, throttling, or GraphQL errors.
- Let MCP constrain the SDK contract.
- change v0.1 semantics without updating `SPEC.md`, the contract manifest, and
  conformance scenarios together.
