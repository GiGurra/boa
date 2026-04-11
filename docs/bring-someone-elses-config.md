# Bring Someone Else's Config

Sometimes the struct you want to expose as a CLI isn't yours. It comes from a third-party library, a generated protobuf type, a shared internal package you don't want to fork — anything where adding `boa:` struct tags isn't an option. BOA supports this: **anything configurable via a struct tag is also configurable programmatically** via `HookContext.GetParam` / `GetParamT`, so you can take a tag-less struct and wire it up from an `InitFuncCtx` hook.

This page covers the two common shapes:

1. **Pure embed** — the external struct *is* your whole CLI config.
2. **Composition** — the external struct is one field inside your own params.

## Pure embed: the external struct is your CLI

If the third-party config already has the shape you want to expose, pass it straight to `CmdT[T]` and configure every field in `InitFuncCtx`:

```go
package main

import (
    "fmt"

    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
    "github.com/third/party/httpserver"
)

// httpserver.Config is defined in a package we don't control — no boa tags.
//
// type Config struct {
//     Host       string
//     Port       int
//     AdminToken string
//     Verbose    bool
// }

func main() {
    boa.CmdT[httpserver.Config]{
        Use:   "serve",
        Short: "Run the HTTP server",

        // Auto-derive --flag-name from field name and $HOST / $PORT / ... from flag name.
        ParamEnrich: boa.ParamEnricherCombine(
            boa.ParamEnricherName,
            boa.ParamEnricherEnv,
            boa.ParamEnricherBool,
        ),

        InitFuncCtx: func(ctx *boa.HookContext, p *httpserver.Config, cmd *cobra.Command) error {
            // Descriptions / defaults / required-ness
            boa.GetParamT(ctx, &p.Host).SetDescription("listen address")
            boa.GetParamT(ctx, &p.Host).SetDefaultT("0.0.0.0")

            port := boa.GetParamT(ctx, &p.Port)
            port.SetDescription("TCP port")
            port.SetDefaultT(8080)
            port.SetMinT(1)
            port.SetMaxT(65535)

            // Hide the admin token from --help, still read from env/config
            token := boa.GetParamT(ctx, &p.AdminToken)
            token.SetDescription("admin API token")
            token.SetNoFlag(true)
            token.SetEnv("ADMIN_TOKEN")
            token.SetRequired(true)

            return nil
        },

        RunFunc: func(p *httpserver.Config, cmd *cobra.Command, args []string) {
            fmt.Printf("listening on %s:%d\n", p.Host, p.Port)
            // httpserver.Run(p) ...
        },
    }.Run()
}
```

The CLI, without ever touching the third-party package:

```
$ serve --help
Usage:
  serve [flags]

Flags:
      --host string   listen address (env: HOST) (default "0.0.0.0")
      --port int      TCP port (env: PORT) (default 8080)
      --verbose       (env: VERBOSE) (default false)

$ ADMIN_TOKEN=secret serve
```

`--admin-token` is gone (noflag), `$ADMIN_TOKEN` is honored, and the `min`/`max` check on `Port` runs even though the third-party struct has no validation tags.

## Composition: external struct as one field inside your own config

More common: you own the overall params struct and want to pull in a third-party config as a sub-section. Named struct fields auto-prefix their children's flag and env names, so the third-party struct's fields land under a clean namespace without collisions.

The example below walks through the **full spectrum** of input-source policies you might want on a realistic config — CLI-only, env-only, config-file-only, everything-at-once, dynamic shell completion, per-field validation, and whole-config validation:

