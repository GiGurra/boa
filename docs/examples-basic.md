# Basic Examples

Practical, copy-paste-ready examples for common BOA CLI patterns.

## Minimal CLI Tool (Hello World)

The simplest possible BOA CLI tool. One required parameter, one action.

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    Name string `descr:"Your name"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "hello",
        Short: "Say hello",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Hello, %s!\n", p.Name)
        },
    }.Run()
}
```

```bash
$ go run . --name World
Hello, World!

$ go run .
# Error: required flag "name" not set
```

## Required and Optional Flags

By default, all plain-type fields are **required**. Mark fields optional with `optional:"true"`.

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    Host    string `descr:"Server hostname"`
    Port    int    `descr:"Server port" optional:"true"`
    Verbose bool   `descr:"Enable verbose logging" optional:"true"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "server",
        Short: "Start a server",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Starting server on %s:%d (verbose=%v)\n", p.Host, p.Port, p.Verbose)
        },
    }.Run()
}
```

```bash
$ go run . --host localhost
Starting server on localhost:0 (verbose=false)

$ go run . --host localhost --port 8080 --verbose
Starting server on localhost:8080 (verbose=true)

$ go run .
# Error: required flag "host" not set
```

## Pointer Fields for Optional Parameters

Use pointer types (`*string`, `*int`, `*bool`) when you need to distinguish "not provided" from "zero value". Pointer fields are always optional by default.

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    Name    string  `descr:"Your name"`
    Retries *int    `descr:"Retry count"`
    Output  *string `descr:"Output file"`
    Force   *bool   `descr:"Force overwrite"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "app",
        Short: "Demo pointer fields",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Name: %s\n", p.Name)

            if p.Retries != nil {
                fmt.Printf("Retries: %d\n", *p.Retries)
            } else {
                fmt.Println("Retries: not set (nil)")
            }

            if p.Output != nil {
                fmt.Printf("Output: %s\n", *p.Output)
            } else {
                fmt.Println("Output: not set (nil)")
            }

            if p.Force != nil {
                fmt.Printf("Force: %v\n", *p.Force)
            } else {
                fmt.Println("Force: not set (nil)")
            }
        },
    }.Run()
}
```

```bash
$ go run . --name alice
Name: alice
Retries: not set (nil)
Output: not set (nil)
Force: not set (nil)

$ go run . --name alice --retries 3 --output results.json --force
Name: alice
Retries: 3
Output: results.json
Force: true
```

Use `required:"true"` to override the default-optional behavior of pointer fields:

```go
type Params struct {
    Token *string `descr:"API token" required:"true"`
}
```

## Boolean Flags

Boolean fields default to `false` when optional. On the CLI, `--verbose` (without a value) sets the flag to `true`.

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    DryRun  bool `descr:"Simulate without making changes" optional:"true"`
    Verbose bool `descr:"Enable verbose output" short:"v" optional:"true"`
    Force   bool `descr:"Skip confirmation prompts" short:"f" optional:"true"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "deploy",
        Short: "Deploy the application",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            if p.DryRun {
                fmt.Println("[DRY RUN] Would deploy...")
            } else {
                fmt.Println("Deploying...")
            }
            if p.Verbose {
                fmt.Println("  verbose output enabled")
            }
            if p.Force {
                fmt.Println("  skipping confirmations")
            }
        },
    }.Run()
}
```

```bash
$ go run . --dry-run --verbose
[DRY RUN] Would deploy...
  verbose output enabled

$ go run . -v -f
Deploying...
  verbose output enabled
  skipping confirmations
```

## Positional Arguments

Use `positional:"true"` to accept arguments by position instead of flags. Positional arguments are matched in struct field order.

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    Source string `positional:"true" descr:"Source file"`
    Dest   string `positional:"true" descr:"Destination file"`
    Force  bool   `descr:"Overwrite existing" optional:"true"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "copy",
        Short: "Copy a file",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Copying %s -> %s (force=%v)\n", p.Source, p.Dest, p.Force)
        },
    }.Run()
}
```

