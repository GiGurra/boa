# Config File Examples

Practical examples for loading configuration from files with BOA.

## Basic Config File Loading

Tag a `string` field with `configfile:"true"` and BOA automatically loads the file before validation. CLI and env var values always take precedence over config file values.

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    ConfigFile string `configfile:"true" optional:"true" default:"config.json"`
    Host       string `descr:"Server host" default:"localhost"`
    Port       int    `descr:"Server port" default:"8080"`
    Debug      bool   `descr:"Debug mode" optional:"true"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "server",
        Short: "Start the server",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Host: %s\nPort: %d\nDebug: %v\n", p.Host, p.Port, p.Debug)
        },
    }.Run()
}
```

Create `config.json`:

```json
{
    "Host": "api.example.com",
    "Port": 3000,
    "Debug": true
}
```

```bash
# Uses config.json from default path:
$ go run .
Host: api.example.com
Port: 3000
Debug: true

# Point to a different config file:
$ go run . --config-file /etc/myapp/prod.json
Host: prod.example.com
Port: 443
Debug: false

# CLI flags override config file values:
$ go run . --port 9090
Host: api.example.com
Port: 9090
Debug: true

# No config file? No problem (it's optional):
$ go run . --config-file "" --host localhost
Host: localhost
Port: 8080
Debug: false
```

## Config File with CLI Overrides

The value priority is: **CLI flags > env vars > root config > substruct config > defaults > zero value**.

```go
package main

import (
    "fmt"
    "os"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    ConfigFile string `configfile:"true" optional:"true"`
    Host       string `descr:"Server host" env:"APP_HOST" default:"localhost"`
    Port       int    `descr:"Server port" env:"APP_PORT" default:"8080"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "server",
        Short: "Demonstrate value priority",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Host: %s (source priority: CLI > env > config > default)\n", p.Host)
            fmt.Printf("Port: %d\n", p.Port)
        },
    }.Run()
}
```

With `config.json`:

```json
{
    "Host": "from-config",
    "Port": 3000
}
```

```bash
# Config file only:
$ go run . --config-file config.json
Host: from-config
Port: 3000

# Env overrides config:
$ APP_HOST=from-env go run . --config-file config.json
Host: from-env
Port: 3000

# CLI overrides everything:
$ APP_HOST=from-env go run . --config-file config.json --host from-cli
Host: from-cli
Port: 3000
```

## Substruct Config Files

Nested structs can each have their own `configfile:"true"` field, loading from separate files. The root config overrides substruct configs when they overlap.

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type DBConfig struct {
    ConfigFile string `configfile:"true" optional:"true"`
    Host       string `descr:"Database host" default:"localhost"`
    Port       int    `descr:"Database port" default:"5432"`
    Name       string `descr:"Database name" default:"mydb"`
}

type CacheConfig struct {
    ConfigFile string `configfile:"true" optional:"true"`
    Host       string `descr:"Cache host" default:"localhost"`
    Port       int    `descr:"Cache port" default:"6379"`
    TTL        int    `descr:"Cache TTL seconds" default:"300"`
}

type Params struct {
    ConfigFile string      `configfile:"true" optional:"true"`
    AppName    string      `descr:"Application name" default:"myapp"`
    DB         DBConfig
    Cache      CacheConfig
}

func main() {
    boa.CmdT[Params]{
        Use:   "app",
        Short: "Multi-config demo",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("App: %s\n", p.AppName)
            fmt.Printf("DB:    %s:%d/%s\n", p.DB.Host, p.DB.Port, p.DB.Name)
            fmt.Printf("Cache: %s:%d (TTL=%ds)\n", p.Cache.Host, p.Cache.Port, p.Cache.TTL)
        },
    }.Run()
}
```

Create `db.json`:

```json
{
    "Host": "db.internal",
    "Port": 5432,
    "Name": "production"
}
```

Create `cache.json`:

```json
{
    "Host": "redis.internal",
    "Port": 6379,
    "TTL": 600
}
```

Create `app.json` (root config -- overrides substruct values when fields overlap):

```json
{
    "AppName": "production-app",
    "DB": {
        "Host": "db-primary.internal"
    }
}
```

