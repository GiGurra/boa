# Getting Started

## Installation

```bash
go get github.com/GiGurra/boa@latest
```

## Basic Usage

### Minimum Setup

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

Help output:

```
A generic cli tool that has a longer description

Usage:
  hello-world <path> <baz> [f-b] [flags]

Flags:
  -b, --bar int      a bar (env: BAR_X)
  -f, --foo string   a foo (required)
  -h, --help         help for hello-world
```

## Sub-commands

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

## Struct Composition

Compose structs to create complex parameter structures:

```go
type Base struct {
    Foo  string
    Bar  int
    File string
}

type DBConfig struct {
    Host string `default:"localhost"`
    Port int    `default:"5432"`
}

type Combined struct {
    Base               // embedded (anonymous): --foo, --bar, --file (no prefix)
    DB     DBConfig    // named field: --db-host, --db-port (auto-prefixed)
    Baz    string
    Time   time.Time `optional:"true"`
}
```

**Embedded (anonymous) fields** are not prefixed. `Base.Foo` becomes `--foo`, not `--base-foo`.

**Named struct fields** auto-prefix their children with the field name in kebab-case. `DB.Host` becomes `--db-host`. This prevents collisions when the same struct type is used in multiple fields:

```go
type Params struct {
    Primary DBConfig  // --primary-host, --primary-port
    Replica DBConfig  // --replica-host, --replica-port
}
```

Deep nesting chains prefixes: `Infra.Primary.Host` becomes `--infra-primary-host`. Explicit `name:"..."` and `env:"..."` tags also get prefixed inside named fields. See [Struct Tags](struct-tags.md#named-struct-auto-prefixing) for full details.

## Array/Slice Parameters

BOA supports array/slice types:

```go
type Params struct {
    Numbers []int    `descr:"list of numbers"`
    Tags    []string `descr:"tags" default:"[a,b,c]"`
    Ports   []int64  `descr:"ports" default:"[8080,8081,8082]"`
}
```

## Pointer Fields

Use pointer types for truly optional parameters where you need to distinguish "not set" from "zero value":

```go
type Params struct {
    Name    *string `descr:"user name"`   // nil if not provided
    Retries *int    `descr:"retry count"` // nil if not provided
    Verbose *bool   `descr:"verbose mode"`
}

func main() {
    boa.CmdT[Params]{
        Use: "app",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            if p.Name != nil {
                fmt.Println("Name:", *p.Name)
            } else {
                fmt.Println("No name provided")
            }
        },
    }.Run()
}
```

Pointer fields are always optional by default, even without `boa.WithDefaultOptional()`. Use `required:"true"` to override.

## Map Fields

Maps are supported as CLI flags with `key=val,key=val` syntax:

```go
type Params struct {
    Labels map[string]string `descr:"key=value labels"`
    Limits map[string]int    `descr:"resource limits"`
}
// Usage: myapp --labels env=prod,team=backend --limits cpu=4,memory=8192
```

Maps default to optional. For complex map types like `map[string][]string`, pass JSON on the CLI:

```
myapp --meta '{"tags":["a","b"],"owners":["alice"]}'
```

## Value Priority

When multiple sources provide values, BOA uses this priority order:

1. **Command-line flags** - Highest priority
2. **Environment variables**
3. **Config files** (via `configfile` tag or PreValidate hook)
4. **Default values**
5. **Zero value** - Lowest priority

## Config File Support

Tag a string field with `configfile:"true"` and boa loads it automatically:

```go
type Params struct {
    ConfigFile string `configfile:"true" optional:"true" default:"config.json"`
    Host       string
    Port       int
}

func main() {
    boa.CmdT[Params]{
        Use: "my-app",
        RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Host: %s, Port: %d\n", params.Host, params.Port)
        },
    }.Run()
}
```

The config file (JSON by default) is loaded before validation. CLI and env var values always take precedence over config file values. See [Advanced Usage](advanced.md#config-file-loading) for custom formats and the explicit `LoadConfigFile` API.

## Accessing Cobra

Access the underlying Cobra command for advanced customization:

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
