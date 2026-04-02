package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

type disabledParamParams struct {
	Foo string `optional:"true"`
	Bar int    `optional:"true"`
	Baz string `optional:"true"`
}

func main() {

	var params disabledParamParams

	if err := (boa.CmdT[disabledParamParams]{
		Use:   "hello-world",
		Short: "a generic cli tool",
		Long:  `A generic cli tool that has a longer description. See the README.MD for more information`,
		ParamEnrich: boa.ParamEnricherCombine(
			boa.ParamEnricherName,
			boa.ParamEnricherShort,
		),
		Params: &params,
		InitFuncCtx: func(ctx *boa.HookContext, p *disabledParamParams, cmd *cobra.Command) error {
			ctx.GetParam(&p.Bar).SetIsEnabledFn(func() bool {
				return ctx.HasValue(&p.Foo)
			})
			ctx.GetParam(&p.Baz).SetRequiredFn(func() bool {
				return ctx.HasValue(&p.Foo)
			})
			return nil
		},
		RunFunc: func(p *disabledParamParams, cmd *cobra.Command, args []string) {
			fmt.Printf("Hello World!\n")
		},
	}.RunE()); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
