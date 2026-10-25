# GiaDB Monorepo Constitution

**Status:** Normative repository policy for maintainers and coding agents.  
**Applies to:** The specification, conformance assets, every SDK, MCP adapter, tests, documentation, and CI/release automation in this repository.

Agents must read this file before making changes. The root `AGENTS.md` must explicitly direct agents to read and obey it.

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
7. A capability is not in GiaDB merely because GitHub exposes an endpoint. Relationships, reactions, label discovery, or other native GitHub capabilities may be added only when a defined GiaDB use case and a specification/conformance change justify them.

## Article 3 — Caller-owned content and data integrity

1. Issue and comment bodies are caller-owned payloads. Pass them through unchanged.
2. Never inject markers, metadata, wrappers, separators, or tracking text into issue/comment bodies.
3. Do not silently trim, normalize, reflow, re-encode, rewrite Markdown, or alter newlines/whitespace in caller-supplied bodies.
4. Preserve the distinction between an omitted field and a supplied empty string. Partial-update semantics must follow the specification; do not collapse omitted and empty values.
5. Tests must cover payload pass-through for representative edge cases: empty strings, leading/trailing whitespace, multiple newlines, Unicode, Markdown, JSON-looking text, and strings that resemble internal markers.
6. Creating or updating one resource/section must not accidentally mutate another resource, the parent issue, or another comment.
7. If a GitHub mutation may have succeeded but the response was lost, preserve the specified ambiguous-outcome behavior. Do not blindly retry a non-idempotent write unless the contract and implementation establish safe idempotency.
8. Never edit production/user data or attempt automatic data repair as part of ordinary code changes. Data recovery requires explicit scope and a verified recovery plan.

## Article 4 — Repository and component boundaries

The intended repository layout is:

- `db/`: specification, versioned contract/schema/conformance assets, spec tooling.
- `go/`: Go SDK, its module file, tests, and Go-specific docs.
- `py/`: Python SDK as an independently buildable Python distribution.
- `mcp/`: Python MCP adapter as an independently buildable distribution depending on `gia-py`.
- `ts/`: TypeScript SDK as an independently buildable npm package.
- `.github/workflows/`: CI and release workflows.
- root docs: project-level README, constitution, contribution and release guidance.

Rules:

1. Each implementation owns its language-specific dependencies, tests, package metadata, build, and release lifecycle.
2. Do not create a root-level build/dependency setup that forces all languages to use the same toolchain.
3. Go must build/test from `go/` with its own `go.mod`.
4. Python SDK and MCP must each have their own `pyproject.toml`, environment/dependency definition, and lockfile. Do not add every Python dependency to one global list.
5. During local development, MCP may use a local path override for the Python SDK. Published metadata must depend on a versioned package resolvable by users, not a filesystem path.
6. TypeScript must use its own npm-compatible metadata and lockfile. It is not required to participate in the Python environment.
7. No component may import another component's private source files or rely on another component's development environment. Cross-component coupling must happen through a declared public package dependency or the shared GiaDB contract.
8. Keep changes scoped. A Go change should not reformat or rewrite Python/TypeScript code; a documentation-only change should not trigger broad refactoring.
9. Preserve existing files and history unless a reviewed migration step explicitly moves them. Never overwrite an existing agent instruction file wholesale without inspecting and preserving its current content.

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
2. All database operations must delegate to the public `gia-py` SDK.
3. MCP code may define tool names, input/output schemas, argument validation required by the MCP protocol, and conversion of SDK results/errors to MCP responses.
4. MCP code must not call GitHub APIs directly, duplicate pagination, recreate SDK error classification, reinterpret resource state, rewrite bodies, or implement separate search behavior.
5. The initial transport is the approved Streamable HTTP design. Unless explicitly changed by a reviewed decision, do not add stdio or claim compatibility with clients that require stdio-spawned servers.
6. A local server must bind to loopback by default. Binding to non-loopback interfaces must require explicit configuration and be clearly documented with its security implications.
7. `uvx gia-mcp` should install/run the Python distribution in an isolated environment and must not require a Go compiler/runtime or a checked-out source repository.
8. The server is a long-running process. Document its host/port, endpoint URL, startup/shutdown behavior, and how clients connect. Do not claim it automatically starts as a background service unless that behavior is implemented.

## Article 7 — Python and TypeScript ecosystem rules

1. Use standard project metadata: `pyproject.toml` for each Python package and `package.json` plus one committed lockfile for TypeScript.
2. Use `uv` for Python environment/dependency/build operations as documented by the project. Do not introduce Poetry, Pipenv, or another package manager without a specific, approved reason.
3. Prefer a synchronous Python SDK for the initial version unless a concrete use case requires async support. Do not build both APIs merely for theoretical flexibility.
4. Keep dependencies minimal. Add a dependency only when it solves a concrete need; record the reason in the PR when it materially affects packaging, runtime, or security.
5. Public models should have explicit type annotations and stable semantics. Do not add a validation/modeling framework only for simple dataclasses unless the SDK contract requires it.
6. Verify PyPI/npm package-name availability before first publication. Distinguish a distribution name (for example, `gia-py`) from its Python import name (for example, `giadb`). Never assume those names are identical.
7. Keep development-only path sources out of built package metadata. Test the built distribution, not just imports from the repository working tree.

## Article 8 — Conformance and testing