```bash
$ go run . input.txt output.txt
Copying input.txt -> output.txt (force=false)

$ go run . input.txt output.txt --force
Copying input.txt -> output.txt (force=true)
```

Help output shows positional arguments in the usage line:

```
Usage:
  copy <source> <dest> [flags]
```

### Optional Positional Arguments

```go
type Params struct {
    File   string `positional:"true" descr:"Input file"`
    Output string `positional:"true" optional:"true" default:"stdout" descr:"Output file"`
}
```

```bash
$ go run . data.csv
# Output defaults to "stdout"

$ go run . data.csv results.json
# Output is "results.json"
```

### Slice Positional Arguments

A slice positional argument consumes all remaining positional arguments:

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    Names []string `positional:"true" descr:"Names to greet"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "greet",
        Short: "Greet people",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            for _, name := range p.Names {
                fmt.Printf("Hello, %s!\n", name)
            }
        },
    }.Run()
}
```

```bash
$ go run . alice bob carol
Hello, alice!
Hello, bob!
Hello, carol!
```

## Default Values

Set defaults with the `default` struct tag. Fields with defaults are automatically optional since they always have a value.

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    Host     string   `descr:"Server host" default:"localhost"`
    Port     int      `descr:"Server port" default:"8080"`
    LogLevel string   `descr:"Log level" default:"info"`
    Tags     []string `descr:"Tags" default:"[web,api]"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "server",
        Short: "Start the server",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Host: %s\nPort: %d\nLog: %s\nTags: %v\n",
                p.Host, p.Port, p.LogLevel, p.Tags)
        },
    }.Run()
}
```

```bash
$ go run .
Host: localhost
Port: 8080
Log: info
Tags: [web api]

$ go run . --port 3000 --log-level debug
Host: localhost
Port: 3000
Log: debug
Tags: [web api]
```

### Programmatic Defaults

Set defaults dynamically in an `InitFuncCtx` hook:

```go
boa.CmdT[Params]{
    Use: "server",
    InitFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command) error {
        ctx.GetParam(&p.Port).SetDefault(boa.Default(8080))
        ctx.GetParam(&p.Host).SetDefault(boa.Default("localhost"))
        return nil
    },
    RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
        // ...
    },
}.Run()
```

## Short Flags

By default, BOA auto-assigns short flags from the first letter of the flag name (skipping `-h` which is reserved for help). Override with the `short` tag.

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    Name    string `descr:"Your name" short:"n"`
    Output  string `descr:"Output file" short:"o"`
    Verbose bool   `descr:"Verbose output" short:"v" optional:"true"`
    Count   int    `descr:"Repeat count" short:"c" default:"1"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "greet",
        Short: "Greet someone",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            for i := 0; i < p.Count; i++ {
                fmt.Printf("Hello, %s!\n", p.Name)
            }
        },
    }.Run()
}
```

```bash
$ go run . -n World -c 3 -v
Hello, World!
Hello, World!
Hello, World!
```

## Environment Variables

Bind parameters to environment variables with the `env` tag. Environment variables take precedence over defaults but are overridden by CLI flags.

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    Host     string `descr:"Server host" env:"APP_HOST" default:"localhost"`
    Port     int    `descr:"Server port" env:"APP_PORT" default:"8080"`
    APIKey   string `descr:"API key" env:"APP_API_KEY"`
    LogLevel string `descr:"Log level" env:"APP_LOG_LEVEL" default:"info"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "server",
        Short: "Start the server",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Host: %s\nPort: %d\nAPI Key: %s\nLog: %s\n",
                p.Host, p.Port, p.APIKey, p.LogLevel)
        },
    }.Run()
}
```

