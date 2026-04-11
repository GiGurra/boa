# BOA - Declarative Go CLI Framework

BOA is a declarative abstraction layer on top of `github.com/spf13/cobra` that uses struct-based parameter definitions with plain Go types and struct tags to automatically generate CLI flags, environment variable bindings, validation, and help text.

## Project Structure

```
pkg/boa/           # Main package
  api_base.go      # Cmd struct, ParamEnricher, HookContext, LoadConfigFile, ConfigFormatExtensions
  api_typed_base.go # CmdT[T] typed command (primary API)
  api_typed_param.go # ParamT[T] typed parameter view
  param_meta.go    # paramMeta: non-generic Param implementation
  type_handler.go  # Type handler registry, RegisterType[T], TypeDef[T]
  internal.go      # Core processing: reflection, traversal, validation (incl. min/max/pattern)
  defaults.go      # Global configuration (Init, WithDefaultOptional)
  *_test.go        # Unit tests

pkg/boaviper/      # Optional subpackage for Viper-like config discovery
  boaviper.go      # AutoConfig, FindConfig, SetEnvPrefix

internal/          # Example programs and integration tests
  example*/        # Various usage examples
  testmain*/       # Test fixtures with different feature combinations
```

## Key Patterns

### Primary API
```go
boa.CmdT[MyParams]{
    Use:   "command-name",
    Short: "description",
    SubCmds: boa.SubCmds(subCmd1, subCmd2),
    RunFunc: func(params *MyParams, cmd *cobra.Command, args []string) {
        // use params.Field directly
    },
}.Run()
```

### Parameter Definition
```go
type Params struct {
    Name    string            `descr:"User name" env:"USER_NAME"`
    Port    int               `descr:"Port number" default:"8080" optional:"true"`
    Verbose *bool             `short:"v"`           // pointer = optional, nil = not set
    Labels  map[string]string `descr:"Key-value labels"` // maps default optional
    Matrix  [][]int           `descr:"Data matrix" boa:"ignore"` // config-file only
}
```

### Supported Field Types
- **Primitives**: `string`, `int`, `int32`, `int64`, `float32`, `float64`, `bool`
- **Pointer types**: `*string`, `*int`, `*bool`, etc. — optional by default, nil = not set
- **Special types**: `time.Time`, `time.Duration`, `net.IP`, `*url.URL`
- **Slices**: `[]string`, `[]int`, etc. — all basic slice types
- **Maps**: `map[string]string`, `map[string]int` — CLI syntax: `key=val,key=val`
- **Complex types**: `[][]string`, `map[string][]int`, etc. — CLI syntax: JSON strings
- **Type aliases**: `type MyString string` works transparently

### Struct Tags
- `descr` / `desc` - Description text
- `default` - Default value
- `env` - Environment variable name
- `short` - Short flag (single char)
- `positional` - Marks positional argument
- `required` / `req` - Marks as required (default for plain types)
- `optional` / `opt` - Marks as optional
- `alts` - Allowed values (enum validation)
- `strict` - Validate against alts
- `min` - Minimum value (numeric) or minimum length (string/slice)
- `max` - Maximum value (numeric) or maximum length (string/slice)
- `pattern` - Regex pattern for string validation
- `configfile` - Auto-load config file from this field's path (works in root and nested structs)
- `boa:"ignore"` (aliases `boa:"ignored"`, `boa:"-"`) - Fully excluded from boa: no mirror, no CLI flag, no env read, no validation. Only raw config-file unmarshal writes to the field. Use for opaque blobs.
- `boa:"configonly"` - Hidden from CLI and env but the **mirror is preserved**: `min`/`max`/`pattern`, custom validators, and required checks still run. Desugars to `noflag + noenv`. Use for validated config-file-only fields. (Was an alias for `boa:"ignore"` in older releases — the current semantics are strictly more useful.)
- `boa:"noflag"` / `boa:"nocli"` - Skip CLI flag registration **only**; env vars, config files, validation, and defaults still apply. Cannot be combined with `positional`.
- `boa:"noenv"` - Skip env var reading **only**; CLI flags and config files still apply. Most useful with `ParamEnricherEnv` to opt individual fields out of the auto-generated env binding.

### Programmatic configuration (struct-tag parity)

