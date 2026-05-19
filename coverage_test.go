// coverage_test.go — exhaustive run() path coverage: every subcommand, every
// exit code (0/1/2), -json variants, namespace isolation, flag-parse error,
// and backend-unreachable error. Driven through run(...) with bytes buffers
// like main_test.go's exec helper; asserts exit code + stdout/stderr.

package main

import (
	"strings"
	"testing"
)

// badAddr points the redis client at a closed/bogus address so backend calls
// fail fast (exit 1). 127.0.0.1:1 is reliably refused.
const badAddr = "127.0.0.1:1"

func TestGetHitJSON(t *testing.T) {
	addr := mini(t)
	if code, _, _ := exec(t, addr, "set", "k", "hello"); code != 0 {
		t.Fatal("set failed")
	}
	code, out, _ := exec(t, addr, "-json", "get", "k")
	if code != 0 {
		t.Fatalf("get -json exit %d", code)
	}
	if !strings.Contains(out, `"key":"k"`) || !strings.Contains(out, `"value":"hello"`) {
		t.Fatalf("get -json output wrong: %q", out)
	}
}

func TestGetMissExit1(t *testing.T) {
	addr := mini(t)
	code, out, errb := exec(t, addr, "get", "absent")
	if code != 1 {
		t.Fatalf("get miss should exit 1, got %d", code)
	}
	if out != "" {
		t.Fatalf("miss should produce no stdout, got %q", out)
	}
	if !strings.Contains(errb, "not found") {
		t.Fatalf("expected 'not found' on stderr, got %q", errb)
	}
}

func TestGetBackendErrorExit1(t *testing.T) {
	code, _, errb := exec(t, badAddr, "get", "k")
	if code != 1 {
		t.Fatalf("get on unreachable backend should exit 1, got %d", code)
	}
	if !strings.Contains(errb, "error:") {
		t.Fatalf("expected 'error:' on stderr, got %q", errb)
	}
}

func TestGetWrongArgCountExit2(t *testing.T) {
	addr := mini(t)
	code, _, errb := exec(t, addr, "get")
	if code != 2 {
		t.Fatalf("get with no key should exit 2, got %d", code)
	}
	if !strings.Contains(errb, "need exactly one") {
		t.Fatalf("expected usage message, got %q", errb)
	}
	if code, _, _ := exec(t, addr, "get", "a", "b"); code != 2 {
		t.Fatalf("get with two keys should exit 2, got %d", code)
	}
}

func TestSetWithTTL(t *testing.T) {
	addr := mini(t)
	// Go's flag package stops at the first non-flag arg, so -ttl must precede
	// the subcommand (the README's trailing-flag example is non-functional).
	code, out, _ := exec(t, addr, "-ttl", "5m", "set", "k", "v")
	if code != 0 {
		t.Fatalf("set -ttl exit %d", code)
	}
	if strings.TrimSpace(out) != "OK" {
		t.Fatalf("expected OK, got %q", out)
	}
}

func TestSetJSON(t *testing.T) {
	addr := mini(t)
	code, out, _ := exec(t, addr, "-json", "-ttl", "1s", "set", "k", "v")
	if code != 0 {
		t.Fatalf("set -json exit %d", code)
	}
	if !strings.Contains(out, `"key":"k"`) || !strings.Contains(out, `"ttl":"1s"`) {
		t.Fatalf("set -json output wrong: %q", out)
	}
}

func TestSetWrongArgCountExit2(t *testing.T) {
	addr := mini(t)
	if code, _, _ := exec(t, addr, "set", "k"); code != 2 {
		t.Fatalf("set with one arg should exit 2, got %d", code)
	}
	if code, _, _ := exec(t, addr, "set"); code != 2 {
		t.Fatalf("set with no args should exit 2, got %d", code)
	}
}

func TestSetBackendErrorExit1(t *testing.T) {
	code, _, errb := exec(t, badAddr, "set", "k", "v")
	if code != 1 {
		t.Fatalf("set on unreachable backend should exit 1, got %d", code)
	}
	if !strings.Contains(errb, "error:") {
		t.Fatalf("expected 'error:' on stderr, got %q", errb)
	}
}

func TestDelOKAndJSON(t *testing.T) {
	addr := mini(t)
	_, _, _ = exec(t, addr, "set", "k", "v")
	code, out, _ := exec(t, addr, "del", "k")
	if code != 0 || strings.TrimSpace(out) != "OK" {
		t.Fatalf("del got code=%d out=%q", code, out)
	}
	// Idempotent: deleting an absent key still exits 0.
	if code, _, _ := exec(t, addr, "del", "k"); code != 0 {
		t.Fatalf("del absent should exit 0, got %d", code)
	}
	code, out, _ = exec(t, addr, "-json", "del", "k")
	if code != 0 || !strings.Contains(out, `"deleted":"k"`) {
		t.Fatalf("del -json got code=%d out=%q", code, out)
	}
}

func TestDelWrongArgCountExit2(t *testing.T) {
	addr := mini(t)
	if code, _, _ := exec(t, addr, "del"); code != 2 {
		t.Fatalf("del with no key should exit 2, got %d", code)
	}
	if code, _, _ := exec(t, addr, "del", "a", "b"); code != 2 {
		t.Fatalf("del with two keys should exit 2, got %d", code)
	}
}

func TestDelBackendErrorExit1(t *testing.T) {
	code, _, errb := exec(t, badAddr, "del", "k")
	if code != 1 {
		t.Fatalf("del on unreachable backend should exit 1, got %d", code)
	}
	if !strings.Contains(errb, "error:") {
		t.Fatalf("expected 'error:' on stderr, got %q", errb)
	}
}