```bash
# Load all three config files:
$ go run . --config-file app.json --db-config-file db.json --cache-config-file cache.json
App: production-app
DB:    db-primary.internal:5432/production
Cache: redis.internal:6379 (TTL=600s)

# Note: DB.Host is "db-primary.internal" (root config overrides db.json's "db.internal")
# DB.Port and DB.Name come from db.json since the root config didn't set them.

# Only substruct configs, no root config:
$ go run . --db-config-file db.json --cache-config-file cache.json
App: myapp
DB:    db.internal:5432/production
Cache: redis.internal:6379 (TTL=600s)

# CLI overrides both configs:
$ go run . --db-config-file db.json --db-host cli-override
App: myapp
DB:    cli-override:5432/production
Cache: localhost:6379 (TTL=300s)
```

Priority for substruct values: **CLI > env > root config > substruct config > defaults**.

## Config Format Registry

JSON is the only format built in. BOA has no third-party parser dependencies — you bring your own library (`gopkg.in/yaml.v3`, `github.com/BurntSushi/toml`, …) and register it.

The primary model is **register once, dispatch by file extension**. Register every format your app might ever load at startup, and BOA picks the right parser for each `--config-file` argument at runtime. The same compiled binary transparently handles JSON today and YAML tomorrow — there is no per-command locking.

### The One-Liner

For every mainstream Go config parser (`yaml.Unmarshal`, `toml.Unmarshal`, `hcl.Decode`, `json.Unmarshal`, …), a single call is all you need:

```go
boa.RegisterConfigFormat(".yaml", yaml.Unmarshal)
boa.RegisterConfigFormat(".yml",  yaml.Unmarshal)
boa.RegisterConfigFormat(".toml", toml.Unmarshal)
```

That one call gets you:

