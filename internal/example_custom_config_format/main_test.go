package main

import (
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

// registerOnce keeps the global format registration idempotent across tests.
var registerOnce sync.Once

func testdataPath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", name)
}

// runWith drives the command via RunArgsE (not main()), captures the parsed
// state via the Observed struct, and surfaces any error to the test.
func runWith(t *testing.T, args ...string) Observed {
	t.Helper()
	registerOnce.Do(registerKVFormat)
	var obs Observed
	cmd := newServerCmd(&obs)
	if err := cmd.RunArgsE(args); err != nil {
		t.Fatalf("RunArgsE(%v): %v", args, err)
	}
	return obs
}

// TestCustomFormatRoundTrip proves the custom .kv format is dispatched via the
// extension registry and that every field in the config file makes it into
// the parsed struct, including optional struct-pointer group fields with
// same-as-default writes.
func TestCustomFormatRoundTrip(t *testing.T) {
	got := runWith(t, "--config-file", testdataPath("config.kv"))

	if got.Host != "api.example.com" {
		t.Errorf("Host = %q, want api.example.com", got.Host)
	}
	if got.Port != 3000 {
		t.Errorf("Port = %d, want 3000", got.Port)
	}
	if !got.DBPresent {
		t.Fatal("DB pointer group should survive cleanup (KeyTree sees same-as-default writes)")
	}
	if got.DBHost != "localhost" {
		t.Errorf("DB.Host = %q, want localhost", got.DBHost)
	}
	if got.DBPort != 5432 {
		t.Errorf("DB.Port = %d, want 5432", got.DBPort)
	}
	// Both DB leaves were written with their default values; only a working
	// KeyTree-based detector can report HasValue=true here.
	if !got.DBHostViaCfg {
		t.Error("DB.Host should report set-by-config=true (same-as-default write via KeyTree)")
	}
	if !got.DBPortViaCfg {
		t.Error("DB.Port should report set-by-config=true (same-as-default write via KeyTree)")
	}
}

// TestBuiltinJSONSameBinary loads a .json file with the same compiled binary,
// proving the program is not locked to the custom format just because it
// registered one. This is the "deploy with json today, yaml tomorrow" scenario.
func TestBuiltinJSONSameBinary(t *testing.T) {
	got := runWith(t, "--config-file", testdataPath("config.json"))

	if got.Host != "api.example.com" {
		t.Errorf("Host = %q, want api.example.com", got.Host)
	}
	if got.Port != 3000 {
		t.Errorf("Port = %d, want 3000", got.Port)
	}
	if !got.DBPresent {
		t.Fatal("DB pointer group should survive cleanup under built-in JSON handler")
	}
	if got.DBHost != "localhost" || got.DBPort != 5432 {
		t.Errorf("DB values mismatch: host=%q port=%d", got.DBHost, got.DBPort)
	}
}

// TestCustomFormatCLIOverride asserts that a CLI flag actually overrides the
// config file value, not just that the program exits cleanly.
func TestCustomFormatCLIOverride(t *testing.T) {
	got := runWith(t, "--config-file", testdataPath("config.kv"), "--port", "9999")

	if got.Port != 9999 {
		t.Errorf("Port = %d, want 9999 (CLI should override config)", got.Port)
	}
	// Config-file Host should still apply since CLI didn't set it.
	if got.Host != "api.example.com" {
		t.Errorf("Host = %q, want api.example.com (config file value, CLI didn't override)", got.Host)
	}
}
