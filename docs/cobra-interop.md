# Cobra Interoperability

BOA is built on top of [Cobra](https://github.com/spf13/cobra) and provides full access to Cobra's primitives. This design allows you to:

- Access the underlying `*cobra.Command` when you need low-level control
- Mix BOA commands with existing Cobra commands in the same command tree
- Migrate existing Cobra applications to BOA incrementally, one command at a time
- Use existing Cobra plugins and ecosystem libraries

## Exposed Cobra Types

BOA's `CmdT` struct directly exposes Cobra types in its API - no wrapping or abstraction:

```go
type CmdT[Struct any] struct {
    // ...
    GroupID string              // Cobra's group ID for help categorization
    Groups  []*cobra.Group      // Cobra's Group type directly
    Args    cobra.PositionalArgs // Cobra's positional args validation
    SubCmds []*cobra.Command    // Cobra's Command type directly
    // ...
}
```

This means you can use Cobra types directly when configuring BOA commands:

```go
boa.CmdT[Params]{
    Use:     "myapp",
    Groups:  []*cobra.Group{{ID: "admin", Title: "Admin Commands:"}},
    Args:    cobra.ExactArgs(2),
    SubCmds: []*cobra.Command{existingCobraCmd, anotherCobraCmd},
    // ...
}
```

## Accessing the Underlying Cobra Command

BOA commands can be converted to Cobra commands using `ToCobra()`:

```go
boaCmd := boa.CmdT[Params]{
    Use:   "myapp",
    Short: "My application",
    RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
        // ...
    },
}

// Get the underlying *cobra.Command
cobraCmd := boaCmd.ToCobra()

// Now you can use any Cobra API
cobraCmd.SetHelpTemplate("Custom help template...")
cobraCmd.SetUsageFunc(customUsageFunc)
```

## Cobra Access in Run Functions

The `*cobra.Command` is available in run functions and lifecycle hooks:

```go
boa.CmdT[Params]{
    Use: "myapp",
    RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
        // cmd is the *cobra.Command
        fmt.Println("Command name:", cmd.Name())
        fmt.Println("Positional args:", args)

        // Access Cobra features
        cmd.Println("Using Cobra's output methods")
    },
}.Run()
```

## Mixing BOA and Cobra Commands

The `SubCmds` field accepts `[]*cobra.Command`, meaning you can freely mix BOA commands with pure Cobra commands:

### Adding Cobra Commands to a BOA Parent

```go
// Existing Cobra command (from your codebase or a library)
legacyCmd := &cobra.Command{
    Use:   "legacy",
    Short: "A legacy Cobra command",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Println("Running legacy command")
    },
}

// BOA root command with mixed subcommands
boa.CmdT[RootParams]{
    Use:   "myapp",
    Short: "Application with mixed commands",
    SubCmds: []*cobra.Command{
        legacyCmd,  // Pure Cobra command
        boa.CmdT[ServeParams]{
            Use:   "serve",
            Short: "Start the server",
            RunFunc: func(p *ServeParams, cmd *cobra.Command, args []string) { /* ... */ },
        }.ToCobra(),  // BOA command converted to Cobra
    },
}.Run()
```

### Adding BOA Commands to a Cobra Parent

```go
// Existing Cobra root command
rootCmd := &cobra.Command{
    Use:   "myapp",
    Short: "My application",
}

// Add BOA subcommands to Cobra parent
rootCmd.AddCommand(
    boa.CmdT[ServeParams]{
        Use:   "serve",
        Short: "Start the server",
        RunFunc: func(p *ServeParams, cmd *cobra.Command, args []string) { /* ... */ },
    }.ToCobra(),

    boa.CmdT[MigrateParams]{
        Use:   "migrate",
        Short: "Run migrations",
        RunFunc: func(p *MigrateParams, cmd *cobra.Command, args []string) { /* ... */ },
    }.ToCobra(),
)

rootCmd.Execute()
```

## Incremental Migration Strategy

BOA's Cobra interoperability enables gradual migration of existing Cobra applications. You can migrate one command at a time without disrupting the entire codebase.

### Step 1: Start with Your Existing Cobra Tree

```go
// Your existing Cobra command structure
func main() {
    rootCmd := &cobra.Command{Use: "myapp"}

    rootCmd.AddCommand(serveCmd)   // Cobra command
    rootCmd.AddCommand(migrateCmd) // Cobra command
    rootCmd.AddCommand(configCmd)  // Cobra command

    rootCmd.Execute()
}
```

### Step 2: Migrate One Command to BOA

```go
func main() {
    rootCmd := &cobra.Command{Use: "myapp"}

    rootCmd.AddCommand(serveCmd)   // Still Cobra
    rootCmd.AddCommand(migrateCmd) // Still Cobra

    // Migrated to BOA - now with type-safe params!
    rootCmd.AddCommand(
        boa.CmdT[ConfigParams]{
            Use:   "config",
            Short: "Manage configuration",
            RunFunc: func(p *ConfigParams, cmd *cobra.Command, args []string) {
                // Type-safe access to parameters
            },
        }.ToCobra(),
    )

    rootCmd.Execute()
}
```

### Step 3: Eventually Migrate the Root

```go
func main() {
    // Root is now BOA, subcommands can be either
    boa.CmdT[RootParams]{
        Use: "myapp",
        SubCmds: []*cobra.Command{
            serveCmd,   // Legacy Cobra commands
            migrateCmd,
            boa.CmdT[ConfigParams]{
                Use: "config",
                RunFunc: func(p *ConfigParams, cmd *cobra.Command, args []string) { /* ... */ },
            }.ToCobra(),
        },
    }.Run()
}
```

## Using Cobra's PositionalArgs

BOA supports Cobra's positional argument validation:

```go
boa.CmdT[Params]{
    Use:  "greet [names...]",
    Args: cobra.MinimumNArgs(1), // Cobra's validation
    RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
        for _, name := range args {
            fmt.Printf("Hello, %s!\n", name)
        }
    },
}.Run()
```

## Using Cobra's Command Groups

Organize subcommands with Cobra's grouping feature:

```go
boa.CmdT[boa.NoParams]{
    Use:   "myapp",
    Groups: []*cobra.Group{
        {ID: "core", Title: "Core Commands:"},
        {ID: "util", Title: "Utility Commands:"},
    },
    SubCmds: boa.SubCmds(
        boa.CmdT[ServeParams]{
            Use:     "serve",
            GroupID: "core",
            RunFunc: func(p *ServeParams, cmd *cobra.Command, args []string) { /* ... */ },
        },
        boa.CmdT[StatusParams]{
            Use:     "status",
            GroupID: "util",
            RunFunc: func(p *StatusParams, cmd *cobra.Command, args []string) { /* ... */ },
        },
    ),
}.Run()
```

## Cobra Ecosystem Compatibility

Since BOA commands convert to standard `*cobra.Command`, you can use the entire Cobra ecosystem:

### Shell Completion

Cobra's built-in completion generators work with BOA:

```go
cmd := boa.CmdT[Params]{
    Use: "myapp",
    SubCmds: boa.SubCmds(/* ... */),
}.ToCobra()

// Add Cobra's completion command
cmd.AddCommand(completionCmd) // Your standard Cobra completion command
```

### Documentation Generation

Use Cobra's doc generation packages:

```go
import "github.com/spf13/cobra/doc"

cmd := boa.CmdT[Params]{Use: "myapp"}.ToCobra()

// Generate markdown docs
doc.GenMarkdownTree(cmd, "./docs")

// Generate man pages
doc.GenManTree(cmd, &doc.GenManHeader{Title: "MYAPP"}, "./man")
```

### Interactive Help with Bubbletea

Libraries like [elewis787/boa](https://github.com/elewis787/boa) add interactive TUI help to Cobra (yes, we accidentally picked the same name - theirs adds Bubbletea-powered help to Cobra, ours adds declarative parameter handling):

```go
import eboa "github.com/elewis787/boa"

boa.CmdT[Params]{
    Use: "myapp",
    PostCreateFunc: func(params *Params, cmd *cobra.Command) error {
        cmd.SetUsageFunc(eboa.UsageFunc)
        cmd.SetHelpFunc(eboa.HelpFunc)
        return nil
    },
}.Run()
```

## Summary

| Task | Method |
|------|--------|
| Convert BOA -> Cobra | `boaCmd.ToCobra()` |
| Add Cobra subcommands | Set `SubCmds` field with `[]*cobra.Command` |
| Add BOA subcommands | Use `boa.SubCmds()` helper or call `.ToCobra()` |
| Access `*cobra.Command` in run | Use `RunFunc` with full signature |
| Use Cobra arg validation | Set `Args` field |
| Use Cobra groups | Set `Groups` and `GroupID` fields |
| Use Cobra ecosystem libs | Call `ToCobra()` then use standard Cobra APIs |
