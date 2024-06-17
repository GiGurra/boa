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
		{
			Name:     "success when all flags provided (including hidden flag)",
			Args:     []string{"--foo", "foo", "--baz", "baz", "--bar", "1"},
			Contains: []string{"Hello World!"},
		},
		{
			Name:        "error when trying to use disabled flag",
			Args:        []string{"--bar", "1"},
			NotContains: []string{"Hello World!"},
		},
		{
			Name:        "fail when missing conditionally required flag",
			Args:        []string{"--foo", "xyz"},
			NotContains: []string{"Hello World!"},
		},
		{
			Name:     "succeed when no flags and all are optional",
			Args:     []string{},
			Contains: []string{"Hello World!"},
		},
		{
			Name:     "succeed when including required flag",
			Args:     []string{"--foo", "xyz", "--baz", "baz"},
			Contains: []string{"Hello World!"},
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
