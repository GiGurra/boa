package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

func main() {

	var subCommand1Params = struct {
		Foo  boa.Required[string]
		Bar  boa.Required[int]    `descr:"a bar" env:"BAR_X" default:"111"`
		Path boa.Required[string] `positional:"true"`
		Baz  boa.Required[string]
		FB   boa.Optional[string] `positional:"true"`
	}{
		Foo: boa.Required[string]{Descr: "a foo"},                                                          // add additional info if you like. This means we get "a foo [required] (env: FOO)" in the help text
		Bar: boa.Required[int]{Default: boa.Default(4), CustomValidator: func(x int) error { return nil }}, // optional custom validation logic
		Baz: boa.Required[string]{Positional: true, Default: boa.Default("cba")},                           // positional arguments
	}

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
					p1 := subCommand1Params.Foo.Value()
					p2 := subCommand1Params.Bar.Value()
					p3 := subCommand1Params.Path.Value()
					p4 := subCommand1Params.Baz.Value()
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
	}.RunH(boa.ResultHandler{
		Failure: func(err error) {
			fmt.Printf("Error: %v\n", err)
		},
	})
}
