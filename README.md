# BOA

[![CI Status](https://github.com/GiGurra/boa/actions/workflows/ci.yml/badge.svg)](https://github.com/GiGurra/boa/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/GiGurra/boa)](https://goreportcard.com/report/github.com/GiGurra/boa)

Boa adds a declarative layer on top of `github.com/spf13/cobra`.

The goal is making the process of creating a command line interface as simple as possible, while still providing access
to cobra primitives when needed.

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
		RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
			fmt.Printf(
				"Hello world with params: %v, %v, %v, %v, %v\n",
				params.Baz,          // string
				params.FB,           // string
				params.Foo.Value(),  // string
				params.Bar.Value(),  // int
				params.File.Value(), // *string
			)
		},
	}.Run()
}

```

Help output for the above:

```
A generic cli tool that has a longer description. See the README.MD for more information

Usage:
  hello-world [flags]

Flags:
  -b, --baz string     (env: BAZ, required)
  -f, --f-b string     (env: F_B)
      --foo string     (env: FOO, required)
      --bar int        (env: BAR) (default 4)
      --file string    (env: FILE)
  -h, --help          help for hello-world
```

### Sub-commands and tags

Most customization is available through field tags:

```go
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

var params struct {
	Foo  boa.Required[string] `descr:"a foo"`
	Bar  boa.Required[int]    `descr:"a bar" env:"BAR_X" default:"4"`
	Path boa.Required[string] `pos:"true"`
	Baz  boa.Required[string] `pos:"true" default:"cba"`
	FB   boa.Optional[string] `pos:"true"`
}

type OtherParams struct {
	Foo2 boa.Required[string] `descr:"a foo"`
}

