# Advanced Examples

Examples covering custom types, complex data structures, nested structs, subcommands, validation, and shell completion.

## Custom Types with RegisterType

Register any Go type as a CLI parameter by providing parse and format functions.

### SemVer Example

```go
package main

import (
    "fmt"
    "strconv"
    "strings"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type SemVer struct {
    Major, Minor, Patch int
}

func (v SemVer) String() string {
    return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func parseSemVer(s string) (SemVer, error) {
    parts := strings.SplitN(s, ".", 3)
    if len(parts) != 3 {
        return SemVer{}, fmt.Errorf("expected MAJOR.MINOR.PATCH, got %q", s)
    }
    major, err := strconv.Atoi(parts[0])
    if err != nil {
        return SemVer{}, fmt.Errorf("invalid major version: %w", err)
    }
    minor, err := strconv.Atoi(parts[1])
    if err != nil {
        return SemVer{}, fmt.Errorf("invalid minor version: %w", err)
    }
    patch, err := strconv.Atoi(parts[2])
    if err != nil {
        return SemVer{}, fmt.Errorf("invalid patch version: %w", err)
    }
    return SemVer{Major: major, Minor: minor, Patch: patch}, nil
}

type Params struct {
    Version SemVer `descr:"Application version" default:"0.1.0"`
}

func main() {
    boa.RegisterType[SemVer](boa.TypeDef[SemVer]{
        Parse:  parseSemVer,
        Format: func(v SemVer) string { return v.String() },
    })

    boa.CmdT[Params]{
        Use:   "release",
        Short: "Create a release",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Releasing version %s\n", p.Version)
            fmt.Printf("Major: %d, Minor: %d, Patch: %d\n",
                p.Version.Major, p.Version.Minor, p.Version.Patch)
        },
    }.Run()
}
```

```bash
$ go run . --version 2.1.0
Releasing version 2.1.0
Major: 2, Minor: 1, Patch: 0

$ go run .
Releasing version 0.1.0
Major: 0, Minor: 1, Patch: 0

$ go run . --version not-valid
# Error: invalid value for param version: expected MAJOR.MINOR.PATCH, got "not-valid"
```

### LogLevel Example

```go
package main

import (
    "fmt"
    "strings"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type LogLevel int

const (
    Debug LogLevel = iota
    Info
    Warn
    Error
)

func (l LogLevel) String() string {
    switch l {
    case Debug:
        return "debug"
    case Info:
        return "info"
    case Warn:
        return "warn"
    case Error:
        return "error"
    default:
        return "unknown"
    }
}

func parseLogLevel(s string) (LogLevel, error) {
    switch strings.ToLower(s) {
    case "debug":
        return Debug, nil
    case "info":
        return Info, nil
    case "warn", "warning":
        return Warn, nil
    case "error":
        return Error, nil
    default:
        return Info, fmt.Errorf("unknown log level %q (use debug, info, warn, error)", s)
    }
}

type Params struct {
    Level LogLevel `descr:"Log level" default:"info"`
}

func main() {
    boa.RegisterType[LogLevel](boa.TypeDef[LogLevel]{
        Parse:  parseLogLevel,
        Format: func(l LogLevel) string { return l.String() },
    })

    boa.CmdT[Params]{
        Use:   "logger",
        Short: "Demo custom log level type",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Log level: %s (%d)\n", p.Level, p.Level)
        },
    }.Run()
}
```

```bash
$ go run . --level warn
Log level: warn (2)
```

### Optional Custom Types with Pointers

Pointer-to-custom-type fields are optional by default, just like pointer-to-primitive fields. `nil` means not provided.

```go
type Params struct {
    Version *SemVer `descr:"App version"` // optional, nil if not set
}
```

## Map Fields

### Simple Key-Value Maps

Simple maps use the ergonomic `key=val,key=val` syntax on the CLI. Map fields default to optional.

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    Labels map[string]string `descr:"Key-value labels"`
    Ports  map[string]int    `descr:"Named ports"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "deploy",
        Short: "Deploy with labels and ports",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Println("Labels:")
            for k, v := range p.Labels {
                fmt.Printf("  %s = %s\n", k, v)
            }
            fmt.Println("Ports:")
            for k, v := range p.Ports {
                fmt.Printf("  %s = %d\n", k, v)
            }
        },
    }.Run()
}
```

```bash
$ go run . --labels env=prod,team=backend --ports http=80,https=443
Labels:
  env = prod
  team = backend
Ports:
  http = 80
  https = 443
```

