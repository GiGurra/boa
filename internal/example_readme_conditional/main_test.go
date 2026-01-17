package main

import (
	"os"
	"testing"
)

func TestModeStream(t *testing.T) {
	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	// Mode=stream, FilePath not required
	os.Args = []string{"hello-world", "--mode", "stream"}
	main()
}

func TestModeFileWithPath(t *testing.T) {
	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	// Mode=file, FilePath required and provided
	os.Args = []string{"hello-world", "--mode", "file", "--file-path", "/path/to/file"}
	main()
}

func TestDebugWithVerbose(t *testing.T) {
	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	// Debug enabled, so Verbose flag is available
	os.Args = []string{"hello-world", "--mode", "stream", "--debug", "--verbose"}
	main()
}
