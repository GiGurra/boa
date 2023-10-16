# BOA

Boa is a compact CLI and environment variable parameter utility. It enhances and simplifies aspects of `github.com/spf13/cobra`, facilitating the creation of straightforward and declarative CLI interfaces.

The primary goal of Boa is to maintain a declarative approach. In its simplest form, you only need to define a struct with parameter fields, and Boa handles the rest.

## Features

* **Declarative Design**: Boa allows for fully declarative definition and validation.
* **Optional Values**: Boa supports true optional values and provides knowledge if a field was set. It also offers opt-in default values built into the type system.
    * A `boa.Required[string]`'s `.Value()` is type aware and returns a `string`.
    * A `boa.Optional[string]`'s `.Value()` is type aware and returns a `*string`.
* **Auto-Generated Properties**: Boa generates flag/param properties from field name, type, tags, and more.
    * For instance, `Foo boa.Required[string]` will generate:
        * flags `--foo` (and short version `-f` if it is not already taken)
        * `FOO` env var mapping
        * `[required] (env: FOO)` in the help text
        * You can supplement this with your own help text, custom generation logic, etc.
    * You can opt out of auto generation, override specific properties, and cherry-pick and/or add your own auto-generation logic.
* **Input Validation**: Boa validates all inputs before the `Run` function is invoked.
* **Config Flexibility**: Use explicit fields for config or tags as per your preference.
* **Cobra Compatibility**: Mix and match Boa with regular Cobra code as you see fit. Boa works with regular Cobra commands.

## Installation

To install Boa, use the following command:

`go get github.com/GiGurra/boa@v0.0.17`

## Usage

Refer to the code snippets provided below for minimum setup, sub-commands and tags, and sub-commands, tags and explicit fields.

### Minimum setup

```go
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

var params = struct {
	Foo  boa.Required[string]
	Bar  boa.Required[int]
	File boa.Required[string]
	Baz  boa.Required[string]
	FB   boa.Optional[string]
}{}

func main() {
	boa.Wrap{
		Use:    "hello-world",
		Short:  "a generic cli tool",
		Long:   `A generic cli tool that has a longer description. See the README.MD for more information`,
		Params: &params,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf(
				"Hello world with params: %s, %d, %s, %s, %v\n",
				params.Foo.Value(),  // string
				params.Bar.Value(),  // int
				params.File.Value(), // string
				params.Baz.Value(),  // string
				params.FB.Value(),   // *string
			)
		},
	}.ToApp()
}
```

Help output for the above:

```
A generic cli tool that has a longer description. See the README.MD for more information

Usage:
  hello-world [flags]

Flags:
  -b, --bar int        [required] (env: BAR)
      --baz string     [required] (env: BAZ)
      --f-b string     (env: F_B)
      --file string    [required] (env: FILE)
  -f, --foo string     [required] (env: FOO)
  -h, --help          help for hello-world
```

### Sub-commands and tags

Most customization is available through field tags:

```go
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

var params = struct {
	Foo  boa.Required[string] `descr:"a foo"`
	Bar  boa.Required[int]    `descr:"a bar" env:"BAR_X" default:"4"`
	Path boa.Required[string] `pos:"true"`
	Baz  boa.Required[string] `pos:"true" default:"cba"`
	FB   boa.Optional[string] `pos:"true"`
}{}

func main() {
	boa.Wrap{
		Use:   "hello-world",
		Short: "a generic cli tool",
		Long:  `A generic cli tool that has a longer description.See the README.MD for more information`,
		SubCommands: []*cobra.Command{
			boa.Wrap{
				Use:         "subcommand1",
				Short:       "a subcommand",
				Params:      &params,
				ParamEnrich: boa.ParamEnricherCombine(boa.ParamEnricherName, boa.ParamEnricherEnv),
				Run: func(cmd *cobra.Command, args []string) {
					p1 := params.Foo.Value()
					p2 := params.Bar.Value()
					p3 := params.Path.Value()
					p4 := params.Baz.Value()
					fmt.Printf("Hello world from subcommand1 with params: %s, %d, %s, %s\n", p1, p2, p3, p4)
				},
			}.ToCmd(),
			boa.Wrap{
				Use:   "subcommand2",
				Short: "a subcommand",
				Run: func(cmd *cobra.Command, args []string) {
					fmt.Println("Hello world from subcommand2")
				},
			}.ToCmd(),
		},
	}.ToApp()
}

```

Help output for the above:

```
a subcommand

Usage:
  hello-world subcommand1 <path> <baz> [f-b] [flags]

Flags:
      --bar int      a bar [required] (env: BAR_X) (default 4)
      --foo string   a foo [required] (env: FOO)
  -h, --help         help for subcommand1
```

### Sub-commands, tags and explicit fields

Some customization is only available through explicit field definitions:

```go
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

var params = struct {
	Foo  boa.Required[string]
	Bar  boa.Required[int]    `descr:"a bar" env:"BAR_X" default:"111"`
	Path boa.Required[string] `pos:"true"`
	Baz  boa.Required[string]
	FB   boa.Optional[string] `pos:"true"`
}{
	Foo: boa.Required[string]{Descr: "a foo"},                                                          // add additional info if you like. This means we get "a foo [required] (env: FOO)" in the help text
	Bar: boa.Required[int]{Default: boa.Default(4), CustomValidator: func(x int) error { return nil }}, // optional custom validation logic
	Baz: boa.Required[string]{Positional: true, Default: boa.Default("cba")},                           // positional arguments
}

func main() {
	boa.Wrap{
		Use:   "hello-world",
		Short: "a generic cli tool",
		Long:  `A generic cli tool that has a longer description.See the README.MD for more information`,
		SubCommands: []*cobra.Command{
			boa.Wrap{
				Use:         "subcommand1",
				Short:       "a subcommand",
				Params:      &params,
				ParamEnrich: boa.ParamEnricherCombine(boa.ParamEnricherName, boa.ParamEnricherEnv),
				Run: func(cmd *cobra.Command, args []string) {
					p1 := params.Foo.Value()
					p2 := params.Bar.Value()
					p3 := params.Path.Value()
					p4 := params.Baz.Value()
					fmt.Printf("Hello world from subcommand1 with params: %s, %d, %s, %s\n", p1, p2, p3, p4)
				},
			}.ToCmd(),
			boa.Wrap{
				Use:   "subcommand2",
				Short: "a subcommand",
				Run: func(cmd *cobra.Command, args []string) {
					fmt.Println("Hello world from subcommand2")
				},
			}.ToCmd(),
		},
	}.ToApp()
}

```

Help output for the above:

```
a subcommand

Usage:
  hello-world subcommand1 <path> <baz> [f-b] [flags]

Flags:
      --bar int      a bar [required] (env: BAR_X) (default 111)
      --foo string   a foo [required] (env: FOO)
  -h, --help         help for subcommand1
```

## Missing features

- [ ] Slices
- [ ] Nested config
- [ ] Struct composition/reuse
- [ ] Probably lots

## State

- [x] Very early. Use at your own risk.
