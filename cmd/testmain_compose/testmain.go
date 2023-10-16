package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
	"time"
)

var params1 struct {
	Foo  boa.Required[string]
	Bar  boa.Required[int]
	File boa.Required[string]
}

var params2 struct {
	Baz  boa.Required[string]
	FB   boa.Optional[string]
	Time boa.Optional[time.Time]
}

func main() {
	boa.Wrap{
		Use:    "hello-world",
		Short:  "a generic cli tool",
		Long:   `A generic cli tool that has a longer description. See the README.MD for more information`,
		Params: boa.Compose(&params1, &params2),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf(
				"Hello world from subcommand1 with params: %s, %d, %s, %s, %v, %v\n",
				params1.Foo.Value(),  // string
				params1.Bar.Value(),  // int
				params1.File.Value(), // string
				params2.Baz.Value(),  // string
				params2.FB.Value(),   // *string
				params2.Time.Value(), // *time.Time
			)
		},
	}.ToApp()
}
