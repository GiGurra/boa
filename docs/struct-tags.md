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
| `min` | | Min value (numeric) or min length (string/slice) | `min:"1"` |
| `max` | | Max value (numeric) or max length (string/slice) | `max:"65535"` |
| `pattern` | | Regex pattern (strings only) | `pattern:"^[a-z]+$"` |
| `configfile` | | Auto-load config file (root or substruct) | `configfile:"true"` |
| `boa` | | Special directives | `boa:"ignore"`, `boa:"configonly"`, `boa:"noflag"`, `boa:"nocli"`, `boa:"noenv"` |

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

### The `boa:"ignore"` and `boa:"configonly"` Tags

Fields tagged `boa:"ignore"` (or its alias `boa:"configonly"`) are skipped during CLI flag and environment variable registration. They do not appear in `--help` output and cannot be set via the command line or env vars.

However, these fields **still receive values from config file loading**, since config files are loaded via `json.Unmarshal` (or the configured unmarshal function) which writes directly to struct fields. This makes these tags useful for config-file-only fields:

```go
type Params struct {
    ConfigFile string `configfile:"true" optional:"true" default:"config.json"`
    Host       string `descr:"server host"`
    Port       int    `descr:"server port"`
    InternalID string `boa:"ignore"`     // only loaded from config file
    Metadata   map[string]string `boa:"configonly"` // clearer intent: config-file-only
}
```

`boa:"configonly"` is functionally identical to `boa:"ignore"` but communicates intent more clearly.

### The `boa:"noflag"` / `boa:"nocli"` Tag

Fields tagged `boa:"noflag"` (or its alias `boa:"nocli"`) are **excluded from CLI flag registration only**. They do not appear in `--help` and cannot be set with a `--flag`, but they are fully processed in every other way: env vars, config files, defaults, `min`/`max`/`pattern` validation, and custom validators all still apply.

```go
type Params struct {
    Name   string `descr:"public name"`
    Secret string `descr:"api token" boa:"noflag" env:"API_TOKEN"`
}
```

With the above, `--secret` is not a valid flag, but `API_TOKEN=...` still populates the field, and the user still sees a `missing required param 'secret' (env: API_TOKEN)` error if it is required and unset.

Combining `boa:"noflag"` with `positional` is an error — a positional argument is, by definition, a CLI argument.

Difference from `boa:"ignore"`: `ignore` skips boa processing entirely (no env reads, no validation) and only supports config-file unmarshal; `noflag` skips just the CLI flag layer.

### The `boa:"noenv"` Tag

Mirror image of `noflag`: the field is exposed as a CLI flag and still loads from config files, but **env var reading is suppressed**. This is mostly useful in combination with `ParamEnricherEnv`, where you want the enricher to auto-bind most fields to env vars but opt a few out:

```go
type Params struct {
    Host     string `descr:"hostname"`
    Internal string `descr:"internal knob" boa:"noenv"`
}
// With ParamEnricherEnv: $HOST populates Host, but $INTERNAL is ignored.
```

### Programmatic parity

Anything configurable with a struct tag is also configurable programmatically through `HookContext.GetParam(&p.Field)` (or the typed `GetParamT`). This is the escape hatch for parameter structs you don't own and can't add tags to:

```go
boa.CmdT[ExternalConfig]{
    Use: "cmd",
    InitFuncCtx: func(ctx *boa.HookContext, p *ExternalConfig, cmd *cobra.Command) error {
        secret := boa.GetParamT(ctx, &p.Secret)
        secret.SetDescription("auth token (env or config only)")
        secret.SetNoFlag(true)    // equivalent to `boa:"noflag"`
        secret.SetEnv("APP_TOKEN")

        port := boa.GetParamT(ctx, &p.Port)
        min, max := 1.0, 65535.0
        port.SetMin(&min)         // equivalent to `min:"1"`
        port.SetMax(&max)         // equivalent to `max:"65535"`
        return nil
    },
}
```

Available setters include `SetDescription`, `SetName`, `SetShort`, `SetEnv`, `SetPositional`, `SetRequired(bool)` / `SetRequiredFn`, `SetNoFlag`, `SetNoEnv`, `SetIgnored`, `SetMin`, `SetMax`, `SetPattern`, `SetAlternatives`, `SetAlternativesFunc`, `SetStrictAlts`, `SetDefault` / `SetDefaultT`, `SetCustomValidator` / `SetCustomValidatorT`, and `SetIsEnabledFn`.

All programmatic setters must be called from `InitFunc` / `InitFuncCtx` (or `CfgStructInit` / `CfgStructInitCtx`) so they take effect before cobra flag binding and env parsing.

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

### Min/Max Validation

For numeric types, `min` and `max` validate the value itself. For strings and slices, they validate the length:

```go
type Params struct {
    Port    int      `descr:"port" min:"1" max:"65535"`
    Rate    float64  `descr:"rate" min:"0.0" max:"1.0"`
    Name    string   `descr:"name" min:"3" max:"20"`
    Retries int      `descr:"retries" max:"10"`
    Tags    []string `descr:"tags" min:"1" max:"5"`
    Files   []string `positional:"true" min:"2" max:"10"`
}
```

Optional (pointer) fields are only validated when a value is actually provided.

### Pattern Validation

Use `pattern` to validate string fields against a regular expression:

```go
type Params struct {
    Name string `descr:"name" pattern:"^[a-z][a-z0-9-]*$"`
    Tag  string `descr:"tag" pattern:"^v[0-9]+\\.[0-9]+\\.[0-9]+$"`
}
```

Optional (pointer) fields are only validated when a value is actually provided.

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

The tagged field must be a `string`. Only one `configfile` field per struct level. Nested structs can also have their own `configfile:"true"` field for substruct-level config files. See [Advanced](advanced.md#substruct-config-files) for details.

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

The `camelToKebabCase` conversion handles acronyms correctly:

| Field | Flag | Short | Env Var |
|-------|------|-------|---------|
| `ServerHost` | `--server-host` | `-s` | (none by default) |
| `DBHost` | `--db-host` | `-d` | (none by default) |
| `HTTPPort` | `--http-port` | `-h` (skipped, reserved) | (none by default) |
| `MaxRetries` | `--max-retries` | `-m` | (none by default) |
| `FB` | `--fb` | `-f` | (none by default) |

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

## Named Struct Auto-Prefixing

Named (non-anonymous) struct fields auto-prefix their children's flag names and env var names. This prevents collisions when the same struct type is used in multiple fields.

```go
type DBConfig struct {
    Host string `descr:"database host" default:"localhost"`
    Port int    `descr:"database port" default:"5432"`
}

type Params struct {
    Primary DBConfig  // named → --primary-host, --primary-port
    Replica DBConfig  // named → --replica-host, --replica-port
}
```

### Rules

- **Embedded (anonymous) fields** are not prefixed: `CommonFlags` → `--verbose`
- **Named fields** auto-prefix: `DB DBConfig` → `--db-host`
- **Deep nesting chains**: `Infra.Primary.Host` → `--infra-primary-host`
- **Env vars also prefixed**: `DB.Host` with `ParamEnricherEnv` → `DB_HOST`
- **Explicit tags also prefixed**: `name:"host"` inside named field `API` → `--api-host`
- **Explicit env tags also prefixed**: `env:"HOST"` inside named field `API` → `API_HOST`

### Example with Env Vars

```go
type ServerConfig struct {
    Host string `env:"SERVER_HOST" default:"localhost"`
    Port int    `env:"SERVER_PORT" default:"8080"`
}

type Params struct {
    API ServerConfig  // env vars become API_SERVER_HOST, API_SERVER_PORT
}
```

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