1. **Parsing** — the extension now dispatches to your library's unmarshal function.
2. **Key-presence detection** — including zero-valued and same-as-default writes to optional struct-pointer parameter groups (see ["Why key-presence detection matters"](#why-key-presence-detection-matters) below for what that means in practice).

The second one comes from a helper called [`UniversalConfigFormat`](#the-universalconfigformat-helper) that `RegisterConfigFormat` uses under the hood: it asks the same parser to additionally decode the file into a `map[string]any`, which is how BOA reads the literal key structure. Every mainstream Go parser can do that, so this all works transparently.

### Register Multiple Formats, Dispatch by Extension

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
    "gopkg.in/yaml.v3"
)

type Params struct {
    ConfigFile string `configfile:"true" optional:"true"`
    Host       string `descr:"Server host" default:"localhost"`
    Port       int    `descr:"Server port" default:"8080"`
}

func init() {
    // One line per non-JSON format. JSON is registered by default, so the
    // binary now handles .yaml, .yml, AND .json transparently — dispatch
    // is decided per --config-file invocation by filepath.Ext.
    boa.RegisterConfigFormat(".yaml", yaml.Unmarshal)
    boa.RegisterConfigFormat(".yml", yaml.Unmarshal)
}

func main() {
    boa.CmdT[Params]{
        Use:   "server",
        Short: "Server that accepts either JSON or YAML config files",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Host: %s, Port: %d\n", p.Host, p.Port)
        },
    }.Run()
}
```

The same binary handles all three of these without any code change:

```bash
# Production deploy today: JSON
$ ./server --config-file prod.json

# Production redeploy tomorrow: YAML, same binary, just a different argument
$ ./server --config-file prod.yaml

# Ops-style one-off with a YAML sidecar file
$ ./server --config-file /etc/myapp/overrides.yml
```

BOA picks the parser per-call from `filepath.Ext(filePath)`, so there is no global "current format" and no rebuild required to switch.

> A complete runnable template — using a trivial dep-free "KV" format so the example doesn't drag a YAML/TOML dependency into this repo — lives at [`internal/example_custom_config_format`](https://github.com/GiGurra/boa/tree/main/internal/example_custom_config_format). Its tests load **both** a `.json` file and a `.kv` file through the same `main()`, proving the multi-format-per-binary story end-to-end. Swap the KV functions for `yaml.Unmarshal` + a yaml-backed `KeyTree` and you have the YAML example verbatim.

### Why Key-Presence Detection Matters

Consider:

```go
type DBConfig struct {
    Host string `descr:"db host" default:"localhost"`
    Port int    `descr:"db port" default:"5432"`
}

type Params struct {
    ConfigFile string `configfile:"true" optional:"true"`
    DB         *DBConfig // optional parameter group
}
```

With this `config.yaml`:

```yaml
DB:
  Host: ""     # zero value
  Port: 5432   # same as the default
```

Without a `KeyTree`, BOA falls back to snapshot comparison — it compares struct values before and after loading. Those writes don't change anything, so BOA concludes "nothing set": `p.DB` is nil'd back out after cleanup, and `HookContext.HasValue(&p.DB.Port)` keeps reporting `false` (since the snapshot saw no change) even for flat top-level fields with a zero-value write. With a `KeyTree`, BOA sees the literal key structure, recognises that `DB`, `DB.Host`, and `DB.Port` were mentioned, keeps the pointer group alive, and correctly reports `HasValue` / set-by-config for every leaf the file actually wrote.

This matters whenever you care about the difference between "the config file mentioned this field" and "the field kept its default". It applies both to optional struct-pointer parameter groups (where the difference decides whether the group survives cleanup) and to plain top-level fields (where the difference is visible via `HookContext.HasValue`). `RegisterConfigFormat` already wires up the `KeyTree` for you whenever the parser can decode into `map[string]any` — which is every mainstream Go parser.

**Field name matching is format-aware.** Because the dump and load paths share a single extension→struct-tag mapping, BOA looks up each field using the struct tag the parser itself respects: `json` for `.json`, `yaml` for `.yaml` / `.yml`, `toml` for `.toml`, `hcl` for `.hcl`, and for any other registered extension the tag defaults to the extension name minus its leading dot (so `.mycustom` consults the `mycustom` tag). Renames like `Host string \`yaml:"host_name"\`` are therefore picked up by set-by-config detection too, not just by the Go-side unmarshaler. Tag value `"-"` skips the field, and tag value `"name,opt,opt"` uses just `name` — same conventions every mainstream Go config parser already follows.

### The `UniversalConfigFormat` Helper

`RegisterConfigFormat` uses it internally; you only ever call it directly when you want to set a format inline on `Cmd.ConfigFormat`:

```go
boa.CmdT[Params]{
    Use:          "server",
    ConfigFormat: boa.UniversalConfigFormat(yaml.Unmarshal),
    RunFunc: func(p *Params, cmd *cobra.Command, args []string) { ... },
}.Run()
```

`UniversalConfigFormat(fn)` returns a `ConfigFormat` whose `Unmarshal` is `fn` and whose `KeyTree` invokes the same `fn` against a `map[string]any` target. That's enough for every parser that treats `any`/`interface{}` targets uniformly — i.e. every mainstream Go config library. Passing `nil` panics, so typos surface immediately.

### When You Genuinely Need the Full Form

Reach for the verbose `boa.ConfigFormat{Unmarshal: ..., KeyTree: ...}` literal (and `RegisterConfigFormatFull`) only when your parser **cannot** decode into `map[string]any` — for example, a custom format whose unmarshaler only knows how to populate specific struct types. In that case you have to hand-write the `KeyTree` yourself because `UniversalConfigFormat` would fail at parse time.

The runnable example at [`internal/example_custom_config_format`](https://github.com/GiGurra/boa/tree/main/internal/example_custom_config_format) shows exactly this case: a tiny KV format whose `kvUnmarshal` only populates structs, so it registers via `RegisterConfigFormatFull` with a hand-written `kvKeyTree`. If you're using a mainstream library like `yaml.v3`, you'll never write code that looks like that — `RegisterConfigFormat(".yaml", yaml.Unmarshal)` is all you need.

> `KeyTree` can return nested maps as either `map[string]any` (yaml.v3, json) or `map[any]any` (yaml.v2) — BOA coerces transparently.

### Per-Command Override (Escape Hatch)

Setting a format on `Cmd.ConfigFormat` (or the legacy `ConfigUnmarshal`) **bypasses** the extension registry for that one command and locks it to a single format. That is almost never what you want — prefer the registry so the same binary stays format-agnostic — but the escape hatch is there for niche cases like:

- A command that must accept a custom-extension blob from a legacy system.
- Tests that want to inject a fake parser without polluting the global registry.

```go
boa.CmdT[Params]{
    Use: "ingest-legacy-blob",
    ConfigFormat: boa.ConfigFormat{
        Unmarshal: myLegacyUnmarshal,
        // KeyTree optional
    },
    RunFunc: func(p *Params, cmd *cobra.Command, args []string) { ... },
}.Run()
```

Resolution order for each `loadConfigFileInto` call:

1. `Cmd.ConfigFormat` (if `Unmarshal` is non-nil) — locks this one command to a single format
2. `Cmd.ConfigUnmarshal` (legacy; unmarshal-only, also command-locked)
3. Format registered by file extension — **the default path; supports any number of formats in one binary**
4. Built-in JSON fallback (with `KeyTree`)

## Config-File-Only Fields (`boa:"configonly"` and `boa:"ignore"`)

Two tags hide fields from the CLI and env but differ in whether boa still runs a mirror/validation:

- `boa:"configonly"` — mirror preserved, validation still runs (use for validated config-file-only fields)
- `boa:"ignore"` — field fully excluded from boa, only raw config unmarshal writes to it (use for opaque blobs)

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    ConfigFile string            `configfile:"true" optional:"true" default:"config.json"`
    Host       string            `descr:"Server host" default:"localhost"`
    Port       int               `descr:"Server port" default:"8080"`
    InternalID string            `boa:"configonly" min:"8"` // config file only, validated
    Metadata   map[string]string `boa:"configonly"`         // config file only
    Routes     []RouteConfig     `boa:"ignore"`             // opaque, not validated by boa
}

type RouteConfig struct {
    Path    string `json:"path"`
    Backend string `json:"backend"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "server",
        Short: "Server with config-only fields",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Host: %s:%d\n", p.Host, p.Port)
            fmt.Printf("Internal ID: %s\n", p.InternalID)
            fmt.Printf("Metadata: %v\n", p.Metadata)
            for _, r := range p.Routes {
                fmt.Printf("Route: %s -> %s\n", r.Path, r.Backend)
            }
        },
    }.Run()
}
```

Create `config.json`:

```json
{
    "Host": "api.example.com",
    "Port": 8080,
    "InternalID": "svc-abc-123",
    "Metadata": {
        "version": "2.1.0",
        "region": "us-east-1"
    },
    "Routes": [
        {"path": "/api", "backend": "http://backend:3000"},
        {"path": "/static", "backend": "http://cdn:8080"}
    ]
}
```

```bash
$ go run .
Host: api.example.com:8080
Internal ID: svc-abc-123
Metadata: map[region:us-east-1 version:2.1.0]
Route: /api -> http://backend:3000
Route: /static -> http://cdn:8080

# Host and Port can be overridden via CLI:
$ go run . --host localhost --port 3000
Host: localhost:3000
Internal ID: svc-abc-123
...

# InternalID and Routes are NOT available as CLI flags:
$ go run . --internal-id foo
# Error: unknown flag: --internal-id
```

### Ignored Sub-Structs

You can also ignore an entire sub-struct. Its fields will not appear as CLI flags, but the struct is still populated from config files.

```go
type DBConfig struct {
    Host     string `json:"host"`
    Port     int    `json:"port"`
    Password string `json:"password"`
}

type Params struct {
    ConfigFile string   `configfile:"true" optional:"true" default:"config.json"`
    AppName    string   `descr:"App name"`
    DB         DBConfig `boa:"ignore"` // entire struct is config-only
}
```

## Auto-Discovery with boaviper

The `boaviper` package provides Viper-like automatic config file discovery. It searches standard paths for config files without requiring the user to specify `--config-file`.

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/GiGurra/boa/pkg/boaviper"
    "github.com/spf13/cobra"
)

type Params struct {
    ConfigFile string `configfile:"true" optional:"true"`
    Host       string `descr:"Server host" default:"localhost"`
    Port       int    `descr:"Server port" default:"8080"`
    Debug      bool   `descr:"Debug mode" optional:"true"`
}

func main() {
    boa.CmdT[Params]{
        Use:      "myapp",
        Short:    "App with auto-discovery",
        InitFunc: boaviper.AutoConfig[Params]("myapp"),
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Config: %s\nHost: %s, Port: %d, Debug: %v\n",
                p.ConfigFile, p.Host, p.Port, p.Debug)
        },
    }.Run()
}
```

`boaviper.AutoConfig` searches these paths (first match wins):

1. `./myapp.json` (current directory)
2. `$HOME/.config/myapp/config.json`
3. `/etc/myapp/config.json`

All registered config format extensions are tried at each path (e.g., `.json`, `.yaml` if registered).

```bash
# Auto-discovers ./myapp.json:
$ echo '{"Port": 9090}' > myapp.json
$ go run .
Config: myapp.json
Host: localhost, Port: 9090, Debug: false

