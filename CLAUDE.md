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
- `boa:"ignore"` - Skip CLI/env registration (still loaded from config files)
- `boa:"configonly"` - Alias for `boa:"ignore"` (clearer intent for config-file-only fields)

### Config Format Registry
- `boa.RegisterConfigFormat(".yaml", yaml.Unmarshal)` registers custom config formats by file extension
- JSON is the only format shipped by default
- `boa.ConfigFormatExtensions()` returns all registered file extensions (used by `boaviper`)
- Resolution: explicit `Cmd.ConfigUnmarshal` > file extension registry > `json.Unmarshal` fallback
- Substruct `configfile:"true"` fields load their own config files; priority: CLI > env > root config > substruct config > defaults

### Custom Type Registration
- `boa.RegisterType[T](TypeDef[T]{Parse, Format})` registers user-defined types as CLI parameters
- Registered types are stored as string flags in cobra and converted via Parse/Format functions
- See `type_handler.go` for the registry and `custom_type_test.go` for examples

### Boaviper Subpackage (`pkg/boaviper/`)
- `boaviper.AutoConfig[T]("appname")` - InitFunc for auto-discovering config files in standard paths
- `boaviper.FindConfig("appname")` - Searches standard paths (`./<app>.json`, `~/.config/<app>/config.json`, `/etc/<app>/config.json`)
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
