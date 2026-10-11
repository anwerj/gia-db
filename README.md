# GIADB

GIADB (GitHub Issues As Database) is a small database abstraction that uses a
GitHub repository as a database and issues as resources.

This repository is the language-neutral source for GIADB semantics. It contains:

- [`SPEC.md`](SPEC.md): the normative v0.1 contract.
- [`AGENTS.md`](AGENTS.md): implementation constraints for coding agents.
- [`contract/v0.1`](contract/v0.1): schemas and conformance scenarios shared by
  every SDK.

SDKs are separate projects. The v0.1 reference implementation is
[`github.com/anwerj/gia-go`](https://github.com/anwerj/gia-go).

GIADB uses GitHub APIs only. It does not clone a repository, persist a local
database, or require MCP.
