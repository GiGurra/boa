# BOA - Declarative Go CLI Framework

BOA is a declarative abstraction layer on top of `github.com/spf13/cobra` that uses struct-based parameter definitions with Go generics (`Required[T]` and `Optional[T]`) to automatically generate CLI flags, environment variable bindings, validation, and help text.

## Project Structure

```
pkg/boa/           # Main (and only) package
  api_base.go      # Base Cmd struct, ParamEnricher definitions
  api_required.go  # Required[T] generic type
  api_optional.go  # Optional[T] generic type
  api_typed_base.go # CmdT[T] typed command builder (primary API)
  internal.go      # Core processing: reflection, parsing, validation
  *_test.go        # Unit tests

internal/          # Example programs and integration tests
  example*/        # Various usage examples
  testmain*/       # Test fixtures with different feature combinations
```

## Key Patterns

### Primary API
```go
cmd := boa.NewCmdT[MyParams]("command-name").
    WithShort("description").
    WithRunFunc(func(params *MyParams) {
        // use params.Field.Value()
    }).
    WithSubCmds(subCmd1, subCmd2)
cmd.Run()
```

### Parameter Definition
```go
type Params struct {
    Name    boa.Required[string] `descr:"User name" env:"USER_NAME"`
    Port    boa.Optional[int]    `descr:"Port number" default:"8080"`
    Verbose boa.Optional[bool]   `short:"v"`
}
```

### Struct Tags
- `descr` / `desc` - Description text
- `default` - Default value
- `env` - Environment variable name
- `short` - Short flag (single char)
- `positional` - Marks positional argument
- `alts` - Allowed values (enum validation)

## Conventions

- **Naming**: PascalCase for exported types/funcs, camelCase for unexported
- **Builder pattern**: Fluent API with `With*` methods returning self
- **Generics**: Heavy use of generics for type safety
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
- `cmd.RunArgs([]string{...})` for argument testing
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
