# Flags & contracts

All flags must appear **before** the subcommand (standard Go `flag` parsing).

### -addr

`-addr string` (default `localhost:6379`) — the Redis address to connect to.

Use cases:

- Point at a local, staging, or production Redis without recompiling.

```sh
cache-cli -addr 127.0.0.1:6379 stats
cache-cli -addr prod-redis.internal:6379 get user:42
```

### -ns

`-ns string` (default empty) — a namespace prefix applied via
`cache.Namespaced`, identical to what your service applies, so keys line up
exactly.

Use cases:

- Inspect one service's keys on a shared Redis DB.
- Match the exact prefix your app uses so `get`/`keys` see the right keys.

```sh
cache-cli -ns svc:billing get user:42      # reads svc:billing:user:42
cache-cli -ns svc:billing keys ''          # only svc:billing keys
```

### -ttl

`-ttl duration` (default `0` = no expiry) — TTL for `set`. Accepts Go duration
syntax (`5m`, `90s`, `1h30m`).

Use cases:

- Seed a short-lived token or feature flag.
- Set a permanent key (omit the flag / `-ttl 0`).

```sh
cache-cli -ttl 90s set otp:42 123456
cache-cli set config:flag on              # no -ttl → never expires
```

### -json

`-json` — machine-readable JSON output instead of plain text. Applies to
`get`, `set`, `del`, `stats`, and `keys`.

Use cases:

- Pipe into `jq` in scripts and CI.
- Stable, parseable output for automation.

```sh
cache-cli -json get user:42 | jq -r .value
cache-cli -json keys 'user:' | jq 'length'
cache-cli -json stats | jq .Entries
```

### Exit codes

| Code | When |
|---|---|
| `0` | success — includes `del` of an absent key, `stats`, `help`/`-h`/`--help` |
| `1` | runtime/backend error, **or a `get` miss** (`cache.ErrNotFound`) |
| `2` | usage error — bad flags, no command, wrong arg count, unknown command |

Use cases:

- `set -e` scripts: a `get` miss fails the script (exit 1) so you can `||`
  handle it.
- Distinguish "missing key" (1) from "wrong invocation" (2) in automation.

```sh
cache-cli get maybe:absent
case $? in
  0) echo "present" ;;
  1) echo "missing or backend error" ;;
  2) echo "bad usage" ;;
esac
```

### Stream contract

stdout = machine-readable / value output (the value, JSON, key list, `OK`).
stderr = human messages (errors, `not found`, usage). This split means piping
stdout stays clean even on a miss.

Use cases:

- `cache-cli get k 2>/dev/null` to suppress the human "not found" while still
  getting the exit code.
- Redirect `2>>cli.log` to capture errors without polluting piped data.

```sh
value=$(cache-cli get user:42 2>/dev/null) || echo "lookup failed" >&2
```
