package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

func main() {

	var subCommand1Params = struct {
		Foo boa.Required[string] `alts:"abc,cde,fgh"`
	}{}

	boa.Cmd{
		Use:     "hello-world",
		Short:   "a generic cli tool",
		Long:    `A generic cli tool that has a longer description.See the README.MD for more information`,
		Version: "v1.2.3",
		SubCmds: []*cobra.Command{
			boa.Cmd{
				Use:         "subcommand1",
				Short:       "a subcommand",
				Params:      &subCommand1Params,
				ParamEnrich: boa.ParamEnricherCombine(boa.ParamEnricherName, boa.ParamEnricherEnv),
				RunFunc: func(cmd *cobra.Command, args []string) {
					p1 := subCommand1Params.Foo.Value()
					fmt.Printf("Hello world from subcommand1 with params: %s\n", p1)
				},
			}.ToCobra(),
			boa.Cmd{
				Use:   "subcommand2",
				Short: "a subcommand",
				RunFunc: func(cmd *cobra.Command, args []string) {
					fmt.Println("Hello world from subcommand2")
				},
			}.ToCobra(),
		},
	}.RunH(boa.ResultHandler{
		Failure: func(err error) {
			fmt.Printf("Error: %v\n", err)
		},
	})
}
