package main

import (
	"github.com/GiGurra/boa/cmd/test_common"
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
			Args:     []string{"--foo", "foo", "--bar", "1", "--file", "file", "--baz", "baz"},
			Expected: "Hello world from subcommand1 with params: foo, 1, file, baz, <nil>",
		},
	})
}
