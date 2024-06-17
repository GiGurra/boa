package main

import (
	"fmt"
	"github.com/GiGurra/boa/cmd/test_common"
	"testing"
)

type spec struct {
	Name        string
	Args        []string
	Contains    []string
	NotContains []string
}

func TestRunMain(t *testing.T) {

	tests := []spec{
		{
			Name:        "print help",
			Args:        []string{"--help"},
			Contains:    []string{"Usage:", "--foo", "--baz"},
			NotContains: []string{"--bar", "required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			res := test_common.RunSingleTest(main, tt.Args...)

			fmt.Print(res.StdOut)

			for _, c := range tt.Contains {
				if !res.HasMatchingLine(c) {
					t.Errorf("Expected %s in output", c)
				}
			}

			for _, c := range tt.NotContains {
				if res.HasMatchingLine(c) {
					t.Errorf("Expected %s to not be in output", c)
				}
			}

		})
	}
}
