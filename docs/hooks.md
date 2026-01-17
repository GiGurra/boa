# Lifecycle Hooks

BOA provides lifecycle hooks to customize behavior at different stages of command execution.

## Hook Execution Order

1. **Init** - Parameter mirrors exist, cobra flags not yet created
2. **PostCreate** - Cobra flags are now registered
3. **PreValidate** - After flags are parsed but before validation
4. **Validation** - Built-in parameter validation
5. **PreExecute** - After validation but before command execution
6. **Run** - The actual command execution

## Init Hook

Runs during initialization, after BOA creates internal parameter mirrors but before cobra flags are registered.

### Interface-based

```go
func (c *MyConfig) Init() error {
    // Initialize defaults, set up validators
    return nil
}

// With HookContext access
func (c *MyConfig) InitCtx(ctx *boa.HookContext) error {
    ctx.GetParam(&c.Host).SetDefault(boa.Default("localhost"))
    return nil
}
```

### Function-based

=== "Direct API"

    ```go
    boa.CmdT[Params]{
        Use: "cmd",
        InitFunc: func(params *Params, cmd *cobra.Command) error {
            return nil
        },
    }

    // With HookContext
    boa.CmdT[Params]{
        Use: "cmd",
        InitFuncCtx: func(ctx *boa.HookContext, params *Params, cmd *cobra.Command) error {
            ctx.GetParam(&params.Name).SetShort("n")
            return nil
        },
    }
    ```

=== "Builder API"

    ```go
    boa.NewCmdT[Params]("cmd").
        WithInitFuncE(func(params *Params) error {
            return nil
        })

    // With HookContext
    boa.NewCmdT[Params]("cmd").
        WithInitFuncCtx(func(ctx *boa.HookContext, params *Params, cmd *cobra.Command) error {
            ctx.GetParam(&params.Name).SetShort("n")
            return nil
        })
    ```

## PostCreate Hook

Runs after cobra flags are created but before arguments are parsed. Useful for inspecting or modifying cobra flags.

### Interface-based

```go
func (c *MyConfig) PostCreate() error {
    // Flags are now registered
    return nil
}

// With HookContext
func (c *MyConfig) PostCreateCtx(ctx *boa.HookContext) error {
    return nil
}
```

### Function-based

=== "Direct API"

    ```go
    boa.CmdT[Params]{
        Use: "cmd",
        PostCreateFuncCtx: func(ctx *boa.HookContext, params *Params, cmd *cobra.Command) error {
            flag := cmd.Flags().Lookup("my-flag")
            if flag != nil {
                // Inspect or modify flag
            }
            return nil
        },
    }
    ```

=== "Builder API"

    ```go
    boa.NewCmdT[Params]("cmd").
        WithPostCreateFuncCtx(func(ctx *boa.HookContext, params *Params, cmd *cobra.Command) error {
            flag := cmd.Flags().Lookup("my-flag")
            if flag != nil {
                // Inspect or modify flag
            }
            return nil
        })
    ```

## PreValidate Hook

Runs after parameters are parsed but before validation. Ideal for loading config files.

### Interface-based

```go
func (c *MyConfig) PreValidate() error {
    // Manipulate parameters before validation
    return nil
}

// With HookContext
func (c *MyConfig) PreValidateCtx(ctx *boa.HookContext) error {
    return nil
}
```

### Function-based

=== "Direct API"

    ```go
    boa.CmdT[Params]{
        Use: "cmd",
        PreValidateFunc: func(params *Params, cmd *cobra.Command, args []string) error {
            return nil
        },
    }
    ```

=== "Builder API"

    ```go
    boa.NewCmdT[Params]("cmd").
        WithPreValidateFuncE(func(params *Params, cmd *cobra.Command, args []string) error {
            return nil
        })
    ```

## PreExecute Hook

Runs after validation but before the Run function. Use for setup like establishing connections.

### Interface-based

```go
func (c *MyConfig) PreExecute() error {
    // Setup resources
    return nil
}
```

### Function-based

=== "Direct API"

    ```go
    boa.CmdT[Params]{
        Use: "cmd",
        PreExecuteFunc: func(params *Params, cmd *cobra.Command, args []string) error {
            return nil
        },
    }
    ```

=== "Builder API"

    ```go
    boa.NewCmdT[Params]("cmd").
        WithPreExecuteFuncE(func(params *Params, cmd *cobra.Command, args []string) error {
            return nil
        })
    ```

## HookContext

The `HookContext` provides access to parameter mirrors for advanced configuration:

- `GetParam(fieldPtr any) Param` - Get the Param interface for any field
- `HasValue(fieldPtr any) bool` - Check if a parameter has a value
- `AllMirrors() []Param` - Get all auto-generated parameter mirrors

### Example: Programmatic Configuration