Every struct-tag feature has a matching method on `Param` / `ParamT[T]` so fields from third-party structs (which you can't tag) can still be fully configured from `InitFunc` / `InitFuncCtx`:

- `SetDescription(string)` / `GetDescription() string`
- `SetName(string)`, `SetShort(string)`, `SetEnv(string)`
- `SetPositional(bool)` / `IsPositional() bool`
- `SetRequired(bool)` — convenience over `SetRequiredFn`
- `SetNoFlag(bool)` / `IsNoFlag() bool` — mirrors `boa:"noflag"`
- `SetNoEnv(bool)` / `IsNoEnv() bool` — mirrors `boa:"noenv"`
- `SetIgnored(bool)` / `IsIgnored() bool` — post-traversal equivalent of `boa:"ignore"` (the tag itself skips traversal entirely, so the mirror never exists; the programmatic form marks an existing mirror as ignored so CLI/env/validation/sync are all skipped). For `boa:"configonly"`, call `SetNoFlag(true)` + `SetNoEnv(true)` instead.
- `SetMinT(T)` / `SetMaxT(T)` on `ParamT[T]` for numeric `T` (stores at full int64/float64 precision). `SetMinLen(int)` / `SetMaxLen(int)` for string / slice / map fields. `ClearMin()` / `ClearMax()` on both. The non-generic `Param` exposes `GetMin() any` / `SetMin(any)` / `ClearMin()` (same for Max), returning a typed pointer: `*int64` for signed ints, `*float64` for floats, `*int` for length-based fields. `SetPattern(string)` / `GetPattern() string` unchanged.
- `SetDefault(any)` / typed `ParamT[T].SetDefaultT(T)`
- `SetAlternatives([]string)`, `SetAlternativesFunc(...)`, `SetStrictAlts(bool)`
- `SetCustomValidator(func(any) error)` / typed `SetCustomValidatorT(func(T) error)`

Programmatic calls must happen in `InitFunc` / `InitFuncCtx` so they take effect before cobra flag binding (`connect`) and before env parsing.

### Config Format Registry
- Dispatch is **extension-driven**: every `loadConfigFileInto` call resolves the format from `filepath.Ext(filePath)` against the global `configFormats` map, so one binary can load any mix of registered formats at runtime.
- `boa.ConfigFormat{Unmarshal, KeyTree}` describes a full format: an unmarshaler plus an optional `KeyTree func([]byte) (map[string]any, error)` used for set-by-config detection.
- `boa.RegisterConfigFormat(".yaml", yaml.Unmarshal)` is the one-liner for every mainstream Go parser. It wraps the unmarshaler in a `UniversalConfigFormat`, which synthesizes `KeyTree` by decoding the same bytes into `map[string]any`. Full key-presence detection is automatic; panics on nil.
- `boa.UniversalConfigFormat(unmarshalFunc)` is the exported helper for inline use with `Cmd.ConfigFormat`. Panics on nil.
- `boa.RegisterConfigFormatFull(".mycustom", boa.ConfigFormat{...})` is the advanced form — reach for it only when the parser cannot decode into `map[string]any` (e.g., a handwritten format that only populates specific struct types), in which case you supply a hand-written `KeyTree`.
- JSON is the only format shipped by default (built-in `KeyTree` backed by `json.Unmarshal`).
- Registry access is guarded by a `sync.RWMutex`; registration is goroutine-safe. Still, the normal pattern is to register from `init()` / startup for clarity.
- `boa.ConfigFormatExtensions()` returns all registered file extensions, **sorted alphabetically** for deterministic iteration (important for `boaviper.FindConfig` which probes the same search path with every registered extension).
- Snapshot fallback for formats without a usable `KeyTree` is scoped per-load — a failing sub-load only triggers fallback within its own subtree, so it cannot corrupt the precision of sibling loads whose KeyTree succeeded.
- `Cmd.ConfigFormat` / legacy `Cmd.ConfigUnmarshal` are **per-command escape hatches** that lock that one command to a single format, bypassing the extension registry. Prefer registry-based dispatch unless you have a specific reason (legacy blob ingestion, test fixtures).
- Resolution per load: `Cmd.ConfigFormat` > `Cmd.ConfigUnmarshal` > extension-registered format > JSON fallback.
- `KeyTree` may return nested values as `map[string]any` (native) or `map[any]any` (e.g. yaml.v2); boa coerces transparently via `asKeyMap`.
- Substruct `configfile:"true"` fields load their own config files; priority: CLI > env > root config > substruct config > defaults

### Custom Type Registration
- `boa.RegisterType[T](TypeDef[T]{Parse, Format})` registers user-defined types as CLI parameters
- Registered types are stored as string flags in cobra and converted via Parse/Format functions
- See `type_handler.go` for the registry and `custom_type_test.go` for examples

### Boaviper Subpackage (`pkg/boaviper/`)
- `boaviper.AutoConfig[T]("appname")` - InitFunc for auto-discovering config files in standard paths
- `boaviper.FindConfig("appname")` - Searches standard paths (`./<app>`, `~/.config/<app>/config`, `/etc/<app>/config`) trying every extension returned by `boa.ConfigFormatExtensions()` (so `.json` plus anything registered via `RegisterConfigFormat` / `RegisterConfigFormatFull`)
- `boaviper.SetEnvPrefix("PREFIX")` - Enricher that combines `ParamEnricherEnv` + `ParamEnricherEnvPrefix`
- Uses `boa.ConfigFormatExtensions()` to try all registered formats

## Architecture

### Type Handler Registry (`type_handler.go`)
Types are handled via a registry instead of scattered type switches. Each handler provides:
- `bindFlag` — how to create a cobra flag for this type
- `parse` — how to parse a string value into this type
- `convert` — optional post-parse conversion (for types stored as strings in cobra)

Handlers are registered by exact type (special types) or reflect.Kind (basic types).
Maps use composed handlers that delegate value parsing to scalar handlers.
Complex types without native pflag support fall back to `StringP` + `json.Unmarshal`.

### Parameter Mirror (`param_meta.go`)
A single non-generic `paramMeta` struct implements the `Param` interface, replacing the old `required[T]`/`optional[T]` generic types. It uses `reflect.Value` for typed storage and tracks metadata, state, and validation.

### Value Priority
CLI args > Environment vars > Root config file > Substruct config files > Default > Zero value

## Conventions

- **Naming**: PascalCase for exported types/funcs, camelCase for unexported
- **Struct literals**: Direct struct initialization for command configuration
- **Reflection**: Used for struct traversal and dynamic value setting
- **Error handling**: Return errors, don't panic

### Auto-generated Names
- Field `MyParam` becomes flag `--my-param` (kebab-case)
- Acronyms handled correctly: `DBHost` → `db-host`, `HTTPPort` → `http-port`, `FB` → `fb`
- Environment variable: `MY_PARAM` (UPPER_SNAKE_CASE)

### Named Struct Auto-Prefixing
- Named (non-anonymous) struct fields auto-prefix their children's flag names and env var names
- `DB DBConfig` where DBConfig has `Host string` → flag `--db-host`, env `DB_HOST`
- Embedded (anonymous) `DBConfig` → flag `--host`, no prefix
- 3+ levels work: `Infra.Primary.Host` → `--infra-primary-host`, env `INFRA_PRIMARY_HOST`
- Explicit `name:"..."` and `env:"..."` tags also get prefixed inside named fields
- This prevents collisions when the same struct is used in multiple named fields

### Struct Pointer Fields (Optional Parameter Groups)
- `DB *DBConfig` — pointer struct fields act as optional parameter groups
- Nil struct pointers are preallocated during init, so their child flags are registered
- After parsing, if no field within the struct was set (CLI, env, or config), the pointer is nil'd back
- `p.DB == nil` means nothing in the group was configured; `p.DB != nil` means at least one field was set
- Defaults alone don't keep the struct alive — only explicit user input does
- Nested pointer structs work: `Outer *OuterConfig` where OuterConfig has `Inner *InnerConfig`
- Config file key-presence detection handles zero-value and same-as-default config entries for any format whose `ConfigFormat` supplies a `KeyTree` probe. `RegisterConfigFormat(ext, fn)` auto-synthesizes one via `UniversalConfigFormat`, so YAML/TOML/HCL get it for free; only formats whose unmarshaler cannot decode into `map[string]any` need to supply their own.
- Works with all features: validation tags, custom validators, alternatives, HookContext access

## Testing

Run all tests:
```bash
go test ./...
```

Tests use:
- `os.Args` injection for CLI simulation
- `cmd.RunArgsE([]string{...})` for argument testing with error handling
- Integration tests in `internal/testmain*/` that call `main()`

## CI/CD

- **CI**: Runs `go test ./...` on push/PR to main
- **Release**: Auto-increments patch version on main push, creates GitHub release
- **Manual release**: Supports patch/minor/major via workflow dispatch
- **Dependencies**: Managed by Renovate (auto-updates)

## Adding New Types

Register a handler in `type_handler.go`:
1. Add to `exactTypeHandlers` (for specific types) or `kindHandlers` (for kind-based)
2. Provide `bindFlag`, `parse`, and optionally `convert` functions
3. No changes needed elsewhere — the handler registry is the single point of extension