```go
package main

import (
    "fmt"
    "path/filepath"
    "strings"

    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
    "github.com/third/party/dbconfig"
)

// dbconfig.Settings is external — no boa tags.
//
// type Settings struct {
//     Host      string
//     Port      int
//     User      string
//     Password  string
//     SSLMode   string
//     CACert    string  // path to a CA bundle
//     AuditTag  string  // free-form label written into audit rows
//     PoolSize  int
//     DebugMode bool
// }

type Params struct {
    // A plain boa tag — log level is just a CLI/env concern, no external struct.
    ConfigFile string `configfile:"true" default:"" optional:"true" descr:"path to config file"`
    LogLevel   string `descr:"log level" alts:"debug,info,warn,error" default:"info"`

    // Named field → children become --db-host, --db-port, ... with matching
    // env names ($DB_HOST, $DB_PORT, ...) under ParamEnricherEnv.
    DB dbconfig.Settings
}

func main() {
    boa.CmdT[Params]{
        Use:   "app",
        Short: "Run the app with a mixed-source config",

        ParamEnrich: boa.ParamEnricherCombine(
            boa.ParamEnricherName,
            boa.ParamEnricherEnv,
            boa.ParamEnricherBool,
        ),

        InitFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
            // ─── CLI + env + config (the default) ──────────────────────────
            // Host is a classic: operators set it with --db-host, ops sets
            // $DB_HOST in systemd, CI sets it in config.yaml. All three work.
            host := boa.GetParamT(ctx, &p.DB.Host)
            host.SetDescription("database host")
            host.SetDefaultT("localhost")

            // ─── CLI + env + config, with per-field validation ─────────────
            port := boa.GetParamT(ctx, &p.DB.Port)
            port.SetDescription("database port")
            port.SetDefaultT(5432)
            port.SetMinT(1)
            port.SetMaxT(65535)

            // ─── CLI-only (env suppressed) ─────────────────────────────────
            // DebugMode is an interactive knob — we don't want a long-lived
            // $DB_DEBUG_MODE sneaking in from systemd. Flag only.
            dbg := boa.GetParamT(ctx, &p.DB.DebugMode)
            dbg.SetDescription("enable verbose DB driver logging")
            dbg.SetNoEnv(true)
            dbg.SetDefaultT(false)

            // ─── Env-only (no CLI flag) ────────────────────────────────────
            // Password must not land in shell history or process listings.
            // Env or config-file-with-mode-600, never argv.
            pwd := boa.GetParamT(ctx, &p.DB.Password)
            pwd.SetDescription("database password")
            pwd.SetNoFlag(true)
            pwd.SetEnv("DB_PASSWORD")
            pwd.SetRequired(true)

            // ─── Config-file-only (noflag + noenv, validation preserved) ───
            // AuditTag is a deployment-identity string set by the platform,
            // not something a human types. It's only meaningful when loaded
            // from the config file we ship with the deployment, but we still
            // want boa to enforce a length bound on it.
            tag := boa.GetParamT(ctx, &p.DB.AuditTag)
            tag.SetDescription("audit label written to every row")
            tag.SetNoFlag(true)
            tag.SetNoEnv(true)
            tag.SetMinLen(3)
            tag.SetMaxLen(64)

            // ─── Fully ignored by boa ──────────────────────────────────────
            // PoolSize comes from the driver's own config merging inside
            // dbconfig package — we don't want boa to touch it at all.
            // Config files can still populate it via raw unmarshal.
            boa.GetParamT(ctx, &p.DB.PoolSize).SetIgnored(true)

            // ─── Enum with static alternatives ─────────────────────────────
            ssl := boa.GetParamT(ctx, &p.DB.SSLMode)
            ssl.SetDescription("TLS policy")
            ssl.SetAlternatives([]string{"disable", "require", "verify-ca", "verify-full"})
            ssl.SetStrictAlts(true)
            ssl.SetDefaultT("require")

            // ─── Dynamic shell completion for a path ───────────────────────
            // CA bundle path — dynamically suggest PEM/CRT files under the
            // current prefix at completion time. This runs inside the
            // user's shell when they hit <TAB>.
            ca := boa.GetParamT(ctx, &p.DB.CACert)
            ca.SetDescription("path to a CA certificate bundle")
            ca.SetAlternativesFunc(func(cmd *cobra.Command, args []string, toComplete string) []string {
                pems, _ := filepath.Glob(toComplete + "*.pem")
                crts, _ := filepath.Glob(toComplete + "*.crt")
                return append(pems, crts...)
            })

            // ─── Per-field custom validator ────────────────────────────────
            // User must be lowercase (many postgres deployments enforce this).
            user := boa.GetParamT(ctx, &p.DB.User)
            user.SetDescription("database username")
            user.SetCustomValidatorT(func(v string) error {
                if v != strings.ToLower(v) {
                    return fmt.Errorf("user must be lowercase, got %q", v)
                }
                return nil
            })

            return nil
        },

        // ─── Whole-config validation ───────────────────────────────────────
        // PreExecuteFunc runs after all individual params are resolved and
        // validated — it's the place for invariants that span multiple
        // fields. Returning an error here fails the command with a clean
        // user-input-style message.
        PreExecuteFunc: func(p *Params, cmd *cobra.Command, args []string) error {
            // verify-full TLS mode requires a CA bundle
            if p.DB.SSLMode == "verify-full" && p.DB.CACert == "" {
                return fmt.Errorf("--db-ssl-mode=verify-full requires --db-ca-cert")
            }
            // Debug mode is noisy — refuse it against a production host
            if p.DB.DebugMode && strings.HasSuffix(p.DB.Host, ".prod.internal") {
                return fmt.Errorf("refusing to enable DB debug mode against a .prod.internal host")
            }
            return nil
        },

        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("connecting to %s:%d as %s (ssl=%s)\n",
                p.DB.Host, p.DB.Port, p.DB.User, p.DB.SSLMode)
        },
    }.Run()
}
```

