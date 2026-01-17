# Migration Guide

If you're migrating from the deprecated `Required[T]`/`Optional[T]` wrapper types, this guide will help.

## Before (Deprecated)

```go
type Params struct {
    Name boa.Required[string] `descr:"User name"`
    Port boa.Optional[int]    `descr:"Port number" default:"8080"`
}

// Accessing values
fmt.Println(params.Name.Value())       // string
fmt.Println(*params.Port.Value())      // int (via pointer)
```

## After (Recommended)

```go
type Params struct {
    Name string `descr:"User name"`                    // required by default
    Port int    `descr:"Port number" optional:"true"`
}

// Accessing values - direct access
fmt.Println(params.Name)  // string
fmt.Println(params.Port)  // int
```

## Programmatic Configuration

Configuration that was previously done directly on wrapper types now uses `HookContext`:

### Before

```go
params.Port.SetRequiredFn(func() bool { return params.Mode == "server" })
```

### After

=== "Direct API"

    ```go
    cmd := boa.CmdT[Params]{
        Use: "app",
        InitFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
            ctx.GetParam(&p.Port).SetRequiredFn(func() bool { return p.Mode == "server" })
            return nil
        },
    }
    ```

=== "Builder API"

    ```go
    cmd := boa.NewCmdT[Params]("app").
        WithInitFuncCtx(func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
            ctx.GetParam(&p.Port).SetRequiredFn(func() bool { return p.Mode == "server" })
            return nil
        })
    ```

## Deprecated API Reference

The following are deprecated but still functional for backward compatibility:

```go
// DEPRECATED wrapper types
type Params struct {
    Name boa.Required[string]   // Use: Name string
    Port boa.Optional[int]      // Use: Port int `optional:"true"`
}

// DEPRECATED factory functions
name := boa.Req("default")    // Use: struct tag `default:"default"`
port := boa.Opt(8080)         // Use: struct tag `default:"8080" optional:"true"`
def := boa.Default(value)     // Use: struct tag `default:"value"`
```

## Why Migrate?

The wrapper types require calling `.Value()` to access values, adding verbosity. Plain Go types provide:

- Direct field access without method calls
- Cleaner, more idiomatic Go code
- Better IDE autocomplete support
- Simpler serialization/deserialization
