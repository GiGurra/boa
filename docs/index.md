# BOA

**Declarative Go CLI Framework built on Cobra**

[![CI Status](https://github.com/GiGurra/boa/actions/workflows/ci.yml/badge.svg)](https://github.com/GiGurra/boa/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/GiGurra/boa)](https://goreportcard.com/report/github.com/GiGurra/boa)

BOA adds a declarative layer on top of [cobra](https://github.com/spf13/cobra), making CLI creation as simple as defining a struct.

## Features

- **Declarative parameters** - Define CLI flags as struct fields with tags
- **Automatic flag generation** - Field names become kebab-case flags
- **Environment variable binding** - Auto-generated or custom env var names
- **Built-in validation** - Required fields, alternatives, custom validators
- **Cobra compatible** - Access underlying Cobra commands when needed

## Quick Example

=== "Direct API"

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

=== "Builder API"

    ```go
    package main

    import (
        "fmt"
        "github.com/GiGurra/boa/pkg/boa"
    )

    type Params struct {
        Name    string `descr:"User name"`
        Port    int    `descr:"Port number" default:"8080" optional:"true"`
        Verbose bool   `short:"v" optional:"true"`
    }

    func main() {
        boa.NewCmdT[Params]("myapp").
            WithShort("A simple CLI application").
            WithRunFunc(func(params *Params) {
                fmt.Printf("Hello %s on port %d\n", params.Name, params.Port)
            }).
            Run()
    }
    ```

This generates:

```
A simple CLI application

Usage:
  myapp [flags]

Flags:
  -n, --name string   User name (env: NAME, required)
  -p, --port int      Port number (env: PORT) (default 8080)
  -v, --verbose       (env: VERBOSE)
  -h, --help          help for myapp
```

## Installation

```bash
go get github.com/GiGurra/boa@latest
```

## Next Steps

- [Quickstart](quickstart.md) - Get a CLI running in 60 seconds with Claude Code
- [Getting Started](getting-started.md) - Installation and basic usage
- [Struct Tags](struct-tags.md) - Complete reference for all struct tags
- [Lifecycle Hooks](hooks.md) - Customize behavior at different stages
- [Cobra Interoperability](cobra-interop.md) - Access Cobra primitives and migrate incrementally
- [Migration Guide](migration.md) - Migrating from deprecated wrapper types