func TestStatsPlainAndJSON(t *testing.T) {
	addr := mini(t)
	code, out, _ := exec(t, addr, "stats")
	if code != 0 {
		t.Fatalf("stats exit %d", code)
	}
	if !strings.Contains(out, "entries=") || !strings.Contains(out, "hit_ratio=") {
		t.Fatalf("stats plain output wrong: %q", out)
	}
	code, out, _ = exec(t, addr, "-json", "stats")
	if code != 0 {
		t.Fatalf("stats -json exit %d", code)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("stats -json should be a JSON object: %q", out)
	}
}

func TestKeysPlainAndEmptyPrefix(t *testing.T) {
	addr := mini(t)
	_, _, _ = exec(t, addr, "set", "a:1", "x")
	_, _, _ = exec(t, addr, "set", "b:1", "y")

	code, out, _ := exec(t, addr, "keys", "a:")
	if code != 0 {
		t.Fatalf("keys exit %d", code)
	}
	if !strings.Contains(out, "a:1") || strings.Contains(out, "b:1") {
		t.Fatalf("keys plain prefix scan wrong: %q", out)
	}

	// Empty prefix (no arg) iterates everything.
	code, out, _ = exec(t, addr, "keys")
	if code != 0 {
		t.Fatalf("keys (empty prefix) exit %d", code)
	}
	if !strings.Contains(out, "a:1") || !strings.Contains(out, "b:1") {
		t.Fatalf("empty-prefix keys should list all: %q", out)
	}
}

func TestKeysJSONEmpty(t *testing.T) {
	addr := mini(t)
	code, out, _ := exec(t, addr, "-json", "keys", "nomatch:")
	if code != 0 {
		t.Fatalf("keys -json exit %d", code)
	}
	// No matches → json.Marshal(nil slice) yields "null".
	if strings.TrimSpace(out) != "null" {
		t.Fatalf("empty keys -json should be null, got %q", out)
	}
}

func TestKeysBackendErrorExit1(t *testing.T) {
	code, _, errb := exec(t, badAddr, "keys", "x:")
	if code != 1 {
		t.Fatalf("keys on unreachable backend should exit 1, got %d", code)
	}
	if !strings.Contains(errb, "error:") {
		t.Fatalf("expected 'error:' on stderr, got %q", errb)
	}
}

func TestHelpExit0(t *testing.T) {
	addr := mini(t)
	// Bare "help" reaches the switch and prints usage to stdout, exit 0.
	code, out, _ := exec(t, addr, "help")
	if code != 0 {
		t.Fatalf("help should exit 0, got %d", code)
	}
	if !strings.Contains(out, "USAGE:") {
		t.Fatalf("help should print usage to stdout, got %q", out)
	}
}

func TestDashHFlagInterceptedExit2(t *testing.T) {
	addr := mini(t)
	// -h and --help are consumed by flag.Parse (ErrHelp) before reaching the
	// switch, so they exit 2 via the flag-parse-error path. The "-h"/"--help"
	// switch labels are only reachable as a positional after another arg and
	// are effectively defensive (justified-uncovered).
	for _, h := range []string{"-h", "--help"} {
		if code, _, _ := exec(t, addr, h); code != 2 {
			t.Fatalf("%q should exit 2 (flag-intercepted), got %d", h, code)
		}
	}
}

func TestNoCommandExit2(t *testing.T) {
	addr := mini(t)
	code, _, errb := exec(t, addr)
	if code != 2 {
		t.Fatalf("no command should exit 2, got %d", code)
	}
	if !strings.Contains(errb, "USAGE:") {
		t.Fatalf("no command should print usage to stderr, got %q", errb)
	}
}

func TestUnknownCommandPrintsUsageExit2(t *testing.T) {
	addr := mini(t)
	code, _, errb := exec(t, addr, "frobnicate")
	if code != 2 {
		t.Fatalf("unknown command should exit 2, got %d", code)
	}
	if !strings.Contains(errb, "unknown command") || !strings.Contains(errb, "USAGE:") {
		t.Fatalf("unknown command stderr wrong: %q", errb)
	}
}

func TestFlagParseErrorExit2(t *testing.T) {
	addr := mini(t)
	// -nope is not a defined flag → flag.Parse fails → exit 2.
	code, _, _ := exec(t, addr, "-nope", "stats")
	if code != 2 {
		t.Fatalf("unknown flag should exit 2, got %d", code)
	}
	// Bad duration value for -ttl also fails flag parsing.
	if code, _, _ := exec(t, addr, "-ttl", "notaduration", "set", "k", "v"); code != 2 {
		t.Fatalf("bad -ttl should exit 2, got %d", code)
	}
}

func TestNamespaceIsolationJSON(t *testing.T) {
	addr := mini(t)
	if code, _, _ := exec(t, addr, "-ns", "svc:a", "set", "k", "v1"); code != 0 {
		t.Fatal("namespaced set a failed")
	}
	if code, _, _ := exec(t, addr, "-ns", "svc:b", "set", "k", "v2"); code != 0 {
		t.Fatal("namespaced set b failed")
	}
	code, out, _ := exec(t, addr, "-ns", "svc:a", "-json", "get", "k")
	if code != 0 || !strings.Contains(out, `"value":"v1"`) {
		t.Fatalf("ns svc:a get wrong: code=%d out=%q", code, out)
	}
	// keys under one namespace must not see the other's key.
	code, _, _ = exec(t, addr, "-ns", "svc:a", "keys", "")
	if code != 0 {
		t.Fatalf("ns keys exit %d", code)
	}
}
