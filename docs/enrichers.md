# Auto-Derivation & Enrichers

BOA automatically derives flag names, environment variables, and other metadata from your struct fields. This is done through **enrichers** - functions that process parameters during initialization.

## ParamEnrich Field

The `ParamEnrich` field controls which enricher is used:

| Value | Behavior |
|-------|----------|
| `nil` | Uses `ParamEnricherDefault` (derives names, short flags, and bool defaults) |
| `ParamEnricherDefault` | Explicit default: derives names, short flags, and bool defaults |
| `ParamEnricherNone` | No enrichment - you must specify everything via struct tags |

## Default Behavior

By default (when `ParamEnrich` is `nil`), BOA applies `ParamEnricherDefault`, which includes:

| Enricher | What it does |
|----------|--------------|
| `ParamEnricherName` | Converts `MyParam` -> `--my-param` (kebab-case) |
| `ParamEnricherShort` | Auto-assigns `-m` from first char (skips conflicts, reserves `-h`) |
| `ParamEnricherBool` | Sets `default: false` for boolean params |

Note: Environment variable binding is **not** included by default. Add `ParamEnricherEnv` explicitly if you want auto-generated env vars.

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
--server-host  (required)
--max-retries  (required)
--verbose      (default: false)
```

## Custom Enrichers

You can compose your own enricher to change the default behavior.

### Enable Auto Env Vars

```go
boa.CmdT[Params]{
    Use: "cmd",
    ParamEnrich: boa.ParamEnricherCombine(
        boa.ParamEnricherName,
        boa.ParamEnricherShort,
        boa.ParamEnricherEnv,
        boa.ParamEnricherBool,
    ),
}
```

### Prefix Env Vars

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

This turns `MY_PARAM` into `MYAPP_MY_PARAM`.

### Disable Auto Short Flags

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

### Disable All Enrichment

```go
boa.CmdT[Params]{
    Use:         "cmd",
    ParamEnrich: boa.ParamEnricherNone,
}
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
| `ParamEnricherDefault` | Combines Name + Short + Bool |
| `ParamEnricherName` | Derives flag name from field name |
| `ParamEnricherShort` | Auto-assigns short flags |
| `ParamEnricherEnv` | Derives env var from flag name |
| `ParamEnricherEnvPrefix(prefix)` | Adds prefix to env vars |
| `ParamEnricherBool` | Sets false default for booleans |
| `ParamEnricherCombine(...)` | Combines multiple enrichers |

## Interaction with Named Struct Prefixing

When a field lives inside a named (non-anonymous) struct field, enrichers operate on the already-prefixed name. For example:

```go
type DBConfig struct {
    Host string `default:"localhost"`
    Port int    `default:"5432"`
}

type Params struct {
    DB DBConfig  // named field
}
```

With `ParamEnricherDefault`, `DB.Host` gets:

1. **Prefix applied**: field name becomes `DBHost` internally
2. **`ParamEnricherName`**: converts `DBHost` → `--db-host`
3. **`ParamEnricherShort`**: assigns `-d` (if available)

With `ParamEnricherEnv` added, the flag name `db-host` is converted to env var `DB_HOST`.

Explicit `env:"..."` tags inside named struct fields are also prefixed: `env:"SERVER_HOST"` inside field `API` becomes `API_SERVER_HOST`.

## Override Auto-Derived Values

Struct tags always take precedence over enrichers:

```go
type Params struct {
    // Enricher would derive --my-host and MY_HOST
    // Tags override to --server and APP_SERVER
    MyHost string `name:"server" env:"APP_SERVER"`
}
```

Note: Inside named struct fields, explicit `name` and `env` tags are also prefixed. `name:"host"` inside field `DB` becomes `--db-host`.
