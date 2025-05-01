package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

type Params struct {
	Foo string `required:"true"`
	Bar string `required:"false"`
}

func main() {
	boa.CmdT[Params]{
		Use:  "hello-world",
		Long: `A generic cli tool that has a longer description. See the README.MD for more information`,
		RunFunc: func(p *Params, _ *cobra.Command, _ []string) {
			fmt.Printf("Hello: %v, %v\n", p.Foo, p.Bar)
		},
	}.Run()
}