```go
type ServerConfig struct {
    Host     string
    Port     int
    LogLevel string
}

func (c *ServerConfig) InitCtx(ctx *boa.HookContext) error {
    hostParam := ctx.GetParam(&c.Host)
    hostParam.SetDefault(boa.Default("localhost"))
    hostParam.SetEnv("SERVER_HOST")

    portParam := ctx.GetParam(&c.Port)
    portParam.SetDefault(boa.Default(8080))
    portParam.SetEnv("SERVER_PORT")

    logParam := ctx.GetParam(&c.LogLevel)
    logParam.SetDefault(boa.Default("info"))
    logParam.SetAlternatives([]string{"debug", "info", "warn", "error"})
    logParam.SetStrictAlts(true)

    return nil
}
```

### Example: Checking Parameter Sources at Runtime

=== "Direct API"

    ```go
    type Params struct {
        Host string `default:"localhost"`
        Port int    `optional:"true"`
    }

    func main() {
        boa.CmdT[Params]{
            Use: "server",
            RunFuncCtx: func(ctx *boa.HookContext, params *Params, cmd *cobra.Command, args []string) {
                if ctx.HasValue(&params.Port) {
                    fmt.Printf("Starting on %s:%d\n", params.Host, params.Port)
                } else {
                    fmt.Printf("Starting on %s (no port)\n", params.Host)
                }
            },
        }.Run()
    }
    ```

=== "Builder API"

    ```go
    type Params struct {
        Host string `default:"localhost"`
        Port int    `optional:"true"`
    }

    func main() {
        boa.NewCmdT[Params]("server").
            WithRunFuncCtx(func(ctx *boa.HookContext, params *Params) {
                if ctx.HasValue(&params.Port) {
                    fmt.Printf("Starting on %s:%d\n", params.Host, params.Port)
                } else {
                    fmt.Printf("Starting on %s (no port)\n", params.Host)
                }
            }).
            Run()
    }
    ```

!!! note
    You can only use one run function variant per command: `RunFunc`, `RunFuncCtx`, `RunFuncE`, or `RunFuncCtxE`.

## Error-Returning Run Functions

BOA provides error-returning variants of run functions for better error handling:

=== "Direct API"

    ```go
    boa.CmdT[Params]{
        Use: "cmd",
        RunFuncE: func(params *Params, cmd *cobra.Command, args []string) error {
            if err := doSomething(); err != nil {
                return fmt.Errorf("failed: %w", err)
            }
            return nil
        },
    }

    // With HookContext
    boa.CmdT[Params]{
        Use: "cmd",
        RunFuncCtxE: func(ctx *boa.HookContext, params *Params, cmd *cobra.Command, args []string) error {
            return nil
        },
    }
    ```

=== "Builder API"

    ```go
    boa.NewCmdT[Params]("cmd").
        WithRunFuncE(func(params *Params) error {
            return nil
        })

    // With full signature
    boa.NewCmdT[Params]("cmd").
        WithRunFuncE3(func(params *Params, cmd *cobra.Command, args []string) error {
            return nil
        })

    // With HookContext
    boa.NewCmdT[Params]("cmd").
        WithRunFuncCtxE(func(ctx *boa.HookContext, params *Params) error {
            return nil
        })
    ```

### Execution Methods

| Method | Description |
|--------|-------------|
| `Run()` | Executes command, panics on any error |
| `RunE()` | Executes command, returns error |
| `RunArgs(args)` | Executes with args, panics on any error |
| `RunArgsE(args)` | Executes with args, returns error |
| `ToCobra()` | Returns `*cobra.Command` with `cmd.Run` set (panics on setup error) |
| `ToCobraE()` | Returns `(*cobra.Command, error)` with `cmd.RunE` set |

## Error Handling

### Run() vs RunE() Behavior

The two execution methods handle errors differently:

| Method | Hook Errors | RunFuncE Errors | Config Errors |
|--------|-------------|-----------------|---------------|
| `Run()` | Panic | Panic | Panic |
| `RunE()` | Return error | Return error | Return error |

### Using Run()

With `Run()`, all errors cause panics. This is suitable for simple CLIs where errors should terminate the program:

```go
// All errors panic with Run()
boa.NewCmdT[Params]("cmd").
    WithInitFuncE(func(p *Params) error {
        return fmt.Errorf("init failed") // Panics
    }).
    Run()
```

### Using RunE()

With `RunE()`, all errors are returned for programmatic handling. This includes:

- Hook errors (`InitFunc`, `PreValidate`, `PreExecute`)
- Runtime errors from `RunFuncE`
- Configuration errors (e.g., setting multiple run functions)

```go
// All errors are returned with RunE()
err := boa.NewCmdT[Params]("cmd").
    WithInitFuncE(func(p *Params) error {
        return fmt.Errorf("init failed")
    }).
    WithRunFuncE(func(p *Params) error {
        return fmt.Errorf("run failed")
    }).
    RunE()

if err != nil {
    // Handle any error programmatically
    fmt.Printf("Command failed: %v\n", err)
}
```

This makes `RunE()` ideal for:

- Testing command logic
- Embedding commands in larger applications
- Custom error handling and logging
