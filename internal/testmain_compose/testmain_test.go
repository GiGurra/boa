package main

import (
	"github.com/GiGurra/boa/internal/test_common"
	"testing"
)

func TestRunMain(t *testing.T) {
	test_common.RunTests(t, main, []test_common.TestSpec{
		{
			Name:     "no args",
			Args:     []string{},
			Expected: "missing required param 'base-foo'",
		},
		{
			Name: "with args",
			Args: []string{
				"--base-foo", "foo", "--base-bar", "1", "--base-file", "file",
				"--foo2", "foo", "--bar2", "1", "--file2", "file",
				"--foo3", "foo", "--bar3", "1", "--file3", "file",
				"--foo24", "foo", "--bar24", "1", "--file24", "file",
				"--baz", "baz",
			},
			Expected: "Hello world from subcommand1 with params: foo, 1, file, baz, \"\", 0001-01-01",
		},
	})

}
