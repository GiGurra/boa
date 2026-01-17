# Error Handling

BOA provides two execution modes with different error handling behaviors: `Run()` which panics on errors, and `RunE()` which returns errors for programmatic handling.

## Run() vs RunE()

| Method | Setup Errors | Hook Errors | Runtime Errors |
|--------|--------------|-------------|----------------|
| `Run()` | Panic | Panic | Panic |
| `RunE()` | Panic | Return | Return |
| `RunArgs(args)` | Panic | Panic | Panic |
| `RunArgsE(args)` | Panic | Return | Return |

**Setup Errors** are programming mistakes that should be caught during development - they always panic regardless of whether you use `Run()` or `RunE()`. This includes:

- Invalid struct tags (e.g., `default:"abc"` on an `int` field)
- Malformed tag syntax
- Setting multiple run functions
- Unsupported field types

### Using Run()

`Run()` is the simplest way to execute a command. All errors cause panics, which is suitable for top-level CLI execution where errors should terminate the program:

```go
boa.NewCmdT[Params]("app").
    WithRunFunc(func(p *Params) {
        // Your command logic
    }).
    Run() // Panics on any error
```

### Using RunE()

`RunE()` returns all errors for programmatic handling. This is ideal for testing, embedding commands in larger applications, or custom error handling:

```go
err := boa.NewCmdT[Params]("app").
    WithRunFuncE(func(p *Params) error {
        if p.Port < 1024 {
            return fmt.Errorf("port must be >= 1024")
        }
        return nil
    }).
    RunE()

if err != nil {
    log.Printf("Command failed: %v", err)
    os.Exit(1)
}
```

## Error-Returning Run Functions

BOA provides error-returning variants of all run function types:

| Non-Error Variant | Error Variant | Description |
|-------------------|---------------|-------------|
| `RunFunc` | `RunFuncE` | Basic run function |
| `RunFuncCtx` | `RunFuncCtxE` | Run function with HookContext access |

### Builder Methods

=== "Simple Signature"

    ```go
    boa.NewCmdT[Params]("app").
        WithRunFuncE(func(p *Params) error {
            return doWork(p)
        })
    ```

=== "Full Signature"

    ```go
    boa.NewCmdT[Params]("app").
        WithRunFuncE3(func(p *Params, cmd *cobra.Command, args []string) error {
            return doWork(p)
        })
    ```

=== "With HookContext"

    ```go
    boa.NewCmdT[Params]("app").
        WithRunFuncCtxE(func(ctx *boa.HookContext, p *Params) error {
            if ctx.HasValue(&p.OptionalField) {
                // Field was explicitly set
            }
            return nil
        })
    ```

=== "Full Signature with HookContext"

    ```go
    boa.NewCmdT[Params]("app").
        WithRunFuncCtxE4(func(ctx *boa.HookContext, p *Params, cmd *cobra.Command, args []string) error {
            return nil
        })
    ```

## Error Types

BOA handles three categories of errors:

### 1. Setup Errors (Always Panic)

These are programming mistakes that indicate bugs in your code. They always panic regardless of whether you use `Run()` or `RunE()`:

- Invalid default value types (e.g., `default:"abc"` on an `int` field)
- Malformed struct tag syntax
- Unsupported field types
- Setting multiple run functions
- Positional argument ordering errors

```go
type Params struct {
    Port int `default:"not-a-number"` // Will panic during setup
}

// This also panics - multiple run functions configured
boa.CmdT[Params]{
    RunFunc:  func(p *Params, cmd *cobra.Command, args []string) {},
    RunFuncE: func(p *Params, cmd *cobra.Command, args []string) error { return nil },
}.RunE() // Panics!
```

These should be caught during development and fixed in the source code.

### 2. Hook Errors

These occur during the command lifecycle:

- `InitFunc` / `InitFuncCtx` errors
- `PostCreateFunc` / `PostCreateFuncCtx` errors
- `PreValidateFunc` / `PreValidateFuncCtx` errors
- `PreExecuteFunc` / `PreExecuteFuncCtx` errors

