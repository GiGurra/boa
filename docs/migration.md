# Migration from Cobra

If you're migrating an existing Cobra application to BOA, this guide will help.

## From Cobra to BOA

### Before (Pure Cobra)

```go
var port int
var host string

var rootCmd = &cobra.Command{
    Use:   "myapp",
    Short: "My application",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Printf("Host: %s, Port: %d\n", host, port)
    },
}

func init() {
    rootCmd.Flags().StringVarP(&host, "host", "H", "localhost", "Server hostname")
    rootCmd.Flags().IntVarP(&port, "port", "p", 8080, "Server port")
    rootCmd.MarkFlagRequired("host")
}

func main() {
    rootCmd.Execute()
}
```

### After (BOA)

```go
type Params struct {
    Host string `descr:"Server hostname" default:"localhost"`
    Port int    `descr:"Server port" default:"8080" optional:"true"`
}

func main() {
    boa.CmdT[Params]{
        Use:   "myapp",
        Short: "My application",
        RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
            fmt.Printf("Host: %s, Port: %d\n", params.Host, params.Port)
        },
    }.Run()
}
```

## Incremental Migration

You don't have to migrate everything at once. BOA commands produce standard `*cobra.Command` objects, so you can mix them freely:

```go
// Start: all Cobra
rootCmd.AddCommand(serveCmd, migrateCmd, configCmd)

// Migrate one at a time
rootCmd.AddCommand(
    serveCmd,   // Still Cobra
    migrateCmd, // Still Cobra
    boa.CmdT[ConfigParams]{
        Use: "config",
        RunFunc: func(p *ConfigParams, cmd *cobra.Command, args []string) { /* ... */ },
    }.ToCobra(), // Now BOA
)
```

See [Cobra Interoperability](cobra-interop.md) for the full incremental migration strategy.

## Why Migrate?

BOA provides:

- **Declarative parameters** - Define flags as struct fields, no manual registration
- **Automatic flag generation** - Field names become kebab-case flags automatically
- **Type safety** - Parameters are typed struct fields, not `interface{}`
- **Built-in validation** - Required fields, alternatives, custom validators
- **Environment variable binding** - Automatic or custom env var support
- **Cleaner code** - No scattered `init()` functions or global variables
