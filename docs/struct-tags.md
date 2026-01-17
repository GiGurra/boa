# Struct Tags Reference

BOA uses struct tags to configure CLI parameters.

## Available Tags

| Tag | Description | Example |
|-----|-------------|---------|
| `descr` / `desc` / `description` / `help` | Description text for help | `descr:"User name"` |
| `name` / `long` | Override flag name | `name:"user-name"` |
| `default` | Default value | `default:"8080"` |
| `env` | Environment variable name | `env:"PORT"` |
| `short` | Short flag (single char) | `short:"p"` |
| `positional` / `pos` | Marks positional argument | `positional:"true"` |
| `required` / `req` | Marks as required | `required:"true"` |
| `optional` / `opt` | Marks as optional | `optional:"true"` |
| `alts` / `alternatives` | Allowed values (enum) | `alts:"debug,info,warn,error"` |
| `strict-alts` / `strict` | Validate against alts | `strict:"true"` |

## Auto-Generated Names

BOA automatically generates flag and environment variable names from field names:

- **Flag name**: `MyParam` becomes `--my-param` (kebab-case)
- **Env var**: `MyParam` becomes `MY_PARAM` (UPPER_SNAKE_CASE)

## Enrichers

BOA automatically enriches parameters using `ParamEnricherDefault`, which includes:

| Enricher | Behavior |
|----------|----------|
| `ParamEnricherName` | Converts field name to kebab-case flag |
| `ParamEnricherShort` | Auto-assigns short flag from first character |
| `ParamEnricherEnv` | Generates env var from flag name |
| `ParamEnricherBool` | Sets default `false` for boolean params |

### Custom Enrichers

Compose your own enricher if you don't want auto-generated env vars:

```go
// Without auto env vars
boa.NewCmdT[Params]("cmd").WithParamEnrich(
    boa.ParamEnricherCombine(
        boa.ParamEnricherName,
        boa.ParamEnricherShort,
        boa.ParamEnricherBool,
    ),
)

// With prefixed env vars
boa.NewCmdT[Params]("cmd").WithParamEnrich(
    boa.ParamEnricherCombine(
        boa.ParamEnricherName,
        boa.ParamEnricherEnv,
        boa.ParamEnricherEnvPrefix("MYAPP"), // MY_PARAM → MYAPP_MY_PARAM
    ),
)
```

## Constraining Values

Use `alts` to specify allowed values:

```go
type Params struct {
    LogLevel string `alts:"debug,info,warn,error" strict:"true"`
    Format   string `alts:"json,yaml,toml"` // strict defaults to true
}
```

## Conditional Parameters

Make parameters conditionally required or enabled using `HookContext`:

```go
type Params struct {
    Mode     string
    FilePath string `optional:"true"`
    Verbose  bool   `optional:"true"`
    Debug    bool   `optional:"true"`
}

func main() {
    boa.NewCmdT[Params]("hello-world").
        WithInitFuncCtx(func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
            // FilePath is required when Mode is "file"
            ctx.GetParam(&p.FilePath).SetRequiredFn(func() bool {
                return p.Mode == "file"
            })

            // Verbose is only enabled when Debug is true
            ctx.GetParam(&p.Verbose).SetIsEnabledFn(func() bool {
                return p.Debug
            })

            return nil
        }).
        WithRunFunc(func(params *Params) {
            fmt.Printf("Mode=%s\n", params.Mode)
        }).
        Run()
}
```
