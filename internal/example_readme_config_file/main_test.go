package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func getTestdataPath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", "config.json")
}

func TestWithConfigFile(t *testing.T) {
	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	configPath := getTestdataPath()
	os.Args = []string{"my-app", "--file", configPath}
	main()
}

func TestWithoutConfigFile(t *testing.T) {
	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	// Without config file, use CLI args
	os.Args = []string{"my-app", "--host", "example.com", "--port", "9000"}
	main()
}

func TestCliOverridesConfigFile(t *testing.T) {
	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	// CLI args take precedence over config file
	configPath := getTestdataPath()
	os.Args = []string{"my-app", "--file", configPath, "--port", "9999"}
	main()
}