### Source-by-source summary of the example

| Field | CLI flag | Env var | Config file | Notes |
|---|---|---|---|---|
| `LogLevel` | `--log-level` | `$LOG_LEVEL` | yes | plain boa tag, own struct |
| `DB.Host` | `--db-host` | `$DB_HOST` | yes | default + description via `InitFuncCtx` |
| `DB.Port` | `--db-port` | `$DB_PORT` | yes | programmatic `SetMinT` / `SetMaxT` |
| `DB.User` | `--db-user` | `$DB_USER` | yes | custom validator (lowercase) |
| `DB.Password` | — | `$DB_PASSWORD` | yes | `SetNoFlag(true)`, required |
| `DB.SSLMode` | `--db-ssl-mode` | `$DB_SSL_MODE` | yes | enum via `SetAlternatives` + `SetStrictAlts` |
| `DB.CACert` | `--db-ca-cert` | `$DB_CA_CERT` | yes | dynamic shell completion |
| `DB.AuditTag` | — | — | yes | `SetNoFlag` + `SetNoEnv`, length bounds |
| `DB.PoolSize` | — | — | yes (raw) | `SetIgnored(true)`, boa doesn't touch it |
| `DB.DebugMode` | `--db-debug-mode` | — | yes | `SetNoEnv(true)` |

### Why whole-config validation lives in `PreExecuteFunc`

`PreExecuteFunc` runs **after** per-field validation (required checks, `min`/`max`/`pattern`, custom validators, alts enforcement) but **before** `RunFunc`. That's the right spot for cross-field invariants like "mode X requires flag Y" or "these two durations can't both be zero" — by the time it runs, you know every field has already passed its own checks, so you can write assertions in terms of validated values without defensive nil/zero handling.

Returning an error from `PreExecuteFunc` aborts the command and prints the error like any other validation failure; you don't need to panic or call `os.Exit`.

### Optional sub-configurations via pointer fields

Making the external struct a **pointer field** turns it into an optional parameter group: if the user doesn't set any of its fields (via CLI, env, or config), the pointer stays `nil` after parsing. This is useful when a feature is optional:

```go
type Params struct {
    LogLevel string             `descr:"log level" default:"info"`
    DB       *dbconfig.Settings // optional: nil if no --db-* flag or $DB_* env was set
}

// ... inside InitFuncCtx, configure &p.DB.Host etc. the same way.
// After Run, `if p.DB != nil { useDatabase(p.DB) }`.
```