# Explicit --config-file overrides auto-discovery:
$ go run . --config-file /etc/myapp/prod.json
Config: /etc/myapp/prod.json
...

# No config file found? Uses defaults:
$ rm myapp.json
$ go run .
Config:
Host: localhost, Port: 8080, Debug: false
```

### Custom Search Paths

```go
boaviper.AutoConfig[Params]("myapp", "./config", "/opt/myapp/etc")
```

This searches:

1. `./config/myapp.json`
2. `/opt/myapp/etc/config.json`
3. `/opt/myapp/etc/myapp.json`

### Auto-Discover with Env Prefix

Combine auto-discovery with prefixed environment variables for a fully Viper-like experience:

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/GiGurra/boa/pkg/boaviper"
    "github.com/spf13/cobra"
)

type Params struct {
    ConfigFile string `configfile:"true" optional:"true"`
    Host       string `descr:"Server host" default:"localhost"`
    Port       int    `descr:"Server port" default:"8080"`
}

func main() {
    boa.CmdT[Params]{
        Use:  "myapp",
        Short: "Viper-like CLI",
        // Auto-discover config files
        InitFunc: boaviper.AutoConfig[Params]("myapp"),
        // Prefix all env vars with MYAPP_
        ParamEnrich: boa.ParamEnricherCombine(
            boa.ParamEnricherDefault,
            boaviper.SetEnvPrefix("MYAPP"),
        ),
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Host: %s, Port: %d\n", p.Host, p.Port)
        },
    }.Run()
}
```

