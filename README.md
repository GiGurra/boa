# BOA

Boa is a small cute cli and env var parameter utility. It extends/wraps/constrains/simplifies parts of
github.com/spf13/cobra for building dead simple and declarative cli interfaces.

Boa tries to be as declarative as possible. For the simplest case, all you need to do is to define a struct with
parameter fields, and boa will take care of the rest.

## Convenience

* Fully declarative for definition and validation.
* True optional values and knowledge if a field was set. Opt-in default values built into the type system
    * A `boa.Required[string]`'s `.Value()` is type aware and returns a `string`
    * A `boa.Optional[string]`'s `.Value()` is type aware and returns a `*string`
* Results in generated `--param-name`, `-p`, `[required] (env: ..) (default ...)` from field name and type.
    * example: `Foo boa.Required[string]` will generate
        * `--foo`, and `-f` if it is available
        * `FOO` env var
        * `[required] (env: FOO)` in the help text
        * You can complement this with your own help text
    * You can opt out of auto generation, or cherry-pick and/or add your own auto generation logic
* Validates all inputs before the `Run` function is invoked
* Use explicit fields for config or tags, you decide

## Usage

`go get github.com/GiGurra/boa@v0.0.4`

Short Example:

```go
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

var subCommand1Params = struct {
	Foo  boa.Required[string]
	Bar  boa.Required[int]    `descr:"a bar" env:"BAR_X" default:"111"`
	Path boa.Required[string] `positional:"true"`
	Baz  boa.Required[string]
	FB   boa.Optional[string] `positional:"true"`
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
				Params:      &subCommand1Params,
				ParamEnrich: boa.ParamEnricherCombine(boa.ParamEnricherName, boa.ParamEnricherEnv),
				Run: func(cmd *cobra.Command, args []string) {
					p1 := subCommand1Params.Foo.Value()
					p2 := subCommand1Params.Bar.Value()
					p3 := subCommand1Params.Path.Value()
					p4 := subCommand1Params.Baz.Value()
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


Output for `go run ./cmd/testmain/ subcommand1 --help` on the above:

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

Example just using tags:

```go
var subCommand1Params = struct {
Foo  boa.Required[string] `descr:"a foo"`
Bar  boa.Required[int]    `descr:"a bar" env:"BAR_X" default:"4"`
Path boa.Required[string] `positional:"true"`
Baz  boa.Required[string] `positional:"true" default:"cba"`
FB   boa.Optional[string] `positional:"true"`
}{}
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

## Missing features

- [ ] Slices
- [ ] Nested config
- [ ] Probably lots
