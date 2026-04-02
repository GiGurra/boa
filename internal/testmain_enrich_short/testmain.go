package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
	"time"
)

func main() {

	var params = struct {
		Foo  string
		Bar  int
		File string
		Baz  string
		FB   string    `optional:"true"`
		Time time.Time `optional:"true"`
	}{}

	if err := (boa.Cmd{
		Use:   "hello-world",
		Short: "a generic cli tool",
		Long:  `A generic cli tool that has a longer description. See the README.MD for more information`,
		ParamEnrich: boa.ParamEnricherCombine(
			boa.ParamEnricherName,
			boa.ParamEnricherShort,
		),
		Params: &params,
		RunFunc: func(cmd *cobra.Command, args []string) {
			fmt.Printf(
				"Hello world from subcommand1 with params: %s, %d, %s, %s, %q\n",
				params.Foo,  // string
				params.Bar,  // int
				params.File, // string
				params.Baz,  // string
				params.FB,   // string
			)
		},
	}.RunE()); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