### Maps from Environment Variables

```go
type Params struct {
    Labels map[string]string `descr:"Labels" env:"APP_LABELS"`
}
```

```bash
$ APP_LABELS="env=staging,region=us-east" go run .
```

### Maps from Config Files

Maps are populated naturally from JSON config files:

```go
type Params struct {
    ConfigFile string            `configfile:"true" optional:"true" default:"config.json"`
    Labels     map[string]string `descr:"Labels" optional:"true"`
}
```

```json
{
    "Labels": {"app": "myapp", "version": "v2"}
}
```

## Complex Types with JSON on CLI

Types without native pflag support (nested slices, maps with complex values) automatically fall back to JSON parsing.

### Nested Slices (Matrix)

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    Matrix [][]int `descr:"Data matrix"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "matrix",
        Short: "Process a data matrix",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            for i, row := range p.Matrix {
                fmt.Printf("Row %d: %v\n", i, row)
            }
        },
    }.Run()
}
```

```bash
$ go run . --matrix '[[1,2,3],[4,5,6],[7,8,9]]'
Row 0: [1 2 3]
Row 1: [4 5 6]
Row 2: [7 8 9]
```

### Complex Map Types

```go
type Params struct {
    Config map[string][]string `descr:"Multi-value config"`
}
```

```bash
$ go run . --config '{"tags":["a","b"],"owners":["alice","bob"]}'
```

### Arbitrary Nested Structures

```go
type Params struct {
    Data map[string]any `descr:"Arbitrary JSON data" optional:"true"`
}
```

```bash
$ go run . --data '{"debug":true,"retries":3,"servers":["a","b"]}'
```

### JSON via Environment Variables

The same JSON syntax works for env vars:

```bash
$ MATRIX='[[1,2],[3,4]]' go run .
```

## Nested Struct Composition with Auto-Prefixing

Named (non-anonymous) struct fields auto-prefix their children's flag names. This prevents collisions when reusing the same struct type.

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type ConnectionConfig struct {
    Host     string `descr:"Hostname" default:"localhost"`
    Port     int    `descr:"Port number" default:"5432"`
    Username string `descr:"Username" default:"admin"`
}

type Params struct {
    Primary ConnectionConfig // --primary-host, --primary-port, --primary-username
    Replica ConnectionConfig // --replica-host, --replica-port, --replica-username
}

func main() {
    boa.CmdT[Params]{
        Use:   "db",
        Short: "Database connection manager",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Primary: %s@%s:%d\n", p.Primary.Username, p.Primary.Host, p.Primary.Port)
            fmt.Printf("Replica: %s@%s:%d\n", p.Replica.Username, p.Replica.Host, p.Replica.Port)
        },
    }.Run()
}
```

```bash
$ go run . --primary-host db1.internal --replica-host db2.internal
Primary: admin@db1.internal:5432
Replica: admin@db2.internal:5432

$ go run . --primary-host db1 --primary-port 5433 --replica-host db2 --replica-username readonly
Primary: admin@db1:5433
Replica: readonly@db2:5432
```

### Deep Nesting (3+ Levels)

Prefixes chain at every named level:

```go
type ConnectionConfig struct {
    Host string `descr:"Hostname" default:"localhost"`
    Port int    `descr:"Port number" default:"5432"`
}

type ClusterConfig struct {
    Primary ConnectionConfig
    Replica ConnectionConfig
}

type Params struct {
    Infra ClusterConfig
}
// Flags: --infra-primary-host, --infra-primary-port,
//        --infra-replica-host, --infra-replica-port
```

```bash
$ go run . --infra-primary-host primary.db --infra-replica-host replica.db
```

### Explicit Tags Are Also Prefixed

Inside named struct fields, explicit `name` and `env` tags are prefixed too. This avoids collisions when the same struct appears multiple times:

```go
type ServerConfig struct {
    Host string `name:"host" env:"SERVER_HOST" default:"localhost"`
    Port int    `name:"port" env:"SERVER_PORT" default:"8080"`
}

