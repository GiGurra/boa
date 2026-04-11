# Advanced Features

This page covers advanced BOA features for power users.

## The Param Interface

Every parameter (whether using struct tags or programmatic configuration) implements the `Param` interface. Access it via `HookContext.GetParam()`:

```go
boa.CmdT[Params]{
    Use: "cmd",
    InitFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
        param := ctx.GetParam(&p.SomeField)
        // Now use param methods...
        return nil
    },
}
```

### Param Methods

| Method | Description |
|--------|-------------|
| `SetName(string)` | Override flag name |
| `SetShort(string)` | Set short flag |
| `SetEnv(string)` | Set environment variable |
| `SetDefault(any)` | Set default value |
| `SetAlternatives([]string)` | Set allowed values |
| `SetAlternativesFunc(func(...) []string)` | Set dynamic completion function |
| `SetStrictAlts(bool)` | Enable/disable strict validation |
| `SetRequiredFn(func() bool)` | Dynamic required condition |
| `SetIsEnabledFn(func() bool)` | Dynamic visibility |
| `GetName() string` | Get current flag name |
| `GetShort() string` | Get current short flag |
| `GetEnv() string` | Get current env var |
| `GetAlternatives() []string` | Get allowed values |
| `GetAlternativesFunc()` | Get dynamic completion function |
| `HasValue() bool` | Check if value was set |
| `IsRequired() bool` | Check if required |
| `IsEnabled() bool` | Check if visible |

## Typed Parameter API (ParamT)

For type-safe parameter configuration, use `boa.GetParamT[T]()` instead of `GetParam()`. This returns a `ParamT[T]` interface with typed methods:

```go
type Params struct {
    Port int    `descr:"Server port"`
    Host string `descr:"Server host"`
}

boa.CmdT[Params]{
    Use: "server",
    InitFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
        // Type-safe: compiler ensures correct types
        portParam := boa.GetParamT(ctx, &p.Port)
        portParam.SetDefaultT(8080)  // Takes int, not any
        portParam.SetCustomValidatorT(func(port int) error {
            if port < 1 || port > 65535 {
                return fmt.Errorf("port must be between 1 and 65535")
            }
            return nil
        })

        hostParam := boa.GetParamT(ctx, &p.Host)
        hostParam.SetDefaultT("localhost")  // Takes string
        hostParam.SetAlternatives([]string{"localhost", "0.0.0.0"})

        return nil
    },
}
```

### ParamT Methods

The `ParamT[T]` interface provides typed methods plus all pass-through methods from `Param`:

| Typed Methods | Description |
|---------------|-------------|
| `SetDefaultT(T)` | Set default value with compile-time type checking |
| `SetCustomValidatorT(func(T) error)` | Set validation function that receives the typed value |

| Pass-through Methods | Description |
|---------------------|-------------|
| `Param()` | Access the underlying untyped `Param` interface |
| `SetAlternatives([]string)` | Set allowed values |
| `SetStrictAlts(bool)` | Enable/disable strict validation |
| `SetAlternativesFunc(...)` | Set dynamic completion function |
| `SetEnv(string)` | Set environment variable |
| `SetShort(string)` | Set short flag |
| `SetName(string)` | Set flag name |
| `SetIsEnabledFn(func() bool)` | Dynamic visibility |
| `SetRequiredFn(func() bool)` | Dynamic required condition |

### Conditional Requirements with ParamT

```go
type DeployParams struct {
    Environment string `descr:"Target environment" default:"dev"`
    ProdKey     string `descr:"Production API key" optional:"true"`
}

boa.CmdT[DeployParams]{
    Use: "deploy",
    InitFuncCtx: func(ctx *boa.HookContext, p *DeployParams, cmd *cobra.Command) error {
        // ProdKey is only required when deploying to production
        prodKeyParam := boa.GetParamT(ctx, &p.ProdKey)
        prodKeyParam.SetRequiredFn(func() bool {
            return p.Environment == "prod"
        })
        return nil
    },
}
```

## Dynamic Shell Completion

### AlternativesFunc

For completion suggestions that depend on runtime state (like fetching from an API), use `SetAlternativesFunc` via `HookContext`:

