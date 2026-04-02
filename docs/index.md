# BOA

**Declarative Go CLI Framework built on Cobra**

[![CI Status](https://github.com/GiGurra/boa/actions/workflows/ci.yml/badge.svg)](https://github.com/GiGurra/boa/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/GiGurra/boa)](https://goreportcard.com/report/github.com/GiGurra/boa)

BOA adds a declarative layer on top of [cobra](https://github.com/spf13/cobra), making CLI creation as simple as defining a struct.

## Quick Example

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    Name    string `descr:"User name"`
    Port    int    `descr:"Port number" default:"8080" optional:"true"`
    Verbose bool   `short:"v" optional:"true"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "myapp",
        Short: "A simple CLI application",
        RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Hello %s on port %d\n", params.Name, params.Port)
        },
    }.Run()
}
```

This generates:

```
A simple CLI application

Usage:
  myapp [flags]

Flags:
  -n, --name string   User name (required)
  -p, --port int      Port number (default 8080)
  -v, --verbose
  -h, --help          help for myapp
```

## With Config File

Add a `configfile` tag and boa loads it automatically — CLI flags and env vars still take priority:

```go
type Params struct {
    ConfigFile string `configfile:"true" optional:"true" default:"config.json"`
    Host       string `descr:"Server host" default:"localhost"`
    Port       int    `descr:"Port number" default:"8080"`
    Labels     map[string]string `descr:"Key-value labels" optional:"true"`
    Matrix     [][]int           `boa:"configonly"` // config-file only, no CLI flag
}
```

```json
{"Host": "prod.example.com", "Port": 443, "Labels": {"env": "prod"}, "Matrix": [[1,2],[3,4]]}
```

```bash
myapp                                    # uses config.json values
myapp --host override.local              # CLI wins over config file
myapp --config-file staging.json         # different config file
HOST=ci.local myapp                      # env var wins over config file
```

## Installation

```bash
go get github.com/GiGurra/boa@latest
```

## Features

- **Declarative parameters** - Define CLI flags as struct fields with tags
- **Plain Go types** - No wrapper types; use `string`, `int`, `*string`, `map[string]string`, etc.
- **Automatic flag generation** - Field names become kebab-case flags (acronym-aware: `DBHost` → `--db-host`)
- **Struct composition** - Named struct fields auto-prefix children (`DB.Host` → `--db-host`), embedded fields stay flat
- **Environment variable binding** - Via struct tags or auto-generated with enrichers
- **Built-in validation** - Required fields, alternatives, custom validators
- **Config file support** - Automatic loading via `configfile` tag with value priority, substruct config files, and pluggable format registry
- **JSON fallback** - Complex types (nested slices, maps) parsed as JSON on CLI
- **Pointer fields** - `*string`, `*int` etc. for truly optional params (nil = not set)
- **Validation tags** - `min`/`max` for range checks, `pattern` for regex matching
- **Custom types** - `RegisterType[T]` for user-defined CLI parameter types
- **Viper-like config discovery** - Optional `boaviper` subpackage for auto-locating config files
- **Cobra compatible** - Access underlying Cobra commands when needed

## Next Steps

- [Getting Started](getting-started.md) - Installation, basic usage, and all parameter types
- [Struct Tags](struct-tags.md) - Complete reference for all struct tags
- [Validation](validation.md) - Required/optional, alternatives, conditional requirements
- [Lifecycle Hooks](hooks.md) - Customize behavior at different stages
- [Enrichers](enrichers.md) - Auto-derivation of flag names, env vars, and short flags
- [Error Handling](error-handling.md) - Run() vs RunE() and error propagation
- [Advanced](advanced.md) - Config files, JSON fallback, ParamT, testing
- [Global Config](global-config.md) - Init() and WithDefaultOptional()
- [Cobra Interoperability](cobra-interop.md) - Access Cobra primitives and migrate incrementally
- [Migration](migration.md) - Migrating from old BOA or Cobra
