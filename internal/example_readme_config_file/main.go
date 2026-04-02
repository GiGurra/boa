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
	boa.CmdT[ConfigFromFile]{
		Use: "my-app",
		PreValidateFunc: func(params *ConfigFromFile, cmd *cobra.Command, args []string) error {
			// Load configuration from file if provided
			// CLI and env var values take precedence over file values
			return boa.LoadConfigFile(params.File, &params.AppConfig, nil)
		},
		RunFunc: func(params *ConfigFromFile, cmd *cobra.Command, args []string) {
			// Use parameters loaded from the file
			fmt.Printf("Host: %s, Port: %d\n",
				params.Host,
				params.Port,
			)
		},
	}.Run()
}
