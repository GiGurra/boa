// Example matching the README "Sub-commands" section.
// This demonstrates hierarchical CLI tools with sub-commands.
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

type SubParams struct {
	Foo  string `descr:"a foo"`
	Bar  int    `descr:"a bar" env:"BAR_X" default:"4"`
	Path string `positional:"true"`
	Baz  string `positional:"true" default:"cba"`
	FB   string `positional:"true" optional:"true"`
}

type OtherParams struct {
	Foo2 string `descr:"a foo"`
}

func main() {
	boa.CmdT[boa.NoParams]{
		Use:   "hello-world",
		Short: "a generic cli tool",
		Long:  "A generic cli tool that has a longer description",
		SubCmds: boa.SubCmds(
			boa.CmdT[SubParams]{
				Use:   "subcommand1",
				Short: "a subcommand",
				RunFunc: func(params *SubParams, cmd *cobra.Command, args []string) {
					fmt.Printf("Hello world from subcommand1 with params: %s, %d, %s, %s\n",
						params.Foo, params.Bar, params.Path, params.Baz)
				},
			},
			boa.CmdT[OtherParams]{
				Use:   "subcommand2",
				Short: "a subcommand",
				RunFunc: func(params *OtherParams, cmd *cobra.Command, args []string) {
					fmt.Println("Hello world from subcommand2")
				},
			},
		),
	}.Run()
}
