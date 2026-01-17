// Example matching the README "Conditional parameters" section.
// This demonstrates making parameters conditionally required or enabled using HookContext.
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

type Params struct {
	Mode     string // when "file", FilePath is required
	FilePath string `optional:"true"`
	Verbose  bool   `optional:"true"` // only enabled when Debug is true
	Debug    bool   `optional:"true"`
}

func main() {
	boa.NewCmdT[Params]("hello-world").
		WithShort("a generic cli tool").
		WithInitFuncCtx(func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
			// FilePath is required when Mode is "file"
			ctx.GetParam(&p.FilePath).SetRequiredFn(func() bool {
				return p.Mode == "file"
			})

			// Verbose is only enabled when Debug is true
			ctx.GetParam(&p.Verbose).SetIsEnabledFn(func() bool {
				return p.Debug
			})

			return nil
		}).
		WithRunFunc(func(params *Params) {
			fmt.Printf("Hello World! Mode=%s\n", params.Mode)
		}).
		Run()
}
