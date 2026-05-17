# Changelog

All notable changes to `github.com/ubgo/cache-cli` are documented here.
Format follows Keep a Changelog; the project follows SemVer (pre-GA in `v0.x`).

## [Unreleased]

### Added

- `cache-cli` inspector for a ubgo/cache Redis backend: `get`, `set`, `del`,
  `stats`, `keys`.
- Flags: `-addr`, `-ns`, `-ttl`, `-json`; `help` subcommand with examples.
- Scriptable exit codes (0 ok / 1 runtime-or-miss / 2 usage).
- Tested end-to-end in-process via miniredis (no Docker).

[Unreleased]: https://github.com/ubgo/cache-cli/commits/main