```bash
$ APP_API_KEY=secret123 APP_PORT=3000 go run .
Host: localhost
Port: 3000
API Key: secret123
Log: info

# CLI overrides env var:
$ APP_PORT=3000 go run . --port 9090
Host: localhost
Port: 9090
API Key:
...
```

Environment variables are shown in help output:

```
Flags:
      --api-key string      API key (env: APP_API_KEY) (required)
      --host string         Server host (env: APP_HOST) (default "localhost")
      --log-level string    Log level (env: APP_LOG_LEVEL) (default "info")
      --port int            Server port (env: APP_PORT) (default 8080)
```

### Auto-Deriving Environment Variables

Instead of tagging each field, use `ParamEnricherEnv` to auto-derive env var names from flag names:

```go
boa.CmdT[Params]{
    Use: "server",
    ParamEnrich: boa.ParamEnricherCombine(
        boa.ParamEnricherDefault,
        boa.ParamEnricherEnv,
    ),
    RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
        // ServerHost -> --server-host -> SERVER_HOST
        // MaxRetries -> --max-retries -> MAX_RETRIES
    },
}.Run()
```

### Env Var Prefix

Add a prefix to all auto-generated env var names:

```go
boa.CmdT[Params]{
    Use: "server",
    ParamEnrich: boa.ParamEnricherCombine(
        boa.ParamEnricherDefault,
        boa.ParamEnricherEnv,
        boa.ParamEnricherEnvPrefix("MYAPP"),
    ),
    RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
        // Port -> --port -> PORT -> MYAPP_PORT
    },
}.Run()
```

## Slice Parameters

Slice fields accept comma-separated values or repeated flags.

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    Numbers []int    `descr:"List of numbers"`
    Tags    []string `descr:"Tags to apply" default:"[web,api]"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "demo",
        Short: "Demo slice flags",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Numbers: %v\nTags: %v\n", p.Numbers, p.Tags)
        },
    }.Run()
}
```

```bash
$ go run . --numbers 1,2,3
Numbers: [1 2 3]
Tags: [web api]

$ go run . --numbers 1,2,3 --tags prod,backend
Numbers: [1 2 3]
Tags: [prod backend]
```

## Enum Values with Alternatives

Restrict a parameter to specific values with the `alts` tag. By default, alternatives are strict (validation enforced).

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    Format   string `descr:"Output format" alts:"json,yaml,table" default:"table"`
    LogLevel string `descr:"Log level" alts:"debug,info,warn,error" default:"info"`
    Color    string `descr:"Color theme" alts:"light,dark,auto" strict:"false" default:"auto"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "export",
        Short: "Export data",
        RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Format: %s, Log: %s, Color: %s\n", p.Format, p.LogLevel, p.Color)
        },
    }.Run()
}
```

```bash
$ go run . --format json
Format: json, Log: info, Color: auto

$ go run . --format xml
# Error: invalid value "xml" for flag --format: must be one of [json yaml table]

# Color accepts any value since strict:"false":
$ go run . --color custom-theme
Format: table, Log: info, Color: custom-theme
```

Alternatives also power shell completion -- pressing Tab suggests valid values.

## Checking If a Value Was Provided

Use `RunFuncCtx` and `ctx.HasValue()` to check whether a parameter was explicitly set (via CLI, env var, or default) vs left unset.

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    Host string `descr:"Server host" default:"localhost"`
    Port *int   `descr:"Server port"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "server",
        Short: "Start server",
        RunFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Host: %s\n", p.Host)

            if p.Port != nil {
                fmt.Printf("Port: %d\n", *p.Port)
            } else {
                fmt.Println("Port: auto-assign")
            }

            if ctx.HasValue(&p.Host) {
                fmt.Println("(host was explicitly configured)")
            }
        },
    }.Run()
}
```

```bash
$ go run .
Host: localhost
Port: auto-assign
(host was explicitly configured)

$ go run . --port 8080
Host: localhost
Port: 8080
(host was explicitly configured)
```
