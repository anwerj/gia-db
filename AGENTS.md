# GiaDB contributor instructions

This file governs how agents and contributors change the specification,
contract pack, and derived SDKs. It does not define GiaDB semantics.

## Naming

- The project name MUST be written exactly as `GiaDB`.
- No alternative capitalization may be used as the project name.
- Existing repository and package identifiers such as `gia-db` and `gia-go`
  MUST retain their defined forms.

## Sources of truth

- `SPEC.md` is the sole normative source for GiaDB semantics.
- `contract/v0.1` is the machine-readable expression of `SPEC.md`; it MUST NOT
  add, remove, or reinterpret requirements.
- `README.md` is explanatory and non-normative. It MUST NOT override
  `SPEC.md`.
- If these files disagree, contributors MUST correct the contract or
  explanatory material to match `SPEC.md`; they MUST NOT create a second
  semantic rule here.

## Changing GiaDB

- A semantic change MUST update `SPEC.md` first.
- The same change MUST update every affected schema, example, conformance
  scenario, manifest checksum, and SDK.
- A documentation-only change MUST NOT silently alter contract behavior.
- Conformance fixtures MUST cover behavior that SDKs are required to share.
- Contract JSON and manifest checksums MUST be validated before submission.
- Derived SDK tests MUST pass before their contract version is declared
  supported.