func main() {
	boa.Cmd{
		Use:   "hello-world",
		Short: "a generic cli tool",
		Long:  `A generic cli tool that has a longer description.See the README.MD for more information`,
		SubCmds: boa.SubCmds(
			boa.Cmd{
				Use:         "subcommand1",
				Short:       "a subcommand",
				Params:      &params,
				ParamEnrich: boa.ParamEnricherCombine(boa.ParamEnricherName, boa.ParamEnricherEnv),
				RunFunc: func(cmd *cobra.Command, args []string) {
					p1 := params.Foo.Value()
					p2 := params.Bar.Value()
					p3 := params.Path.Value()
					p4 := params.Baz.Value()
					fmt.Printf("Hello world from subcommand1 with params: %s, %d, %s, %s\n", p1, p2, p3, p4)
				},
			},
			boa.CmdT[OtherParams]{
				Use:   "subcommand2",
				Short: "a subcommand",
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
      --foo string   a foo [required] (env: FOO)
  -h, --help         help for subcommand1
```

### Sub-commands, tags and explicit fields

Some customization is only available through explicit field definitions:

```go
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

var params = struct {
	Foo  boa.Required[string]
	Bar  boa.Required[int]    `descr:"a bar" env:"BAR_X" default:"111"`
	Path boa.Required[string] `pos:"true"`
	Baz  boa.Required[string]
	FB   boa.Optional[string] `pos:"true"`
}{
	Foo: boa.Required[string]{Descr: "a foo"},                                                          // add additional info if you like. This means we get "a foo [required] (env: FOO)" in the help text
	Bar: boa.Required[int]{Default: boa.Default(4), CustomValidator: func(x int) error { return nil }}, // optional custom validation logic
	Baz: boa.Required[string]{Positional: true, Default: boa.Default("cba")},                           // positional arguments
}

func main() {
	boa.Cmd{
		Use:   "hello-world",
		Short: "a generic cli tool",
		Long:  `A generic cli tool that has a longer description.See the README.MD for more information`,
		SubCmds: []*cobra.Command{
			boa.Cmd{
				Use:         "subcommand1",
				Short:       "a subcommand",
				Params:      &params,
				ParamEnrich: boa.ParamEnricherCombine(boa.ParamEnricherName, boa.ParamEnricherEnv),
				RunFunc: func(cmd *cobra.Command, args []string) {
					p1 := params.Foo.Value()
					p2 := params.Bar.Value()
					p3 := params.Path.Value()
					p4 := params.Baz.Value()
					fmt.Printf("Hello world from subcommand1 with params: %s, %d, %s, %s\n", p1, p2, p3, p4)
				},
			}.ToCobra(),
			boa.Cmd{
				Use:   "subcommand2",
				Short: "a subcommand",
				RunFunc: func(cmd *cobra.Command, args []string) {
					fmt.Println("Hello world from subcommand2")
				},
			}.ToCobra(),
		},
	}.Run()
}
```

Help output for the above:

```
a subcommand

Usage:
  hello-world subcommand1 <path> <baz> [f-b] [flags]

Flags:
      --bar int      a bar (env: BAR_X) (default 111)
      --foo string   a foo [required] (env: FOO)
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
	Foo  boa.Required[string]
	Bar  boa.Required[int]
	File boa.Required[string]
}

type Base2 struct {
	Foo2  boa.Required[string]
	Bar2  boa.Required[int]
	File2 boa.Required[string]
}

type Base3 struct {
	Foo3  boa.Required[string]
	Bar3  boa.Required[int]
	File3 boa.Required[string]
}

type Base4 struct {
	Foo24  boa.Required[string]
	Bar24  boa.Required[int]
	File24 boa.Required[string]
}

var combined struct {
	Base Base1
	Base2
	Base3
	Base4
	Baz  boa.Required[string]
	FB   boa.Optional[string]
	Time boa.Optional[time.Time]
}

func main() {
	boa.Cmd{
		Use:    "hello-world",
		Short:  "a generic cli tool",
		Long:   `A generic cli tool that has a longer description. See the README.MD for more information`,
		Params: &combined,
		RunFunc: func(cmd *cobra.Command, args []string) {
			fmt.Printf(
				"Hello world from subcommand1 with params: %s, %d, %s, %s, %v, %v\n",
				params.Base.Foo.Value(),  // string
				params.Base.Bar.Value(),  // int
				params.Base.File.Value(), // string
				params.Baz.Value(),       // string
				params.FB.Value(),        // *string
				params.Time.Value(),      // *time.Time
			)
		},
	}.Run()
}
```

### Leverage all of cobra's features:

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
		Long:  `A generic cli tool that has a longer description. See the README.MD for more information`,
		InitFunc: func(params *Params, cmd *cobra.Command) error {
			cmd.Deprecated = "this command is deprecated"
			return nil
		},
		RunFunc: func(params *Params, _ *cobra.Command, _ []string) {
			fmt.Printf(
				"Hello world with params: %s, %d, %s, %s, %v\n",
				params.Baz, // string
				params.FB,  // *string
			)
		},
	}.Run()
}
```

### Conditional parameters

You can make parameters conditionally required or enabled:

```go
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

func main() {
	var params = struct {
		Foo boa.Optional[string]
		Bar boa.Optional[int]
		Baz boa.Optional[string]
	}{}

	// Bar is only enabled if Foo has a value
	params.Bar.SetIsEnabledFn(func() bool {
		return params.Foo.HasValue()
	})

	// Baz is required if Foo has a value
	params.Baz.SetRequiredFn(func() bool {
		return params.Foo.HasValue()
	})

	boa.Cmd{
		Use:    "hello-world",
		Short:  "a generic cli tool",
		Long:   `A generic cli tool that has a longer description.`,
		Params: &params,
		RunFunc: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Hello World!\n")
		},
	}.Run()
}
```

### Constraining parameter values

You can specify that a parameter must be one of a set of values:

```go
package main

import (
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

var params = struct {
	Foo boa.Required[string] `alts:"abc,cde,fgh"`
}{}

```

### Array/slice parameters

Boa supports array/slice types with proper parsing:

```go
package main

import (
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

var params struct {
	WithoutDefaults boa.Required[[]float64]
	WithDefaults    boa.Required[[]int64] `default:"[1,2,3]"`
}

```

### Fluent builder API

A structured builder API is available for more complex command creation:

```go
package main

import (
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

type TestStruct struct {
	Flag1 boa.Required[string]
	Flag2 boa.Required[int]
}

func main() {
	cmd := boa.NewCmdT[TestStruct]("my-command").
		WithShort("A command description").
		WithLong("A longer command description").
		WithRunFunc(func(params *TestStruct) {
			fmt.Printf("Running with: %s, %d\n",
				params.Flag1.Value(),
				params.Flag2.Value(),
			)
		}).
		WithSubCmds(
			boa.NewCmdT[TestStruct]("subcommand1"),
			//...etc
		)

	cmd.Run()
}

```

### Config file serialization and configuration

```go
package main

import (
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

type AppConfig struct {
	Host boa.Required[string]
	Port boa.Required[int]
}

type ConfigFromFile struct {
	File boa.Required[string]
	AppConfig
}

func main() {
	boa.NewCmdT[ConfigFromFile]("my-app").
		WithPreValidateFuncE(func(params *ConfigFromFile, cmd *cobra.Command, args []string) error {
			// boa.UnMarshalFromFileParam is a helper to unmarshal from a file, but you can run
			// any custom code here.
			// boa.Optional and boa.Required have implementations of json.Unmarshaler.
			// This implementation will also respect pre-assigned cli and env var values, 
			// and not overwrite them with the file values.
			return boa.UnMarshalFromFileParam(&params.File, &params.AppConfig, nil /* custom unmarshaller function */)
		}).
		WithRunFunc(func(params *ConfigFromFile) {
			// Use parameters loaded from the file
			fmt.Printf("Host: %s, Port: %d\n",
				params.Host.Value(),
				params.Port.Value(),
			)
		}).
		Run()
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
   parameter type will be used. If you are using the `boa.Required` or `boa.Optional` types, you should use the
   `HasValue` method to check if a value has been set.

## Lifecycle Hooks in Boa

Boa provides several lifecycle hooks that can be implemented or defined to customize behavior at different stages of
command execution. These hooks give you fine-grained control over parameter initialization, validation, and execution.

### Init Hook

The Init hook runs during the initialization phase, before any command-line arguments or environment variables are
processed.

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

// Alternatively, use the InitFunc in Cmd
func main() {
	boa.Cmd{
		Params: &params,
		InitFunc: func(params any) error {
			// Custom initialization logic
			return nil
		},
	}.Run()

	// Or with the builder API
	boa.NewCmdT[MyConfigStruct]("command").
		WithInitFuncE(func(params *MyConfigStruct) error {
			// Custom initialization logic
			return nil
		})
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

// Alternatively, use the PreValidateFunc in Cmd
func main() {
	boa.Cmd{
		Params: &params,
		PreValidateFunc: func(params any, cmd *cobra.Command, args []string) error {
			// Custom pre-validation logic
			return nil
		},
	}.Run()

	// Or with the builder API
	boa.NewCmdT[MyConfigStruct]("command").
		WithPreValidateFuncE(func(params *MyConfigStruct, cmd *cobra.Command, args []string) error {
			// Custom pre-validation logic, such as loading from config files
			return nil
		})
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

// Alternatively, use the PreExecuteFunc in Cmd
func main() {
	boa.Cmd{
		Params: &params,
		PreExecuteFunc: func(params any, cmd *cobra.Command, args []string) error {
			// Custom pre-execution logic
			return nil
		},
	}.Run()

	// Or with the builder API
	boa.NewCmdT[MyConfigStruct]("command").
		WithPreExecuteFuncE(func(params *MyConfigStruct, cmd *cobra.Command, args []string) error {
			// Custom pre-execution logic
			return nil
		})
}

```

### Hook Execution Order

Hooks are executed in the following order:

1. **Init** - During command initialization, before any flags are parsed
2. **PreValidate** - After flags are parsed but before validation
3. **Validation** - Built-in parameter validation
4. **PreExecute** - After validation but before command execution
5. **Run** - The actual command execution

### Common Use Cases

- **Init**: Set up default values, configure custom validators
- **PreValidate**: Load configurations from files, set derived parameters
- **PreExecute**: Establish connections, prepare resources needed for execution

### Error Handling

All hooks can return errors to abort command execution. If any hook returns an error, the command will not proceed to
the next phase, and the error will be reported to the user.

## Experimental/Work in progress

### Raw types

It is currently being evaluated if a sufficiently good API can be provided using raw types. Boa currently has an
implementation of this, but it is not very well tested, and relies on golang's zero value semantics to work.

```go
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

var params struct {
	Foo  string `descr:"a foo"`
	Bar  int    `descr:"a bar" env:"BAR_X" default:"4" required:"false"`
	Path string `pos:"true"`
	Baz  string `pos:"true" default:"cba"`
	FB   string `pos:"true"`
}

func main() {
	boa.Cmd{
		Use:    "hello-world",
		Short:  "a generic cli tool",
		Long:   `A generic cli tool that has a longer description. See the README.MD for more information`,
		Params: &params,
		RunFunc: func(cmd *cobra.Command, args []string) {
			fmt.Printf(
				"Hello world with params: %s, %d, %s, %s, %v\n",
				params.Foo,  // string
				params.Bar,  // int
				params.Path, // string
				params.Baz,  // string
				params.FB,   // *string
			)
		},
	}.Run()
}
```

## Missing features

- [ ] Prefixed nested config
- [ ] Support for custom types as slice elements
- [ ] Support for more complex types as parameters (e.g. maps, custom types)

## State

- [x] Pretty early. Use at your own risk. I'm using it in most of my own projects and some production code at work.
