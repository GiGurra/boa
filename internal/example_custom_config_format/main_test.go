package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func testdataPath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", name)
}

// TestCustomFormatRoundTrip loads the custom .kv format via the extension registry.
func TestCustomFormatRoundTrip(t *testing.T) {
	prev := os.Args
	defer func() { os.Args = prev }()
	os.Args = []string{"server", "--config-file", testdataPath("config.kv")}
	main()
}

// TestBuiltinJSONSameBinary loads a .json file using the same compiled binary,
// proving the program is not locked to the custom format just because it
// registered one. This is the "deploy with json today, yaml tomorrow" scenario.
func TestBuiltinJSONSameBinary(t *testing.T) {
	prev := os.Args
	defer func() { os.Args = prev }()
	os.Args = []string{"server", "--config-file", testdataPath("config.json")}
	main()
}

func TestCustomFormatCLIOverride(t *testing.T) {
	prev := os.Args
	defer func() { os.Args = prev }()
	os.Args = []string{"server", "--config-file", testdataPath("config.kv"), "--port", "9999"}
	main()
}
