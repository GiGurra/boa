// Example matching the README "Sub-commands" section.
// This demonstrates hierarchical CLI tools with sub-commands.
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
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
	boa.NewCmdT[boa.NoParams]("hello-world").
		WithShort("a generic cli tool").
		WithLong("A generic cli tool that has a longer description").
		WithSubCmds(
			boa.NewCmdT[SubParams]("subcommand1").
				WithShort("a subcommand").
				WithRunFunc(func(params *SubParams) {
					fmt.Printf("Hello world from subcommand1 with params: %s, %d, %s, %s\n",
						params.Foo, params.Bar, params.Path, params.Baz)
				}),
			boa.NewCmdT[OtherParams]("subcommand2").
				WithShort("a subcommand").
				WithRunFunc(func(params *OtherParams) {
					fmt.Println("Hello world from subcommand2")
				}),
		).
		Run()
}
