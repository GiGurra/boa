# BOA - Declarative Go CLI Framework

BOA is a declarative abstraction layer on top of `github.com/spf13/cobra` that uses struct-based parameter definitions with plain Go types and struct tags to automatically generate CLI flags, environment variable bindings, validation, and help text.

## Project Structure

```
pkg/boa/           # Main (and only) package
  api_base.go      # Cmd struct, ParamEnricher, HookContext, LoadConfigFile
  api_required.go  # required[T] internal type (unexported)
  api_optional.go  # optional[T] internal type (unexported)
  api_typed_base.go # CmdT[T] typed command (primary API)
  api_typed_param.go # ParamT[T] typed parameter view
  internal.go      # Core processing: reflection, parsing, validation
  defaults.go      # Global configuration (Init, WithDefaultOptional)
  *_test.go        # Unit tests

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
    Name    string `descr:"User name" env:"USER_NAME"`
    Port    int    `descr:"Port number" default:"8080" optional:"true"`
    Verbose bool   `short:"v" optional:"true"`
}
```

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

## Conventions

- **Naming**: PascalCase for exported types/funcs, camelCase for unexported
- **Struct literals**: Direct struct initialization for command configuration
- **Generics**: Used internally for type safety (required[T]/optional[T] are unexported)
- **Reflection**: Used for struct traversal and dynamic value setting
- **Error handling**: Return errors, don't panic

### Auto-generated Names
- Field `MyParam` becomes flag `--my-param` (kebab-case)
- Environment variable: `MY_PARAM` (UPPER_SNAKE_CASE)

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

## Value Priority

CLI args > Environment vars > Config file > Default > Zero value

## Adding New Features

1. Define in appropriate `api_*.go` file
2. Processing logic goes in `internal.go`
3. Add unit tests in corresponding `*_test.go`
4. Consider adding example in `internal/` if non-trivial
