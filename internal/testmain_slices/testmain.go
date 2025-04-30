package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

func main() {

	var params struct {
		WithoutDefaults boa.Required[[]float64]
		WithDefaults    boa.Required[[]int64] `default:"[1,2,3]"`
	}

	boa.Cmd{
		Use:    "hello-world",
		Short:  "a generic cli tool",
		Long:   `A generic cli tool that has a longer description. See the README.MD for more information`,
		Params: &params,
		RunFunc: func(cmd *cobra.Command, args []string) {
			fmt.Printf(
				"Hello world from subcommand1 with params: %v, %v\n",
				params.WithoutDefaults.Value(),
				params.WithDefaults.Value(),
			)
		},
	}.RunH(boa.ResultHandler{
		Failure: func(err error) {
			fmt.Printf("Error: %v\n", err)
		},
	})
}
