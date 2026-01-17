package main

import (
	"os"
	"testing"
)

func TestMain(t *testing.T) {
	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	// Test with all required args: --foo value, positional path and baz
	os.Args = []string{"hello-world", "--foo", "test-foo", "my-path", "my-baz"}
	main()
}

func TestMainWithAllArgs(t *testing.T) {
	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	// Test with all args including optional ones
	os.Args = []string{"hello-world", "--foo", "test-foo", "--bar", "42", "my-path", "my-baz", "my-fb"}
	main()
}

func TestMainWithEnvVar(t *testing.T) {
	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	// Test with BAR_X env var
	os.Setenv("BAR_X", "100")
	defer os.Unsetenv("BAR_X")

	os.Args = []string{"hello-world", "--foo", "test-foo", "my-path", "my-baz"}
	main()
}