See [Config File Examples → Why Key-Presence Detection Matters](examples-config.md#why-key-presence-detection-matters) for the full semantics.

## What you can configure programmatically

Every struct-tag feature has a matching method. The table below is the complete tag → programmatic mapping:

| Tag | Programmatic equivalent |
|-----|--------------------------|
| `descr` / `desc` / `help` | `SetDescription(string)` |
| `name` / `long` | `SetName(string)` |
| `short` | `SetShort(string)` |
| `env` | `SetEnv(string)` |
| `default` | `SetDefault(any)` / `ParamT[T].SetDefaultT(T)` |
| `positional` / `pos` | `SetPositional(bool)` |
| `required` / `req` | `SetRequired(bool)` or `SetRequiredFn(func() bool)` |
| `optional` / `opt` | `SetRequired(false)` |
| `alts` / `alternatives` | `SetAlternatives([]string)`, `SetAlternativesFunc(...)` |
| `strict` / `strict-alts` | `SetStrictAlts(bool)` |
| `min` | `ParamT[T].SetMinT(T)` for numeric; `SetMinLen(int)` for string/slice/map. `ClearMin()` removes. Non-generic `Param.SetMin(any)` accepts any numeric. |
| `max` | `ParamT[T].SetMaxT(T)` / `SetMaxLen(int)` / `ClearMax()`. Symmetric with `min`. |
| `pattern` | `SetPattern(string)` |
| `boa:"noflag"` / `"nocli"` | `SetNoFlag(bool)` |
| `boa:"noenv"` | `SetNoEnv(bool)` |
| `boa:"configonly"` | `SetNoFlag(true)` + `SetNoEnv(true)` |
| `boa:"ignore"` | `SetIgnored(bool)` (post-traversal equivalent) |
| `configfile:"true"` | `SetConfigFile(bool)` — field must be a string |

All of these must be called from a hook that runs **before cobra flag binding** — that is, `InitFunc`, `InitFuncCtx`, or the `CfgStructInit` / `CfgStructInitCtx` interfaces. Calling them later (in `PostCreate*` or `RunFunc`) is too late: the flags are already wired up.

## Wrapping the configuration in a helper

If you embed the same external type in multiple commands, extract the wiring into a helper so each command only writes it once:

```go
// wireDBConfig attaches descriptions, defaults, validation, and the
// hide-password-from-CLI policy to a dbconfig.Settings sub-field. Works
// with named DB fields and optional *DB pointer fields alike.
func wireDBConfig(ctx *boa.HookContext, db *dbconfig.Settings) {
    boa.GetParamT(ctx, &db.Host).SetDefaultT("localhost")
    boa.GetParamT(ctx, &db.Port).SetDefaultT(5432)

    port := boa.GetParamT(ctx, &db.Port)
    port.SetMinT(1)
    port.SetMaxT(65535)

    pwd := boa.GetParamT(ctx, &db.Password)
    pwd.SetNoFlag(true)
    pwd.SetRequired(true)
}

// ... then in each command:
InitFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
    wireDBConfig(ctx, &p.DB)
    return nil
},
```

## Auto-config-file loading for an external struct

You have **two ways** to point `configfile` at a field inside an external struct:

1. **Programmatic `SetConfigFile(true)`** — add a plain string field to the params struct (or to a wrapper) and mark it from `InitFuncCtx`.
2. **Anonymous embedding** — wrap the external struct in your own type and put a `configfile:"true"` field on the wrapper.

Both end up at the same place (BOA's config-file registry); pick whichever reads better for the call site. Below are three flavors of the embedding variant, from most-ceremonial to least.

### Variant A — named wrapper type

Anonymous embedding lets you bolt a tagged config-file field onto an external struct without modifying it. The wrapper becomes your params type; boa walks the embedded fields at the root level (no prefix), and JSON/YAML/TOML unmarshal flattens embedded fields the same way, so a single config file populates them directly:

```go
// externalpkg.Settings is third-party — no boa tags, no configfile field.
//
// type Settings struct {
//     Host string
//     Port int
// }

type Params struct {
    externalpkg.Settings        // anonymous embed — fields land at the root
    ConfigFile           string `configfile:"true" optional:"true" descr:"path to config file"`
}

func main() {
    boa.CmdT[Params]{
        Use: "app",
        ParamEnrich: boa.ParamEnricherCombine(
            boa.ParamEnricherName,
            boa.ParamEnricherEnv,
            boa.ParamEnricherBool,
        ),
        InitFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
            // Descriptions / defaults / validation for the embedded fields come
            // from the programmatic API since you can't tag them.
            boa.GetParamT(ctx, &p.Host).SetDefaultT("localhost")
            port := boa.GetParamT(ctx, &p.Port)
            port.SetDefaultT(5432)
            port.SetMinT(1)
            port.SetMaxT(65535)
            return nil
        },
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("%s:%d\n", p.Host, p.Port)
        },
    }.Run()
}
```

Now `app --config-file app.json` works, `--host` / `--port` / `$HOST` / `$PORT` work, and CLI still beats config-file — the full precedence chain is preserved because boa treats the embedded fields exactly like fields declared on the wrapper itself.

### Variant B — inline anonymous struct

If the wrapper is only used in a single `CmdT` call, you don't even need to name it. Declare it inline at the call site:

```go
boa.CmdT[struct {
    externalpkg.Settings
    ConfigFile string `configfile:"true" optional:"true"`
}]{
    Use: "app",
    RunFunc: func(p *struct {
        externalpkg.Settings
        ConfigFile string `configfile:"true" optional:"true"`
    }, cmd *cobra.Command, args []string) {
        fmt.Printf("%s:%d\n", p.Host, p.Port)
    },
}.Run()
```

Same semantics as the named version — just fewer declarations. The tradeoff is that you have to repeat the anonymous type in each hook's signature (Go has no type-alias shortcut here), which gets noisy past one or two hooks. Fine for throwaway commands; for anything with `InitFuncCtx`, name the type.

### Variant C — programmatic, no wrapper at all

If the external struct already has the exact shape you want to expose, skip wrapping and configure the tagless struct directly. Add a `ConfigFile` field to a surrounding struct (or use the wrapper pattern above), then flip the flag from `InitFuncCtx`:

```go
type Params struct {
    ConfigFile string `optional:"true" descr:"path to config file"`
    externalpkg.Settings
}

boa.CmdT[Params]{
    InitFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
        // Equivalent to `configfile:"true"` on Params.ConfigFile, but set
        // at runtime — useful when the surrounding struct comes from a
        // package you don't want to add boa-specific tags to.
        boa.GetParamT(ctx, &p.ConfigFile).SetConfigFile(true)
        return nil
    },
    // ...
}.Run()
```

`SetConfigFile(true)` is the programmatic equivalent of `configfile:"true"` — the field must still be a string, and the call must happen in `InitFunc` / `InitFuncCtx` (before boa builds the config-file registry). Calling it on a non-string field produces a clean user-input-style error, not a panic.

### Caveats

- **The embedded type must be exported** (`externalpkg.Settings`, not `externalpkg.settings`). Anonymous embedding of an unexported type produces an unexported field, which boa silently skips along with all of its children — CLI flags and env binding for those fields simply won't register. (Earlier versions panicked here; the current build skips cleanly.)
- **Name collisions** between the wrapper's own fields and the embedded type's fields are resolved by Go's shallower-wins rule at the type level, but BOA registers flags by field name — so if both the wrapper and the embedded struct declare a `Name` field, boa will error on duplicate flag registration. Rename one, or switch to a named (non-anonymous) field, which auto-prefixes the embedded children and eliminates the collision.

## Limitations

A smaller handful of things are **not** currently available programmatically:

- **`boa:"ignore"` at the tag level skips traversal entirely** so the mirror never exists; the programmatic equivalent `SetIgnored(true)` marks an existing mirror as ignored instead. The observable behavior is the same (no CLI, no env, no validation — only raw config-file unmarshal writes), but there's one subtle difference: with the tag, the field is not even walked, so deeply nested ignored sub-trees have zero cost at startup.
- **`InitFuncCtx` only sees fields from the live params tree.** If the third-party struct contains its own nested pointer fields that start as `nil`, BOA preallocates them before `InitFuncCtx` runs (so you can take `&p.DB.Inner.Field` freely), but if the third-party code itself reassigns one of those pointers later, the mirror index may go stale — call into boa early in the lifecycle and let it own the tree.

## See also

- [Advanced → The Param Interface](advanced.md#the-param-interface) — full list of `Param` methods
- [Advanced → Programmatic Configuration (Tag Parity)](advanced.md#programmatic-configuration-tag-parity) — the same mapping table with more detail
- [Lifecycle Hooks](hooks.md) — when each hook runs
- [Struct Tags](struct-tags.md) — the equivalent tag-based reference
