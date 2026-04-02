package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

func main() {

	var subCommand1Params = struct {
		Foo  string `descr:"a foo"`
		Bar  int    `descr:"a bar" env:"BAR_X" default:"4"`
		Path string `positional:"true"`
		Baz  string `positional:"true" default:"cba"`
		FB   string `positional:"true" optional:"true"`
	}{}

	boa.Cmd{
		Use:     "hello-world",
		Short:   "a generic cli tool",
		Long:    `A generic cli tool that has a longer description.See the README.MD for more information`,
		Version: "v1.2.3",
		SubCmds: boa.SubCmds(
			boa.Cmd{
				Use:         "subcommand1",
				Short:       "a subcommand",
				Params:      &subCommand1Params,
				ParamEnrich: boa.ParamEnricherCombine(boa.ParamEnricherName, boa.ParamEnricherEnv),
				RunFunc: func(cmd *cobra.Command, args []string) {
					p1 := subCommand1Params.Foo
					p2 := subCommand1Params.Bar
					p3 := subCommand1Params.Path
					p4 := subCommand1Params.Baz
					fmt.Printf("Hello world from subcommand1 with params: %s, %d, %s, %s\n", p1, p2, p3, p4)
				},
			},
			boa.Cmd{
				Use:   "subcommand2",
				Short: "a subcommand",
				RunFunc: func(cmd *cobra.Command, args []string) {
					fmt.Println("Hello world from subcommand2")
				},
			},
		),
	}.Run()
}