```bash
# All three sources work:
$ MYAPP_PORT=3000 go run .
Host: localhost, Port: 3000

# Priority: CLI > env > config file > default
$ echo '{"Port": 5000}' > myapp.json
$ MYAPP_PORT=3000 go run . --port 9090
Host: localhost, Port: 9090
```

## Explicit Config File Loading

For full control over config file loading, use `boa.LoadConfigFile` in a `PreValidateFunc` hook. This is useful when you need to load into a sub-struct or apply custom logic.

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type AppConfig struct {
    Host string
    Port int
}

type Params struct {
    ConfigFile string `descr:"Path to config file" optional:"true"`
    AppConfig         // embedded -- fields become --host, --port
}

func main() {
    boa.CmdT[Params]{
        Use:   "app",
        Short: "Explicit config loading",
        PreValidateFunc: func(p *Params, cmd *cobra.Command, args []string) error {
            // Load config file into the embedded AppConfig
            return boa.LoadConfigFile(p.ConfigFile, &p.AppConfig, nil)
        },
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Host: %s, Port: %d\n", p.Host, p.Port)
        },
    }.Run()
}
```

`LoadConfigFile` signature:

```go
func LoadConfigFile[T any](filePath string, target *T, unmarshalFunc func([]byte, any) error) error
```

- `filePath`: path to the config file (empty string is a no-op)
- `target`: pointer to the struct to populate
- `unmarshalFunc`: custom unmarshal function (`nil` uses file extension detection, then falls back to `json.Unmarshal`)

## Multi-File Overlay (Base + Local Cascade)

For the classic 12-factor `config.json` + `config.local.json` pattern, declare the configfile field as a `[]string` — later files overlay earlier ones at the key level:

```go
type Params struct {
    ConfigFiles []string `configfile:"true" optional:"true"`
    Host        string   `optional:"true"`
    Port        int      `optional:"true"`
}

