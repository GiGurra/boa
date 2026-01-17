package main

import (
	"os"
	"testing"
)

func TestSubcommand1(t *testing.T) {
	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	os.Args = []string{"hello-world", "subcommand1", "--foo", "test-foo", "my-path", "my-baz"}
	main()
}

func TestSubcommand1WithAllArgs(t *testing.T) {
	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	os.Args = []string{"hello-world", "subcommand1", "--foo", "test-foo", "--bar", "42", "my-path", "my-baz", "my-fb"}
	main()
}

func TestSubcommand2(t *testing.T) {
	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	os.Args = []string{"hello-world", "subcommand2", "--foo2", "test-foo2"}
	main()
}