1. Tests are evidence, not decoration. Every behavior change needs tests that would fail under the previous incorrect behavior.
2. When `db/` changes, run spec/schema validation and conformance against every implemented SDK. Do not release an SDK that fails the selected contract.
3. SDK tests must be deterministic and mock HTTP interactions by default. CI must not require personal tokens or live GitHub network access.
4. Live integration tests, if present, must be opt-in, clearly marked, safe to run, and excluded from ordinary PR CI unless explicitly configured.
5. Pagination tests must check the actual result sequence/IDs, duplicates, omissions, continuation-token resume behavior, final-page behavior, and truncation boundaries—not only helper return values or HTTP call counts.
6. Body preservation tests must compare exact strings, not trimmed or normalized forms.
7. Test errors and edge cases, including missing/unexpected API metadata, rate limits, permission failures, invalid continuation tokens, not-found resources, and ambiguous mutation outcomes where relevant to the contract.
8. A package must build from a clean checkout with only its declared dependencies installed. Do not rely on an agent's global Python packages, cached Go modules, local `.env`, or undocumented files.
9. Report the exact commands run and actual results. If a test was not run, say so. Never claim CI, lint, type checking, or release publishing succeeded without evidence.

## Article 9 — CI isolation and required checks

1. CI jobs must be component-isolated and follow the documented path/dependency matrix.
2. Changes under `db/` are cross-cutting and must trigger conformance validation for all implemented SDKs.
3. Changes under `py/` must trigger relevant `mcp/` integration checks because MCP depends on the Python SDK.
4. Changes to one implementation should not run unrelated language builds except where the dependency/contract matrix requires them.
5. Do not use workflow-level path filters that cause a required workflow to be absent and leave a required check pending. Prefer an always-triggered CI workflow with job-level path conditions and a stable final summary job.
6. The summary job must fail when any applicable job fails or is unexpectedly cancelled. It may treat a component job as skipped only when the path detector confirms that the component is unaffected.
7. Include `workflow_dispatch` for diagnosing CI when useful. If merge queues are enabled, add the required `merge_group` trigger and verify it.
8. Actions must use least-privilege permissions and reviewed, full-SHA-pinned third-party actions. Keep routine PR permissions read-only unless a specific, justified operation requires more.
9. Never make publishing credentials available to untrusted pull-request code.

## Article 10 — Versioning and releases

Use independently versioned components and component-prefixed tags:

- `db/vX.Y.Z`
- `go/vX.Y.Z`
- `py/vX.Y.Z`
- `mcp/vX.Y.Z`
- `ts/vX.Y.Z`

Rules:

1. A specification version and a package version are different concepts. An SDK release must state which spec version(s) it conforms to.
2. The tag must match the component package manifest/version source. Reject mismatches rather than repairing them silently in CI.
3. A release workflow tests and builds the exact tagged commit before publication. A green `main` branch is not proof that an arbitrary tag is releasable.
4. Release workflows publish only their own component. No all-packages release side effect is allowed.
5. Do not publish on ordinary pushes to `main`.
6. Use OIDC Trusted Publishing for PyPI/npm where supported, and grant `id-token: write` only to the publish job. Keep release environments and publisher configuration explicit.
7. A package version is immutable. Never force-move, delete, or reuse a released tag/version to patch it; publish a new version.
8. Before publishing, verify package names, version metadata, built artifact contents, and registry configuration.
9. Do not create release tags, publish packages, archive repositories, or change registry settings unless the user explicitly authorizes the release operation.
10. Release automation must fail closed if any prerequisite or version validation is unclear.

## Article 11 — Documentation truthfulness

1. The root README explains GiaDB's purpose, repository structure, contract-first approach, and links to each component.
2. Each package README documents installation, minimum runtime/toolchain, configuration, API examples, tests, and release status.
3. Clearly separate implemented, planned, and experimental features. Do not describe future operations as available.
4. Do not imply that GitHub's rate limits, search ceilings, permission model, or payload limits disappear behind GiaDB.
5. Do not describe GiaDB as free without qualification; GitHub plans, usage limits, and package registry policies apply.
6. Pseudocode must be labelled as pseudocode. Executable snippets must use real, tested method signatures.
7. Keep release and migration notes accurate. If a module path changes, document the new path and known compatibility impact.

## Article 12 — Agent workflow and reporting

Before editing, an agent must:

1. Read this constitution and the relevant root `AGENTS.md` instructions.
2. Inspect the current branch, working tree, package manifest, relevant spec/conformance files, tests, workflows, and open changes that affect the requested work.
3. Record the relevant paths and public APIs from the actual repository; do not assume file paths, branches, PR status, package versions, or CI status from memory.
4. Identify the exact behavior requested and map it to an existing spec clause or a proposed spec change.
5. State a bounded implementation plan before making a multi-file change.

During work, the agent must:

1. Make the smallest coherent change and avoid unrelated refactoring.
2. Preserve uncommitted/user changes. Never reset, clean, force-push, or overwrite files without explicit authorization.
3. Avoid broad textual replacements when a structured/code-aware edit is safer.
4. Verify package metadata and imports after moving directories or changing module paths.
5. Use actual test output as the source of truth. Do not infer a test passed because a similar test passed earlier.
6. Do not silently work around failing tests by weakening assertions, skipping cases, or changing conformance fixtures to match faulty code.
7. If scope expands or the contract is ambiguous, stop and report the specific blocker rather than guessing.

When reporting completion, the agent must include:

- Summary of behavior changed and why.
- Files/paths changed.
- Public API, schema, dependency, and release impact.
- Tests and commands actually run, with pass/fail results.
- Tests not run and why.
- Remaining risks, unresolved decisions, and any divergence from this constitution.
- Confirmation that no package was published and no release tag was created unless explicitly requested.

## Article 13 — Change to this constitution

This constitution may evolve, but changes must be reviewed as repository policy rather than slipped into an unrelated implementation diff. If a requested task conflicts with a constitutional rule, call out the conflict and propose an explicit amendment or user-approved exception. Do not silently ignore the rule.

---

**Core principle:** One specification, independently buildable implementations, shared conformance, explicit releases, and no invented behavior.
