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
			Name:     "with args",
			Args:     []string{"-f", "foo", "-b", "1", "--file", "file", "--baz", "baz"},
			Expected: "Hello world from subcommand1 with params: foo, 1, file, baz, <nil>",
		},
	})
}
