package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

type Params struct {
	// Simple parameter declarations
	Baz string `required:"true"`
	FB  string `required:"false"`
	// More flexible declarations
	Foo  boa.Required[string]
	Bar  boa.Required[int] `default:"4"`
	File boa.Optional[string]
}

func main() {
	boa.CmdT[Params]{
		Use:   "hello-world",
		Short: "a generic cli tool",
		Long:  `A generic cli tool that has a longer description. See the README.MD for more information`,
		RunFunc: func(params *Params, _ *cobra.Command, _ []string) {
			fmt.Printf(
				"Hello world with params: %s, %d, %s, %s, %v\n",
				params.Foo.Value(),  // string
				params.Bar.Value(),  // int
				params.File.Value(), // *string
				params.Baz,          // string
				params.FB,           // *string
			)
		},
	}.Run()
}
