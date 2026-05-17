# Contributing to ubgo/cache-cli

`cache-cli` is a small `package main` binary. It is its own Go module that imports `github.com/ubgo/cache` and `github.com/ubgo/cache-redis`.

## Module layout

This is an independent module, but it is developed and released alongside the parent `ubgo/cache` family. It tracks the same `ubgo/cache` API version it ships with. Treat a `cache-cli` release as coupled to the `ubgo/cache` release it was built against.

## Local development setup

When developing against an unreleased `ubgo/cache` or `ubgo/cache-redis`, add a local `replace` directive so you build against your working tree:

```sh
go mod edit -replace github.com/ubgo/cache=../cache
go mod edit -replace github.com/ubgo/cache-redis=../cache-redis
go mod tidy
```

Do **not** commit machine-specific `replace` directives unless the release process expects them. Revert them before opening a PR if they were only for local iteration.

## Build / test / lint gate

Every change must pass the full gate before it is considered done:

```sh
gofmt -w .
go build ./...
go test -race -count=1 ./...
golangci-lint run ./...
```

All four must be clean: zero `gofmt` diff, zero build errors, zero test failures (race detector on), zero lint findings.

Tests use [`miniredis`](https://github.com/alicebob/miniredis) — no real Redis is required to run `go test`.

### Linter notes

`.golangci.yml` already excludes `errcheck` for `fmt.Fprintln` / `fmt.Fprintf` / `fmt.Fprint` (writes to stdout/stderr practically never fail; checking every one in a CLI is noise). Keep that exclusion — do not start checking those errors. The `revive` `unused-parameter: parameter 'ctx'` exclusion exists because `ctx` is part of the `cache.Cache` contract.

## Doc-comment style

- Every exported symbol has a doc comment that starts with the symbol name.
- The package doc comment (top of `main.go`) is a `Command cache-cli ...` summary with a couple of runnable example invocations.
- Inline comments explain **why**, not what — especially around exit-code choices and the stdout/stderr split (machine-readable data to stdout, human messages to stderr).
- Keep comments accurate if behavior changes; a stale comment is worse than none.

## Pull requests

- Keep the surface minimal. New subcommands or flags need a clear operator use case.
- Update `README.md` (usage table, examples, FAQ) and `CHANGELOG.md` in the same PR as a behavior change.
- Preserve the exit-code contract (`0` ok, `1` runtime/not-found, `2` usage) — scripts depend on it.
