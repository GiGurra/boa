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
| `configfile` | | Auto-load config file | `configfile:"true"` |
| `boa` | | Special directives | `boa:"ignore"` |

## Special Field Types

### Pointer Fields

Pointer types (`*string`, `*int`, `*bool`, etc.) are always optional by default, regardless of the global configuration. A `nil` value means the flag was not provided. Use `required:"true"` to override.

```go
type Params struct {
    Name  *string `descr:"user name"`              // optional, nil if not set
    Count *int    `descr:"item count"`              // optional, nil if not set
    Force *bool   `required:"true" descr:"force"`   // required even though it's a pointer
}
```

### The `boa:"ignore"` Tag

Fields tagged `boa:"ignore"` are skipped during CLI flag and environment variable registration. They do not appear in `--help` output and cannot be set via the command line or env vars.

However, these fields **still receive values from config file loading**, since config files are loaded via `json.Unmarshal` (or the configured unmarshal function) which writes directly to struct fields. This makes `boa:"ignore"` useful for config-file-only fields:

```go
type Params struct {
    ConfigFile string `configfile:"true" optional:"true" default:"config.json"`
    Host       string `descr:"server host"`
    Port       int    `descr:"server port"`
    InternalID string `boa:"ignore"` // only loaded from config file, not exposed as CLI flag
}
```

### Map Fields

Map types with string keys (`map[string]string`, `map[string]int`, `map[string]int64`) use `key=val,key=val` syntax on the CLI. Maps default to optional.

```go
type Params struct {
    Labels map[string]string `descr:"key=value labels"`
    Limits map[string]int    `descr:"resource limits"`
}
// Usage: myapp --labels env=prod,team=backend --limits cpu=4,memory=8192
```

For complex map value types (e.g., `map[string][]string`), the CLI uses JSON syntax. See [Advanced](advanced.md#json-fallback-for-complex-types).

### Complex Types (JSON Fallback)

Any field type without native pflag support (nested slices, complex maps, etc.) automatically falls back to JSON parsing on the CLI:

```go
type Params struct {
    Matrix [][]int             `descr:"nested matrix" optional:"true"`
    Meta   map[string][]string `descr:"metadata" optional:"true"`
}
// Usage: --matrix '[[1,2],[3,4]]' --meta '{"tags":["a","b"]}'
```

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

### Config File

```go
type Params struct {
    Config string `configfile:"true" optional:"true" default:"config.json" descr:"Path to config file"`
    Host   string `descr:"Server hostname"`
    Port   int    `descr:"Server port"`
}
// Usage: myapp --config myconfig.json
// Or just: myapp (loads config.json by default)
```

The tagged field must be a `string`. Only one `configfile` field per struct. See [Advanced](advanced.md#config-file-loading) for details.

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

Without explicit tags, BOA derives flag names and short flags automatically. Environment variables are **not** auto-derived by default -- add `ParamEnricherEnv` to your enricher chain or use `env` struct tags.

| Field | Flag | Short | Env Var |
|-------|------|-------|---------|
| `ServerHost` | `--server-host` | `-s` | (none by default) |
| `MaxRetries` | `--max-retries` | `-m` | (none by default) |
| `V` | `--v` | (none - too short) | (none by default) |

To auto-derive env vars, set `ParamEnrich`:

```go
boa.CmdT[Params]{
    Use: "app",
    ParamEnrich: boa.ParamEnricherCombine(
        boa.ParamEnricherName,
        boa.ParamEnricherShort,
        boa.ParamEnricherEnv,   // adds SERVER_HOST, MAX_RETRIES, etc.
        boa.ParamEnricherBool,
    ),
}
```

See [Enrichers](enrichers.md) to customize this behavior.

## Beyond Struct Tags

Struct tags cover the most common use cases, but for dynamic behavior you'll need programmatic configuration via `HookContext`:

- **Dynamic defaults** - Set defaults based on runtime values
- **Conditional requirements** - Make fields required based on other fields
- **Dynamic completions** - Shell completions from APIs, files, or computed at runtime
- **AlternativesFunc** - Generate completion suggestions dynamically (e.g., list files, query databases)
- **Custom validation** - Complex validation logic

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

See [Advanced](advanced.md) for the full `Param` interface and examples.

## See Also

- [Enrichers](enrichers.md) - Auto-derivation of names
- [Validation](validation.md) - Constraints and conditional requirements
- [Advanced](advanced.md) - Programmatic configuration
