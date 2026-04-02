package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

func main() {

	var params struct {
		WithoutDefaults []float64
		WithDefaults    []int64 `default:"[1,2,3]"`
	}

	if err := (boa.Cmd{
		Use:    "hello-world",
		Short:  "a generic cli tool",
		Long:   `A generic cli tool that has a longer description. See the README.MD for more information`,
		Params: &params,
		RunFunc: func(cmd *cobra.Command, args []string) {
			fmt.Printf(
				"params: without=%v, with=%v\n",
				params.WithoutDefaults,
				params.WithDefaults,
			)
		},
	}.RunE()); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
