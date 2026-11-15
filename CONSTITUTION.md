# GiaDB contract constitution

**Status:** Normative policy for the GiaDB specification and implementations.

This constitution defines how implementations stay aligned with the approved
contract. It does not define GiaDB semantics; `db/SPEC.md` does that.

## Article 1 — Contract and authority

1. `db/SPEC.md` and the versioned contract/conformance assets under `db/` define GiaDB behavior. Implementations must conform to them; they must not independently redefine the product.
2. The specification, conformance fixtures, SDKs, and docs must not silently disagree. If they disagree, identify the exact conflicting statements/files and stop the affected implementation work until the contract is resolved.
3. A coding agent must not invent a behavior, method, error code, schema field, endpoint, or compatibility promise to fill a gap. If the contract is ambiguous, report the ambiguity and propose a spec change separately from implementation.
4. Conformance assets are executable contract material. Do not casually rename, delete, weaken, or regenerate them to make a failing implementation pass.
5. A request to add or change behavior starts with the spec and conformance contract. Update those first, then SDK implementations and their tests. A path-only move must not also change semantics unless each change is called out and reviewed separately.
6. The project plan, README, examples, and implementation comments are subordinate to the approved specification. When documentation becomes stale, correct it; do not treat stale text as authority.

## Article 2 — Product identity and scope

1. GiaDB is a language-neutral specification and a family of SDKs over GitHub Issues. It is not a general-purpose database server and does not replace PostgreSQL, MySQL, DynamoDB, or similar systems.
2. The configured GitHub repository defines the database boundary.
3. According to the approved GiaDB contract, every non-pull-request issue in that repository is a resource. An SDK must not require an injected body marker, a reserved label, or a title prefix to recognize a resource.
4. Issue number is the repository-scoped resource ID. Do not confuse it with GitHub's internal database ID or a comment ID.
5. Pull requests must be excluded from resource operations, even when GitHub returns them in an issues/search result. Use the authoritative API fields to detect pull requests; do not rely on title conventions.
6. Labels are optional GitHub metadata. They are not resource identity, mandatory type declarations, or an implicit table system unless a future approved specification explicitly changes the contract.

## Article 3 — Caller-owned content and data integrity

1. Issue and comment bodies are caller-owned payloads. Pass them through unchanged.
2. Never inject markers, metadata, wrappers, separators, or tracking text into issue/comment bodies.
3. Do not silently trim, normalize, reflow, re-encode, rewrite Markdown, or alter newlines/whitespace in caller-supplied bodies.
4. Preserve the distinction between an omitted field and a supplied empty string. Partial-update semantics must follow the specification; do not collapse omitted and empty values.
5. Tests must cover payload pass-through for representative edge cases: empty strings, leading/trailing whitespace, multiple newlines, Unicode, Markdown, JSON-looking text, and strings that resemble internal markers.
6. Creating or updating one resource/section must not accidentally mutate another resource, the parent issue, or another comment.
7. If a GitHub mutation may have succeeded but the response was lost, preserve the specified ambiguous-outcome behavior. Do not blindly retry a non-idempotent write unless the contract and implementation establish safe idempotency.
8. Never edit production/user data or attempt automatic data repair as part of ordinary code changes. Data recovery requires explicit scope and a verified recovery plan.

## Article 4 — Component boundaries

Each SDK owns its language-specific dependencies, public models, tests, and
build. Components may depend on another component only through its declared
public package or the shared GiaDB contract. The MCP adapter delegates all
database behavior to the public Go SDK and must not become a second
implementation.

## Article 5 — SDK responsibilities

1. Each SDK implements the same public GiaDB contract in its language's normal conventions. Language idioms may differ; GiaDB semantics must not.
2. Each SDK owns the translation from its public API to GitHub API calls, response models, pagination, and GiaDB error semantics for that language.
3. Public APIs return GiaDB-owned models and errors rather than leaking raw third-party GitHub SDK objects unless the specification deliberately requires otherwise.
4. Follow the contract for pagination, opaque continuation tokens, search result ceilings, truncation indicators, pull-request exclusion, omitted fields, update semantics, and error mapping.
5. Do not add automatic retries, request pacing, in-memory caching, local persistence, background workers, or altered concurrency behavior without a spec-approved requirement. Such behavior affects observable semantics and operational safety.
6. Do not add a convenience method only because the underlying GitHub endpoint exists. Keep the API minimal and use-case-driven.
7. SDK-specific README examples must compile/run against the actual released API. Do not use pseudocode that appears to be a real method signature without labeling it illustrative.

## Article 6 — MCP adapter boundary

1. `gia-mcp` is an adapter, not a second GiaDB implementation.
2. All database operations must delegate to the public Go SDK `github.com/anwerj/gia-db/go`.
3. MCP code may define tool names, input/output schemas, argument validation required by the MCP protocol, and conversion of SDK results/errors to MCP responses.
4. MCP code must not call GitHub APIs directly, duplicate pagination, recreate SDK error classification, reinterpret resource state, rewrite bodies, or implement separate search behavior.
5. The transports are stdio and Streamable HTTP. Both must serve the same registered MCP server. Do not implement MCP transport protocol handling in this repository.
6. The HTTP server must bind to loopback by default. A non-loopback listen address requires the explicit `--allow-non-loopback` flag and must be documented with its security implications. Leave DNS-rebinding protection enabled for loopback connections.
7. `gia-mcp` is the Go binary built from `mcp/`. It must not require a Python or JavaScript runtime, and this adapter must not add npm or PyPI launchers.
8. The server is a long-running process. Document its host/port, endpoint URL, startup/shutdown behavior, and how clients connect. Do not claim it automatically starts as a background service unless that behavior is implemented.
9. Stdio protocol messages go to stdout. Diagnostics go to stderr.

## Article 7 — Conformance and testing

1. Tests are evidence, not decoration. Every behavior change needs tests that would fail under the previous incorrect behavior.
2. When `db/` changes, run spec/schema validation and conformance against every implemented SDK. Do not release an SDK that fails the selected contract.
3. SDK tests must be deterministic and mock HTTP interactions by default. CI must not require personal tokens or live GitHub network access.
4. Live integration tests, if present, must be opt-in, clearly marked, safe to run, and excluded from ordinary PR CI unless explicitly configured.
5. Pagination tests must check the actual result sequence/IDs, duplicates, omissions, continuation-token resume behavior, final-page behavior, and truncation boundaries—not only helper return values or HTTP call counts.
6. Body preservation tests must compare exact strings, not trimmed or normalized forms.
7. Test errors and edge cases, including missing/unexpected API metadata, rate limits, permission failures, invalid continuation tokens, not-found resources, and ambiguous mutation outcomes where relevant to the contract.
8. A package must build from a clean checkout with only its declared dependencies installed. Do not rely on an agent's global Python packages, cached Go modules, local `.env`, or undocumented files.
9. Report the exact commands run and actual results. If a test was not run, say so. Never claim CI, lint, type checking, or release publishing succeeded without evidence.
