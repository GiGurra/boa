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

For YAML, TOML, or other formats, set `ConfigUnmarshal`:

```go
import "gopkg.in/yaml.v3"

boa.CmdT[Params]{
    Use:             "app",
    ConfigUnmarshal: yaml.Unmarshal,
    RunFunc:         func(p *Params, cmd *cobra.Command, args []string) { ... },
}.Run()
```

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
