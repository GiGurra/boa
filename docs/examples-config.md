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

JSON is the only format built in. Register additional formats with `boa.RegisterConfigFormat`.

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
    "gopkg.in/yaml.v3"
)

type Params struct {
    ConfigFile string `configfile:"true" optional:"true" default:"config.yaml"`
    Host       string `descr:"Server host"`
    Port       int    `descr:"Server port" default:"8080"`
}

func main() {
    // Register YAML support -- file extension determines which unmarshal to use
    boa.RegisterConfigFormat(".yaml", yaml.Unmarshal)
    boa.RegisterConfigFormat(".yml", yaml.Unmarshal)

    boa.CmdT[Params]{
        Use:   "server",
        Short: "Server with YAML config",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Host: %s, Port: %d\n", p.Host, p.Port)
        },
    }.Run()
}
```

Create `config.yaml`:

```yaml
Host: api.example.com
Port: 3000
```

```bash
$ go run .
Host: api.example.com, Port: 3000

# Can also use JSON files -- BOA picks the right parser by extension:
$ go run . --config-file config.json
Host: ...
```

### Override Unmarshal Per Command

Use `ConfigUnmarshal` on the command to bypass file extension detection entirely:

```go
boa.CmdT[Params]{
    Use:             "server",
    ConfigUnmarshal: yaml.Unmarshal,  // Always use YAML regardless of file extension
    RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
        // ...
    },
}.Run()
```

Resolution order for choosing the unmarshal function:

1. Explicit `ConfigUnmarshal` on the command
2. Registered format matched by file extension (`.yaml` -> `yaml.Unmarshal`)
3. `json.Unmarshal` (default fallback)

## Config-File-Only Fields (`boa:"ignore"`)

Fields tagged `boa:"ignore"` (or `boa:"configonly"`) are not exposed as CLI flags or env vars. They only get populated from config files.

This is useful for complex settings that make sense in a file but not on the command line.

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
    InternalID string            `boa:"ignore"`     // config file only
    Metadata   map[string]string `boa:"configonly"` // config file only (clearer alias)
    Routes     []RouteConfig     `boa:"ignore"`     // complex nested config
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