```go
err := boa.NewCmdT[Params]("app").
    WithInitFuncE(func(p *Params) error {
        return fmt.Errorf("initialization failed")
    }).
    RunE()
// err: "error in InitFunc: initialization failed"
```

### 3. Runtime Errors

These occur during command execution from your `RunFuncE`:

```go
err := boa.NewCmdT[Params]("app").
    WithRunFuncE(func(p *Params) error {
        return fmt.Errorf("something went wrong")
    }).
    RunArgsE([]string{"--name", "test"})
// err: "something went wrong"
```

## ToCobra() vs ToCobraE()

When you need the underlying Cobra command:

| Method | Returns | Setup Errors | Hook Errors |
|--------|---------|--------------|-------------|
| `ToCobra()` | `*cobra.Command` | Panic | Panic |
| `ToCobraE()` | `(*cobra.Command, error)` | Panic | Return |

```go
// ToCobra - panics on any error
cmd := boa.NewCmdT[Params]("app").
    WithRunFunc(func(p *Params) {}).
    ToCobra()

// ToCobraE - panics on setup errors, returns hook errors
cmd, err := boa.NewCmdT[Params]("app").
    WithInitFuncE(func(p *Params) error { return nil }).
    WithRunFuncE(func(p *Params) error { return nil }).
    ToCobraE()
if err != nil {
    // Handle hook error (InitFunc ran during ToCobraE)
}
```

## Testing with Error Handling

`RunE()` and `RunArgsE()` are ideal for testing:

```go
func TestMyCommand_InvalidPort(t *testing.T) {
    err := boa.NewCmdT[Params]("app").
        WithRunFuncE(func(p *Params) error {
            if p.Port < 1024 {
                return fmt.Errorf("port must be >= 1024, got %d", p.Port)
            }
            return nil
        }).
        RunArgsE([]string{"--port", "80"})

    if err == nil {
        t.Fatal("expected error for port < 1024")
    }
    if !strings.Contains(err.Error(), "port must be >= 1024") {
        t.Fatalf("unexpected error: %v", err)
    }
}

func TestMyCommand_HookError(t *testing.T) {
    err := boa.NewCmdT[Params]("app").
        WithPreValidateFuncE(func(p *Params, cmd *cobra.Command, args []string) error {
            return fmt.Errorf("validation failed")
        }).
        WithRunFuncE(func(p *Params) error {
            t.Fatal("should not reach here")
            return nil
        }).
        RunArgsE([]string{"--name", "test"})

    if err == nil {
        t.Fatal("expected error from PreValidateFunc")
    }
}
```

## Best Practices

### Use Run() for simple CLIs

When errors should just terminate the program:

```go
func main() {
    boa.NewCmdT[Params]("app").
        WithRunFunc(func(p *Params) {
            // Errors here can use log.Fatal or panic
        }).
        Run()
}
```

### Use RunE() for testable commands

When you need to verify error conditions:

```go
func runApp(args []string) error {
    return boa.NewCmdT[Params]("app").
        WithRunFuncE(func(p *Params) error {
            return doWork(p)
        }).
        RunArgsE(args)
}

// In tests
func TestApp(t *testing.T) {
    err := runApp([]string{"--invalid-flag"})
    // Assert on err
}
```

### Use RunE() for embedded commands

When your CLI is part of a larger application:

```go
func (s *Server) handleCLI(args []string) error {
    return boa.NewCmdT[Params]("admin").
        WithRunFuncE(func(p *Params) error {
            return s.adminOperation(p)
        }).
        RunArgsE(args)
}
```

## Only One Run Function

You can only set one run function per command. Setting multiple will cause an error:

```go
// This will error - can't use both RunFunc and RunFuncE
boa.CmdT[Params]{
    RunFunc:  func(p *Params, cmd *cobra.Command, args []string) {},
    RunFuncE: func(p *Params, cmd *cobra.Command, args []string) error { return nil },
}
```

Choose the variant that matches your error handling needs:

- `RunFunc` / `RunFuncCtx` - when using `Run()`
- `RunFuncE` / `RunFuncCtxE` - when using `RunE()` or need error returns
