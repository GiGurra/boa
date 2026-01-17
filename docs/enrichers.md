# Auto-Derivation & Enrichers

BOA automatically derives flag names, environment variables, and other metadata from your struct fields. This is done through **enrichers** - functions that process parameters during initialization.

## ParamEnrich Field

The `ParamEnrich` field controls which enricher is used:

| Value | Behavior |
|-------|----------|
| `nil` | Uses `ParamEnricherDefault` (enriches everything including env vars) |
| `ParamEnricherDefault` | Explicit default: derives names, short flags, env vars, and bool defaults |
| `ParamEnricherNone` | No enrichment - you must specify everything via struct tags |

## Default Behavior

By default (when `ParamEnrich` is `nil`), BOA applies `ParamEnricherDefault`, which includes:

| Enricher | What it does |
|----------|--------------|
| `ParamEnricherName` | Converts `MyParam` → `--my-param` (kebab-case) |
| `ParamEnricherShort` | Auto-assigns `-m` from first char (skips conflicts, reserves `-h`) |
| `ParamEnricherEnv` | Generates `MY_PARAM` from flag name (UPPER_SNAKE_CASE) |
| `ParamEnricherBool` | Sets `default: false` for boolean params |

Example - this struct:

```go
type Params struct {
    ServerHost string
    MaxRetries int
    Verbose    bool
}
```

Automatically gets:

```
--server-host  (env: SERVER_HOST, required)
--max-retries  (env: MAX_RETRIES, required)
--verbose      (env: VERBOSE, default: false)
```

## Custom Enrichers

You can compose your own enricher to change the default behavior.

### Disable Auto Env Vars

=== "Direct API"

    ```go
    boa.CmdT[Params]{
        Use: "cmd",
        ParamEnrich: boa.ParamEnricherCombine(
            boa.ParamEnricherName,
            boa.ParamEnricherShort,
            boa.ParamEnricherBool,
        ),
    }
    ```

=== "Builder API"

    ```go
    boa.NewCmdT[Params]("cmd").WithParamEnrich(
        boa.ParamEnricherCombine(
            boa.ParamEnricherName,
            boa.ParamEnricherShort,
            boa.ParamEnricherBool,
        ),
    )
    ```

### Prefix Env Vars

=== "Direct API"

    ```go
    boa.CmdT[Params]{
        Use: "cmd",
        ParamEnrich: boa.ParamEnricherCombine(
            boa.ParamEnricherName,
            boa.ParamEnricherEnv,
            boa.ParamEnricherEnvPrefix("MYAPP"),
        ),
    }
    ```

=== "Builder API"

    ```go
    boa.NewCmdT[Params]("cmd").WithParamEnrich(
        boa.ParamEnricherCombine(
            boa.ParamEnricherName,
            boa.ParamEnricherEnv,
            boa.ParamEnricherEnvPrefix("MYAPP"),
        ),
    )
    ```

This turns `MY_PARAM` into `MYAPP_MY_PARAM`.

### Disable Auto Short Flags

=== "Direct API"

    ```go
    boa.CmdT[Params]{
        Use: "cmd",
        ParamEnrich: boa.ParamEnricherCombine(
            boa.ParamEnricherName,
            boa.ParamEnricherEnv,
            boa.ParamEnricherBool,
        ),
    }
    ```

=== "Builder API"

    ```go
    boa.NewCmdT[Params]("cmd").WithParamEnrich(
        boa.ParamEnricherCombine(
            boa.ParamEnricherName,
            boa.ParamEnricherEnv,
            boa.ParamEnricherBool,
        ),
    )
    ```

### Disable All Enrichment

=== "Direct API"

    ```go
    boa.CmdT[Params]{
        Use:         "cmd",
        ParamEnrich: boa.ParamEnricherNone,
    }
    ```

=== "Builder API"

    ```go
    boa.NewCmdT[Params]("cmd").WithParamEnrich(boa.ParamEnricherNone)
    ```

With no enrichment, you must specify everything via struct tags:

```go
type Params struct {
    Host string `name:"host" short:"h" env:"HOST" descr:"Server host"`
}
```

## Available Enrichers

| Enricher | Description |
|----------|-------------|
| `ParamEnricherDefault` | Combines Name + Short + Env + Bool |
| `ParamEnricherName` | Derives flag name from field name |
| `ParamEnricherShort` | Auto-assigns short flags |
| `ParamEnricherEnv` | Derives env var from flag name |
| `ParamEnricherEnvPrefix(prefix)` | Adds prefix to env vars |
| `ParamEnricherBool` | Sets false default for booleans |
| `ParamEnricherCombine(...)` | Combines multiple enrichers |

## Override Auto-Derived Values

Struct tags always take precedence over enrichers:

```go
type Params struct {
    // Enricher would derive --my-host and MY_HOST
    // Tags override to --server and APP_SERVER
    MyHost string `name:"server" env:"APP_SERVER"`
}
```
