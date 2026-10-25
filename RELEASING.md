# Releasing GiaDB

Components are versioned and released independently. A spec release does not publish an SDK, and an SDK release does not publish the spec.

Do not publish on a push to `main`. Create a component-prefixed tag on the reviewed commit. The matching release workflow checks out that tag, tests it, and creates a GitHub Release only after those checks pass. `workflow_dispatch` runs the checks and does not publish.

No release tag is created by documentation or migration work. Publishing a package or creating a tag requires an explicit maintainer action.

## SemVer before 1.0

Before 1.0, a patch release is a non-breaking fix or clarification. A minor release is new or breaking behavior. Pre-1.0 compatibility is not guaranteed.

At 1.0 and later, use conventional SemVer: patch for compatible fixes, minor for compatible additions, and major for incompatible changes.

Never move, delete, or reuse a released tag or version. A correction is a new version.

## Tags

| Component | Tag | Version source | Distribution |
|---|---|---|---|
| Specification and conformance | `db/vX.Y.Z` | `version` in `db/contract/v0.1/manifest.json` | Git tag and GitHub Release. No package registry |
| Go SDK | `go/vX.Y.Z` | The tag. `go/go.mod` declares the module path, not the version | Go module `github.com/anwerj/gia-db/go` |
| Python SDK | `py/vX.Y.Z` | `py/pyproject.toml`, when that package exists | PyPI. Not implemented |
| MCP server | `mcp/vX.Y.Z` | `mcp/pyproject.toml`, when that package exists | PyPI. Not implemented |
| TypeScript SDK | `ts/vX.Y.Z` | `ts/package.json`, when that package exists | npm. Not implemented |

The current spec manifest version is `0.1.0`, so the matching spec tag is `db/v0.1.0`. The tag must match that field exactly. The workflow rejects a mismatch instead of rewriting the manifest.

The Go module path is `github.com/anwerj/gia-db/go`. A 2.0.0 module would use path `github.com/anwerj/gia-db/go/v2` and tag `go/v2.0.0`. Consumers resolve a Go release through the Go module proxy. The release workflow does not upload a Go artifact.

Schema `$id` values remain `https://github.com/anwerj/gia-db/contract/v0.1/...`. They were not rewritten when the contract files moved under `db/`, because rewriting them would change the manifest checksums.

## Maintainer procedure

1. Merge the reviewed change to the default branch.
2. Choose the component version using the policy above.
3. Update that component's version source when it has one, and update release notes when they exist.
4. Confirm the component CI and required conformance checks passed on that commit.
5. Create the tag on that commit, for example `git tag db/v0.1.0` or `git tag go/v0.1.0`.
6. Confirm the release workflow ran on the tag and passed.
7. Confirm the GitHub Release, or a clean consumer module for Go, resolves that version.
8. Update release notes after the artifact is verified.

## Registries that are not configured

Python and TypeScript are not published from this repository. There are no `release-py`, `release-mcp`, or `release-ts` workflows yet.

Before a first PyPI or npm release, a maintainer must check that the package name is available, configure trusted publishing for the exact repository and workflow filename, and use GitHub OIDC (`id-token: write`) on the publish job only. Do not put a long-lived registry token in pull-request workflows. `gia-mcp` must not be published until its `gia-py` dependency is a published version. Published Python metadata must not contain a local filesystem path.

## Migration choices recorded here

- The Go SDK was added under `go/` as new files in this repository.
- `py/`, `mcp/`, and `ts/` are not implemented. Their tag patterns above are reserved names, not release workflows.
- `db/SPEC.md` was moved and not edited. The identifiers `gia-db` and `gia-go` in the specification are unchanged. The Go package name remains `gia`.