// CLI:
//   app --config-files base.json,local.json
//   app --config-files base.json --config-files prod.json
```

`base.json`:
```json
{"Host": "app.example.com", "Port": 80}
```

`local.json` (developer override):
```json
{"Port": 8080}
```

After the chain loads, the resolved parameters are `Host=app.example.com` (from base, unchanged by local) and `Port=8080` (local overlaid base). Keys that are absent in the later file leave the earlier values intact — that's the natural behavior of sequential `json.Unmarshal` calls into the same struct.

### Overlay semantics

- **Order**: left-to-right is lowest-to-highest precedence. The rightmost file wins for any key it mentions.
- **Missing keys**: leave earlier values alone. Not-mentioned ≠ reset.
- **Slices and maps**: *fully replaced* by the later file, not merged. If base has `Tags: [a, b]` and local has `Tags: [c]`, the final value is `[c]`. This is standard `json.Unmarshal` behavior and matches what almost every config-cascade user expects. (Deep merging is deliberately out of scope.)
- **CLI and env still win**: the full precedence chain is unchanged — CLI > env > root config chain > substruct config chain > defaults. The multi-file chain slots in at "root config" (or "substruct config" for nested declarations).
- **Substructs** can declare their own `[]string` configfile chain too, and each chain loads independently. Substruct chains load first, the root chain loads last.
- **Empty strings** in the list are skipped silently — handy when an optional overlay is computed at runtime.
- **Missing files** produce a clean error naming the file that failed.

### `LoadConfigFiles` helper

If you'd rather build the path list yourself (e.g. from environment, computed from the user's home directory, or mixing embedded defaults with on-disk overrides), use the helper in a `PreValidateFunc`:

```go
func main() {
    boa.CmdT[Params]{
        Use: "app",
        PreValidateFunc: func(p *Params, cmd *cobra.Command, args []string) error {
            paths := []string{
                "/etc/myapp/config.json",                  // system defaults
                filepath.Join(os.Getenv("HOME"), ".myapp.json"), // user overrides
                "./myapp.local.json",                      // per-project overrides
            }
            return boa.LoadConfigFiles(paths, p, nil)
        },
    }.Run()
}
```

Signature:

```go
func LoadConfigFiles[T any](paths []string, target *T, unmarshalFunc func([]byte, any) error) error
```

Empty strings in `paths` are skipped; a nil or empty slice is a no-op. Loading stops at the first missing file and returns the underlying error.

## Loading Config From Bytes

When the config does not live on disk — for example, `//go:embed` assets, stdin, an HTTP response body, or a test fixture — use `boa.LoadConfigBytes`. It shares the same format-resolution rules as `LoadConfigFile`, so registered formats like YAML or TOML work exactly the same.

```go
package main

import (
    _ "embed"
    "fmt"

    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
    "gopkg.in/yaml.v3"
)

//go:embed defaults.yaml
var defaultsYAML []byte

type Params struct {
    Host string `optional:"true"`
    Port int    `optional:"true"`
}

func main() {
    boa.RegisterConfigFormat(".yaml", yaml.Unmarshal)

    boa.CmdT[Params]{
        Use: "app",
        PreValidateFunc: func(p *Params, cmd *cobra.Command, args []string) error {
            // Seed defaults from the embedded YAML blob. CLI and env vars
            // still win over whatever is loaded here.
            return boa.LoadConfigBytes(defaultsYAML, ".yaml", p, nil)
        },
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Host=%s Port=%d\n", p.Host, p.Port)
        },
    }.Run()
}
```

`LoadConfigBytes` signature:

```go
func LoadConfigBytes[T any](data []byte, ext string, target *T, unmarshalFunc func([]byte, any) error) error
```

- `data`: the raw config bytes (empty or `nil` is a no-op)
- `ext`: file extension used to pick a registered format — `".yaml"`, `"yaml"`, or `""` (empty falls back to JSON). The leading dot is optional.
- `target`: pointer to the struct to populate
- `unmarshalFunc`: custom unmarshal function — when non-nil, takes precedence over `ext`

Typical sources:

- `//go:embed` — ship a default config file inside the binary
- Piped stdin — `data, _ := io.ReadAll(os.Stdin)`
- HTTP / S3 / secrets manager responses — treat the response body as config bytes
- In-memory test fixtures — no temp files required

## Writing Config Back Out

BOA can also serialize a resolved parameter set back out to a config file. Two variants are provided:

| API | When to use |
|-----|-------------|
| `boa.DumpConfigBytes(v, ext, nil)` / `boa.DumpConfigFile(path, v, nil)` | **Naive dump.** Emits every exported field on `v`, including Go zero values. Good for "generate an example config that shows every option". |
| `ctx.DumpBytes(ext, nil)` / `ctx.DumpFile(path, nil)` (on `HookContext`) | **Source-aware dump.** Emits only fields that have a value from CLI, environment, config file, or a default — fields the user never touched are omitted entirely. This is the right helper for persisting resolved config between runs. |