type Params struct {
    API ServerConfig  // --api-host, --api-port, env: API_SERVER_HOST, API_SERVER_PORT
    Web ServerConfig  // --web-host, --web-port, env: WEB_SERVER_HOST, WEB_SERVER_PORT
}
```

## Embedded Structs for Shared Options

Embedded (anonymous) struct fields are NOT prefixed. Use this to share common options across commands.

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type CommonOpts struct {
    Verbose bool   `descr:"Verbose output" short:"v" optional:"true"`
    Format  string `descr:"Output format" alts:"json,text,table" default:"text"`
}

type ListParams struct {
    CommonOpts           // embedded -- flags: --verbose, --format (no prefix)
    Limit      int       `descr:"Max items" default:"50"`
}

type GetParams struct {
    CommonOpts           // embedded -- same flags, no prefix
    ID         string    `descr:"Item ID"`
}

func main() {
    boa.CmdT[boa.NoParams]{
        Use:   "items",
        Short: "Manage items",
        SubCmds: boa.SubCmds(
            boa.CmdT[ListParams]{
                Use:   "list",
                Short: "List items",
                RunFunc: func(p *ListParams, cmd *cobra.Command, args []string) {
                    fmt.Printf("Listing %d items (format=%s, verbose=%v)\n",
                        p.Limit, p.Format, p.Verbose)
                },
            },
            boa.CmdT[GetParams]{
                Use:   "get",
                Short: "Get an item",
                RunFunc: func(p *GetParams, cmd *cobra.Command, args []string) {
                    fmt.Printf("Getting item %s (format=%s, verbose=%v)\n",
                        p.ID, p.Format, p.Verbose)
                },
            },
        ),
    }.Run()
}
```

```bash
$ go run . list --verbose --limit 10
Listing 10 items (format=text, verbose=true)

$ go run . get --id abc-123 --format json
Getting item abc-123 (format=json, verbose=false)
```

### Mixing Embedded and Named

```go
type Params struct {
    CommonOpts           // embedded -- --verbose, --format (no prefix)
    DB         DBConfig  // named   -- --db-host, --db-port (prefixed)
}
```

## Subcommands

Build hierarchical CLI tools with subcommands. Use `boa.NoParams` for parent commands that have no parameters of their own.

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type AddParams struct {
    Name  string `descr:"Item name"`
    Count int    `descr:"Quantity" default:"1"`
}

type RemoveParams struct {
    ID    string `descr:"Item ID"`
    Force bool   `descr:"Skip confirmation" optional:"true"`
}

type ListParams struct {
    Limit  int    `descr:"Max items" default:"20"`
    Format string `descr:"Output format" alts:"json,table" default:"table"`
}

func main() {
    boa.CmdT[boa.NoParams]{
        Use:   "inventory",
        Short: "Manage inventory items",
        SubCmds: boa.SubCmds(
            boa.CmdT[AddParams]{
                Use:     "add",
                Short:   "Add an item",
                Aliases: []string{"a"},
                RunFunc: func(p *AddParams, cmd *cobra.Command, args []string) {
                    fmt.Printf("Added %d x %s\n", p.Count, p.Name)
                },
            },
            boa.CmdT[RemoveParams]{
                Use:     "remove",
                Short:   "Remove an item",
                Aliases: []string{"rm"},
                RunFunc: func(p *RemoveParams, cmd *cobra.Command, args []string) {
                    if p.Force {
                        fmt.Printf("Removed %s (forced)\n", p.ID)
                    } else {
                        fmt.Printf("Removing %s... confirm? (use --force to skip)\n", p.ID)
                    }
                },
            },
            boa.CmdT[ListParams]{
                Use:     "list",
                Short:   "List items",
                Aliases: []string{"ls"},
                RunFunc: func(p *ListParams, cmd *cobra.Command, args []string) {
                    fmt.Printf("Listing up to %d items (format=%s)\n", p.Limit, p.Format)
                },
            },
        ),
    }.Run()
}
```

```bash
$ go run . add --name "Widget" --count 5
Added 5 x Widget

$ go run . rm --id item-123 --force
Removed item-123 (forced)

$ go run . ls --format json --limit 10
Listing up to 10 items (format=json)

$ go run . --help
Manage inventory items

Usage:
  inventory [command]

Available Commands:
  add         Add an item
  list        List items
  remove      Remove an item
