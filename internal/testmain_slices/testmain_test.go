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
			Expected: "missing required param 'without-defaults'",
		},
		{
			Name:     "with args",
			Args:     []string{"--without-defaults", "1"},
			Expected: "Hello world from subcommand1 with params: [1], [1 2 3]",
		},
	})
}
