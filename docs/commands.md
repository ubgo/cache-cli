# Commands

Usage: `cache-cli [flags] <command> [args]`. Flags must precede the command.
Machine-readable / value output goes to **stdout**; human messages (errors,
"not found", usage) go to **stderr** so `-json` output and piped values stay
clean.

### get

`cache-cli [flags] get <key>` — print the value, or exit `1` if missing.

What it is: `Get` on the (optionally namespaced) Redis cache. A miss prints
`not found` to stderr and exits `1` (not `0`) so scripts can branch on
presence. Wrong arg count → exit `2`.

Use cases:

- Branch in a shell script on whether a key exists.
- Pipe a cached value into another command.

```sh
# Plain value to stdout
cache-cli -addr localhost:6379 get user:42

# Branch on presence (exit 1 = miss)
if cache-cli get session:abc >/dev/null 2>&1; then
  echo "session live"
else
  echo "session absent"
fi

# JSON form
cache-cli -json get user:42      # {"key":"user:42","value":"alice"}
```

### set

`cache-cli [flags] set <key> <value>` — store a value (use `-ttl` for expiry).

What it is: `Set`. Needs exactly two args (else exit `2`). Backend failure →
exit `1`. Prints `OK` (or JSON `{"key":...,"ttl":...}`).

Use cases:

- Seed a cache key from a deploy/CI step.
- Manually override a value for debugging.

```sh
# All flags MUST precede the subcommand; -ttl after `set` is ignored.
cache-cli -ttl 5m set token abc
cache-cli -addr prod-redis:6379 -ttl 5m set token abc
cache-cli -json -ttl 5m set token abc      # {"key":"token","ttl":"5m0s"}
```

### del

`cache-cli [flags] del <key>` — delete a key.

What it is: `Del`. **Idempotent**: deleting an absent key is not an error and
exits `0`. Only a real backend failure exits `1`. Wrong arg count → exit `2`.

Use cases:

- Invalidate a key after a manual data fix.
- Cleanup step in a script (safe to run even if the key is gone).

```sh
cache-cli del user:42            # OK, exit 0 even if it did not exist
cache-cli -json del user:42      # {"deleted":"user:42"}
```

### stats

`cache-cli [flags] stats` — print backend stats. Always exits `0`.

What it is: `Stats()`. For Redis this reports `entries` (DBSIZE);
hits/misses/hit_ratio are shown but are zero for the Redis adapter (Redis
tracks those server-side, not per-adapter).

Use cases:

- Quick health/size glance from a shell or dashboard cron.
- Capture JSON stats into a monitoring pipeline.

```sh
cache-cli -addr localhost:6379 stats
# entries=1234 hits=0 misses=0 hit_ratio=0.000

cache-cli -json stats
# {"Hits":0,"Misses":0,...,"Entries":1234,...}
```

### keys

`cache-cli [flags] keys [prefix]` — list keys under a prefix (`SCAN`-based).

What it is: `Iterate` with the given prefix (empty/no arg = everything the
adapter can see). Redis `Iterate` is cursor `SCAN` (never `KEYS *`), so this is
safe on large keyspaces. The iterator is always closed. A scan that fails
midway still returns the keys it got but exits `1`.

Use cases:

- Audit which keys exist under a namespace.
- Feed a key list into `xargs cache-cli del`.

```sh
cache-cli keys 'user:'                 # one key per line on stdout
cache-cli -json keys 'user:'           # ["user:1","user:42",...]
cache-cli keys                         # every visible key

# Bulk delete a namespace
cache-cli -json keys 'tmp:' | jq -r '.[]' | xargs -n1 cache-cli del
```

### help

`cache-cli help` (also `-h`, `--help`) — print usage to stdout, exit `0`.

What it is: the usage text. Note: `help` as a command exits `0`; *no command at
all* prints usage to **stderr** and exits `2` (that is a usage error).

```sh
cache-cli help
cache-cli --help
```
