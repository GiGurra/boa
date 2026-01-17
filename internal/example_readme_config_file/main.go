// Example matching the README "Config file serialization and configuration" section.
// This demonstrates loading configuration from a file.
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

type AppConfig struct {
	Host string
	Port int
}

type ConfigFromFile struct {
	File string `descr:"config file path" optional:"true"`
	AppConfig
}

func main() {
	boa.NewCmdT[ConfigFromFile]("my-app").
		WithPreValidateFuncCtx(func(ctx *boa.HookContext, params *ConfigFromFile, cmd *cobra.Command, args []string) error {
			// Load configuration from file if provided
			// boa.UnMarshalFromFileParam is a helper to unmarshal from a file
			// CLI and env var values take precedence over file values
			fileParam := ctx.GetParam(&params.File)
			return boa.UnMarshalFromFileParam(fileParam, &params.AppConfig, nil)
		}).
		WithRunFunc(func(params *ConfigFromFile) {
			// Use parameters loaded from the file
			fmt.Printf("Host: %s, Port: %d\n",
				params.Host,
				params.Port,
			)
		}).
		Run()
}
