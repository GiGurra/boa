package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

var subCommand1Params = struct {
	Foo  boa.Required[string] `descr:"a foo"`
	Bar  boa.Required[int]    `descr:"a bar" env:"BAR_X" default:"4"`
	Path boa.Required[string] `positional:"true"`
	Baz  boa.Required[string] `positional:"true" default:"cba"`
	FB   boa.Optional[string] `positional:"true"`
}{}

func main() {
	boa.Wrap{
		Use:   "hello-world",
		Short: "a generic cli tool",
		Long:  `A generic cli tool that has a longer description.See the README.MD for more information`,
		SubCommands: []*cobra.Command{
			boa.Wrap{
				Use:         "subcommand1",
				Short:       "a subcommand",
				Params:      &subCommand1Params,
				ParamEnrich: boa.ParamEnricherCombine(boa.ParamEnricherName, boa.ParamEnricherEnv),
				Run: func(cmd *cobra.Command, args []string) {
					p1 := subCommand1Params.Foo.Value()
					p2 := subCommand1Params.Bar.Value()
					p3 := subCommand1Params.Path.Value()
					p4 := subCommand1Params.Baz.Value()
					fmt.Printf("Hello world from subcommand1 with params: %s, %d, %s, %s\n", p1, p2, p3, p4)
				},
			}.ToCmd(),
			boa.Wrap{
				Use:   "subcommand2",
				Short: "a subcommand",
				Run: func(cmd *cobra.Command, args []string) {
					fmt.Println("Hello world from subcommand2")
				},
			}.ToCmd(),
		},
	}.ToApp()
}
