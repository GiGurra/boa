# BOA

[![CI Status](https://github.com/GiGurra/boa/actions/workflows/ci.yml/badge.svg)](https://github.com/GiGurra/boa/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/GiGurra/boa)](https://goreportcard.com/report/github.com/GiGurra/boa)

Boa is a declarative CLI framework for Go built on top of [cobra](https://github.com/spf13/cobra). Define your CLI parameters as plain Go structs and let boa generate flags, env var bindings, validation, and help text automatically.

**[Full Documentation](https://gigurra.github.io/boa/)**

## Installation

```bash
go get github.com/GiGurra/boa@latest
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    Name string `descr:"your name"`
    Port int    `descr:"port number" default:"8080" optional:"true"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "my-app",
        Short: "a simple CLI tool",
        RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Hello %s on port %d\n", params.Name, params.Port)
        },
    }.Run()
}
```

```
Usage:
  my-app [flags]

Flags:
  -h, --help          help for my-app
  -n, --name string   your name (required)
  -p, --port int      port number (default 8080)
```

## Parameter Types

### Plain types (required by default)

```go
type Params struct {
    Host    string        `descr:"server host"`
    Port    int           `descr:"port number"`
    Verbose bool          `descr:"verbose output" optional:"true"`
    Timeout time.Duration `descr:"request timeout" default:"30s" optional:"true"`
}
```

### Pointer types (optional by default, nil = not set)

```go
type Params struct {
    Name  *string `descr:"user name"`              // nil if not provided
    Count *int    `descr:"count" required:"true"`   // override: make it required
}
```

### Slices

```go
type Params struct {
    Tags  []string `descr:"tags" default:"[a,b,c]"`
    Ports []int    `descr:"ports"`
}
// Usage: --tags a,b,c --ports 8080,8081
```

### Maps (optional by default)

```go
type Params struct {
    Labels map[string]string `descr:"key=value labels"`
    Limits map[string]int    `descr:"resource limits"`
}
// Usage: --labels env=prod,team=backend --limits cpu=4,memory=8192
```

### Complex types (JSON on CLI)

Any type without native pflag support uses JSON parsing automatically:

```go
type Params struct {
    Matrix [][]int             `descr:"nested matrix" optional:"true"`
    Meta   map[string][]string `descr:"metadata" optional:"true"`
}
// Usage: --matrix '[[1,2],[3,4]]' --meta '{"tags":["a","b"]}'
```

### Positional arguments

```go
type Params struct {
    Input  string `positional:"true"`
    Output string `positional:"true" default:"out.txt"`
    Extra  string `positional:"true" optional:"true"`
}
// Usage: my-app <input> <output> [extra]
```

## Struct Tags Reference

| Tag | Description | Example |
|-----|-------------|---------|
| `descr` / `desc` / `description` / `help` | Description text | `descr:"User name"` |
| `name` / `long` | Override flag name | `name:"user-name"` |
| `default` | Default value | `default:"8080"` |
| `env` | Environment variable name | `env:"PORT"` |
| `short` | Short flag (single char) | `short:"p"` |
| `positional` / `pos` | Marks positional argument | `positional:"true"` |
| `required` / `req` | Marks as required | `required:"true"` |
| `optional` / `opt` | Marks as optional | `optional:"true"` |
| `alts` / `alternatives` | Allowed values (enum) | `alts:"debug,info,warn,error"` |
| `strict-alts` / `strict` | Validate against alts | `strict:"true"` |
| `configfile` | Auto-load config from this field's path | `configfile:"true"` |
| `boa` | Special directives | `boa:"ignore"` |

### Auto-generated names

- Field `MyParam` becomes flag `--my-param` (kebab-case)
- Acronyms handled correctly: `DBHost` becomes `--db-host`, `HTTPPort` becomes `--http-port`
- Environment variable: `MY_PARAM` (UPPER_SNAKE_CASE)

## Config Files

Tag a string field with `configfile:"true"` and boa loads the file automatically before validation:

```go
type Params struct {
    ConfigFile string `configfile:"true" optional:"true" default:"config.json"`
    Host       string
    Port       int
    Internal   [][]string `boa:"ignore"` // config-file only, no CLI flag
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

- CLI and env var values always take precedence over config file values.
- Use `boa:"ignore"` to mark fields that should only be loaded from the config file (no CLI flag, no env var).
- For manual control, use `boa.LoadConfigFile` in a `PreValidateFunc` hook.

### Substruct Config Files

Nested structs can have their own `configfile:"true"` field. Each substruct loads its own config file independently:

```go
type DBConfig struct {
    ConfigFile string `configfile:"true" optional:"true"`
    Host       string `default:"localhost"`
    Port       int    `default:"5432"`
}

type Params struct {
    ConfigFile string   `configfile:"true" optional:"true" default:"config.json"`
    DB         DBConfig
}
```

Priority: CLI > env > root config > substruct config > defaults.

### Config Format Registry

JSON is the only format shipped by default. Register additional formats by file extension:

```go
import "gopkg.in/yaml.v3"

boa.RegisterConfigFormat(".yaml", yaml.Unmarshal)
boa.RegisterConfigFormat(".toml", toml.Unmarshal)
```

Resolution order for unmarshal function: explicit `ConfigUnmarshal` on the command > registered format by file extension > `json.Unmarshal` fallback.

## Value Priority

1. **CLI flags** -- highest
2. **Environment variables**
3. **Root config file**
4. **Substruct config files**
5. **Default values**
6. **Zero value** -- lowest

## Subcommands

```go
func main() {
    boa.CmdT[boa.NoParams]{
        Use:   "my-app",
        Short: "a multi-command CLI",
        SubCmds: boa.SubCmds(
            boa.CmdT[ServeParams]{
                Use:   "serve",
                Short: "start the server",
                RunFunc: func(p *ServeParams, cmd *cobra.Command, args []string) {
                    fmt.Printf("Serving on %s:%d\n", p.Host, p.Port)
                },
            },
            boa.CmdT[DeployParams]{
                Use:   "deploy",
                Short: "deploy the app",
                RunFunc: func(p *DeployParams, cmd *cobra.Command, args []string) {
                    fmt.Println("Deploying...")
                },
            },
        ),
    }.Run()
}
```

## Enrichers

The `ParamEnrich` field controls automatic parameter enrichment:

| Value | Behavior |
|-------|----------|
| `nil` (default) | `ParamEnricherDefault` -- derives flag names, short flags, and bool defaults |
| `ParamEnricherNone` | No enrichment -- you must specify everything via struct tags |

`ParamEnricherDefault` includes `ParamEnricherName`, `ParamEnricherShort`, and `ParamEnricherBool`. Environment variable binding is **not** included by default. Add it explicitly:

```go
boa.CmdT[Params]{
    Use: "cmd",
    ParamEnrich: boa.ParamEnricherCombine(
        boa.ParamEnricherName,
        boa.ParamEnricherShort,
        boa.ParamEnricherEnv,                  // auto env vars
        boa.ParamEnricherEnvPrefix("MYAPP"),   // optional: MYAPP_MY_PARAM
        boa.ParamEnricherBool,
    ),
    // ...
}
```

## Global Configuration

```go
func main() {
    boa.Init(
        boa.WithDefaultOptional(), // plain fields default to optional instead of required
    )
    boa.CmdT[Params]{Use: "my-app", /* ... */}.Run()
}
```

Explicit struct tags (`required`, `optional`) always take precedence.

## Hooks

Boa provides lifecycle hooks for customizing behavior at each stage:

| Hook | When it runs |
|------|-------------|
| `InitFunc` / `InitFuncCtx` | After param mirrors created, before cobra flags registered |
| `PostCreateFunc` / `PostCreateFuncCtx` | After cobra flags created, before parsing |
| `PreValidateFunc` / `PreValidateFuncCtx` | After parsing, before validation |
| `PreExecuteFunc` / `PreExecuteFuncCtx` | After validation, before execution |

The `Ctx` variants provide a `*boa.HookContext` for programmatic parameter configuration:

```go
boa.CmdT[Params]{
    Use: "cmd",
    InitFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
        ctx.GetParam(&p.FilePath).SetRequiredFn(func() bool {
            return p.Mode == "file"
        })
        ctx.GetParam(&p.LogLevel).SetAlternatives([]string{"debug", "info", "warn", "error"})
        ctx.GetParam(&p.LogLevel).SetDefault(boa.Default("info"))
        return nil
    },
    // ...
}
```

Parameter structs can also implement hook interfaces directly (`CfgStructInit`, `CfgStructInitCtx`, `CfgStructPreValidate`, etc.).

See [Hooks](https://gigurra.github.io/boa/hooks/) for details.

## Checking if a Value Was Set

Use `RunFuncCtx` to check whether optional parameters were explicitly provided:

```go
boa.CmdT[Params]{
    Use: "server",
    RunFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command, args []string) {
        if ctx.HasValue(&p.Port) {
            fmt.Printf("Port explicitly set to %d\n", p.Port)
        }
    },
}
```

## Error Handling

| Method | Behavior |
|--------|----------|
| `Run()` | User input errors exit(1), other errors panic |
| `RunE()` | Returns errors for programmatic handling |
| `RunArgs(args)` | Like `Run()` with custom args |
| `RunArgsE(args)` | Like `RunE()` with custom args |
| `ToCobra()` | Returns `*cobra.Command` (panics on setup error) |
| `ToCobraE()` | Returns `(*cobra.Command, error)` |

Use `RunFunc` with `Run()` for simple CLIs. Use `RunFuncE` with `RunE()` when you need error returns:

```go
err := boa.CmdT[Params]{
    Use: "process",
    RunFuncE: func(p *Params, cmd *cobra.Command, args []string) error {
        if p.File == "" {
            return fmt.Errorf("file cannot be empty")
        }
        return nil
    },
}.RunE()
```

## Cobra Interop

Access the underlying cobra command for advanced customization:

```go
boa.CmdT[Params]{
    Use: "cmd",
    InitFunc: func(p *Params, cmd *cobra.Command) error {
        cmd.Deprecated = "use new-cmd instead"
        return nil
    },
    RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
        // ...
    },
}
```

## Struct Composition

```go
type CommonFlags struct {
    Verbose bool `optional:"true"`
    Output  string `default:"stdout"`
}

type DBConfig struct {
    Host string `default:"localhost"`
    Port int    `default:"5432"`
}

type Params struct {
    CommonFlags           // embedded (anonymous): --verbose, --output (no prefix)
    DB          DBConfig  // named field: --db-host, --db-port (auto-prefixed)
    File        string
}
```

**Embedded (anonymous) fields** are not prefixed -- `CommonFlags.Verbose` becomes `--verbose`.

**Named struct fields** auto-prefix their children with the field name in kebab-case:

- `DB.Host` becomes `--db-host`, env var `DB_HOST`
- `DB.Port` becomes `--db-port`, env var `DB_PORT`

This prevents collisions when the same struct type is used in multiple fields:

```go
type Params struct {
    Primary DBConfig  // --primary-host, --primary-port
    Replica DBConfig  // --replica-host, --replica-port
}
```

Deep nesting chains prefixes: `Infra.Primary.Host` becomes `--infra-primary-host`.

Explicit `name:"..."` and `env:"..."` tags also get prefixed inside named fields.

## Roadmap

- **Custom type registration** — `RegisterType[T]()` for user-defined types with custom parsing
- **Viper compatibility package** — optional `boa-viper-compat` addon for automatic config file search paths (`~/.config/app/`, `/etc/app/`, `.`)

## Further Reading

- [Full Documentation](https://gigurra.github.io/boa/)
- [Hooks & Lifecycle](https://gigurra.github.io/boa/hooks/)
- [Advanced Features](https://gigurra.github.io/boa/advanced/)
- [Enrichers](https://gigurra.github.io/boa/enrichers/)
- [Struct Tags](https://gigurra.github.io/boa/struct-tags/)
- [Error Handling](https://gigurra.github.io/boa/error-handling/)
- [Migration Guide](https://gigurra.github.io/boa/migration/)
