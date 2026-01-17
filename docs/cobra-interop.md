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
boaCmd := boa.NewCmdT[Params]("myapp").
    WithShort("My application").
    WithRunFunc(func(params *Params) {
        // ...
    })

// Get the underlying *cobra.Command
cobraCmd := boaCmd.ToCobra()

// Now you can use any Cobra API
cobraCmd.SetHelpTemplate("Custom help template...")
cobraCmd.SetUsageFunc(customUsageFunc)
```

## Cobra Access in Run Functions

The `*cobra.Command` is available in run functions and lifecycle hooks:

=== "Direct API"

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

=== "Builder API"

    ```go
    boa.NewCmdT[Params]("myapp").
        WithRunFunc3(func(params *Params, cmd *cobra.Command, args []string) {
            // cmd is the *cobra.Command
            fmt.Println("Command name:", cmd.Name())
            fmt.Println("Positional args:", args)
        }).
        Run()
    ```

## Mixing BOA and Cobra Commands

The `SubCmds` field accepts `[]*cobra.Command`, meaning you can freely mix BOA commands with pure Cobra commands:

### Adding Cobra Commands to a BOA Parent

=== "Direct API"

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
            boa.NewCmdT[ServeParams]("serve").
                WithShort("Start the server").
                WithRunFunc(func(p *ServeParams) { /* ... */ }).
                ToCobra(),  // BOA command converted to Cobra
        },
    }.Run()
    ```

=== "Builder API"

    ```go
    // Existing Cobra command
    legacyCmd := &cobra.Command{
        Use:   "legacy",
        Short: "A legacy Cobra command",
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Println("Running legacy command")
        },
    }

    // Using WithCobraSubCmds for raw Cobra commands
    boa.NewCmdT[RootParams]("myapp").
        WithShort("Application with mixed commands").
        WithCobraSubCmds(legacyCmd).  // Add Cobra command directly
        WithSubCmds(
            boa.NewCmdT[ServeParams]("serve").
                WithShort("Start the server").
                WithRunFunc(func(p *ServeParams) { /* ... */ }),
        ).
        Run()
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
    boa.NewCmdT[ServeParams]("serve").
        WithShort("Start the server").
        WithRunFunc(func(p *ServeParams) { /* ... */ }).
        ToCobra(),

    boa.NewCmdT[MigrateParams]("migrate").
        WithShort("Run migrations").
        WithRunFunc(func(p *MigrateParams) { /* ... */ }).
        ToCobra(),
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
        boa.NewCmdT[ConfigParams]("config").
            WithShort("Manage configuration").
            WithRunFunc(func(p *ConfigParams) {
                // Type-safe access to parameters
            }).
            ToCobra(),
    )

    rootCmd.Execute()
}
```

### Step 3: Eventually Migrate the Root

```go
func main() {
    // Root is now BOA, subcommands can be either
    boa.NewCmdT[RootParams]("myapp").
        WithCobraSubCmds(serveCmd, migrateCmd). // Legacy Cobra commands
        WithSubCmds(
            boa.NewCmdT[ConfigParams]("config").
                WithRunFunc(func(p *ConfigParams) { /* ... */ }),
        ).
        Run()
}
```

## Using Cobra's PositionalArgs

BOA supports Cobra's positional argument validation:

=== "Direct API"

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

=== "Builder API"

    ```go
    boa.NewCmdT[Params]("greet").
        WithUse("greet [names...]").
        WithArgs(cobra.MinimumNArgs(1)).
        WithRunFunc3(func(params *Params, cmd *cobra.Command, args []string) {
            for _, name := range args {
                fmt.Printf("Hello, %s!\n", name)
            }
        }).
        Run()
    ```

## Using Cobra's Command Groups

Organize subcommands with Cobra's grouping feature:

=== "Direct API"

    ```go
    boa.CmdT[NoParams]{
        Use:   "myapp",
        Groups: []*cobra.Group{
            {ID: "core", Title: "Core Commands:"},
            {ID: "util", Title: "Utility Commands:"},
        },
        SubCmds: boa.SubCmds(
            boa.NewCmdT[ServeParams]("serve").
                WithGroupID("core").
                WithRunFunc(func(p *ServeParams) { /* ... */ }),
            boa.NewCmdT[StatusParams]("status").
                WithGroupID("util").
                WithRunFunc(func(p *StatusParams) { /* ... */ }),
        ),
    }.Run()
    ```

=== "Builder API"

    ```go
    boa.NewCmdT[NoParams]("myapp").
        WithGroups(
            &cobra.Group{ID: "core", Title: "Core Commands:"},
            &cobra.Group{ID: "util", Title: "Utility Commands:"},
        ).
        WithSubCmds(
            boa.NewCmdT[ServeParams]("serve").
                WithGroupID("core").
                WithRunFunc(func(p *ServeParams) { /* ... */ }),
            boa.NewCmdT[StatusParams]("status").
                WithGroupID("util").
                WithRunFunc(func(p *StatusParams) { /* ... */ }),
        ).
        Run()
    ```

## Cobra Ecosystem Compatibility

Since BOA commands convert to standard `*cobra.Command`, you can use the entire Cobra ecosystem:

### Shell Completion

Cobra's built-in completion generators work with BOA:

```go
cmd := boa.NewCmdT[Params]("myapp").
    WithSubCmds(/* ... */).
    ToCobra()

// Add Cobra's completion command
cmd.AddCommand(completionCmd) // Your standard Cobra completion command
```

### Documentation Generation

Use Cobra's doc generation packages:

```go
import "github.com/spf13/cobra/doc"

cmd := boa.NewCmdT[Params]("myapp").ToCobra()

// Generate markdown docs
doc.GenMarkdownTree(cmd, "./docs")

// Generate man pages
doc.GenManTree(cmd, &doc.GenManHeader{Title: "MYAPP"}, "./man")
```

### Interactive Help with Bubbletea

Libraries like [elewis787/boa](https://github.com/elewis787/boa) add interactive TUI help to Cobra (yes, we accidentally picked the same name - theirs adds Bubbletea-powered help to Cobra, ours adds declarative parameter handling):

```go
import eboa "github.com/elewis787/boa"

cmd := boa.NewCmdT[Params]("myapp").ToCobra()

// Add interactive help powered by Bubbletea
cmd.SetUsageFunc(eboa.UsageFunc)
cmd.SetHelpFunc(eboa.HelpFunc)
```

## Summary

| Task | Method |
|------|--------|
| Convert BOA → Cobra | `boaCmd.ToCobra()` |
| Add Cobra subcommands | `WithCobraSubCmds(cmd)` or set `SubCmds` field |
| Add BOA subcommands | `WithSubCmds(cmd)` |
| Access `*cobra.Command` in run | Use `WithRunFunc3` or `RunFunc` with full signature |
| Use Cobra arg validation | Set `Args` field or use `WithArgs()` |
| Use Cobra groups | Set `Groups` field or use `WithGroups()` |
| Use Cobra ecosystem libs | Call `ToCobra()` then use standard Cobra APIs |