```

### Nested Subcommands

```go
boa.CmdT[boa.NoParams]{
    Use: "app",
    SubCmds: boa.SubCmds(
        boa.CmdT[boa.NoParams]{
            Use:   "cluster",
            Short: "Cluster management",
            SubCmds: boa.SubCmds(
                boa.CmdT[CreateParams]{Use: "create", ...},
                boa.CmdT[DeleteParams]{Use: "delete", ...},
            ),
        },
    ),
}
```

```bash
$ go run . cluster create --name my-cluster
```

### Command Groups

Organize subcommands into named groups in help output:

```go
boa.CmdT[boa.NoParams]{
    Use: "app",
    Groups: []*cobra.Group{
        {ID: "core", Title: "Core Commands:"},
        {ID: "util", Title: "Utility Commands:"},
    },
    SubCmds: boa.SubCmds(
        boa.CmdT[boa.NoParams]{Use: "init", GroupID: "core", ...},
        boa.CmdT[boa.NoParams]{Use: "run", GroupID: "core", ...},
        boa.CmdT[boa.NoParams]{Use: "version", GroupID: "util", ...},
        boa.CmdT[boa.NoParams]{Use: "config", GroupID: "util", ...},
    ),
}
```

Groups are auto-generated from subcommand `GroupID` values if you do not specify explicit `Groups`.

## Validation with min/max/pattern Tags

BOA provides built-in validation tags for numeric ranges and string patterns.

### Numeric min/max

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    Port    int     `descr:"Server port" min:"1" max:"65535"`
    Retries int     `descr:"Max retries" min:"0" max:"10" default:"3"`
    Rate    float64 `descr:"Request rate" min:"0.0" max:"1.0" default:"0.5"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "server",
        Short: "Start with validated params",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Port: %d, Retries: %d, Rate: %.2f\n", p.Port, p.Retries, p.Rate)
        },
    }.Run()
}
```

```bash
$ go run . --port 8080
Port: 8080, Retries: 3, Rate: 0.50

$ go run . --port 0
# Error: value 0 for param 'port' is below min 1

$ go run . --port 70000
# Error: value 70000 for param 'port' is above max 65535

$ go run . --rate 1.5
# Error: value 1.5 for param 'rate' is above max 1
```

### String min/max (length validation)

For string fields, `min` and `max` validate the string length:

```go
type Params struct {
    Name string `descr:"Project name" min:"3" max:"20"`
}
```

```bash
$ go run . --name "ab"
# Error: length 2 of param 'name' is below min 3

$ go run . --name "a-very-long-name-that-exceeds-twenty"
# Error: length 36 of param 'name' is above max 20

$ go run . --name "my-project"
# OK
```

### String Pattern Validation

Use `pattern` for regex validation on string fields:

```go
type Params struct {
    Name string `descr:"Resource name" pattern:"^[a-z][a-z0-9-]*$"`
    Tag  string `descr:"Version tag" pattern:"^v[0-9]+\\.[0-9]+\\.[0-9]+$" optional:"true"`
}
```

```bash
$ go run . --name my-app-123
# OK

$ go run . --name MyApp
# Error: value "MyApp" for param 'name' does not match pattern ^[a-z][a-z0-9-]*$

$ go run . --name my-app --tag v1.2.3
# OK

$ go run . --name my-app --tag latest
# Error: value "latest" for param 'tag' does not match pattern ...
```

### Validation with Pointer Fields

Validation tags are only checked when a value is actually provided. Pointer fields that are `nil` (not set) skip validation:

```go
type Params struct {
    Port *int    `descr:"Port" min:"1" max:"65535"`
    Tag  *string `descr:"Tag" pattern:"^v[0-9]+\\.[0-9]+\\.[0-9]+$"`
}
```

```bash
# No flags provided -- both nil, no validation errors:
$ go run .

# Values provided -- validation runs:
$ go run . --port 0
# Error: value 0 for param 'port' is below min 1
```

## Custom Validators

Use `SetCustomValidatorT` in `InitFuncCtx` for validation logic beyond what tags offer.

```go
package main

