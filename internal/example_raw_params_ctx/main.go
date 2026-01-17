// Example demonstrating context-aware hooks for customizing raw parameters.
//
// This example shows how to use HookContext to access and configure
// auto-generated parameter mirrors for raw struct fields (string, int, etc.)
// that don't use the Required[T]/Optional[T] wrappers.
package main

import (
	"fmt"

	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

// ServerConfig uses raw fields instead of Required[T]/Optional[T] wrappers.
// The HookContext API allows us to customize these fields programmatically.
type ServerConfig struct {
	Host     string // Will be customized via InitCtx
	Port     int    // Will be customized via InitCtx
	LogLevel string // Will have alternatives set
	Protocol string // Will have strict alternatives
}

// InitCtx is called during initialization with access to parameter mirrors.
// This allows setting defaults, alternatives, env vars, etc. on raw fields.
func (c *ServerConfig) InitCtx(ctx *boa.HookContext) error {
	// Get the parameter mirror for the Host field
	hostParam := ctx.GetParam(&c.Host)
	hostParam.SetDefault(boa.Default("localhost"))
	hostParam.SetEnv("SERVER_HOST")

	// Get the parameter mirror for the Port field
	portParam := ctx.GetParam(&c.Port)
	portParam.SetDefault(boa.Default(8080))
	portParam.SetEnv("SERVER_PORT")

	// Set up alternatives for LogLevel with shell completion
	logParam := ctx.GetParam(&c.LogLevel)
	logParam.SetDefault(boa.Default("info"))
	logParam.SetAlternatives([]string{"debug", "info", "warn", "error"})

	// Set up strict alternatives for Protocol (validation will fail if not in list)
	protoParam := ctx.GetParam(&c.Protocol)
	protoParam.SetDefault(boa.Default("http"))
	protoParam.SetAlternatives([]string{"http", "https", "grpc"})
	protoParam.SetStrictAlts(true)

	return nil
}

func main() {
	boa.NewCmdT[ServerConfig]("server").
		WithShort("Start the server with configurable options").
		WithRunFunc(func(params *ServerConfig) {
			fmt.Printf("Starting server:\n")
			fmt.Printf("  Host:     %s\n", params.Host)
			fmt.Printf("  Port:     %d\n", params.Port)
			fmt.Printf("  LogLevel: %s\n", params.LogLevel)
			fmt.Printf("  Protocol: %s\n", params.Protocol)
		}).
		Run()
}

// Alternative: Using function-based hooks instead of interface
func exampleWithFunctionHook() {
	type Config struct {
		Name    string
		Verbose bool
	}

	boa.NewCmdT[Config]("app").
		WithInitFuncCtx(func(ctx *boa.HookContext, params *Config, cmd *cobra.Command) error {
			// Configure the Name parameter
			nameParam := ctx.GetParam(&params.Name)
			nameParam.SetDefault(boa.Default("default-name"))
			nameParam.SetShort("n")

			// Configure the Verbose parameter
			verboseParam := ctx.GetParam(&params.Verbose)
			verboseParam.SetShort("v")

			return nil
		}).
		WithRunFunc(func(params *Config) {
			fmt.Printf("Name: %s, Verbose: %v\n", params.Name, params.Verbose)
		}).
		Run()
}

// Example showing GetParam works for both raw and wrapped fields
func exampleMixedFields() {
	type MixedConfig struct {
		RawHost     string             // raw field
		WrappedPort boa.Required[int]  // wrapped field
		OptionalTLS boa.Optional[bool] // optional wrapped field
	}

	boa.NewCmdT[MixedConfig]("mixed").
		WithInitFuncCtx(func(ctx *boa.HookContext, params *MixedConfig, cmd *cobra.Command) error {
			// GetParam works for raw fields - returns the auto-generated mirror
			rawParam := ctx.GetParam(&params.RawHost)
			rawParam.SetDefault(boa.Default("0.0.0.0"))

			// GetParam also works for wrapped fields - returns the field itself
			wrappedParam := ctx.GetParam(&params.WrappedPort)
			wrappedParam.SetDefault(boa.Default(443))

			// Works for optional fields too
			optParam := ctx.GetParam(&params.OptionalTLS)
			optParam.SetDefault(boa.Default(true))

			return nil
		}).
		WithRunFunc(func(params *MixedConfig) {
			fmt.Printf("Host: %s, Port: %d, TLS: %v\n",
				params.RawHost,
				params.WrappedPort.Value(),
				params.OptionalTLS.Value())
		}).
		Run()
}
