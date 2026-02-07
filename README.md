# BOA

[![CI Status](https://github.com/GiGurra/boa/actions/workflows/ci.yml/badge.svg)](https://github.com/GiGurra/boa/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/GiGurra/boa)](https://goreportcard.com/report/github.com/GiGurra/boa)

Boa adds a declarative layer on top of `github.com/spf13/cobra`.

The goal is making the process of creating a command line interface as simple as possible, while still providing access
to cobra primitives when needed.

**[Full Documentation](https://gigurra.github.io/boa/)** - This README is a condensed summary. See the docs for detailed guides on enrichers, validation, lifecycle hooks, and advanced features.

## Installation

`go get github.com/GiGurra/boa@latest`

## Usage

### Minimum setup

```go
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

type Params struct {
	Foo  string `descr:"a foo"`
	Bar  int    `descr:"a bar" env:"BAR_X" optional:"true"`
	Path string `positional:"true"`
	Baz  string `positional:"true" default:"cba"`
	FB   string `positional:"true" optional:"true"`
}

func main() {
	boa.CmdT[Params]{
		Use:   "hello-world",
		Short: "a generic cli tool",
		Long:  `A generic cli tool that has a longer description. See the README.MD for more information`,
		RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
			fmt.Printf(
				"Hello world with params: %s, %d, %s, %s, %s\n",
				params.Foo,  // string (access directly)
				params.Bar,  // int (access directly)
				params.Path, // string
				params.Baz,  // string
				params.FB,   // string
			)
		},
	}.Run()
}

```

Help output for the above:

```
A generic cli tool that has a longer description. See the README.MD for more information

Usage:
  hello-world <path> <baz> [f-b] [flags]

Flags:
      --bar int      a bar (env: BAR_X) (default 4)
  -f, --foo string   a foo (env: FOO, required)
  -h, --help         help for hello-world
```

### Struct Tags Reference

| Tag | Description | Example |
|-----|-------------|---------|
| `descr` / `desc` / `description` / `help` | Description text for help | `descr:"User name"` |
| `name` / `long` | Override flag name | `name:"user-name"` |
| `default` | Default value | `default:"8080"` |
| `env` | Environment variable name | `env:"PORT"` |
| `short` | Short flag (single char) | `short:"p"` |
| `positional` / `pos` | Marks positional argument | `positional:"true"` |
| `required` / `req` | Marks as required | `required:"true"` |
| `optional` / `opt` | Marks as optional | `optional:"true"` |
| `alts` / `alternatives` | Allowed values (enum) | `alts:"debug,info,warn,error"` |
| `strict-alts` / `strict` | Validate against alts | `strict:"true"` |

For advanced programmatic configuration (setting defaults, alternatives, conditional requirements),
see the [Context-Aware Hooks](#context-aware-hooks-hookcontext) section.

### Global Configuration

Use `boa.Init()` to configure global behavior before creating commands:

```go
func main() {
    boa.Init(
        boa.WithDefaultOptional(), // plain fields default to optional instead of required
    )

    // Now string, int, etc. fields are optional unless explicitly tagged required:"true"
    boa.CmdT[MyParams]{
        Use: "my-app",
        // ...
    }.Run()
}
```

Without `Init`, the default behavior is unchanged (fields are required). Explicit struct tags (`required`, `req`, `optional`, `opt`) and `Required[T]`/`Optional[T]` wrappers always take precedence.

See [Global Configuration](https://gigurra.github.io/boa/global-config/) for details.

### Enrichers

The `ParamEnrich` field controls parameter enrichment:

| Value | Behavior |
|-------|----------|
| `nil` | Uses `ParamEnricherDefault` (derives names, short flags, and bool defaults) |
| `ParamEnricherDefault` | Explicit default: derives names, short flags, and bool defaults |
| `ParamEnricherNone` | No enrichment - you must specify everything via struct tags |

`ParamEnricherDefault` includes:

| Enricher | Behavior |
|----------|----------|
| `ParamEnricherName` | Converts field name to kebab-case flag (e.g., `MyParam` → `--my-param`) |
| `ParamEnricherShort` | Auto-assigns short flag from first character (skips `h` for help, avoids conflicts) |
| `ParamEnricherBool` | Sets default `false` for boolean params without explicit defaults |

Environment variable binding is **not** included by default. Add `ParamEnricherEnv` explicitly:

```go
// Enable auto env vars
boa.CmdT[Params]{
    Use: "cmd",
    ParamEnrich: boa.ParamEnricherCombine(
        boa.ParamEnricherName,
        boa.ParamEnricherShort,
        boa.ParamEnricherEnv,
        boa.ParamEnricherBool,
    ),
    // ...
}

// With prefixed env vars
boa.CmdT[Params]{
    Use: "cmd",
    ParamEnrich: boa.ParamEnricherCombine(
        boa.ParamEnricherName,
        boa.ParamEnricherEnv,
        boa.ParamEnricherEnvPrefix("MYAPP"), // MY_PARAM → MYAPP_MY_PARAM
    ),
    // ...
}
```

### Sub-commands

Create hierarchical CLI tools with sub-commands:

```go
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

type SubParams struct {
	Foo  string `descr:"a foo"`
	Bar  int    `descr:"a bar" env:"BAR_X" default:"4"`
	Path string `positional:"true"`
	Baz  string `positional:"true" default:"cba"`
	FB   string `positional:"true" optional:"true"`
}

type OtherParams struct {
	Foo2 string `descr:"a foo"`
}

func main() {
	boa.CmdT[boa.NoParams]{
		Use:   "hello-world",
		Short: "a generic cli tool",
		Long:  "A generic cli tool that has a longer description",
		SubCmds: boa.SubCmds(
			boa.CmdT[SubParams]{
				Use:   "subcommand1",
				Short: "a subcommand",
				RunFunc: func(params *SubParams, cmd *cobra.Command, args []string) {
					fmt.Printf("Hello world from subcommand1 with params: %s, %d, %s, %s\n",
						params.Foo, params.Bar, params.Path, params.Baz)
				},
			},
			boa.CmdT[OtherParams]{
				Use:   "subcommand2",
				Short: "another subcommand",
				RunFunc: func(params *OtherParams, cmd *cobra.Command, args []string) {
					fmt.Println("Hello world from subcommand2")
				},
			},
		),
	}.Run()
}
```

Help output for the above:

```
a subcommand

Usage:
  hello-world subcommand1 <path> <baz> [f-b] [flags]

Flags:
      --bar int      a bar (env: BAR_X) (default 4)
  -f, --foo string   a foo (env: FOO, required)
  -h, --help         help for subcommand1
```

### Composition

You can compose structs to create more complex parameter structures:

```go
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
	"time"
)

type Base1 struct {
	Foo  string
	Bar  int
	File string
}

type Base2 struct {
	Foo2  string
	Bar2  int
	File2 string
}

type Combined struct {
	Base Base1
	Base2
	Baz  string
	FB   string    `optional:"true"`
	Time time.Time `optional:"true"`
}

func main() {
	boa.CmdT[Combined]{
		Use:   "hello-world",
		Short: "a generic cli tool",
		Long:  "A generic cli tool that has a longer description",
		RunFunc: func(params *Combined, cmd *cobra.Command, args []string) {
			fmt.Printf(
				"Hello world with params: %s, %d, %s, %s, %s, %v\n",
				params.Base.Foo,  // string
				params.Base.Bar,  // int
				params.Base.File, // string
				params.Baz,       // string
				params.FB,        // string
				params.Time,      // time.Time
			)
		},
	}.Run()
}
```

**Note:** Nested struct fields use their own field names as flags, not prefixed with the parent struct name.
For example, `Base.Foo` becomes `--foo`, not `--base-foo`. See "Missing features" for planned prefix support.

### Leverage all of Cobra's features

Access the underlying Cobra command for advanced customization:

```go
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
		Long:  "A generic cli tool that has a longer description",
		InitFunc: func(params *Params, cmd *cobra.Command) error {
			cmd.Deprecated = "this command is deprecated"
			return nil
		},
		RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
			fmt.Printf("Hello world with params: %s, %s\n",
				params.Baz,
				params.FB,
			)
		},
	}.Run()
}
```

### Conditional parameters

You can make parameters conditionally required or enabled using `HookContext`:

```go
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
	boa.CmdT[Params]{
		Use:   "hello-world",
		Short: "a generic cli tool",
		InitFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
			// FilePath is required when Mode is "file"
			ctx.GetParam(&p.FilePath).SetRequiredFn(func() bool {
				return p.Mode == "file"
			})

			// Verbose is only enabled when Debug is true
			ctx.GetParam(&p.Verbose).SetIsEnabledFn(func() bool {
				return p.Debug
			})

			return nil
		},
		RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
			fmt.Printf("Hello World! Mode=%s\n", params.Mode)
		},
	}.Run()
}
```

### Constraining parameter values

You can specify that a parameter must be one of a set of values using the `alts` tag:

```go
type Params struct {
	LogLevel string `alts:"debug,info,warn,error" strict:"true"`
	Format   string `alts:"json,yaml,toml"` // suggestions only (strict defaults to true)
}
```

### Array/slice parameters

Boa supports array/slice types with proper parsing:

```go
type Params struct {
	Numbers []int    `descr:"list of numbers"`
	Tags    []string `descr:"tags" default:"[a,b,c]"`
	Ports   []int64  `descr:"ports" default:"[8080,8081,8082]"`
}
```

### Config file serialization and configuration

```go
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
		PreValidateFuncCtx: func(ctx *boa.HookContext, params *ConfigFromFile, cmd *cobra.Command, args []string) error {
			// Load configuration from file if provided
			// boa.UnMarshalFromFileParam is a helper to unmarshal from a file
			// CLI and env var values take precedence over file values
			fileParam := ctx.GetParam(&params.File)
			return boa.UnMarshalFromFileParam(fileParam, &params.AppConfig, nil)
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
```

## Parameter value source priority

Boa supports multiple sources for parameter values, including command-line flags, environment variables, and config
files. When multiple sources are available, the following priority order is used:

1. **Command-line flags**: Values provided directly on the command line take precedence over all other sources.
2. **Environment variables**: If a command-line flag is not provided, the corresponding environment variable will be
   used if it exists.
3. **Config files**: If neither a command-line flag nor an environment variable is provided, the value from the
   configuration file will be used.
4. **Default values**: If no value is provided from any source, the default value specified in the parameter
   definition will be used.
5. **Zero value**: If no value is provided from any source and no default value is specified, the zero value for the
   parameter type will be used.

## Lifecycle Hooks in Boa

Boa provides several lifecycle hooks that can be implemented or defined to customize behavior at different stages of
command execution. These hooks give you fine-grained control over parameter initialization, validation, and execution.

### Init Hook

The Init hook runs during the initialization phase, after boa creates internal parameter mirrors but before cobra
flags are registered. This allows you to configure parameters (set defaults, env vars, validators) via `HookContext`
before they become CLI flags.

```go
package main

import (
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

// Implement this interface on your configuration struct
type CfgStructInit interface {
	Init() error
}

// Example implementation
func (i *MyConfigStruct) Init() error {
	// Initialize defaults, set up validators, etc.
	i.SomeParam.Default = boa.Default("default value")
	return nil
}

// Alternatively, use the InitFunc field in CmdT
func main() {
	boa.CmdT[MyConfigStruct]{
		Use: "command",
		InitFunc: func(params *MyConfigStruct, cmd *cobra.Command) error {
			// Custom initialization logic
			return nil
		},
	}.Run()
}

```

### PostCreate Hook

The PostCreate hook runs after cobra flags have been created but before any command-line arguments are parsed.
This is useful when you need to inspect or modify the cobra command after flags are registered.

```go
package main

import (
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

func main() {
	boa.CmdT[MyConfigStruct]{
		Use: "command",
		PostCreateFuncCtx: func(ctx *boa.HookContext, params *MyConfigStruct, cmd *cobra.Command) error {
			// Cobra flags are now available
			flag := cmd.Flags().Lookup("my-flag")
			if flag != nil {
				// Inspect or modify flag properties
			}
			return nil
		},
	}.Run()
}
```

### PreValidate Hook

The PreValidate hook runs after parameters are parsed from the command line and environment variables but before
validation is performed.

```go
package main

import (
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

// Implement this interface on your configuration struct
type CfgStructPreValidate interface {
	PreValidate() error
}

// Example implementation
func (i *MyConfigStruct) PreValidate() error {
	// Manipulate parameters before validation
	return nil
}

// Alternatively, use the PreValidateFunc field in CmdT
func main() {
	boa.CmdT[MyConfigStruct]{
		Use: "command",
		PreValidateFunc: func(params *MyConfigStruct, cmd *cobra.Command, args []string) error {
			// Custom pre-validation logic, such as loading from config files
			return nil
		},
	}.Run()
}
```

### PreExecute Hook

The PreExecute hook runs after parameter validation but before the command's Run function is executed.

```go
package main

import (
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

// Implement this interface on your configuration struct
type CfgStructPreExecute interface {
	PreExecute() error
}

// Example implementation
func (i *MyConfigStruct) PreExecute() error {
	// Setup that should happen after validation but before execution
	return nil
}

// Alternatively, use the PreExecuteFunc field in CmdT
func main() {
	boa.CmdT[MyConfigStruct]{
		Use: "command",
		PreExecuteFunc: func(params *MyConfigStruct, cmd *cobra.Command, args []string) error {
			// Custom pre-execution logic
			return nil
		},
	}.Run()
}

```

### Hook Execution Order

Hooks are executed in the following order:

1. **Init** - Parameter mirrors exist, cobra flags not yet created (configure params here)
2. **PostCreate** - Cobra flags are now registered (inspect/modify flags here)
3. **PreValidate** - After flags are parsed but before validation
4. **Validation** - Built-in parameter validation
5. **PreExecute** - After validation but before command execution
6. **Run** - The actual command execution

### Common Use Cases

- **Init**: Set up default values, configure custom validators
- **PostCreate**: Inspect or modify cobra flags after they're registered
- **PreValidate**: Load configurations from files, set derived parameters
- **PreExecute**: Establish connections, prepare resources needed for execution

### Error Handling

All hooks can return errors to abort command execution. If any hook returns an error, the command will not proceed to
the next phase, and the error will be reported to the user.

### Context-Aware Hooks (HookContext)

For advanced use cases, boa provides context-aware hooks that give access to the underlying parameter mirrors.

The `HookContext` provides:
- `GetParam(fieldPtr any) Param` - Get the Param interface for any field
- `HasValue(fieldPtr any) bool` - Check if a parameter has a value from any source (CLI, env, default, or injection)
- `AllMirrors() []Param` - Get all auto-generated parameter mirrors

#### Interface-based Context Hooks

```go
package main

import (
	"github.com/GiGurra/boa/pkg/boa"
)

type ServerConfig struct {
	Host     string
	Port     int
	LogLevel string
}

// InitCtx is called during initialization with HookContext access
func (c *ServerConfig) InitCtx(ctx *boa.HookContext) error {
	// Configure the Host parameter
	hostParam := ctx.GetParam(&c.Host)
	hostParam.SetDefault(boa.Default("localhost"))
	hostParam.SetEnv("SERVER_HOST")

	// Configure the Port parameter
	portParam := ctx.GetParam(&c.Port)
	portParam.SetDefault(boa.Default(8080))
	portParam.SetEnv("SERVER_PORT")

	// Set up alternatives with shell completion for LogLevel
	logParam := ctx.GetParam(&c.LogLevel)
	logParam.SetDefault(boa.Default("info"))
	logParam.SetAlternatives([]string{"debug", "info", "warn", "error"})
	logParam.SetStrictAlts(true) // Validation fails if value not in list

	return nil
}

func main() {
	boa.CmdT[ServerConfig]{
		Use: "server",
		RunFunc: func(params *ServerConfig, cmd *cobra.Command, args []string) {
			// Use params.Host, params.Port, params.LogLevel
		},
	}.Run()
}
```

Available context-aware interfaces:
- `CfgStructInitCtx` - `InitCtx(ctx *HookContext) error`
- `CfgStructPreValidateCtx` - `PreValidateCtx(ctx *HookContext) error`
- `CfgStructPreExecuteCtx` - `PreExecuteCtx(ctx *HookContext) error`

#### Function-based Context Hooks

```go
package main

import (
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

type Config struct {
	Name    string
	Verbose bool
}

func main() {
	boa.CmdT[Config]{
		Use: "app",
		InitFuncCtx: func(ctx *boa.HookContext, params *Config, cmd *cobra.Command) error {
			// Configure parameters programmatically
			nameParam := ctx.GetParam(&params.Name)
			nameParam.SetDefault(boa.Default("default-name"))
			nameParam.SetShort("n")
			nameParam.SetAlternatives([]string{"alice", "bob", "carol"})
			return nil
		},
		RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
			// Use params
		},
	}.Run()
}
```

Available context hook fields in `CmdT`:
- `InitFuncCtx` - During initialization
- `PostCreateFuncCtx` - After cobra flags are created
- `PreValidateFuncCtx` - After parsing, before validation
- `PreExecuteFuncCtx` - After validation, before execution
- `RunFuncCtx` - Command execution with HookContext access

#### RunFuncCtx - Checking Parameter Sources at Runtime

Use `RunFuncCtx` when you need to check whether optional parameters actually have a value, even if that explicitly is the go default zero value:

```go
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

type Params struct {
	Host string `default:"localhost"`
	Port int    `optional:"true"`
}

func main() {
	boa.CmdT[Params]{
		Use: "server",
		RunFuncCtx: func(ctx *boa.HookContext, params *Params, cmd *cobra.Command, args []string) {
			// Check if parameters have values (from CLI, env, default, or injection)
			if ctx.HasValue(&params.Port) {
				fmt.Printf("Starting server on %s:%d\n", params.Host, params.Port)
			} else {
				fmt.Printf("Starting server on %s (no port specified)\n", params.Host)
			}
		},
	}.Run()
}
```

Note: You cannot use both `RunFunc` and `RunFuncCtx` on the same command - choose one or the other.

### Error-Returning Run Functions (RunFuncE)

Boa provides error-returning variants of the run functions for better error handling in your commands:

```go
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

type Params struct {
	File string
}

func main() {
	err := boa.CmdT[Params]{
		Use: "process",
		RunFuncE: func(params *Params, cmd *cobra.Command, args []string) error {
			if params.File == "" {
				return fmt.Errorf("file cannot be empty")
			}
			// Process file...
			return nil
		},
	}.RunE() // Returns error instead of calling os.Exit

	if err != nil {
		fmt.Printf("Command failed: %v\n", err)
	}
}
```

#### Available Error-Returning Fields

| Field | Signature |
|-------|-----------|
| `RunFuncE` | `func(params *T, cmd *cobra.Command, args []string) error` |
| `RunFuncCtxE` | `func(ctx *HookContext, params *T, cmd *cobra.Command, args []string) error` |

#### Execution Methods

| Method | Description |
|--------|-------------|
| `Run()` | Executes command, panics on any error |
| `RunE()` | Executes command, returns error |
| `RunArgs(args)` | Executes with args, panics on any error |
| `RunArgsE(args)` | Executes with args, returns error |
| `ToCobra()` | Returns `*cobra.Command` with `cmd.Run` set (panics on setup error) |
| `ToCobraE()` | Returns `(*cobra.Command, error)` with `cmd.RunE` set |

#### Run() vs RunE() Error Handling

The two execution methods handle errors differently:

- **`Run()`** - All errors (hook errors, runtime errors, config errors) cause **panics**
- **`RunE()`** - All errors are **returned** for programmatic handling

```go
// With Run() - errors panic
boa.CmdT[Params]{
	Use: "cmd",
	RunFuncE: func(p *Params, cmd *cobra.Command, args []string) error {
		return fmt.Errorf("something went wrong")
	},
}.Run() // Panics with the error

// With RunE() - errors are returned
err := boa.CmdT[Params]{
	Use: "cmd",
	RunFuncE: func(p *Params, cmd *cobra.Command, args []string) error {
		return fmt.Errorf("something went wrong")
	},
}.RunE() // err contains "something went wrong"

if err != nil {
	// Handle error programmatically
}
```

This includes all error types:
- Hook errors (`InitFunc`, `PreValidate`, `PreExecute`)
- Runtime errors from `RunFuncE`
- Configuration errors (e.g., setting multiple run functions)

#### When to Use Which

- Use `RunFunc` / `Run()` for simple CLIs where errors should terminate the program
- Use `RunFuncE` / `RunE()` when you need to:
  - Handle errors programmatically
  - Write testable command logic
  - Integrate with frameworks that expect error returns
  - Build commands that are called from other Go code

## Builder API (Alternative)

BOA also provides a fluent builder API as an alternative to the direct struct style. The builder API may be deprecated in a future release, so the direct API (shown throughout this README) is recommended for new projects.

```go
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
)

type Params struct {
	Name string
	Port int `optional:"true"`
}

func main() {
	boa.NewCmdT[Params]("my-app").
		WithShort("A CLI tool").
		WithRunFunc(func(params *Params) {
			fmt.Printf("Hello, %s!\n", params.Name)
		}).
		WithSubCmds(
			boa.NewCmdT[Params]("sub").
				WithRunFunc(func(params *Params) {
					fmt.Println("Subcommand")
				}),
		).
		Run()
}
```

The builder API provides `With*` methods for all configuration options. Both APIs are functionally equivalent and produce identical CLI behavior.

## Migration Guide

If you're migrating from the deprecated `Required[T]`/`Optional[T]` wrapper types:

### Before (Deprecated)
```go
type Params struct {
	Name boa.Required[string] `descr:"User name"`
	Port boa.Optional[int]    `descr:"Port number" default:"8080"`
}

// Accessing values
fmt.Println(params.Name.Value())       // string
fmt.Println(*params.Port.Value())      // int (via pointer)
```

### After (Recommended)
```go
type Params struct {
	Name string `descr:"User name"`                           // required by default
	Port int    `descr:"Port number" optional:"true"`
}

// Accessing values - direct access
fmt.Println(params.Name)  // string
fmt.Println(params.Port)  // int (direct value)
```

### Programmatic Configuration

For programmatic configuration that was previously done directly on wrapper types:

**Before:**
```go
params.Port.SetRequiredFn(func() bool { return params.Mode == "server" })
```

**After:**
```go
// Use HookContext in InitFuncCtx
boa.CmdT[Params]{
	Use: "app",
	InitFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
		ctx.GetParam(&p.Port).SetRequiredFn(func() bool { return p.Mode == "server" })
		return nil
	},
}
```

## Legacy API (Deprecated)

The `Required[T]` and `Optional[T]` wrapper types are deprecated but still functional for backward compatibility.

```go
// DEPRECATED - prefer plain Go types instead
type Params struct {
	Name boa.Required[string]   // Use: Name string
	Port boa.Optional[int]      // Use: Port int `optional:"true"`
}

// DEPRECATED factory functions
name := boa.Req("default")    // Use: struct tag `default:"default"`
port := boa.Opt(8080)         // Use: struct tag `default:"8080" optional:"true"`
def := boa.Default(value)     // Use: struct tag `default:"value"`
```

The wrapper types require calling `.Value()` to access values, which adds verbosity compared to direct field access.

## Missing features

- [ ] Support for custom types
- [ ] Prefixed nested config

## State

- [x] Stable API with plain Go types as the primary interface
- [x] Used in production projects
