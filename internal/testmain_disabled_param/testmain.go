package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

func main() {

	var params = struct {
		Foo boa.Optional[string]
		Bar boa.Optional[int]
		Baz boa.Optional[string]
	}{}

	params.Bar.SetIsEnabledFn(func() bool {
		return params.Foo.HasValue()
	})
	params.Baz.SetRequiredFn(func() bool {
		return params.Foo.HasValue()
	})

	boa.Cmd{
		Use:   "hello-world",
		Short: "a generic cli tool",
		Long:  `A generic cli tool that has a longer description. See the README.MD for more information`,
		ParamEnrich: boa.ParamEnricherCombine(
			boa.ParamEnricherName,
			boa.ParamEnricherShort,
		),
		Params: &params,
		RunFunc: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Hello World!\n")
		},
	}.RunH(boa.ResultHandler{
		Failure: func(err error) {
			fmt.Printf("Error: %v\n", err)
		},
	})
}
