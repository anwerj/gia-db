# GiaDB contributor instructions

This file governs how agents and contributors change the specification,
contract pack, and derived SDKs. It does not define GiaDB semantics.

## Mandatory monorepo policy

Before making any repository changes, read [`CONSTITUTION.md`](CONSTITUTION.md) and follow it.

`CONSTITUTION.md` defines mandatory rules for contract-first changes, data integrity, package isolation, conformance, CI, versioning, releases, and reporting. If instructions or repository contents conflict, do not guess: identify the conflict and report it before implementing behavior that would be inconsistent with the approved GiaDB specification.

The canonical contract is under `db/`. Read the relevant version of `db/SPEC.md` and its conformance assets before changing SDK or MCP behavior.

## Naming

- The project name MUST be written exactly as `GiaDB`.
- No alternative capitalization may be used as the project name.
- Existing repository and package identifiers such as `gia-db` and `gia-go`
  MUST retain their defined forms.

## Sources of truth

- `db/SPEC.md` is the sole normative source for GiaDB semantics.
- `db/contract/v0.1` is the machine-readable expression of `db/SPEC.md`; it MUST NOT
  add, remove, or reinterpret requirements.
- `README.md` is explanatory and non-normative. It MUST NOT override
  `db/SPEC.md`.
- If these files disagree, contributors MUST correct the contract or
  explanatory material to match `db/SPEC.md`; they MUST NOT create a second
  semantic rule here.

## Changing GiaDB

- A semantic change MUST update `db/SPEC.md` first.
- The same change MUST update every affected schema, example, conformance
  scenario, manifest checksum, and SDK.
- A documentation-only change MUST NOT silently alter contract behavior.
- Conformance fixtures MUST cover behavior that SDKs are required to share.
- Contract JSON and manifest checksums MUST be validated before submission.
- Derived SDK tests MUST pass before their contract version is declared
  supported.
