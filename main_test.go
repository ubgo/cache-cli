package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
)

func mini(t *testing.T) string {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	return mr.Addr()
}

func exec(t *testing.T, addr string, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	full := append([]string{"-addr", addr}, args...)
	code := run(full, &out, &errb)
	return code, out.String(), errb.String()
}

func TestSetGetDelFlow(t *testing.T) {
	addr := mini(t)

	if code, _, e := exec(t, addr, "set", "k", "hello"); code != 0 {
		t.Fatalf("set exit %d: %s", code, e)
	}
	code, out, _ := exec(t, addr, "get", "k")
	if code != 0 || strings.TrimSpace(out) != "hello" {
		t.Fatalf("get got code=%d out=%q", code, out)
	}
	if code, _, _ := exec(t, addr, "del", "k"); code != 0 {
		t.Fatal("del should succeed")
	}
	// Missing key → exit 1.
	if code, _, _ := exec(t, addr, "get", "k"); code != 1 {
		t.Fatalf("get missing should exit 1, got %d", code)
	}
}

func TestKeysJSON(t *testing.T) {
	addr := mini(t)
	_, _, _ = exec(t, addr, "set", "user:1", "a")
	_, _, _ = exec(t, addr, "set", "user:2", "b")
	_, _, _ = exec(t, addr, "set", "post:1", "c")

	code, out, _ := exec(t, addr, "-json", "keys", "user:")
	if code != 0 {
		t.Fatalf("keys exit %d", code)
	}
	if !strings.Contains(out, "user:1") || !strings.Contains(out, "user:2") ||
		strings.Contains(out, "post:1") {
		t.Fatalf("prefix scan wrong: %s", out)
	}
}

func TestUnknownCommandAndUsage(t *testing.T) {
	addr := mini(t)
	if code, _, _ := exec(t, addr, "bogus"); code != 2 {
		t.Fatalf("unknown command should exit 2, got %d", code)
	}
	code, out, _ := exec(t, addr, "help")
	if code != 0 || !strings.Contains(out, "USAGE:") {
		t.Fatalf("help should print usage and exit 0 (code=%d)", code)
	}
}

func TestNamespaceFlag(t *testing.T) {
	addr := mini(t)
	if code, _, _ := exec(t, addr, "-ns", "svc", "set", "k", "v"); code != 0 {
		t.Fatal("namespaced set failed")
	}
	// Same key without the namespace must miss.
	if code, _, _ := exec(t, addr, "get", "k"); code != 1 {
		t.Fatalf("unnamespaced get should miss, got %d", code)
	}
	if code, out, _ := exec(t, addr, "-ns", "svc", "get", "k"); code != 0 || strings.TrimSpace(out) != "v" {
		t.Fatalf("namespaced get failed: code=%d out=%q", code, out)
	}
}
