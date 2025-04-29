# BOA

Boa is a compact CLI and environment variable parameter utility. It enhances and simplifies aspects of `github.com/spf13/cobra`, facilitating the creation of straightforward and declarative CLI interfaces.

The primary goal of Boa is to maintain a declarative approach. In its simplest form, you only need to define a struct with parameter fields, and Boa handles the rest.

## Features

* **Declarative Design**: Boa allows for fully declarative definition and validation.
* **Optional Values**: Boa supports true optional values and provides knowledge if a field was set. It also offers opt-in default values built into the type system.
    * A `boa.Required[string]`'s `.Value()` is type aware and returns a `string`.
    * A `boa.Optional[string]`'s `.Value()` is type aware and returns a `*string`.
* **Auto-Generated Properties**: Boa generates flag/param properties from field name, type, tags, and more.
    * For instance, `Foo boa.Required[string]` will generate:
        * flags `--foo` (and short version `-f` if it is not already taken)
        * `FOO` env var mapping
        * `[required] (env: FOO)` in the help text
        * You can supplement this with your own help text, custom generation logic, etc.
    * You can opt out of auto generation, override specific properties, and cherry-pick and/or add your own auto-generation logic.
* **Input Validation**: Boa validates all inputs before the `Run` function is invoked.
* **Config Flexibility**: Use explicit fields for config or tags as per your preference.
* **Cobra Compatibility**: Mix and match Boa with regular Cobra code as you see fit. Boa works with regular Cobra commands.
* **Conditional Parameters**: Parameters can be conditionally required or enabled based on other parameter values.
* **Flag Alternatives**: Support for alternative flag names and auto-completion.
* **Slices Support**: Handle array types like `[]string`, `[]int`, etc. with proper parsing.
* **Time Support**: Native support for `time.Time` parameters.
* **Config file Support**: Built-in capability to marshal/unmarshal configurations/sub-configurations to/from JSON or other formats.
* **Structured Builder API**: A fluent API for building commands.
* **Custom Validation**: Provide custom validation functions for parameters.
* **Initialization and Lifecycle Hooks**: Support for initialization, pre-validation, and pre-execution hooks.

## Installation

To install Boa, use the following command:

`go get github.com/GiGurra/boa@latest`

## Usage

Refer to the code snippets provided below for minimum setup, sub-commands and tags, and sub-commands, tags and explicit fields.

### Minimum setup

```go
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

var params struct {
	Foo  boa.Required[string]
	Bar  boa.Required[int]
	File boa.Required[string]
	Baz  boa.Required[string]
	FB   boa.Optional[string]
}

func main() {
	boa.Wrap{
		Use:    "hello-world",
		Short:  "a generic cli tool",
		Long:   `A generic cli tool that has a longer description. See the README.MD for more information`,
		Params: &params,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf(
				"Hello world with params: %s, %d, %s, %s, %v\n",
				params.Foo.Value(),  // string
				params.Bar.Value(),  // int
				params.File.Value(), // string
				params.Baz.Value(),  // string
				params.FB.Value(),   // *string
			)
		},
	}.ToApp()
}
```

Help output for the above:

```
A generic cli tool that has a longer description. See the README.MD for more information

Usage:
  hello-world [flags]

Flags:
  -b, --bar int        [required] (env: BAR)
      --baz string     [required] (env: BAZ)
      --f-b string     (env: F_B)
      --file string    [required] (env: FILE)
  -f, --foo string     [required] (env: FOO)
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

func main() {
	boa.Wrap{
		Use:   "hello-world",
		Short: "a generic cli tool",
		Long:  `A generic cli tool that has a longer description.See the README.MD for more information`,
		SubCommands: []*cobra.Command{
			boa.Wrap{
				Use:         "subcommand1",
				Short:       "a subcommand",
				Params:      &params,
				ParamEnrich: boa.ParamEnricherCombine(boa.ParamEnricherName, boa.ParamEnricherEnv),
				Run: func(cmd *cobra.Command, args []string) {
					p1 := params.Foo.Value()
					p2 := params.Bar.Value()
					p3 := params.Path.Value()
					p4 := params.Baz.Value()
					fmt.Printf("Hello world from subcommand1 with params: %s, %d, %s, %s\n", p1, p2, p3, p4)
				},
			}.ToCmd(),
			boa.Wrap{
				Use:   "subcommand2",
				Short: "a subcommand",
				Run: func(cmd *cobra.Command, args []string) {
					fmt.Println("Hello world from subcommand2")
				},
			}.ToCmd(),
		},
	}.ToApp()
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
	boa.Wrap{
		Use:   "hello-world",
		Short: "a generic cli tool",
		Long:  `A generic cli tool that has a longer description.See the README.MD for more information`,
		SubCommands: []*cobra.Command{
			boa.Wrap{
				Use:         "subcommand1",
				Short:       "a subcommand",
				Params:      &params,
				ParamEnrich: boa.ParamEnricherCombine(boa.ParamEnricherName, boa.ParamEnricherEnv),
				Run: func(cmd *cobra.Command, args []string) {
					p1 := params.Foo.Value()
					p2 := params.Bar.Value()
					p3 := params.Path.Value()
					p4 := params.Baz.Value()
					fmt.Printf("Hello world from subcommand1 with params: %s, %d, %s, %s\n", p1, p2, p3, p4)
				},
			}.ToCmd(),
			boa.Wrap{
				Use:   "subcommand2",
				Short: "a subcommand",
				Run: func(cmd *cobra.Command, args []string) {
					fmt.Println("Hello world from subcommand2")
				},
			}.ToCmd(),
		},
	}.ToApp()
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

var base3 struct {
	Foo3  boa.Required[string]
	Bar3  boa.Required[int]
	File3 boa.Required[string]
}

var base4 struct {
	Foo24  boa.Required[string]
	Bar24  boa.Required[int]
	File24 boa.Required[string]
}

var params struct {
	Base Base1
	Base2
	Baz  boa.Required[string]
	FB   boa.Optional[string]
	Time boa.Optional[time.Time]
}

func main() {
	boa.Wrap{
		Use:    "hello-world",
		Short:  "a generic cli tool",
		Long:   `A generic cli tool that has a longer description. See the README.MD for more information`,
		Params: boa.Compose(&params, &base3, &base4),
		Run: func(cmd *cobra.Command, args []string) {
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
	}.ToApp()
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

	boa.Wrap{
		Use:    "hello-world",
		Short:  "a generic cli tool",
		Long:   `A generic cli tool that has a longer description.`,
		Params: &params,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Hello World!\n")
		},
	}.ToApp()
}
```

