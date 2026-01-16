package main

import (
	"fmt"

	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

func main() {
	boa.Cmd{
		Use:   "myapp",
		Short: "An app demonstrating aliases and groups",
		Long:  "An example CLI app that demonstrates command aliases and help grouping",
		// Define a custom group title for "server", let "tools" be auto-generated
		Groups: []*cobra.Group{
			{ID: "server", Title: "Server Commands:"},
		},
		SubCmds: boa.SubCmds(
			// Server commands with aliases
			boa.Cmd{
				Use:     "start",
				Short:   "Start the server",
				Aliases: []string{"up", "run"},
				GroupID: "server",
				RunFunc: func(cmd *cobra.Command, args []string) {
					fmt.Println("Server started!")
				},
			},
			boa.Cmd{
				Use:     "stop",
				Short:   "Stop the server",
				Aliases: []string{"down"},
				GroupID: "server",
				RunFunc: func(cmd *cobra.Command, args []string) {
					fmt.Println("Server stopped!")
				},
			},
			// Tool commands - group will be auto-generated as "tools:"
			boa.Cmd{
				Use:     "lint",
				Short:   "Run linter",
				GroupID: "tools",
				RunFunc: func(cmd *cobra.Command, args []string) {
					fmt.Println("Linting...")
				},
			},
			boa.Cmd{
				Use:     "format",
				Short:   "Format code",
				Aliases: []string{"fmt"},
				GroupID: "tools",
				RunFunc: func(cmd *cobra.Command, args []string) {
					fmt.Println("Formatting...")
				},
			},
		),
	}.Run()
}
