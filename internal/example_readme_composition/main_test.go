package main

import (
	"os"
	"testing"
)

func TestComposition(t *testing.T) {
	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	// Note: nested struct fields use their own names, not prefixed with parent struct name
	os.Args = []string{
		"hello-world",
		"--foo", "foo-val",
		"--bar", "42",
		"--file", "file-val",
		"--foo2", "foo2-val",
		"--bar2", "43",
		"--file2", "file2-val",
		"--baz", "baz-val",
	}
	main()
}

func TestCompositionWithOptional(t *testing.T) {
	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	os.Args = []string{
		"hello-world",
		"--foo", "foo-val",
		"--bar", "42",
		"--file", "file-val",
		"--foo2", "foo2-val",
		"--bar2", "43",
		"--file2", "file2-val",
		"--baz", "baz-val",
		"--f-b", "fb-val",
		"--time", "2024-01-15T10:30:00Z",
	}
	main()
}
