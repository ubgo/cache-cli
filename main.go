// main.go — the cache-cli inspector entrypoint and all command logic (package main, github.com/ubgo/cache-cli).
//
// Package role: cache-cli is the inspector CLI in the ubgo/cache family — it
// talks to a Redis backend through the github.com/ubgo/cache-redis adapter
// (so it is written entirely against the cache.Cache interface). This file
// has no doc.go; the // Command … block below is the package doc.
//
// This file: defines main (a thin os.Exit(run(...)) shell), the usage text,
// and run — which parses flags, builds the redis adapter (optionally
// cache.Namespaced), and dispatches get/set/del/stats/keys/help.
// Contracts an AI must keep: exit codes are 0 success / 1 runtime or backend
// error or get-miss / 2 usage error (bad flags, no command, wrong arg count,
// unknown command) — scripts depend on this, do not change it; machine /
// value output goes to stdout, human messages (errors, "not found", usage)
// go to stderr so -json and piped values stay clean; flag.ContinueOnError
// (not ExitOnError) and injected stdout/stderr keep run unit-testable.
//
// AI-context: the fmt.Fprint* writes to stdout deliberately ignore errors —
// .golangci.yml already excludes them from errcheck; keep that arrangement so
// CI stays green (do not wrap these in error checks).

// Command cache-cli is a minimal inspector for a github.com/ubgo/cache Redis
// backend: get/set/del/stats/keys with an optional --json output and a
// non-zero exit on failure (scriptable).
//
//	cache-cli -addr localhost:6379 stats
//	cache-cli -addr localhost:6379 -ns svc:billing get user:42
//	cache-cli set k v -ttl 5m
//	cache-cli -json keys 'user:*'
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/redis/go-redis/v9"
	"github.com/ubgo/cache"
	rediscache "github.com/ubgo/cache-redis"
)

// main is a thin shell: it delegates everything to run and uses run's return
// value as the process exit code. Keeping all logic and I/O in run (which
// takes explicit writers) is what makes the program testable without
// spawning a subprocess — see main_test.go.
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

const usage = `cache-cli — inspector for a ubgo/cache Redis backend

USAGE:
  cache-cli [flags] <command> [args]

COMMANDS:
  get <key>            print the value (or exit 1 if missing)
  set <key> <value>    store a value (use -ttl for expiry)
  del <key>            delete a key
  stats                print backend stats
  keys <prefix>        list keys under a prefix (SCAN-based)

FLAGS:
  -addr string   redis address (default "localhost:6379")
  -ns string     namespace prefix applied to keys
  -ttl duration  TTL for 'set' (default 0 = no expiry)
  -json          machine-readable JSON output
  -h, -help      show this help

EXAMPLES:
  cache-cli -addr localhost:6379 stats
  cache-cli -ns svc:billing get user:42
  cache-cli set token abc -ttl 5m
  cache-cli -json keys 'user:'`

