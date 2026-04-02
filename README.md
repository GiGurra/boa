# BOA

[![CI Status](https://github.com/GiGurra/boa/actions/workflows/ci.yml/badge.svg)](https://github.com/GiGurra/boa/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/GiGurra/boa)](https://goreportcard.com/report/github.com/GiGurra/boa)

Self-documenting CLIs from Go structs. Define your parameters once and get flags, env vars, validation, config file loading, and help text — all generated automatically. The result is a CLI that's easy to write, easy for humans to use, and easy for LLMs to invoke — because the full parameter schema is right there in `--help`.

Built on top of [cobra](https://github.com/spf13/cobra), not replacing it. Full cobra interop when you need it.

**[Full Documentation](https://gigurra.github.io/boa/)**

## Quick Start

```bash
go get github.com/GiGurra/boa@latest
```

```go
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

All standard Go types work out of the box:

```go
type Params struct {
    Host    string            `descr:"server host"`                    // required by default
    Port    int               `descr:"port" default:"8080"`            // with default
    Name    *string           `descr:"user name"`                      // pointer = optional, nil = not set
    Tags    []string          `descr:"tags" default:"[a,b,c]"`         // --tags a,b,c
    Labels  map[string]string `descr:"labels"`                         // --labels env=prod,team=backend
    Input   string            `positional:"true"`                      // positional arg
    Timeout time.Duration     `descr:"timeout" default:"30s"`          // durations, IPs, URLs, etc.
    Matrix  [][]int           `descr:"matrix" optional:"true"`         // complex types use JSON: '[[1,2],[3,4]]'
}
```

## Subcommands

```go
boa.CmdT[boa.NoParams]{
    Use:   "my-app",
    Short: "a multi-command CLI",
    SubCmds: boa.SubCmds(
        boa.CmdT[ServeParams]{
            Use: "serve", Short: "start the server",
            RunFunc: func(p *ServeParams, cmd *cobra.Command, args []string) { ... },
        },
        boa.CmdT[DeployParams]{
            Use: "deploy", Short: "deploy the app",
            RunFunc: func(p *DeployParams, cmd *cobra.Command, args []string) { ... },
        },
    ),
}.Run()
```

## Config Files

Tag a field with `configfile` and boa loads it automatically. CLI and env vars always win:

```go
type Params struct {
    ConfigFile string            `configfile:"true" optional:"true" default:"config.json"`
    Host       string            `descr:"server host"`
    Port       int               `descr:"port" default:"8080"`
    Internal   [][]string        `boa:"configonly"` // loaded from config only, no CLI flag
}
```

JSON is built in. Register other formats with `boa.RegisterConfigFormat(".yaml", yaml.Unmarshal)`.

## Struct Composition

Named fields auto-prefix their children. Embedded fields stay flat:

```go
type DBConfig struct {
    Host string `default:"localhost"`
    Port int    `default:"5432"`
}

type Params struct {
    CommonFlags           // embedded: --verbose, --output
    Primary DBConfig      // --primary-host, --primary-port
    Replica DBConfig      // --replica-host, --replica-port
}
```

## Validation

```go
type Params struct {
    Port     int    `descr:"port" min:"1" max:"65535"`
    LogLevel string `descr:"log level" alts:"debug,info,warn,error" strict:"true"`
    Name     string `descr:"name" pattern:"^[a-z]+$"`
}
```

## Struct Tags Reference

| Tag | Description | Example |
|-----|-------------|---------|
| `descr` / `desc` | Description text | `descr:"User name"` |
| `name` / `long` | Override flag name | `name:"user-name"` |
| `default` | Default value | `default:"8080"` |
| `env` | Environment variable name | `env:"PORT"` |
| `short` | Short flag (single char) | `short:"p"` |
| `positional` / `pos` | Marks positional argument | `positional:"true"` |
| `required` / `req` | Marks as required | `required:"true"` |
| `optional` / `opt` | Marks as optional | `optional:"true"` |
| `alts` | Allowed values (enum) | `alts:"debug,info,warn,error"` |
| `strict` | Validate against alts | `strict:"true"` |
| `min` | Min value or min length | `min:"1"` |
| `max` | Max value or max length | `max:"65535"` |
| `pattern` | Regex pattern | `pattern:"^[a-z]+$"` |
| `configfile` | Auto-load config from path | `configfile:"true"` |
| `boa` | Special directives | `boa:"ignore"`, `boa:"configonly"` |

## Error Handling

| Method | Behavior |
|--------|----------|
| `Run()` | Shows usage + error on bad input, exits 1 |
| `RunE()` | Returns errors silently for programmatic use |
| `ToCobra()` | Returns `*cobra.Command` for custom execution |

## Further Reading

- [Getting Started](https://gigurra.github.io/boa/getting-started/) — all parameter types, subcommands, config files
- [Struct Tags](https://gigurra.github.io/boa/struct-tags/) — complete tag reference with auto-prefixing
- [Validation](https://gigurra.github.io/boa/validation/) — required/optional, alternatives, conditional requirements
- [Lifecycle Hooks](https://gigurra.github.io/boa/hooks/) — customize behavior at each stage
- [Enrichers](https://gigurra.github.io/boa/enrichers/) — auto-derive flag names, env vars, short flags
- [Error Handling](https://gigurra.github.io/boa/error-handling/) — Run() vs RunE() and error propagation
- [Advanced](https://gigurra.github.io/boa/advanced/) — custom types, config format registry, viper-like discovery
- [Cobra Interop](https://gigurra.github.io/boa/cobra-interop/) — access cobra primitives, migrate incrementally
