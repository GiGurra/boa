# Getting Started

## Installation

```bash
go get github.com/GiGurra/boa@latest
```

## Basic Usage

### Minimum Setup

=== "Direct API"

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
            Long:  "A generic cli tool that has a longer description",
            RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
                fmt.Printf("Hello world with params: %s, %d, %s, %s, %s\n",
                    params.Foo, params.Bar, params.Path, params.Baz, params.FB)
            },
        }.Run()
    }
    ```

=== "Builder API"

    ```go
    package main

    import (
        "fmt"
        "github.com/GiGurra/boa/pkg/boa"
    )

    type Params struct {
        Foo  string `descr:"a foo"`
        Bar  int    `descr:"a bar" env:"BAR_X" optional:"true"`
        Path string `positional:"true"`
        Baz  string `positional:"true" default:"cba"`
        FB   string `positional:"true" optional:"true"`
    }

    func main() {
        boa.NewCmdT[Params]("hello-world").
            WithShort("a generic cli tool").
            WithLong("A generic cli tool that has a longer description").
            WithRunFunc(func(params *Params) {
                fmt.Printf("Hello world with params: %s, %d, %s, %s, %s\n",
                    params.Foo, params.Bar, params.Path, params.Baz, params.FB)
            }).
            Run()
    }
    ```

Help output:

```
A generic cli tool that has a longer description

Usage:
  hello-world <path> <baz> [f-b] [flags]

Flags:
      --bar int      a bar (env: BAR_X) (default 4)
  -f, --foo string   a foo (env: FOO, required)
  -h, --help         help for hello-world
```

## Sub-commands

Create hierarchical CLI tools with sub-commands:

=== "Direct API"

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
    }

    type OtherParams struct {
        Foo2 string `descr:"a foo"`
    }

    func main() {
        boa.CmdT[boa.NoParams]{
            Use:   "hello-world",
            Short: "a generic cli tool",
            SubCmds: boa.SubCmds(
                boa.CmdT[SubParams]{
                    Use:   "subcommand1",
                    Short: "a subcommand",
                    RunFunc: func(params *SubParams, cmd *cobra.Command, args []string) {
                        fmt.Printf("Hello from subcommand1: %s, %d, %s\n",
                            params.Foo, params.Bar, params.Path)
                    },
                },
                boa.CmdT[OtherParams]{
                    Use:   "subcommand2",
                    Short: "another subcommand",
                    RunFunc: func(params *OtherParams, cmd *cobra.Command, args []string) {
                        fmt.Println("Hello from subcommand2")
                    },
                },
            ),
        }.Run()
    }
    ```

=== "Builder API"

    ```go
    package main

    import (
        "fmt"
        "github.com/GiGurra/boa/pkg/boa"
    )

    type SubParams struct {
        Foo  string `descr:"a foo"`
        Bar  int    `descr:"a bar" env:"BAR_X" default:"4"`
        Path string `positional:"true"`
    }

    type OtherParams struct {
        Foo2 string `descr:"a foo"`
    }

    func main() {
        boa.NewCmdT[boa.NoParams]("hello-world").
            WithShort("a generic cli tool").
            WithSubCmds(
                boa.NewCmdT[SubParams]("subcommand1").
                    WithShort("a subcommand").
                    WithRunFunc(func(params *SubParams) {
                        fmt.Printf("Hello from subcommand1: %s, %d, %s\n",
                            params.Foo, params.Bar, params.Path)
                    }),
                boa.NewCmdT[OtherParams]("subcommand2").
                    WithShort("another subcommand").
                    WithRunFunc(func(params *OtherParams) {
                        fmt.Println("Hello from subcommand2")
                    }),
            ).
            Run()
    }
    ```

## Struct Composition

Compose structs to create complex parameter structures:

```go
type Base struct {
    Foo  string
    Bar  int
    File string
}

type Combined struct {
    Base
    Baz  string
    Time time.Time `optional:"true"`
}
```

!!! note
    Nested struct fields use their own field names as flags, not prefixed with the parent struct name.
    For example, `Base.Foo` becomes `--foo`, not `--base-foo`.

## Array/Slice Parameters

BOA supports array/slice types:

```go
type Params struct {
    Numbers []int    `descr:"list of numbers"`
    Tags    []string `descr:"tags" default:"[a,b,c]"`
    Ports   []int64  `descr:"ports" default:"[8080,8081,8082]"`
}
```

## Value Priority

When multiple sources provide values, BOA uses this priority order:

1. **Command-line flags** - Highest priority
2. **Environment variables**
3. **Config files** (when using PreValidate hook)
4. **Default values**
5. **Zero value** - Lowest priority

## Config File Support

Load configuration from files using the PreValidate hook:

=== "Direct API"

    ```go
    type AppConfig struct {
        Host string
        Port int
    }

    type ConfigFromFile struct {
        File string `descr:"config file path"`
        AppConfig
    }

    func main() {
        boa.CmdT[ConfigFromFile]{
            Use: "my-app",
            PreValidateFuncCtx: func(ctx *boa.HookContext, params *ConfigFromFile, cmd *cobra.Command, args []string) error {
                fileParam := ctx.GetParam(&params.File)
                return boa.UnMarshalFromFileParam(fileParam, &params.AppConfig, nil)
            },
            RunFunc: func(params *ConfigFromFile, cmd *cobra.Command, args []string) {
                fmt.Printf("Host: %s, Port: %d\n", params.Host, params.Port)
            },
        }.Run()
    }
    ```

=== "Builder API"

    ```go
    type AppConfig struct {
        Host string
        Port int
    }

    type ConfigFromFile struct {
        File string `descr:"config file path"`
        AppConfig
    }

    func main() {
        boa.NewCmdT[ConfigFromFile]("my-app").
            WithPreValidateFuncCtx(func(ctx *boa.HookContext, params *ConfigFromFile, cmd *cobra.Command, args []string) error {
                fileParam := ctx.GetParam(&params.File)
                return boa.UnMarshalFromFileParam(fileParam, &params.AppConfig, nil)
            }).
            WithRunFunc(func(params *ConfigFromFile) {
                fmt.Printf("Host: %s, Port: %d\n", params.Host, params.Port)
            }).
            Run()
    }
    ```

## Accessing Cobra

Access the underlying Cobra command for advanced customization:

=== "Direct API"

    ```go
    boa.CmdT[Params]{
        Use: "hello-world",
        InitFunc: func(params *Params, cmd *cobra.Command) error {
            cmd.Deprecated = "this command is deprecated"
            return nil
        },
        RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
            // ...
        },
    }.Run()
    ```

=== "Builder API"

    ```go
    boa.NewCmdT[Params]("hello-world").
        WithInitFunc2E(func(params *Params, cmd *cobra.Command) error {
            cmd.Deprecated = "this command is deprecated"
            return nil
        }).
        WithRunFunc(func(params *Params) {
            // ...
        }).
        Run()
    ```