// run executes one invocation and returns the process exit code.
//
// Exit-code contract (relied on by scripts — do not change):
//
//	0  success (includes del of an absent key, stats, help)
//	1  runtime/backend error, or a get miss (cache.ErrNotFound)
//	2  usage error: bad flags, no command, wrong arg count, unknown command
//
// Stream contract: machine-readable / value output goes to stdout; human
// messages (errors, "not found", usage) go to stderr so -json output and
// piped values stay clean. stdout/stderr are injected so tests can capture
// them with bytes.Buffer instead of spawning a process.
func run(args []string, stdout, stderr io.Writer) int {
	// ContinueOnError (not ExitOnError) so a flag parse failure returns 2
	// from run instead of calling os.Exit inside the flag package — that is
	// what keeps run unit-testable.
	fs := flag.NewFlagSet("cache-cli", flag.ContinueOnError)
	// Route the flag package's own error text to the injected stderr.
	fs.SetOutput(stderr)
	addr := fs.String("addr", "localhost:6379", "redis address")
	ns := fs.String("ns", "", "namespace prefix")
	ttl := fs.Duration("ttl", 0, "TTL for set")
	asJSON := fs.Bool("json", false, "JSON output")
	fs.Usage = func() { fmt.Fprintln(stderr, usage) }
	if err := fs.Parse(args); err != nil {
		return 2
	}
	// Positional args after flag parsing: rest[0] is the subcommand.
	rest := fs.Args()
	if len(rest) == 0 {
		fmt.Fprintln(stderr, usage)
		return 2 // no command given is a usage error, not a runtime error
	}

	rdb := redis.NewClient(&redis.Options{Addr: *addr})
	defer func() { _ = rdb.Close() }()
	// The redis adapter implements cache.Cache; everything below is written
	// against the interface, so namespacing is a transparent wrapper and the
	// command logic never knows whether a prefix is applied.
	var c cache.Cache = rediscache.New(rdb)
	if *ns != "" {
		c = cache.Namespaced(c, *ns)
	}
	ctx := context.Background()

	cmd := rest[0]
	cmdArgs := rest[1:]

	// emit is the single output path for get/set/del/stats: JSON-marshal v to
	// stdout under -json, otherwise write the plain string. Errors are
	// intentionally ignored — see .golangci.yml; stdout writes don't fail in
	// practice and a CLI checking every Fprint is noise.
	emit := func(v any, plain string) {
		if *asJSON {
			b, _ := json.Marshal(v)
			fmt.Fprintln(stdout, string(b))
		} else {
			fmt.Fprintln(stdout, plain)
		}
	}

	switch cmd {
	case "get":
		if len(cmdArgs) != 1 {
			fmt.Fprintln(stderr, "get: need exactly one <key>")
			return 2
		}
		v, err := c.Get(ctx, cmdArgs[0])
		// A miss is exit 1 (not 0) so scripts can branch on presence; the
		// human reason goes to stderr to keep stdout clean for piping.
		if errors.Is(err, cache.ErrNotFound) {
			fmt.Fprintln(stderr, "not found")
			return 1
		}
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		emit(map[string]string{"key": cmdArgs[0], "value": string(v)}, string(v))
		return 0

	case "set":
		if len(cmdArgs) != 2 {
			fmt.Fprintln(stderr, "set: need <key> <value>")
			return 2
		}
		if err := c.Set(ctx, cmdArgs[0], []byte(cmdArgs[1]), *ttl); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		emit(map[string]any{"key": cmdArgs[0], "ttl": ttl.String()}, "OK")
		return 0

	case "del":
		if len(cmdArgs) != 1 {
			fmt.Fprintln(stderr, "del: need exactly one <key>")
			return 2
		}
		// Del is idempotent at the backend level: deleting an absent key is
		// not an error and exits 0. Only a real backend failure exits 1.
		if err := c.Del(ctx, cmdArgs[0]); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		emit(map[string]string{"deleted": cmdArgs[0]}, "OK")
		return 0

	case "stats":
		s := c.Stats()
		emit(s, fmt.Sprintf("entries=%d hits=%d misses=%d hit_ratio=%.3f",
			s.Entries, s.Hits, s.Misses, s.HitRatio()))
		return 0

	case "keys":
		// Empty prefix (no arg) iterates everything the adapter can see.
		prefix := ""
		if len(cmdArgs) == 1 {
			prefix = cmdArgs[0]
		}
		// Iterate is SCAN-based on Redis (never KEYS *), so this is safe on
		// large keyspaces. The iterator is always Closed via defer.
		it := c.Iterate(ctx, cache.IterateOpts{Prefix: prefix})
		defer func() { _ = it.Close() }()
		var found []string
		for it.Next() {
			found = append(found, it.Key())
		}
		// Err must be checked after the loop: a partial scan that failed
		// midway still returned some keys but is a runtime error (exit 1).
		if err := it.Err(); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		if *asJSON {
			b, _ := json.Marshal(found)
			fmt.Fprintln(stdout, string(b))
		} else {
			for _, k := range found {
				fmt.Fprintln(stdout, k)
			}
		}
		return 0

	case "-h", "help", "--help":
		fmt.Fprintln(stdout, usage)
		return 0

	default:
		fmt.Fprintf(stderr, "unknown command %q\n", cmd)
		fmt.Fprintln(stderr, usage)
		return 2
	}
}