### Alternative flag names

You can specify alternative flag names:

```go
var params = struct {
	Foo boa.Required[string] `alts:"abc,cde,fgh"`
}{}
```

### Array/slice parameters

Boa supports array/slice types with proper parsing:

```go
var params struct {
	WithoutDefaults boa.Required[[]float64]
	WithDefaults    boa.Required[[]int64] `default:"[1,2,3]"`
}
```

### Fluent builder API

A structured builder API is available for more complex command creation:

```go
import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
)

type TestStruct struct {
	Flag1 boa.Required[string]
	Flag2 boa.Required[int]
}

func main() {
	builder := boa.NewCmdBuilder[TestStruct]("my-command").
		WithShort("A command description").
		WithLong("A longer command description").
		WithRunFunc(func(params *TestStruct) {
			fmt.Printf("Running with: %s, %d\n",
				params.Flag1.Value(),
				params.Flag2.Value(),
			)
		})

	builder.Run()
}
```

### Config file serialization and configuration

```go
import (
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

type AppConfig struct {
	Host boa.Required[string]
	Port boa.Required[int]
}

type ConfigFromFile struct {
	File    boa.Required[string]
	AppConfig
}

func main() {
	boa.NewCmdBuilder[ConfigFromFile]("my-app").
		WithPreValidateFuncE(func(params *ConfigFromFile, cmd *cobra.Command, args []string) error {
			// boa.UnMarshalFromFileParam is a helper to unmarshal from a file, but you can run
			// any custom code here.
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
# Lifecycle Hooks in Boa

Boa provides several lifecycle hooks that can be implemented or defined to customize behavior at different stages of command execution. These hooks give you fine-grained control over parameter initialization, validation, and execution.

## Available Hooks

### Init Hook

The Init hook runs during the initialization phase, before any command-line arguments or environment variables are processed.

```go
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

// Alternatively, use the InitFunc in Wrap
boa.Wrap{
    Params: &params,
    InitFunc: func(params any) error {
        // Custom initialization logic
        return nil
    },
}.ToApp()

// Or with the builder API
boa.NewCmdBuilder[MyConfigStruct]("command").
    WithInitFuncE(func(params *MyConfigStruct) error {
        // Custom initialization logic
        return nil
    })
```

### PreValidate Hook

The PreValidate hook runs after parameters are parsed from the command line and environment variables but before validation is performed.

```go
// Implement this interface on your configuration struct
type CfgStructPreValidate interface {
    PreValidate() error
}

// Example implementation
func (i *MyConfigStruct) PreValidate() error {
    // Manipulate parameters before validation
    return nil
}

// Alternatively, use the PreValidateFunc in Wrap
boa.Wrap{
    Params: &params,
    PreValidateFunc: func(params any, cmd *cobra.Command, args []string) error {
        // Custom pre-validation logic
        return nil
    },
}.ToApp()

// Or with the builder API
boa.NewCmdBuilder[MyConfigStruct]("command").
    WithPreValidateFuncE(func(params *MyConfigStruct, cmd *cobra.Command, args []string) error {
        // Custom pre-validation logic, such as loading from config files
        return nil
    })
```

### PreExecute Hook

The PreExecute hook runs after parameter validation but before the command's Run function is executed.

```go
// Implement this interface on your configuration struct
type CfgStructPreExecute interface {
    PreExecute() error
}

// Example implementation
func (i *MyConfigStruct) PreExecute() error {
    // Setup that should happen after validation but before execution
    return nil
}

// Alternatively, use the PreExecuteFunc in Wrap
boa.Wrap{
    Params: &params,
    PreExecuteFunc: func(params any, cmd *cobra.Command, args []string) error {
        // Custom pre-execution logic
        return nil
    },
}.ToApp()

// Or with the builder API
boa.NewCmdBuilder[MyConfigStruct]("command").
    WithPreExecuteFuncE(func(params *MyConfigStruct, cmd *cobra.Command, args []string) error {
        // Custom pre-execution logic
        return nil
    })
```

## Hook Execution Order

Hooks are executed in the following order:

1. **Init** - During command initialization, before any flags are parsed
2. **PreValidate** - After flags are parsed but before validation
3. **Validation** - Built-in parameter validation
4. **PreExecute** - After validation but before command execution
5. **Run** - The actual command execution

## Common Use Cases

- **Init**: Set up default values, configure custom validators
- **PreValidate**: Load configurations from files, set derived parameters
- **PreExecute**: Establish connections, prepare resources needed for execution

## Error Handling

All hooks can return errors to abort command execution. If any hook returns an error, the command will not proceed to the next phase, and the error will be reported to the user.

## Missing features

- [ ] Nested config

## State

- [x] Pretty early. Use at your own risk. I'm using it in most of my own projects and some production code at work.
