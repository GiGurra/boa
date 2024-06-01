package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
	"time"
)

func main() {

	var params = struct {
		Foo  boa.Required[string]
		Bar  boa.Required[int]
		File boa.Required[string]
		Baz  boa.Required[string]
		FB   boa.Optional[string]
		Time boa.Optional[time.Time]
	}{}

	boa.Wrap{
		Use:   "hello-world",
		Short: "a generic cli tool",
		Long:  `A generic cli tool that has a longer description. See the README.MD for more information`,
		ParamEnrich: boa.ParamEnricherCombine(
			boa.ParamEnricherName,
			boa.ParamEnricherShort,
		),
		Params: &params,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf(
				"Hello world from subcommand1 with params: %s, %d, %s, %s, %v\n",
				params.Foo.Value(),  // string
				params.Bar.Value(),  // int
				params.File.Value(), // string
				params.Baz.Value(),  // string
				params.FB.Value(),   // *string
			)
		},
	}.ToAppH(boa.Handler{
		Failure: func(err error) {
			fmt.Printf("Error: %v\n", err)
		},
	})
}