```go
type Params struct {
    Region string `descr:"AWS region"`
}

func main() {
    boa.CmdT[Params]{
        Use: "app",
        InitFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
            ctx.GetParam(&p.Region).SetAlternativesFunc(
                func(cmd *cobra.Command, args []string, toComplete string) []string {
                    // Could fetch from API, read from file, etc.
                    return []string{"us-east-1", "us-west-2", "eu-west-1"}
                },
            )
            return nil
        },
        RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
            // ...
        },
    }.Run()
}
```

### ValidArgsFunc

For positional argument completion:

```go
boa.CmdT[Params]{
    Use: "app",
    ValidArgsFunc: func(p *Params, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
        // Return suggestions for positional args
        return []string{"option1", "option2"}, cobra.ShellCompDirectiveDefault
    },
}
```

## Config File Loading

### Using the `configfile` Tag

Tag a string field with `configfile:"true"` to automatically load a config file before validation:

```go
type Params struct {
    ConfigFile string `configfile:"true" optional:"true" default:"config.json"`
    Host       string
    Port       int
}

boa.CmdT[Params]{
    Use: "app",
    RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
        fmt.Printf("Host: %s, Port: %d\n", p.Host, p.Port)
    },
}.Run()
```

CLI and env var values always take precedence over config file values.

### Config Format Registry

JSON is the only format shipped by default. BOA has no third-party parser dependencies — you bring your own library and register it in the global registry. The registry is keyed by file extension, so the same compiled binary can accept any mix of formats at runtime (e.g. `--config-file prod.json` today, `--config-file prod.yaml` tomorrow, no rebuild required).

#### The one-liner

```go
import "gopkg.in/yaml.v3"

boa.RegisterConfigFormat(".yaml", yaml.Unmarshal)
boa.RegisterConfigFormat(".toml", toml.Unmarshal)
```

That's the whole story for every mainstream Go config parser. The one-liner gets you both parsing **and** full key-presence detection — including zero-valued and same-as-default writes to optional struct-pointer parameter groups (`DB *DBConfig`). Under the hood `RegisterConfigFormat` wraps the unmarshaler in a `UniversalConfigFormat`, which asks the same parser to also decode the file into a `map[string]any` so BOA can read the literal key structure. Every mainstream Go parser supports that.

Without a `KeyTree`, BOA would fall back to snapshot comparison for those struct-pointer groups, which can't tell "user wrote the default" apart from "user wrote nothing". With the auto-synthesized one you avoid that gap entirely.

`KeyTree` accepts nested maps in either `map[string]any` (yaml.v3, json, toml) or `map[any]any` (yaml.v2) shape — BOA coerces transparently.

#### The `UniversalConfigFormat` helper

Use it when you want to attach a format inline to `Cmd.ConfigFormat` without touching the global registry:

```go
boa.CmdT[Params]{
    Use:          "app",
    ConfigFormat: boa.UniversalConfigFormat(yaml.Unmarshal),
    RunFunc:      func(p *Params, cmd *cobra.Command, args []string) { ... },
}.Run()
```

`UniversalConfigFormat(nil)` panics, so typos surface at the construction site rather than silently falling through to the JSON handler at parse time.

#### When to reach for `RegisterConfigFormatFull`

Only when your parser **cannot** decode into `map[string]any`. Most of the time, "write a parser that fills only specific struct types" is an app-specific custom format, not a third-party library. In that case the auto-synthesized `KeyTree` would fail at parse time, so you hand-write it yourself:

```go
boa.RegisterConfigFormatFull(".mycustom", boa.ConfigFormat{
    Unmarshal: mycustom.Decode,
    KeyTree:   mycustom.KeysOnly, // returns map[string]any of key structure
})
```

Resolution order for each config file load:

1. `Cmd.ConfigFormat` — per-command escape hatch; locks that one command to a single format (rarely what you want)
2. `Cmd.ConfigUnmarshal` — legacy, unmarshal-only, also command-locked
3. **Registered format matched by file extension — the default path; any number of formats can coexist in one binary**
4. Built-in JSON fallback

#### Per-command escape hatch

