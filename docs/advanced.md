# Advanced Features

This page covers advanced BOA features for power users.

## The Param Interface

Every parameter (whether using struct tags or programmatic configuration) implements the `Param` interface. Access it via `HookContext.GetParam()`:

=== "Direct API"

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

=== "Builder API"

    ```go
    boa.NewCmdT[Params]("cmd").
        WithInitFuncCtx(func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
            param := ctx.GetParam(&p.SomeField)
            // Now use param methods...
            return nil
        })
    ```

### Param Methods

| Method | Description |
|--------|-------------|
| `SetName(string)` | Override flag name |
| `SetShort(string)` | Set short flag |
| `SetEnv(string)` | Set environment variable |
| `SetDefault(any)` | Set default value |
| `SetAlternatives([]string)` | Set allowed values |
| `SetStrictAlts(bool)` | Enable/disable strict validation |
| `SetRequiredFn(func() bool)` | Dynamic required condition |
| `SetIsEnabledFn(func() bool)` | Dynamic visibility |
| `GetName() string` | Get current flag name |
| `GetShort() string` | Get current short flag |
| `GetEnv() string` | Get current env var |
| `GetAlternatives() []string` | Get allowed values |
| `HasValue() bool` | Check if value was set |
| `IsRequired() bool` | Check if required |
| `IsEnabled() bool` | Check if visible |

## Dynamic Shell Completion

### AlternativesFunc

For completion suggestions that depend on runtime state (like fetching from an API):

=== "Direct API"

    ```go
    type Params struct {
        // Using the deprecated wrapper type for AlternativesFunc
        Region boa.Required[string]
    }

    func main() {
        var p Params
        p.Region.AlternativesFunc = func(cmd *cobra.Command, args []string, toComplete string) []string {
            // Could fetch from API, read from file, etc.
            return []string{"us-east-1", "us-west-2", "eu-west-1"}
        }

        boa.CmdT[Params]{
            Use:    "app",
            Params: &p,
            RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
                // ...
            },
        }.Run()
    }
    ```

=== "Builder API"

    ```go
    type Params struct {
        // Using the deprecated wrapper type for AlternativesFunc
        Region boa.Required[string]
    }

    func main() {
        var p Params
        p.Region.AlternativesFunc = func(cmd *cobra.Command, args []string, toComplete string) []string {
            // Could fetch from API, read from file, etc.
            return []string{"us-east-1", "us-west-2", "eu-west-1"}
        }

        boa.NewCmdT2("app", &p).
            WithRunFunc(func(params *Params) {
                // ...
            }).
            Run()
    }
    ```

### ValidArgsFunc

For positional argument completion:

=== "Direct API"

    ```go
    boa.CmdT[Params]{
        Use: "app",
        ValidArgsFunc: func(p *Params, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
            // Return suggestions for positional args
            return []string{"option1", "option2"}, cobra.ShellCompDirectiveDefault
        },
    }
    ```

=== "Builder API"

    ```go
    boa.NewCmdT[Params]("app").
        WithValidArgsFunc(func(p *Params, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
            // Return suggestions for positional args
            return []string{"option1", "option2"}, cobra.ShellCompDirectiveDefault
        })
    ```

## Config File Loading

Load configuration from files in the PreValidate hook:

=== "Direct API"

    ```go
    type AppConfig struct {
        Host string
        Port int
    }

    type Params struct {
        ConfigFile string `descr:"Path to config file" optional:"true"`
        AppConfig
    }

    func main() {
        boa.CmdT[Params]{
            Use: "app",
            PreValidateFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command, args []string) error {
                fileParam := ctx.GetParam(&p.ConfigFile)
                return boa.UnMarshalFromFileParam(fileParam, &p.AppConfig, nil)
            },
            RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
                fmt.Printf("Host: %s, Port: %d\n", p.Host, p.Port)
            },
        }.Run()
    }
    ```

=== "Builder API"

    ```go
    type AppConfig struct {
        Host string
        Port int
    }

    type Params struct {
        ConfigFile string `descr:"Path to config file" optional:"true"`
        AppConfig
    }

    func main() {
        boa.NewCmdT[Params]("app").
            WithPreValidateFuncCtx(func(ctx *boa.HookContext, p *Params, cmd *cobra.Command, args []string) error {
                fileParam := ctx.GetParam(&p.ConfigFile)
                return boa.UnMarshalFromFileParam(fileParam, &p.AppConfig, nil)
            }).
            WithRunFunc(func(p *Params) {
                fmt.Printf("Host: %s, Port: %d\n", p.Host, p.Port)
            }).
            Run()
    }
    ```

Custom unmarshal functions (YAML, TOML, etc.):

```go
import "gopkg.in/yaml.v3"

boa.UnMarshalFromFileParam(fileParam, &p.AppConfig, yaml.Unmarshal)
```

## Checking Value Sources

Use `HookContext` in your run function to check how values were set:

=== "Direct API"

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

=== "Builder API"

    ```go
    boa.NewCmdT[Params]("app").
        WithRunFuncCtx(func(ctx *boa.HookContext, p *Params) {
            if ctx.HasValue(&p.Port) {
                fmt.Printf("Port explicitly set to %d\n", p.Port)
            } else {
                fmt.Println("Using default port")
            }
        })
    ```

## Accessing Cobra Directly

Access the underlying Cobra command for features BOA doesn't wrap:

=== "Direct API"

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

=== "Builder API"

    ```go
    boa.NewCmdT[Params]("app").
        WithInitFunc2E(func(p *Params, cmd *cobra.Command) error {
            cmd.Deprecated = "use 'new-app' instead"
            cmd.Hidden = true
            cmd.SilenceUsage = true
            return nil
        })
    ```

Or after flags are created:

=== "Direct API"

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

=== "Builder API"

    ```go
    boa.NewCmdT[Params]("app").
        WithPostCreateFuncCtx(func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
            flag := cmd.Flags().Lookup("verbose")
            flag.NoOptDefVal = "true"  // --verbose without value means true
            return nil
        })
    ```

## Command Groups

Organize subcommands into groups in help output:

=== "Direct API"

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

=== "Builder API"

    ```go
    boa.NewCmdT[boa.NoParams]("app").
        WithGroups(
            &cobra.Group{ID: "core", Title: "Core Commands:"},
            &cobra.Group{ID: "util", Title: "Utility Commands:"},
        ).
        WithSubCmds(
            boa.NewCmdT[Params]("init").WithGroupID("core"),
            boa.NewCmdT[Params]("run").WithGroupID("core"),
            boa.NewCmdT[Params]("version").WithGroupID("util"),
        )
    ```

## Testing Commands

### Inject Arguments

=== "Direct API"

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

=== "Builder API"

    ```go
    cmd := boa.NewCmdT[Params]("app").
        WithRunFunc(func(p *Params) {
            // ...
        })

    // Test with specific args
    cmd.RunArgs([]string{"--name", "test", "--port", "8080"})
    ```

### Validate Without Running

=== "Direct API"

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

=== "Builder API"

    ```go
    cmd := boa.NewCmdT[Params]("app").
        WithRawArgs([]string{"--name", "test"})

    err := cmd.Validate()
    if err != nil {
        // Validation failed
    }
    ```

## Interface-Based Hooks

Implement interfaces on your config struct instead of using `With*` methods:

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
