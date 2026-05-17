# cache-cli — feature cookbook

Exhaustive, example-driven reference for the `cache-cli` command: every
subcommand, every flag, the exit-code and stream contracts, and scripting use
cases.

`cache-cli` is a scriptable inspector for a [`github.com/ubgo/cache`](https://github.com/ubgo/cache)
**Redis** backend. It connects with `cache-redis`, optionally applies a
`cache.Namespaced` prefix, and exits with meaningful codes.

Install / run:

```sh
go install github.com/ubgo/cache-cli@latest
cache-cli -addr localhost:6379 stats
```

## Pages

- [Commands](commands.md) — `get`, `set`, `del`, `stats`, `keys`, `help`.
- [Flags & contracts](flags.md) — `-addr`, `-ns`, `-ttl`, `-json`, exit codes, stdout/stderr split.

## Capability matrix

| Surface | Kind | Page |
|---|---|---|
| `get <key>` | command | [Commands](commands.md#get) |
| `set <key> <value>` | command | [Commands](commands.md#set) |
| `del <key>` | command | [Commands](commands.md#del) |
| `stats` | command | [Commands](commands.md#stats) |
| `keys [prefix]` | command | [Commands](commands.md#keys) |
| `help` / `-h` / `--help` | command | [Commands](commands.md#help) |
| `-addr string` | flag | [Flags](flags.md#-addr) |
| `-ns string` | flag | [Flags](flags.md#-ns) |
| `-ttl duration` | flag | [Flags](flags.md#-ttl) |
| `-json` | flag | [Flags](flags.md#-json) |
| exit codes 0 / 1 / 2 | contract | [Flags](flags.md#exit-codes) |
| stdout vs stderr | contract | [Flags](flags.md#stream-contract) |

## Exit-code contract (relied on by scripts — do not change)

| Code | Meaning |
|---|---|
| `0` | success (includes `del` of an absent key, `stats`, `help`) |
| `1` | runtime/backend error, **or a `get` miss** (`cache.ErrNotFound`) |
| `2` | usage error: bad flags, no command, wrong arg count, unknown command |
