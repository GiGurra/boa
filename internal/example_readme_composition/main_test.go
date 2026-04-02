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

	// Named struct fields (Base Base1) get auto-prefixed: Foo → --base-foo
	// Embedded struct fields (Base2) do NOT get prefixed: Foo2 → --foo2
	os.Args = []string{
		"hello-world",
		"--base-foo", "foo-val",
		"--base-bar", "42",
		"--base-file", "file-val",
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
		"--base-foo", "foo-val",
		"--base-bar", "42",
		"--base-file", "file-val",
		"--foo2", "foo2-val",
		"--bar2", "43",
		"--file2", "file2-val",
		"--baz", "baz-val",
		"--fb", "fb-val",
		"--time", "2024-01-15T10:30:00Z",
	}
	main()
}