Setting a format directly on a command **bypasses** the extension registry and locks that command to one format. This is almost never what you want — prefer the global registry so your binary stays format-agnostic — but the escape hatch exists for custom-extension blobs from legacy systems and for injecting fake parsers in tests.

```go
boa.CmdT[Params]{
    Use: "ingest-legacy-blob",
    ConfigFormat: boa.ConfigFormat{
        Unmarshal: myLegacyUnmarshal,
        // KeyTree optional
    },
    RunFunc: func(p *Params, cmd *cobra.Command, args []string) { ... },
}.Run()
```

The legacy unmarshal-only field still works:

```go
boa.CmdT[Params]{
    Use:             "app",
    ConfigUnmarshal: yaml.Unmarshal,
    RunFunc:         func(p *Params, cmd *cobra.Command, args []string) { ... },
}.Run()
```

### Substruct Config Files

The `configfile:"true"` tag works on fields inside nested structs, not just at root. Each substruct can have its own config file:

```go
type DBConfig struct {
    ConfigFile string `configfile:"true" optional:"true"`
    Host       string `descr:"host" default:"localhost"`
    Port       int    `descr:"port" default:"5432"`
}

type CacheConfig struct {
    ConfigFile string `configfile:"true" optional:"true"`
    TTL        int    `descr:"cache TTL" default:"300"`
}

type Params struct {
    ConfigFile string      `configfile:"true" optional:"true" default:"config.json"`
    DB         DBConfig
    Cache      CacheConfig
}
```

Substruct configs load first, then root config loads and overrides any overlapping values. The full merge priority is:

1. **CLI flags** -- highest
2. **Environment variables**
3. **Root config file**
4. **Substruct config files**
5. **Default values**
6. **Zero value** -- lowest

This lets you split configuration across multiple files while maintaining a clear override hierarchy.

### Using `LoadConfigFile` Explicitly

For more control (e.g., loading into a sub-struct, multiple config files), use `LoadConfigFile` in a PreValidate hook:

```go
type AppConfig struct {
    Host string
    Port int
}

type Params struct {
    ConfigFile string `descr:"Path to config file" optional:"true"`
    AppConfig
}

boa.CmdT[Params]{
    Use: "app",
    PreValidateFunc: func(p *Params, cmd *cobra.Command, args []string) error {
        return boa.LoadConfigFile(p.ConfigFile, &p.AppConfig, nil)
    },
    RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
        fmt.Printf("Host: %s, Port: %d\n", p.Host, p.Port)
    },
}.Run()
```

### Loading Config From In-Memory Bytes

When the config lives somewhere other than a local file — an embedded `//go:embed` asset, stdin, an HTTP response, a secrets-manager blob, or a test fixture — use `boa.LoadConfigBytes`. It shares `LoadConfigFile`'s format-resolution rules (overrides → registered format for `ext` → JSON fallback), so any format registered via `RegisterConfigFormat` works identically.

```go
//go:embed defaults.yaml
var defaultsYAML []byte

boa.CmdT[Params]{
    Use: "app",
    PreValidateFunc: func(p *Params, cmd *cobra.Command, args []string) error {
        // Seed defaults from the embedded YAML blob.
        // CLI and env vars still override whatever is loaded here.
        return boa.LoadConfigBytes(defaultsYAML, ".yaml", p, nil)
    },
}.Run()
```

`ext` accepts `".yaml"`, `"yaml"`, or `""` (empty falls back to JSON). Empty or `nil` `data` is a no-op, so callers can hand in the result of an optional read without a preceding length check.

### Writing Resolved Config Back Out

Two serializers are available for the other direction:

- `boa.DumpConfigBytes(v, ext, nil)` / `boa.DumpConfigFile(path, v, nil)` — naive dump, emits every exported field including Go zero values. Good for "generate an example config that shows every option".
- `ctx.DumpBytes(ext, nil)` / `ctx.DumpFile(path, nil)` on `HookContext` — **source-aware**, emits only fields where `HasValue` is true (CLI, env, config file, default). Fields the user never touched are omitted entirely. This is the right helper for persisting resolved config between runs.

Source-aware dump pins defaults into the file, which is deliberate: a future release can change the app's built-in defaults without silently changing behavior for users whose saved config said "I'm happy with what shipped in version 1.0". The `configfile` path parameter itself is omitted so the dumped file doesn't reference its own path on the next load.

