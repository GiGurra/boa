# Validation & Constraints

BOA provides several ways to validate and constrain parameter values.

## Required vs Optional

By default, all parameters are **required**. Mark parameters as optional explicitly:

```go
type Params struct {
    Name    string                        // required
    Port    int    `optional:"true"`      // optional
    Verbose bool   `optional:"true"`      // optional
}
```

## Allowed Values (Alternatives)

Use `alts` to restrict a parameter to specific values:

```go
type Params struct {
    LogLevel string `alts:"debug,info,warn,error"`
    Format   string `alts:"json,yaml,toml"`
}
```

This provides:

1. **Shell completion** - Tab completion suggests valid values
2. **Validation** - Invalid values are rejected (when `strict` is true, which is the default)

### Strict Mode

By default, `alts` enforces validation. To allow any value while still providing completion hints:

```go
type Params struct {
    // Validation enforced (default)
    LogLevel string `alts:"debug,info,warn,error" strict:"true"`

    // Suggestions only, any value accepted
    Color string `alts:"red,green,blue" strict:"false"`
}
```

## Conditional Requirements

Make parameters conditionally required based on other values using `HookContext`:

```go
type Params struct {
    Mode     string
    FilePath string `optional:"true"`
    URL      string `optional:"true"`
}

func main() {
    boa.CmdT[Params]{
        Use: "app",
        InitFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
            // FilePath required when Mode is "file"
            ctx.GetParam(&p.FilePath).SetRequiredFn(func() bool {
                return p.Mode == "file"
            })

            // URL required when Mode is "http"
            ctx.GetParam(&p.URL).SetRequiredFn(func() bool {
                return p.Mode == "http"
            })

            return nil
        },
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            // ...
        },
    }.Run()
}
```

## Conditional Visibility

Hide parameters entirely based on conditions:

```go
type Params struct {
    Debug   bool   `optional:"true"`
    Verbose bool   `optional:"true"`
}

func main() {
    boa.CmdT[Params]{
        Use: "app",
        InitFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
            // Verbose flag only visible when Debug is true
            ctx.GetParam(&p.Verbose).SetIsEnabledFn(func() bool {
                return p.Debug
            })
            return nil
        },
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            // ...
        },
    }.Run()
}
```

## Dynamic Alternatives

For alternatives that depend on runtime state, use `SetAlternatives` in a hook:

```go
func (c *Config) InitCtx(ctx *boa.HookContext) error {
    // Set alternatives based on available options
    envs := []string{"dev", "staging", "prod"}
    ctx.GetParam(&c.Environment).SetAlternatives(envs)
    ctx.GetParam(&c.Environment).SetStrictAlts(true)
    return nil
}
```

## Value Priority

When multiple sources provide values, BOA uses this priority:

1. **CLI flags** - `--port 8080`
2. **Environment variables** - `PORT=8080`
3. **Root config file** - (via `configfile` tag at root or PreValidate hook)
4. **Substruct config files** - (via `configfile` tag in nested structs)
5. **Default values** - `default:"8080"`
6. **Zero value** - `0` for int, `""` for string, etc.

Higher priority sources override lower ones.
