// Example matching the README "Minimum setup" section.
// This demonstrates raw Go types with struct tags.
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

type Params struct {
	Foo  string `descr:"a foo"`
	Bar  int    `descr:"a bar" env:"BAR_X" optional:"true"`
	Path string `positional:"true"`
	Baz  string `positional:"true" default:"cba"`
	FB   string `positional:"true" optional:"true"`
}

func main() {
	boa.CmdT[Params]{
		Use:   "hello-world",
		Short: "a generic cli tool",
		Long:  `A generic cli tool that has a longer description. See the README.MD for more information`,
		RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
			fmt.Printf(
				"Hello world with params: %s, %d, %s, %s, %s\n",
				params.Foo,  // string (access directly)
				params.Bar,  // int (access directly)
				params.Path, // string
				params.Baz,  // string
				params.FB,   // string
			)
		},
	}.Run()
}
