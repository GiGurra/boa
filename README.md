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
* Can generate everything except descriptions from field names and types
  * `Foo boa.Required[string]` will generate
    * `--foo`, and `-f` if it is available
    * `FOO` env var
    * `[required] (env: FOO)` in the help text
    * You can complement this with your own help text 
  * You can opt out of auto generation, or cherry pick and/or add your own auto generation logic

## Usage

Short Example:

```go
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

var subCommand1Params = struct {
	Foo boa.Required[string]  // tags not supported, yet perhaps
	Bar boa.Required[int]     // tags not supported, yet perhaps
	Baz boa.Optional[float64] // tags not supported, yet perhaps
}{
	Foo: boa.Required[string]{Descr: "a foo"},                                 // add additional info if you like. This means we get "a foo [required] (env: FOO)" in the help text
	Bar: boa.Required[int]{CustomValidator: func(x int) error { return nil }}, // optional custom validation logic
}

func main() {
	boa.Wrap{
		Use:   "hello-world",
		Short: "a generic cli tool",
		Long:  `A generic cli tool that has a longer description. See the README.MD for more information`,
		SubCommands: []*cobra.Command{
			boa.Wrap{
				Use:         "subcommand1",
				Short:       "a subcommand",
				Params:      &subCommand1Params,
				ParamEnrich: boa.ParamEnricherCombine(boa.ParamEnricherName, boa.ParamEnricherEnv),
				Run: func(cmd *cobra.Command, args []string) {
					var p1 string = subCommand1Params.Foo.Value()
					var p2 int = subCommand1Params.Bar.Value()
					var p3 *float64 = subCommand1Params.Baz.Value()
					fmt.Printf("Hello world from subcommand1 with params: %s, %d, %v\n", p1, p2, p3)
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

## Missing features

- [ ] Support for positional arguments
- [ ] Support for field tags
- [ ] Support setting values by tag
- [ ] Probably more things