import (
    "fmt"
    "net"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    Host string `descr:"Server hostname"`
    Port int    `descr:"Server port"`
    CIDR string `descr:"Allowed CIDR range" optional:"true"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "server",
        Short: "Server with custom validation",
        InitFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
            portParam := boa.GetParamT(ctx, &p.Port)
            portParam.SetCustomValidatorT(func(port int) error {
                if port < 1024 && port != 80 && port != 443 {
                    return fmt.Errorf("non-standard privileged port %d (use 80, 443, or >= 1024)", port)
                }
                return nil
            })

            cidrParam := boa.GetParamT(ctx, &p.CIDR)
            cidrParam.SetCustomValidatorT(func(cidr string) error {
                if cidr == "" {
                    return nil
                }
                _, _, err := net.ParseCIDR(cidr)
                if err != nil {
                    return fmt.Errorf("invalid CIDR: %w", err)
                }
                return nil
            })

            return nil
        },
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Listening on %s:%d\n", p.Host, p.Port)
            if p.CIDR != "" {
                fmt.Printf("Allowed CIDR: %s\n", p.CIDR)
            }
        },
    }.Run()
}
```

```bash
$ go run . --host 0.0.0.0 --port 8080
Listening on 0.0.0.0:8080

$ go run . --host 0.0.0.0 --port 22
# Error: non-standard privileged port 22 (use 80, 443, or >= 1024)

$ go run . --host 0.0.0.0 --port 443 --cidr not-a-cidr
# Error: invalid CIDR: invalid CIDR address: not-a-cidr
```

## Conditional Required Fields

Make fields required only when certain conditions are met. The field must be `optional:"true"` in the struct tag, then use `SetRequiredFn` to add the condition.

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    Mode     string `descr:"Input mode" alts:"file,http,stdin" default:"stdin"`
    FilePath string `descr:"File path" optional:"true"`
    URL      string `descr:"HTTP URL" optional:"true"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "ingest",
        Short: "Ingest data from various sources",
        InitFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
            ctx.GetParam(&p.FilePath).SetRequiredFn(func() bool {
                return p.Mode == "file"
            })
            ctx.GetParam(&p.URL).SetRequiredFn(func() bool {
                return p.Mode == "http"
            })
            return nil
        },
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            switch p.Mode {
            case "file":
                fmt.Printf("Reading from file: %s\n", p.FilePath)
            case "http":
                fmt.Printf("Fetching from URL: %s\n", p.URL)
            case "stdin":
                fmt.Println("Reading from stdin...")
            }
        },
    }.Run()
}
```

```bash
$ go run . --mode file --file-path data.csv
Reading from file: data.csv

$ go run . --mode file
# Error: required flag "file-path" not set (required because mode=file)

$ go run . --mode http --url https://api.example.com/data
Fetching from URL: https://api.example.com/data

$ go run . --mode stdin
Reading from stdin...

$ go run .
Reading from stdin...
```

## Conditional Visibility

Hide parameters entirely when they are not relevant:

```go
type Params struct {
    Debug     bool   `descr:"Debug mode" optional:"true"`
    DebugPort int    `descr:"Debug port" optional:"true" default:"6060"`
    TraceFile string `descr:"Trace output file" optional:"true"`
}

boa.CmdT[Params]{
    Use: "server",
    InitFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
        // DebugPort and TraceFile only visible when Debug is true
        ctx.GetParam(&p.DebugPort).SetIsEnabledFn(func() bool {
            return p.Debug
        })
        ctx.GetParam(&p.TraceFile).SetIsEnabledFn(func() bool {
            return p.Debug
        })
        return nil
    },
    RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
        // ...
    },
}
```

## Shell Completion with Alternatives

### Static Alternatives

The `alts` tag provides shell completion suggestions automatically:

```go
type Params struct {
    Region string `descr:"AWS region" alts:"us-east-1,us-west-2,eu-west-1,ap-southeast-1"`
    Env    string `descr:"Environment" alts:"dev,staging,prod" default:"dev"`
}
```

With `strict:"true"` (the default when `alts` is set), invalid values are rejected. Use `strict:"false"` to provide suggestions without enforcing them.

### Dynamic Alternatives (AlternativesFunc)

For completions that depend on runtime state, use `SetAlternativesFunc`:

```go
package main

import (
    "fmt"
    "os"
    "path/filepath"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    Config string `descr:"Config file to use"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "app",
        Short: "App with dynamic completion",
        InitFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
            ctx.GetParam(&p.Config).SetAlternativesFunc(
                func(cmd *cobra.Command, args []string, toComplete string) []string {
                    // List JSON files in the current directory
                    matches, _ := filepath.Glob("*.json")
                    return matches
                },
            )
            ctx.GetParam(&p.Config).SetStrictAlts(false) // suggestions only
            return nil
        },
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Using config: %s\n", p.Config)
        },
    }.Run()
}
```

### ValidArgsFunc for Positional Arguments

For positional argument completion:

```go
boa.CmdT[Params]{
    Use: "deploy",
    ValidArgsFunc: func(p *Params, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
        if len(args) == 0 {
            return []string{"web", "api", "worker"}, cobra.ShellCompDirectiveNoFileComp
        }
        return nil, cobra.ShellCompDirectiveDefault
    },
    RunFunc: func(p *Params, cmd *cobra.Command, args []string) { ... },
}
```

## Interface-Based Hooks

Instead of using function fields on the command, implement interfaces on your config struct. This keeps configuration logic co-located with the struct definition.

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type ServerConfig struct {
    Host     string `descr:"Server host"`
    Port     int    `descr:"Server port"`
    LogLevel string `descr:"Log level" optional:"true"`
}

// InitCtx runs during initialization -- configure defaults and validation
func (c *ServerConfig) InitCtx(ctx *boa.HookContext) error {
    ctx.GetParam(&c.Host).SetDefault(boa.Default("localhost"))

    portParam := boa.GetParamT(ctx, &c.Port)
    portParam.SetDefaultT(8080)
    portParam.SetCustomValidatorT(func(port int) error {
        if port < 1 || port > 65535 {
            return fmt.Errorf("port must be between 1 and 65535")
        }
        return nil
    })

    logParam := ctx.GetParam(&c.LogLevel)
    logParam.SetDefault(boa.Default("info"))
    logParam.SetAlternatives([]string{"debug", "info", "warn", "error"})
    logParam.SetStrictAlts(true)

    return nil
}

// PreExecute runs after validation, before the command runs
func (c *ServerConfig) PreExecute() error {
    fmt.Printf("[pre-execute] Will start server on %s:%d\n", c.Host, c.Port)
    return nil
}

func main() {
    boa.CmdT[ServerConfig]{
        Use:   "server",
        Short: "Server with interface hooks",
        RunFunc: func(p *ServerConfig, cmd *cobra.Command, args []string) {
            fmt.Printf("Server running on %s:%d (log=%s)\n", p.Host, p.Port, p.LogLevel)
        },
    }.Run()
}
```

