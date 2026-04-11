# BOA

[![CI Status](https://github.com/GiGurra/boa/actions/workflows/ci.yml/badge.svg)](https://github.com/GiGurra/boa/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/GiGurra/boa)](https://goreportcard.com/report/github.com/GiGurra/boa)
[![Docs](https://img.shields.io/badge/docs-gigurra.github.io%2Fboa-blue)](https://gigurra.github.io/boa/)

Like if [kong](https://github.com/alecthomas/kong) and [urfave/cli](https://github.com/urfave/cli) had a baby and made it [cobra](https://github.com/spf13/cobra) compatible.

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

This is what you get — flag names, short flags, defaults, required/optional, descriptions, and usage line all generated from the struct:

```
$ my-app --help
a simple CLI tool

Usage:
  my-app [flags]

Flags:
  -h, --help          help for my-app
  -n, --name string   your name (required)
  -p, --port int      port number (default 8080)
```

And this is how you interact with it:

```
$ my-app --name Alice
Hello Alice on port 8080

$ my-app --name Bob --port 3000
Hello Bob on port 3000

$ my-app
Usage:
  my-app [flags]

Flags:
  -h, --help          help for my-app
  -n, --name string   your name (required)
  -p, --port int      port number (default 8080)

Error: required flag "name" not set
```

<details>
<summary><b>Parameter Types</b></summary>

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

Pointer fields are optional by default — `nil` means "not set", so you can distinguish between "user passed zero" and "user didn't pass anything":

```go
type Params struct {
    Retries *int `descr:"retry count"` // nil if not provided, *0 if --retries 0
}

boa.CmdT[Params]{
    Use: "app",
    RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
        if p.Retries != nil {
            fmt.Printf("Retrying %d times\n", *p.Retries)
        } else {
            fmt.Println("Using default retry strategy")
        }
    },
}.Run()
```
</details>

<details>
<summary><b>Subcommands</b></summary>

```go
type ServeParams struct {
    Host string `descr:"bind address" default:"0.0.0.0"`
    Port int    `descr:"port" default:"8080"`
}

type DeployParams struct {
    Target string `descr:"deploy target" alts:"staging,production" strict:"true"`
    DryRun bool   `descr:"dry run mode" optional:"true"`
}

func main() {
    boa.CmdT[boa.NoParams]{
        Use:   "my-app",
        Short: "a multi-command CLI",
        SubCmds: boa.SubCmds(
            boa.CmdT[ServeParams]{
                Use: "serve", Short: "start the server",
                RunFunc: func(p *ServeParams, cmd *cobra.Command, args []string) {
                    fmt.Printf("Serving on %s:%d\n", p.Host, p.Port)
                },
            },
            boa.CmdT[DeployParams]{
                Use: "deploy", Short: "deploy the app",
                RunFunc: func(p *DeployParams, cmd *cobra.Command, args []string) {
                    fmt.Printf("Deploying to %s (dry-run: %v)\n", p.Target, p.DryRun)
                },
            },
        ),
    }.Run()
}
```

```
$ my-app --help
a multi-command CLI

Usage:
  my-app [command]

Available Commands:
  serve       start the server
  deploy      deploy the app

$ my-app deploy --target staging --dry-run
Deploying to staging (dry-run: true)

$ my-app bogus
Error: unknown command "bogus" for "my-app"
```
</details>

<details>
<summary><b>Config Files</b></summary>

Tag a field with `configfile` and boa loads it automatically. CLI and env vars always win:

```go
type Params struct {
    ConfigFile string     `configfile:"true" optional:"true" default:"config.json"`
    Host       string     `descr:"server host"`
    Port       int        `descr:"port" default:"8080"`
    Internal   [][]string `boa:"configonly"` // loaded from config only, no CLI flag
}
```

```bash
$ cat config.json
{"Host": "prod.example.com", "Port": 443, "Internal": [["a","b"],["c","d"]]}

$ my-app                              # uses config.json values
$ my-app --host override.local        # CLI wins over config file
$ my-app --config-file staging.json   # different config file
$ HOST=ci.local my-app                # env var wins over config file
```

Nested structs can have their own config files. Priority: CLI > env > root config > substruct config > defaults:

```go
type DBConfig struct {
    ConfigFile string `configfile:"true" optional:"true"`
    Host       string `default:"localhost"`
    Port       int    `default:"5432"`
}

type Params struct {
    ConfigFile string   `configfile:"true" optional:"true" default:"app.json"`
    DB         DBConfig
}
```

JSON is built in. Register other formats with one line:

```go
boa.RegisterConfigFormat(".yaml", yaml.Unmarshal)
boa.RegisterConfigFormat(".toml", toml.Unmarshal)
```
</details>

<details>
<summary><b>Struct Composition</b></summary>

Named fields auto-prefix their children. Embedded fields stay flat:

```go
type DBConfig struct {
    Host string `default:"localhost"`
    Port int    `default:"5432"`
}

type CommonFlags struct {
    Verbose bool `optional:"true"`
}

type Params struct {
    CommonFlags           // embedded: --verbose (no prefix)
    Primary DBConfig      // named: --primary-host, --primary-port
    Replica DBConfig      // named: --replica-host, --replica-port
}
```

```
$ my-app --help
Flags:
  --verbose              (default false)
  --primary-host string  (default "localhost")
  --primary-port int     (default 5432)
  --replica-host string  (default "localhost")
  --replica-port int     (default 5432)
```

Deep nesting chains prefixes: `Infra.Primary.Host` becomes `--infra-primary-host`. Env vars follow the same pattern: `INFRA_PRIMARY_HOST`.
</details>

<details>
<summary><b>Validation</b></summary>

Struct tag validation:

```go
type Params struct {
    Port     int    `descr:"port" min:"1" max:"65535"`
    LogLevel string `descr:"log level" alts:"debug,info,warn,error" strict:"true"`
    Name     string `descr:"name" pattern:"^[a-z]+$"`
    Tags     []string `descr:"tags" min:"1" max:"5"` // min/max = slice length
}
```

Programmatic validation with `HookContext` — for cases where struct tags aren't enough:

```go
type Params struct {
    Host string `descr:"Server hostname"`
    Port int    `descr:"Server port"`
    CIDR string `descr:"Allowed CIDR range" optional:"true"`
}

boa.CmdT[Params]{
    Use: "server",
    InitFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
        // Type-safe custom validator
        boa.GetParamT(ctx, &p.Port).SetCustomValidatorT(func(port int) error {
            if port < 1024 && port != 80 && port != 443 {
                return fmt.Errorf("non-standard privileged port %d", port)
            }
            return nil
        })

        // Conditional required: CIDR only required when host is not localhost
        ctx.GetParam(&p.CIDR).SetRequiredFn(func() bool {
            return p.Host != "localhost"
        })

        return nil
    },
    RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
        fmt.Printf("Listening on %s:%d\n", p.Host, p.Port)
    },
}.Run()
```

You can also implement validation directly on your params struct:

```go
type ServerConfig struct {
    Host     string
    Port     int
    LogLevel string
}

func (c *ServerConfig) InitCtx(ctx *boa.HookContext) error {
    ctx.GetParam(&c.Port).SetDefault(boa.Default(8080))
    ctx.GetParam(&c.LogLevel).SetAlternatives([]string{"debug", "info", "warn", "error"})
    ctx.GetParam(&c.LogLevel).SetStrictAlts(true)
    return nil
}
```
</details>

<details>
<summary><b>Error Handling</b></summary>

| Method | Behavior |
|--------|----------|
| `Run()` | Shows usage + error on bad input, exits 1. Other errors panic. |
| `RunE()` | Returns all errors silently for programmatic use |
| `ToCobra()` | Returns `*cobra.Command` for custom execution via `boa.Execute(cmd)` |

`Run()` for simple CLIs:

```go
boa.CmdT[Params]{
    Use: "app",
    RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
        fmt.Println("Success!")
    },
}.Run()
// Bad input → prints usage + error, exits 1
// Runtime panic → crashes with stack trace
```

`RunE()` when you need error handling:

```go
err := boa.CmdT[Params]{
    Use: "app",
    RunFuncE: func(p *Params, cmd *cobra.Command, args []string) error {
        if p.Port < 1024 {
            return fmt.Errorf("port must be >= 1024")
        }
        return nil
    },
}.RunE()

if err != nil {
    log.Fatalf("Command failed: %v", err)
}
```

`ToCobra()` when embedding boa in a larger cobra app:

```go
cmd := boa.CmdT[Params]{
    Use: "sub",
    RunFunc: func(p *Params, cmd *cobra.Command, args []string) { ... },
}.ToCobra()

rootCmd.AddCommand(cmd)
boa.Execute(rootCmd) // prints usage + error on failure
```
</details>

<details>
<summary><b>Shell Completions</b></summary>

Every boa CLI gets shell completions for free via cobra. No extra code needed:

```
$ my-app completion bash   # bash completions
$ my-app completion zsh    # zsh completions
$ my-app completion fish   # fish completions
$ my-app completion powershell  # powershell completions
```

Install once and get tab completion for all flags, subcommands, and enum values:

```bash
# bash
my-app completion bash > /etc/bash_completion.d/my-app

# zsh
my-app completion zsh > "${fpath[1]}/_my-app"

# fish
my-app completion fish > ~/.config/fish/completions/my-app.fish
```

Fields with `alts` automatically complete to their allowed values.

Static completions from struct tags:

```go
type Params struct {
    LogLevel string `descr:"log level" alts:"debug,info,warn,error"` // tab-completes to these values
}
```

Dynamic completions — e.g. completing based on output from other CLIs:

```go
boa.CmdT[Params]{
    Use: "deploy",
    InitFunc: func(p *Params, cmd *cobra.Command) error {
        cmd.RegisterFlagCompletionFunc("namespace", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
            // Call kubectl to get real namespace list
            out, err := exec.Command("kubectl", "get", "namespaces", "-o", "jsonpath={.items[*].metadata.name}").Output()
            if err != nil {
                return nil, cobra.ShellCompDirectiveError
            }
            return strings.Fields(string(out)), cobra.ShellCompDirectiveNoFileComp
        })
        return nil
    },
    RunFunc: func(p *Params, cmd *cobra.Command, args []string) { ... },
}.Run()
// $ deploy --namespace <TAB>
// default    kube-system    production    staging
```
</details>

<details>
<summary><b>Struct Tags Reference</b></summary>

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
| `boa` | Special directives | `boa:"ignore"` (no mirror), `boa:"configonly"` (no CLI/env, mirror + validation preserved), `boa:"noflag"` / `"nocli"`, `boa:"noenv"` |

</details>

## Further Reading

- [Getting Started](https://gigurra.github.io/boa/getting-started/) — all parameter types, subcommands, config files
- [Struct Tags](https://gigurra.github.io/boa/struct-tags/) — complete tag reference with auto-prefixing
- [Bring Someone Else's Config](https://gigurra.github.io/boa/bring-someone-elses-config/) — wire up third-party / tag-less structs programmatically
- [Validation](https://gigurra.github.io/boa/validation/) — required/optional, alternatives, conditional requirements
- [Lifecycle Hooks](https://gigurra.github.io/boa/hooks/) — customize behavior at each stage
- [Enrichers](https://gigurra.github.io/boa/enrichers/) — auto-derive flag names, env vars, short flags
- [Error Handling](https://gigurra.github.io/boa/error-handling/) — Run() vs RunE() and error propagation
- [Advanced](https://gigurra.github.io/boa/advanced/) — custom types, config format registry, viper-like discovery
- [Cobra Interop](https://gigurra.github.io/boa/cobra-interop/) — access cobra primitives, migrate incrementally
