package main

import (
	"fmt"
	"testing"

	"github.com/GiGurra/boa/internal/test_common"
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
			Name: "help shows grouped commands",
			Args: []string{"--help"},
			Contains: []string{
				"Server Commands:", // explicit group title
				"tools:",           // auto-generated group title
				"start",
				"stop",
				"lint",
				"format",
			},
		},
		{
			Name: "subcommand help shows aliases for start",
			Args: []string{"start", "--help"},
			Contains: []string{
				"Aliases:",
				"start, up, run",
			},
		},
		{
			Name: "subcommand help shows aliases for format",
			Args: []string{"format", "--help"},
			Contains: []string{
				"Aliases:",
				"format, fmt",
			},
		},
		{
			Name:     "run command by name",
			Args:     []string{"start"},
			Contains: []string{"Server started!"},
		},
		{
			Name:     "run command by alias 'up'",
			Args:     []string{"up"},
			Contains: []string{"Server started!"},
		},
		{
			Name:     "run command by alias 'run'",
			Args:     []string{"run"},
			Contains: []string{"Server started!"},
		},
		{
			Name:     "run stop by alias 'down'",
			Args:     []string{"down"},
			Contains: []string{"Server stopped!"},
		},
		{
			Name:     "run format by alias 'fmt'",
			Args:     []string{"fmt"},
			Contains: []string{"Formatting..."},
		},
		{
			Name:     "run lint command",
			Args:     []string{"lint"},
			Contains: []string{"Linting..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			res := test_common.RunSingleTest(main, tt.Args...)

			fmt.Print(res.StdOut)

			for _, c := range tt.Contains {
				if !res.HasMatchingLine(c) {
					t.Errorf("Expected '%s' in output", c)
				}
			}

			for _, c := range tt.NotContains {
				if res.HasMatchingLine(c) {
					t.Errorf("Expected '%s' to not be in output", c)
				}
			}
		})
	}
}
