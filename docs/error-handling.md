# Error Handling

BOA provides two execution modes: `Run()` for simple CLI apps, and `RunE()` for programmatic error handling.

## Run() vs RunE()

| Method | Setup Errors | User Input Errors | Hook Errors | Runtime Errors |
|--------|--------------|-------------------|-------------|----------------|
| `Run()` | Panic | Exit(1) | Exit(1) | Panic |
| `RunE()` | Panic | Return | Return | Return |
| `RunArgs(args)` | Panic | Exit(1) | Exit(1) | Panic |
| `RunArgsE(args)` | Panic | Return | Return | Return |

### Using Run()

`Run()` is the simplest way to execute a command. User input errors print a message and exit cleanly. See the table above for full behavior.

=== "Direct API"

    ```go
    func main() {
        boa.CmdT[Params]{
            Use: "app",
            RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
                // Your command logic
            },
        }.Run()
    }
    ```

=== "Builder API"

    ```go
    func main() {
        boa.NewCmdT[Params]("app").
            WithRunFunc(func(p *Params) {
                // Your command logic
            }).
            Run()
    }
    ```

### Using RunE()

`RunE()` returns all non-setup errors for programmatic handling. Ideal for testing, embedding, or custom error handling.

=== "Direct API"

    ```go
    err := boa.CmdT[Params]{
        Use: "app",
        RunFuncE: func(p *Params, cmd *cobra.Command, args []string) error {
            if p.Port < 1024 {
                return fmt.Errorf("port must be >= 1024")
            }
            return nil
        },
    }.RunE()

    if err != nil {
        log.Printf("Command failed: %v", err)
        os.Exit(1)
    }
    ```

=== "Builder API"

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

## Error Types

BOA handles four categories of errors (see table above for behavior):

### 1. Setup Errors

Programming mistakes caught during command setup. Always panic.

- Invalid default value types (e.g., `default:"abc"` on an `int` field)
- Malformed struct tag syntax
- Unsupported field types
- Setting multiple run functions
- Positional argument ordering errors

```go
type Params struct {
    Port int `default:"not-a-number"` // Will panic during setup
}
```

### 2. User Input Errors

Invalid input from the CLI user. With `Run()`: prints error and exits(1). With `RunE()`: returns error.

- Missing required parameters
- Invalid flag values (e.g., `--port abc` for an integer flag)
- Unknown flags (e.g., `--unknown-flag`)
- Invalid alternatives (enum validation failures)
- Custom validator failures
- Invalid environment variable values
- Missing positional arguments

```go
type Params struct {
    Name string `short:"n" required:"true"`
    Mode string `default:"fast" alts:"fast,slow"`
}

// User runs: myapp --mode=invalid
// Output: Error: invalid value for param 'mode': 'invalid' is not in the list of allowed values: [fast slow]
// Exit code: 1
```

#### Creating User Input Errors in Hooks

Use `NewUserInputError` or `NewUserInputErrorf` to return user input errors from hooks:

=== "Direct API"

    ```go
    boa.CmdT[Params]{
        Use: "app",
        PreValidateFunc: func(p *Params, cmd *cobra.Command, args []string) error {
            if p.StartPort > p.EndPort {
                return boa.NewUserInputErrorf("start port must be less than end port")
            }
            return nil
        },
    }
    ```

=== "Builder API"

    ```go
    boa.NewCmdT[Params]("app").
        WithPreValidateFuncE(func(p *Params, cmd *cobra.Command, args []string) error {
            if p.StartPort > p.EndPort {
                return boa.NewUserInputErrorf("start port must be less than end port")
            }
            return nil
        })
    ```

#### Checking for User Input Errors

```go
err := cmd.RunArgsE([]string{"--invalid-flag"})
if boa.IsUserInputError(err) {
    // Handle user input error
}
```

### 3. Hook Errors

Errors from lifecycle hooks (Init, PostCreate, PreValidate, PreExecute). Behavior depends on `Run()` vs `RunE()` - see table.

=== "Direct API"

    ```go
    err := boa.CmdT[Params]{
        Use: "app",
        InitFunc: func(p *Params, cmd *cobra.Command) error {
            return fmt.Errorf("initialization failed")
        },
    }.RunE()
    // err: "error in InitFunc: initialization failed"
    ```

=== "Builder API"

    ```go
    err := boa.NewCmdT[Params]("app").
        WithInitFuncE(func(p *Params) error {
            return fmt.Errorf("initialization failed")
        }).
        RunE()
    // err: "error in InitFunc: initialization failed"
    ```

### 4. Runtime Errors

Errors from your `RunFuncE`. Behavior depends on `Run()` vs `RunE()` - see table.

=== "Direct API"

    ```go
    err := boa.CmdT[Params]{
        Use: "app",
        RunFuncE: func(p *Params, cmd *cobra.Command, args []string) error {
            return fmt.Errorf("something went wrong")
        },
    }.RunArgsE([]string{"--name", "test"})
    // err: "something went wrong"
    ```

=== "Builder API"

    ```go
    err := boa.NewCmdT[Params]("app").
        WithRunFuncE(func(p *Params) error {
            return fmt.Errorf("something went wrong")
        }).
        RunArgsE([]string{"--name", "test"})
    // err: "something went wrong"
    ```

## Error-Returning Run Functions

| Non-Error Variant | Error Variant | Description |
|-------------------|---------------|-------------|
| `RunFunc` | `RunFuncE` | Basic run function |
| `RunFuncCtx` | `RunFuncCtxE` | Run function with HookContext access |

## ToCobra() vs ToCobraE()

| Method | Returns | Setup Errors | Hook Errors |
|--------|---------|--------------|-------------|
| `ToCobra()` | `*cobra.Command` | Panic | Panic |
| `ToCobraE()` | `(*cobra.Command, error)` | Panic | Return |

## Testing

Use `RunE()` and `RunArgsE()` for testing:

=== "Direct API"

    ```go
    func TestMyCommand_InvalidPort(t *testing.T) {
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

=== "Builder API"

    ```go
    func TestMyCommand_InvalidPort(t *testing.T) {
        err := boa.NewCmdT[Params]("app").
            WithRunFuncE(func(p *Params) error {
                if p.Port < 1024 {
                    return fmt.Errorf("port must be >= 1024")
                }
                return nil
            }).
            RunArgsE([]string{"--port", "80"})

        if err == nil {
            t.Fatal("expected error for port < 1024")
        }
    }
    ```

## Only One Run Function

You can only set one run function per command. Setting multiple causes a setup error (panic):

=== "Direct API"

    ```go
    // This will panic - can't use both RunFunc and RunFuncE
    boa.CmdT[Params]{
        Use:      "app",
        RunFunc:  func(p *Params, cmd *cobra.Command, args []string) {},
        RunFuncE: func(p *Params, cmd *cobra.Command, args []string) error { return nil },
    }
    ```

=== "Builder API"

    ```go
    // This will panic - can't use both WithRunFunc and WithRunFuncE
    boa.NewCmdT[Params]("app").
        WithRunFunc(func(p *Params) {}).
        WithRunFuncE(func(p *Params) error { return nil })
    ```
