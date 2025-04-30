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
			Expected: "missing required param 'foo'",
		},
		{
			Name: "with args",
			Args: []string{
				"--foo", "foo", "--bar", "1", "--file", "file",
				"--foo2", "foo", "--bar2", "1", "--file2", "file",
				"--foo3", "foo", "--bar3", "1", "--file3", "file",
				"--foo24", "foo", "--bar24", "1", "--file24", "file",
				"--baz", "baz",
			},
			Expected: "Hello world from subcommand1 with params: foo, 1, file, baz, <nil>, <nil>",
		},
	})

}
