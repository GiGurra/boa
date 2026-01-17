# Struct Tags Reference

Quick reference for all BOA struct tags.

## Tag Reference

| Tag | Aliases | Description | Example |
|-----|---------|-------------|---------|
| `descr` | `desc`, `description`, `help` | Help text | `descr:"User name"` |
| `name` | `long` | Override flag name | `name:"server-host"` |
| `short` | | Single-char flag | `short:"n"` |
| `env` | | Environment variable | `env:"APP_HOST"` |
| `default` | | Default value | `default:"8080"` |
| `required` | `req` | Mark as required | `required:"true"` |
| `optional` | `opt` | Mark as optional | `optional:"true"` |
| `positional` | `pos` | Positional argument | `positional:"true"` |
| `alts` | `alternatives` | Allowed values | `alts:"a,b,c"` |
| `strict-alts` | `strict` | Validate alts | `strict:"true"` |

## Examples

### Basic Flags

```go
type Params struct {
    Host string `descr:"Server hostname" default:"localhost"`
    Port int    `descr:"Server port" short:"p" default:"8080"`
}
```

### Environment Variables

```go
type Params struct {
    APIKey   string `env:"API_KEY" descr:"API authentication key"`
    LogLevel string `env:"LOG_LEVEL" default:"info"`
}
```

### Positional Arguments

```go
type Params struct {
    Source string `positional:"true" descr:"Source file"`
    Dest   string `positional:"true" descr:"Destination file"`
}
// Usage: myapp <source> <dest>
```

### Optional Positional Arguments

```go
type Params struct {
    File   string `positional:"true" descr:"Input file"`
    Output string `positional:"true" optional:"true" default:"out.txt"`
}
// Usage: myapp <file> [output]
```

### Enum Values

```go
type Params struct {
    Format string `alts:"json,yaml,toml" default:"json"`
    Level  string `alts:"debug,info,warn,error" strict:"true"`
}
```

### Combined Example

```go
type Params struct {
    // Required flag with short form and env var
    Config string `short:"c" env:"APP_CONFIG" descr:"Config file path"`

    // Optional flag with default
    Port int `short:"p" optional:"true" default:"8080" descr:"Listen port"`

    // Positional argument
    Command string `positional:"true" descr:"Command to run"`

    // Boolean flag (defaults to false automatically)
    Verbose bool `short:"v" optional:"true" descr:"Verbose output"`

    // Enum with validation
    Mode string `alts:"dev,prod" default:"dev" descr:"Run mode"`
}
```

## Auto-Generated Values

Without explicit tags, BOA derives:

| Field | Flag | Env Var | Short |
|-------|------|---------|-------|
| `ServerHost` | `--server-host` | `SERVER_HOST` | `-s` |
| `MaxRetries` | `--max-retries` | `MAX_RETRIES` | `-m` |
| `V` | `--v` | `V` | (none - too short) |

See [Enrichers](enrichers.md) to customize this behavior.

## Beyond Struct Tags

Struct tags cover the most common use cases, but for dynamic behavior you'll need programmatic configuration via `HookContext`:

- **Dynamic defaults** - Set defaults based on runtime values
- **Conditional requirements** - Make fields required based on other fields
- **Dynamic completions** - Shell completions from APIs, files, or computed at runtime
- **AlternativesFunc** - Generate completion suggestions dynamically (e.g., list files, query databases)
- **Custom validation** - Complex validation logic

=== "Direct API"

    ```go
    boa.CmdT[Params]{
        Use: "app",
        InitFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
            param := ctx.GetParam(&p.Region)
            param.SetAlternatives(fetchRegionsFromAPI())
            param.SetStrictAlts(true)
            return nil
        },
    }
    ```

=== "Builder API"

    ```go
    boa.NewCmdT[Params]("app").
        WithInitFuncCtx(func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
            param := ctx.GetParam(&p.Region)
            param.SetAlternatives(fetchRegionsFromAPI())
            param.SetStrictAlts(true)
            return nil
        })
    ```

See [Advanced](advanced.md) for the full `Param` interface and examples.

## See Also

- [Enrichers](enrichers.md) - Auto-derivation of names
- [Validation](validation.md) - Constraints and conditional requirements
- [Advanced](advanced.md) - Programmatic configuration
