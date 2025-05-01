package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

type Params struct {
	Baz string
	FB  string
}

func main() {
	boa.CmdT[Params]{
		Use:   "hello-world",
		Short: "a generic cli tool",
		Long:  `A generic cli tool that has a longer description. See the README.MD for more information`,
		InitFunc: func(params *Params, cmd *cobra.Command) error {
			cmd.Deprecated = "this command is deprecated"
			return nil
		},
		RunFunc: func(params *Params, _ *cobra.Command, _ []string) {
			fmt.Printf(
				"Hello world with params: %s, %s\n",
				params.Baz, // string
				params.FB,  // *string
			)
		},
	}.Run()
}
