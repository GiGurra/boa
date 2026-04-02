# Migration Guide

## From Old BOA (pre-v1.0) to BOA v1.0

BOA v1.0 removes the builder pattern and the `Required[T]`/`Optional[T]` generic wrapper types. Commands are now configured via struct literals, and parameters use plain Go types.

### Summary of Breaking Changes

| Old API (pre-v1.0) | New API (v1.0) |
|---------------------|----------------|
| `boa.NewCmdT[P]("name")` | `boa.CmdT[P]{Use: "name"}` |
| `.WithShort("desc")` | `Short: "desc"` |
| `.WithLong("desc")` | `Long: "desc"` |
| `.WithRunFunc(func(p *P) { ... })` | `RunFunc: func(p *P, cmd *cobra.Command, args []string) { ... }` |
| `.WithSubCmds(...)` | `SubCmds: boa.SubCmds(...)` |
| `boa.Required[string]` | `string` (required by default) |
| `boa.Optional[int]` | `int` with `optional:"true"` tag, or `*int` |
| `params.Name.Value()` | `params.Name` (direct field access) |
| `SupportedTypes` constraint | Removed -- `any` is used |

### Command Definition

**Before:**

```go
cmd := boa.NewCmdT[Params]("myapp").
    WithShort("My application").
    WithLong("A detailed description").
    WithRunFunc(func(params *Params) {
        fmt.Println(params.Name.Value())
    }).
    WithSubCmds(subCmd1, subCmd2)
cmd.Run()
```

**After:**

```go
boa.CmdT[Params]{
    Use:   "myapp",
    Short: "My application",
    Long:  "A detailed description",
    SubCmds: boa.SubCmds(subCmd1, subCmd2),
    RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
        fmt.Println(params.Name)
    },
}.Run()
```

### RunFunc Signature

The run function now receives the cobra command and args, matching cobra's own pattern:

**Before:**

```go
RunFunc: func(params *Params) {
    // no access to cmd or args
}
```

**After:**

```go
RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
    // full access to cobra command and positional args
}
```

The context-aware variant also changed:

**Before:**

```go
RunFuncCtx: func(ctx *boa.HookContext, params *Params) {
    // ...
}
```

**After:**

```go
RunFuncCtx: func(ctx *boa.HookContext, params *Params, cmd *cobra.Command, args []string) {
    // ...
}
```

### Parameter Types

`Required[T]` and `Optional[T]` are removed. Use plain Go types instead.

**Before:**

```go
type Params struct {
    Name    boa.Required[string] `descr:"User name" env:"USER_NAME"`
    Port    boa.Optional[int]    `descr:"Port number" default:"8080"`
    Verbose boa.Optional[bool]   `short:"v"`
}

// Accessing values:
fmt.Println(params.Name.Value())
if params.Port.HasValue() {
    fmt.Println(params.Port.Value())
}
```

**After:**

```go
type Params struct {
    Name    string `descr:"User name" env:"USER_NAME"`
    Port    int    `descr:"Port number" default:"8080" optional:"true"`
    Verbose bool   `short:"v" optional:"true"`
}

// Accessing values -- direct field access:
fmt.Println(params.Name)
fmt.Println(params.Port)
```

### Optional Parameters: Pointer Fields

For truly optional parameters where you need to distinguish "not set" from "zero value", use pointer types:

**Before:**

```go
type Params struct {
    Retries boa.Optional[int] `descr:"retry count"`
}

if params.Retries.HasValue() {
    fmt.Println(params.Retries.Value())
}
```

**After:**

```go
type Params struct {
    Retries *int `descr:"retry count"`
}

if params.Retries != nil {
    fmt.Println(*params.Retries)
}
```

Pointer fields are always optional by default, even without `optional:"true"` or `boa.WithDefaultOptional()`.

### New Features in v1.0

#### Map Fields

```go
type Params struct {
    Labels map[string]string `descr:"key=value labels"`
}
// Usage: --labels env=prod,team=backend
```

#### Config File Support

```go
type Params struct {
    ConfigFile string `configfile:"true" optional:"true" default:"config.json"`
    Host       string
    Port       int
}
```

#### Config-File-Only Fields

```go
type Params struct {
    ConfigFile string            `configfile:"true" optional:"true" default:"config.json"`
    Host       string            `descr:"server host"`
    InternalID string            `boa:"ignore"` // only loaded from config file
    Metadata   map[string]string `boa:"ignore"` // not exposed as CLI flag
}
```

#### JSON Fallback for Complex Types

```go
type Params struct {
    Matrix [][]int             `descr:"nested matrix" optional:"true"`
    Meta   map[string][]string `descr:"metadata" optional:"true"`
}
// Usage: --matrix '[[1,2],[3,4]]' --meta '{"tags":["a","b"]}'
```