Key names honour format-appropriate struct tags (`json:"..."` for `.json`, `yaml:"..."` for `.yaml`/`.yml`, `toml:"..."` for `.toml`, `hcl:"..."` for `.hcl`); untagged fields fall back to Go field names. `json:"-"` skips the field entirely.

Non-JSON formats need a marshaler registered alongside the unmarshaler:

```go
boa.RegisterConfigFormat(".yaml", yaml.Unmarshal)
boa.RegisterConfigMarshaler(".yaml", yaml.Marshal)
```

JSON comes with both directions pre-registered (pretty-printed, 2-space indent, trailing newline). See [examples-config.md](examples-config.md#writing-config-back-out) for full examples.

## Checking Value Sources

Use `HookContext` in your run function to check how values were set:

```go
boa.CmdT[Params]{
    Use: "app",
    RunFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command, args []string) {
        if ctx.HasValue(&p.Port) {
            fmt.Printf("Port explicitly set to %d\n", p.Port)
        } else {
            fmt.Println("Using default port")
        }
    },
}
```

## Accessing Cobra Directly

Access the underlying Cobra command for features BOA doesn't wrap:

```go
boa.CmdT[Params]{
    Use: "app",
    InitFunc: func(p *Params, cmd *cobra.Command) error {
        cmd.Deprecated = "use 'new-app' instead"
        cmd.Hidden = true
        cmd.SilenceUsage = true
        return nil
    },
}
```

Or after flags are created:

```go
boa.CmdT[Params]{
    Use: "app",
    PostCreateFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
        flag := cmd.Flags().Lookup("verbose")
        flag.NoOptDefVal = "true"  // --verbose without value means true
        return nil
    },
}
```

## Command Groups

Organize subcommands into groups in help output:

```go
boa.CmdT[boa.NoParams]{
    Use: "app",
    Groups: []*cobra.Group{
        {ID: "core", Title: "Core Commands:"},
        {ID: "util", Title: "Utility Commands:"},
    },
    SubCmds: boa.SubCmds(
        boa.CmdT[Params]{Use: "init", GroupID: "core"},
        boa.CmdT[Params]{Use: "run", GroupID: "core"},
        boa.CmdT[Params]{Use: "version", GroupID: "util"},
    ),
}
```

## Testing Commands

### Inject Arguments

```go
cmd := boa.CmdT[Params]{
    Use: "app",
    RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
        // ...
    },
}

// Test with specific args
cmd.RunArgs([]string{"--name", "test", "--port", "8080"})
```

### Testing with Error Returns

Use `RunFuncE` and `RunArgsE` for testable commands that return errors:

```go
func TestMyCommand(t *testing.T) {
    err := boa.CmdT[Params]{
        Use: "app",
        RunFuncE: func(p *Params, cmd *cobra.Command, args []string) error {
            if p.Port < 1024 {
                return fmt.Errorf("port must be >= 1024")
            }
            return nil
        },
    }.RunArgsE([]string{"--port", "80"})

    if err == nil {
        t.Fatal("expected error for port < 1024")
    }
}
```

Use `ToCobraE()` when you need the underlying cobra command with `RunE` set:

```go
func TestMyCommand(t *testing.T) {
    cmd, err := boa.CmdT[Params]{
        Use: "app",
        RunFuncE: func(p *Params, cmd *cobra.Command, args []string) error {
            return nil
        },
    }.ToCobraE()
    if err != nil {
        t.Fatalf("setup failed: %v", err)
    }

    cmd.SetArgs([]string{"--name", "test"})
    err = cmd.Execute()
    // Assert on err
}
```

### Validate Without Running

```go
cmd := boa.CmdT[Params]{
    Use:     "app",
    RawArgs: []string{"--name", "test"},
}

err := cmd.Validate()
if err != nil {
    // Validation failed
}
```

## Interface-Based Hooks

Implement interfaces on your config struct instead of using function fields:

```go
type Config struct {
    Host string
    Port int
}

// Called during initialization
func (c *Config) Init() error {
    return nil
}

// Called during initialization with HookContext
func (c *Config) InitCtx(ctx *boa.HookContext) error {
    ctx.GetParam(&c.Port).SetDefault(boa.Default(8080))
    return nil
}

// Called before validation
func (c *Config) PreValidate() error {
    return nil
}

// Called before execution
func (c *Config) PreExecute() error {
    return nil
}
```

Available interfaces:

- `CfgStructInit` - `Init() error`
- `CfgStructInitCtx` - `InitCtx(ctx *HookContext) error`
- `CfgStructPostCreate` - `PostCreate() error`
- `CfgStructPostCreateCtx` - `PostCreateCtx(ctx *HookContext) error`
- `CfgStructPreValidate` - `PreValidate() error`
- `CfgStructPreValidateCtx` - `PreValidateCtx(ctx *HookContext) error`
- `CfgStructPreExecute` - `PreExecute() error`
- `CfgStructPreExecuteCtx` - `PreExecuteCtx(ctx *HookContext) error`

## JSON Fallback for Complex Types

Any field type that doesn't have native pflag support (e.g., nested slices, complex maps) automatically falls back to JSON parsing. BOA registers the flag as a `StringP` and uses `json.Unmarshal` to parse the value.

This means you can use arbitrarily complex types in your params struct:

```go
type Params struct {
    Matrix  [][]int              `descr:"nested matrix" optional:"true"`
    Meta    map[string][]string  `descr:"metadata" optional:"true"`
    Config  map[string]any       `descr:"arbitrary config" optional:"true"`
}

// CLI usage:
//   --matrix '[[1,2],[3,4]]'
//   --meta '{"tags":["a","b"],"owners":["alice"]}'
//   --config '{"debug":true,"retries":3}'
```

The same JSON syntax works for environment variables. Config files work natively since they are already unmarshaled from JSON/YAML/etc.

### Simple Maps

Simple map types like `map[string]string` and `map[string]int` use the more ergonomic `key=val,key=val` syntax on the CLI, and only fall back to JSON for complex value types:

```go
type Params struct {
    Labels map[string]string   `descr:"simple key=val syntax"`
    Deep   map[string][]string `descr:"JSON syntax required"`
}
// --labels env=prod,team=backend
// --deep '{"groups":["admin","users"]}'
```

## Config-File-Only Fields with `boa:"configonly"` and `boa:"ignore"`

For fields that should only come from a config file — not `--flag`, not `$ENV` — boa offers two tags with slightly different semantics:

- **`boa:"configonly"`** — the field is hidden from the CLI and env but its mirror is **still created** and still runs validation (`min`/`max`/`pattern`, custom validators, required checks). Use this when you want config-file-only fields to still be validated.
- **`boa:"ignore"`** — the field is **fully excluded** from boa processing. No mirror, no validation, no required check; only raw config-file unmarshal can write to it. Use this when you want to opt out of boa entirely for that field (for example, nested config blobs that boa has no opinion on).

```go
type Params struct {
    ConfigFile string            `configfile:"true" optional:"true" default:"config.json"`
    Host       string            `descr:"server host"`
    Port       int               `descr:"server port"`
    InternalID string            `boa:"configonly" min:"8"` // validated
    Metadata   map[string]string `boa:"ignore"`             // opaque
}
```

With `config.json`:
```json
{
    "Host": "example.com",
    "Port": 8080,
    "InternalID": "abc-123",
    "Metadata": {"version": "2", "region": "us-east-1"}
}
```

`Host` and `Port` can be overridden via CLI flags; `InternalID` and `Metadata` are only loaded from the config file. `InternalID`'s `min:"8"` length check still runs; `Metadata` is passed through untouched.

Prior to this release, `boa:"configonly"` was an alias for `boa:"ignore"`. If you used `configonly` purely to hide a field from CLI/env and didn't rely on validation being skipped, the new behavior is a strict upgrade. If you need the old behavior (no mirror, no validation), switch to `boa:"ignore"`.

## Finer-Grained Skipping: `boa:"noflag"` and `boa:"noenv"`

`boa:"configonly"` covers the common "config-file only" case by hiding a field from both CLI and env. When you only want to skip one of those channels, boa provides two orthogonal directives that preserve the mirror (and therefore validation):

- **`boa:"noflag"` (alias `boa:"nocli"`)** — do not register a CLI flag for this field, but keep env vars, config file loading, defaults, and `min`/`max`/`pattern` validation active.
- **`boa:"noenv"`** — do not read this field from environment variables, but keep the CLI flag and config-file loading.

`boa:"configonly"` is exactly `noflag,noenv` (plus the mirror-preserving behavior described in the previous section).

```go
type Params struct {
    Host   string `descr:"server host"`
    Secret string `descr:"api token" boa:"noflag" env:"API_TOKEN" min:"20"`
    Debug  bool   `descr:"debug mode" boa:"noenv" optional:"true"`
}
```

Here `--secret` is not a flag, but `API_TOKEN=...` still sets it and the `min:"20"` length check still runs. `--debug` is a flag, but no `$DEBUG` env var is ever consulted (useful when you use `ParamEnricherEnv` to auto-derive env names but want this field excluded).

`boa:"noflag"` cannot be combined with `positional:"true"` — a positional arg is, by definition, a CLI argument.

## Programmatic Configuration (Tag Parity)

All struct-tag features have a matching setter on the `Param` / `ParamT[T]` interface, reachable via `HookContext.GetParam(&field)` or `boa.GetParamT(ctx, &field)`. This matters when your parameter struct comes from a third-party package and you cannot add tags:

```go
boa.CmdT[ThirdPartyConfig]{
    Use: "cmd",
    InitFuncCtx: func(ctx *boa.HookContext, p *ThirdPartyConfig, cmd *cobra.Command) error {
        token := boa.GetParamT(ctx, &p.Token)
        token.SetNoFlag(true)               // like boa:"noflag"
        token.SetEnv("MYAPP_TOKEN")         // like env:"MYAPP_TOKEN"
        token.SetDescription("API token")   // like descr:"API token"

        port := boa.GetParamT(ctx, &p.Port)
        port.SetMinT(1)                     // like min:"1"
        port.SetMaxT(65535)                 // like max:"65535"
        port.SetRequired(true)              // like required:"true"
        return nil
    },
}
```

| Tag | Programmatic equivalent |
|-----|--------------------------|
| `descr` / `desc` / `help` | `SetDescription(string)` |
| `name` / `long` | `SetName(string)` |
| `short` | `SetShort(string)` |
| `env` | `SetEnv(string)` |
| `default` | `SetDefault(any)` / `SetDefaultT[T](T)` |
| `positional` / `pos` | `SetPositional(bool)` |
| `required` / `req` | `SetRequired(bool)` or `SetRequiredFn(func() bool)` |
| `optional` / `opt` | `SetRequired(false)` |
| `alts` / `alternatives` | `SetAlternatives([]string)`, `SetAlternativesFunc(...)` |
| `strict` / `strict-alts` | `SetStrictAlts(bool)` |
| `min` | `ParamT[T].SetMinT(T)` for numeric, `SetMinLen(int)` for string/slice/map. `ClearMin()` removes. Non-generic `Param.SetMin(any)` accepts any numeric (coerced to `*int64` / `*float64` / `*int` per field kind). |
| `max` | `ParamT[T].SetMaxT(T)` / `SetMaxLen(int)` / `ClearMax()`. Symmetric with `min`. |
| `pattern` | `SetPattern(string)` |
| `boa:"noflag"` / `"nocli"` | `SetNoFlag(bool)` |
| `boa:"noenv"` | `SetNoEnv(bool)` |
| `boa:"ignore"` | `SetIgnored(bool)` (post-traversal equivalent; the tag form skips traversal entirely) |
| `boa:"configonly"` | `SetNoFlag(true)` + `SetNoEnv(true)` |

All programmatic setters must run in `InitFunc` / `InitFuncCtx` (or `CfgStructInit` / `CfgStructInitCtx`) so they take effect before cobra flag binding and env-var reading.

## Named Struct Auto-Prefixing

Named (non-anonymous) struct fields automatically prefix their children's flag names and env var names with the field name in kebab-case. This is the primary mechanism for avoiding flag name collisions when reusing struct types.

### How It Works

```go
type DBConfig struct {
    Host string `descr:"database host" default:"localhost"`
    Port int    `descr:"database port" default:"5432"`
}

type Params struct {
    Primary DBConfig  // --primary-host, --primary-port, env: PRIMARY_HOST, PRIMARY_PORT
    Replica DBConfig  // --replica-host, --replica-port, env: REPLICA_HOST, REPLICA_PORT
}
```

### Prefixing Rules

| Scenario | Flag Name | Env Var |
|----------|-----------|---------|
| Embedded `DBConfig` with `Host` | `--host` | `HOST` |
| Named `DB DBConfig` with `Host` | `--db-host` | `DB_HOST` |
| Deep `Infra.Primary.Host` | `--infra-primary-host` | `INFRA_PRIMARY_HOST` |
| Named field + explicit `name:"host"` | `--db-host` (prefixed) | N/A |
| Named field + explicit `env:"SERVER_HOST"` | N/A | `DB_SERVER_HOST` (prefixed) |
| Embedded + explicit `env:"MY_HOST"` | N/A | `MY_HOST` (not prefixed) |

### Deep Nesting

Prefixes chain at every named (non-anonymous) level:

```go
type ConnectionConfig struct {
    Host string `default:"localhost"`
    Port int    `default:"5432"`
}

type ClusterConfig struct {
    Primary ConnectionConfig
    Replica ConnectionConfig
}

type Params struct {
    Infra ClusterConfig
}
// Flags: --infra-primary-host, --infra-primary-port, --infra-replica-host, --infra-replica-port
// Env vars: INFRA_PRIMARY_HOST, INFRA_PRIMARY_PORT, etc.
```

### Explicit Tags Are Also Prefixed

Inside a named struct field, both auto-generated and explicit tag values get the parent prefix. This is intentional -- it avoids collisions when the same struct type appears multiple times:

```go
type ServerConfig struct {
    Host string `name:"host" env:"SERVER_HOST" default:"localhost"`
}

type Params struct {
    API ServerConfig  // flag: --api-host, env: API_SERVER_HOST
    Web ServerConfig  // flag: --web-host, env: WEB_SERVER_HOST
}
```

## Custom Type Registration

Register user-defined types as CLI parameters with `RegisterType`. The type is stored as a string flag in cobra and converted via your provided `Parse`/`Format` functions:

```go
type SemVer struct {
    Major, Minor, Patch int
}

func init() {
    boa.RegisterType[SemVer](boa.TypeDef[SemVer]{
        Parse: func(s string) (SemVer, error) {
            var v SemVer
            _, err := fmt.Sscanf(s, "%d.%d.%d", &v.Major, &v.Minor, &v.Patch)
            return v, err
        },
        Format: func(v SemVer) string {
            return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
        },
    })
}

type Params struct {
    Version SemVer `descr:"app version" default:"1.0.0"`
}
```

`TypeDef[T]` fields:

| Field | Type | Description |
|-------|------|-------------|
| `Parse` | `func(string) (T, error)` | Converts a CLI string into the typed value (required) |
| `Format` | `func(T) string` | Converts the typed value back to a string for default display. If nil, `fmt.Sprintf("%v", val)` is used |

## ConfigFormatExtensions

`boa.ConfigFormatExtensions()` returns the file extensions that have registered config format handlers. Always includes `.json` (registered by default). This is used by the `boaviper` subpackage for auto-discovery:

```go
exts := boa.ConfigFormatExtensions() // e.g., [".json", ".yaml"]
```

## Type Handler Registry (Architecture)

Internally, BOA uses a type handler registry (`type_handler.go`) instead of scattered type switches. Each handler provides:

- **`bindFlag`** -- how to create a cobra/pflag flag for this type
- **`parse`** -- how to convert a string value into the target type
- **`convert`** -- optional post-parse conversion (e.g., for types stored as strings in cobra)

Handlers are registered by exact type (for special types like `time.Time`, `net.IP`) or by `reflect.Kind` (for basic types like `string`, `int`). Map types use composed handlers that delegate value parsing to the appropriate scalar handler for their value type.

Types without a registered handler fall back to `StringP` + `json.Unmarshal`, which is how nested slices and complex maps are supported automatically.