Source-aware dump is the useful one for most production apps: because defaults count as "set", the dumped file pins the current default values in place, so a future release that ships different built-in defaults won't silently change behavior for users whose saved config said "I'm happy with what you shipped in version 1.0".

```go
package main

import (
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
    "gopkg.in/yaml.v3"
)

type Params struct {
    Name    string `optional:"true"`
    Host    string `optional:"true" default:"localhost"`
    Port    int    `optional:"true" default:"8080"`
    Verbose bool   `optional:"true"`
}

func main() {
    boa.RegisterConfigFormat(".yaml", yaml.Unmarshal)
    boa.RegisterConfigMarshaler(".yaml", yaml.Marshal)

    boa.CmdT[Params]{
        Use: "app",
        RunFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command, args []string) {
            // Persist the resolved config so the next run reuses it.
            // CLI-set values, env-set values, config-file values, and
            // defaults all get written out. Fields the user never
            // touched stay omitted.
            if err := ctx.DumpFile("~/.myapp/config.yaml", nil); err != nil {
                // ...
            }
        },
    }.Run()
}
```

API signatures:

```go
// Naive — whole struct, zero values and all.
func DumpConfigBytes[T any](v *T, ext string, marshalFunc func(v any) ([]byte, error)) ([]byte, error)
func DumpConfigFile[T any](filePath string, v *T, marshalFunc func(v any) ([]byte, error)) error

// Source-aware — only fields with HasValue=true.
func (c *HookContext) DumpBytes(ext string, marshalFunc func(v any) ([]byte, error)) ([]byte, error)
func (c *HookContext) DumpFile(filePath string, marshalFunc func(v any) ([]byte, error)) error
```

### Enabling dump for non-JSON formats

`RegisterConfigFormat` only installs an unmarshaler. To also enable `Dump*`, pair it with `RegisterConfigMarshaler`:

```go
boa.RegisterConfigFormat(".yaml", yaml.Unmarshal)
boa.RegisterConfigMarshaler(".yaml", yaml.Marshal)
```

JSON comes with both directions pre-registered, indented with two spaces and terminated with a trailing newline.

If you try to dump to a format with no registered marshaler, `Dump*` returns a clear error rather than silently falling through to JSON — writing JSON bytes to a file named `.yaml` would be a nasty surprise.

### What gets omitted from a source-aware dump

- **Unset fields** — fields where nothing (CLI, env, config, inject, default) has ever supplied a value. `Silent bool` without a default stays out of the dump rather than appearing as `"Silent": false` for every user who never touched it.
- **The `configfile` field** — the path to the config file itself is never written into the dumped file, because a config file that references its own path is self-referential and surprising on the next load.
- **Nested structs with no set descendants** — if nothing in `DB` is set, the whole `DB` key is omitted (not emitted as an empty object).
- **Nil optional struct-pointer groups** (`DB *DBConfig`) — same as the above.

### What gets kept

- **CLI / env / config / inject values** — always.
- **Defaults** — always, to pin them against future app upgrades that might change them. The one exception is bool fields whose only claim to "set" is the auto-installed `false` default from `ParamEnricherBool`; those are treated as unset unless the user explicitly flipped them.

## Mixed Config Formats

Different config files can use different formats. The format is detected by file extension when using the registry.

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
    "gopkg.in/yaml.v3"
)

type DBConfig struct {
    ConfigFile string `configfile:"true" optional:"true"`
    Host       string `descr:"DB host" default:"localhost"`
    Port       int    `descr:"DB port" default:"5432"`
}

type Params struct {
    ConfigFile string   `configfile:"true" optional:"true"`
    AppName    string   `descr:"App name" default:"myapp"`
    DB         DBConfig
}

func main() {
    // Register YAML in addition to built-in JSON
    boa.RegisterConfigFormat(".yaml", yaml.Unmarshal)

    boa.CmdT[Params]{
        Use:   "app",
        Short: "Mixed format configs",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("App: %s\nDB: %s:%d\n", p.AppName, p.DB.Host, p.DB.Port)
        },
    }.Run()
}
```

```bash
# Root config as YAML, DB config as JSON:
$ go run . --config-file app.yaml --db-config-file db.json
App: from-yaml
DB: db-host:5432
```

BOA picks the correct unmarshal function based on each file's extension independently.