```bash
$ go run .
[pre-execute] Will start server on localhost:8080
Server running on localhost:8080 (log=info)

$ go run . --port 3000 --log-level debug
[pre-execute] Will start server on localhost:3000
Server running on localhost:3000 (log=debug)
```

Available interfaces:

| Interface | Method | When it runs |
|-----------|--------|-------------|
| `CfgStructInit` | `Init() error` | During initialization |
| `CfgStructInitCtx` | `InitCtx(ctx *HookContext) error` | During initialization (with context) |
| `CfgStructPostCreate` | `PostCreate() error` | After cobra flags created |
| `CfgStructPostCreateCtx` | `PostCreateCtx(ctx *HookContext) error` | After cobra flags created (with context) |
| `CfgStructPreValidate` | `PreValidate() error` | After parsing, before validation |
| `CfgStructPreValidateCtx` | `PreValidateCtx(ctx *HookContext) error` | After parsing, before validation (with context) |
| `CfgStructPreExecute` | `PreExecute() error` | After validation, before run |
| `CfgStructPreExecuteCtx` | `PreExecuteCtx(ctx *HookContext) error` | After validation, before run (with context) |

## Testing Commands

### Basic Test with RunArgsE

```go
func TestMyCommand(t *testing.T) {
    type Params struct {
        Name string `descr:"Name"`
        Port int    `descr:"Port" default:"8080"`
    }

    err := boa.CmdT[Params]{
        Use: "test",
        RunFuncE: func(p *Params, cmd *cobra.Command, args []string) error {
            if p.Name != "alice" {
                return fmt.Errorf("expected alice, got %s", p.Name)
            }
            if p.Port != 9090 {
                return fmt.Errorf("expected 9090, got %d", p.Port)
            }
            return nil
        },
    }.RunArgsE([]string{"--name", "alice", "--port", "9090"})

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}
```

### Validation-Only Test

Test that validation catches bad input without running the command:

```go
func TestValidation(t *testing.T) {
    type Params struct {
        Port int `descr:"Port" min:"1" max:"65535"`
    }

    err := boa.CmdT[Params]{
        Use:     "test",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
        RawArgs: []string{"--port", "0"},
    }.Validate()

    if err == nil {
        t.Fatal("expected validation error for port=0")
    }
}
```

### Testing with ToCobraE

Get the underlying cobra command for advanced test scenarios:

```go
func TestWithCobra(t *testing.T) {
    type Params struct {
        Name string
    }

    cobraCmd, err := boa.CmdT[Params]{
        Use: "test",
        RunFuncE: func(p *Params, cmd *cobra.Command, args []string) error {
            return nil
        },
    }.ToCobraE()
    if err != nil {
        t.Fatalf("setup failed: %v", err)
    }

    cobraCmd.SetArgs([]string{"--name", "test"})
    if err := cobraCmd.Execute(); err != nil {
        t.Fatalf("execution failed: %v", err)
    }
}
```
