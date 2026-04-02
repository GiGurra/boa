package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

type Params struct {
	// Simple parameter declarations
	Baz  string `required:"true"`
	FB   string `required:"false"`
	Foo  string
	Bar  int    `default:"4"`
	File string `optional:"true"`
}

func main() {
	boa.CmdT[Params]{
		Use:   "hello-world",
		Short: "a generic cli tool",
		Long:  `A generic cli tool that has a longer description. See the README.MD for more information`,
		RunFunc: func(params *Params, _ *cobra.Command, _ []string) {
			fmt.Printf(
				"Hello world with params: %s, %d, %v, %s, %v\n",
				params.Foo,  // string
				params.Bar,  // int
				params.File, // string
				params.Baz,  // string
				params.FB,   // string
			)
		},
	}.Run()
}
