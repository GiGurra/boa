package main

import (
	"os"
	"testing"
)

func TestSlicesWithDefaults(t *testing.T) {
	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	// Test with just numbers (tags and ports use defaults)
	os.Args = []string{"hello-world", "--numbers", "1,2,3"}
	main()
}

func TestSlicesOverrideDefaults(t *testing.T) {
	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	// Test overriding defaults
	os.Args = []string{
		"hello-world",
		"--numbers", "10,20,30",
		"--tags", "x,y,z",
		"--ports", "9000,9001",
	}
	main()
}

func TestSlicesSingleValue(t *testing.T) {
	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	// Test with single value for numbers
	os.Args = []string{"hello-world", "--numbers", "42"}
	main()
}