#### Substruct Config Files

The `configfile:"true"` tag now works on fields inside nested structs. Each substruct can have its own config file. Priority: CLI > env > root config > substruct config > defaults.

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

#### Config Format Registry

Register custom config file formats by extension. JSON is the only format shipped by default:

```go
boa.RegisterConfigFormat(".yaml", yaml.Unmarshal)
boa.RegisterConfigFormat(".toml", toml.Unmarshal)
```

Resolution: explicit `ConfigUnmarshal` on the command > registered format by file extension > `json.Unmarshal` fallback.

#### Named Struct Auto-Prefixing

Named (non-anonymous) struct fields now auto-prefix their children's flag names and env var names. This is a behavioral change from pre-v1.0 where all nested struct fields were unprefixed.

```go
type DBConfig struct {
    Host string `default:"localhost"`
    Port int    `default:"5432"`
}

type Params struct {
    DB DBConfig  // v1.0: --db-host, --db-port (auto-prefixed)
                 // pre-v1.0: --host, --port (no prefix)
}
```

Embedded (anonymous) fields remain unprefixed as before. If you rely on the old unprefixed behavior for named fields, either embed the struct anonymously or use explicit `name:"..."` tags (noting that explicit tags are also prefixed inside named fields).

#### Global Default Optional

```go
boa.Init(boa.WithDefaultOptional())

type Params struct {
    Name   string `descr:"user name"`             // now optional
    Port   int    `descr:"port" required:"true"`   // still required
}
```

### HookContext and GetParam

The `HookContext` API is largely the same, but `GetParamT` no longer requires a `SupportedTypes` constraint -- it works with `any`:

**Before:**

```go
// GetParamT required SupportedTypes constraint
nameParam := boa.GetParamT[string](ctx, &params.Name)
```

**After:**

```go
// Works with any type
nameParam := boa.GetParamT(ctx, &params.Name)
```

### Step-by-Step Migration

1. **Replace command construction**: Change `boa.NewCmdT[P]("name").WithX(...)` chains to `boa.CmdT[P]{Use: "name", X: ...}` struct literals.

2. **Update RunFunc signatures**: Add `cmd *cobra.Command, args []string` parameters.

3. **Replace Required[T] with plain types**: `boa.Required[string]` becomes `string`. Fields are required by default.

4. **Replace Optional[T] with tagged types or pointers**: `boa.Optional[int]` becomes either `int` with `optional:"true"` tag, or `*int` for nil-distinguishable optionality.

5. **Remove .Value() calls**: Access fields directly (`params.Name` instead of `params.Name.Value()`).

6. **Remove .HasValue() calls**: Use `HookContext.HasValue(&params.Field)` in `RunFuncCtx`, or use pointer fields (`params.Field != nil`).

7. **Update imports**: Remove any imports of removed types.

## From Cobra to BOA

### Before (Pure Cobra)

```go
var port int
var host string

var rootCmd = &cobra.Command{
    Use:   "myapp",
    Short: "My application",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Printf("Host: %s, Port: %d\n", host, port)
    },
}

func init() {
    rootCmd.Flags().StringVarP(&host, "host", "H", "localhost", "Server hostname")
    rootCmd.Flags().IntVarP(&port, "port", "p", 8080, "Server port")
    rootCmd.MarkFlagRequired("host")
}

func main() {
    rootCmd.Execute()
}
```

### After (BOA)

```go
type Params struct {
    Host string `descr:"Server hostname" default:"localhost"`
    Port int    `descr:"Server port" default:"8080" optional:"true"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "myapp",
        Short: "My application",
        RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Host: %s, Port: %d\n", params.Host, params.Port)
        },
    }.Run()
}
```

## Incremental Migration

You don't have to migrate everything at once. BOA commands produce standard `*cobra.Command` objects, so you can mix them freely:

```go
// Start: all Cobra
rootCmd.AddCommand(serveCmd, migrateCmd, configCmd)

// Migrate one at a time
rootCmd.AddCommand(
    serveCmd,   // Still Cobra
    migrateCmd, // Still Cobra
    boa.CmdT[ConfigParams]{
        Use: "config",
        RunFunc: func(p *ConfigParams, cmd *cobra.Command, args []string) { /* ... */ },
    }.ToCobra(), // Now BOA
)
```

See [Cobra Interoperability](cobra-interop.md) for the full incremental migration strategy.

## Why Migrate?

BOA provides:

- **Declarative parameters** - Define flags as struct fields, no manual registration
- **Automatic flag generation** - Field names become kebab-case flags automatically
- **Type safety** - Parameters are typed struct fields, not `interface{}`
- **Built-in validation** - Required fields, alternatives, custom validators
- **Environment variable binding** - Automatic or custom env var support
- **Cleaner code** - No scattered `init()` functions or global variables
