package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
	"time"
)

func main() {
	type Base1 struct {
		Foo  boa.Required[string]
		Bar  boa.Required[int]
		File boa.Required[string]
	}

	type Base2 struct {
		Foo2  boa.Required[string]
		Bar2  boa.Required[int]
		File2 boa.Required[string]
	}

	var base3 struct {
		Foo3  boa.Required[string]
		Bar3  boa.Required[int]
		File3 boa.Required[string]
	}

	var base4 struct {
		Foo24  boa.Required[string]
		Bar24  boa.Required[int]
		File24 boa.Required[string]
	}

	var params struct {
		Base Base1
		Base2
		Baz  boa.Required[string]
		FB   boa.Optional[string]
		Time boa.Optional[time.Time]
	}

	boa.Wrap{
		Use:    "hello-world",
		Short:  "a generic cli tool",
		Long:   `A generic cli tool that has a longer description. See the README.MD for more information`,
		Params: boa.Compose(&params, &base3, &base4),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf(
				"Hello world from subcommand1 with params: %s, %d, %s, %s, %v, %v\n",
				params.Base.Foo.Value(),  // string
				params.Base.Bar.Value(),  // int
				params.Base.File.Value(), // string
				params.Baz.Value(),       // string
				params.FB.Value(),        // *string
				params.Time.Value(),      // *time.Time
			)
		},
	}.ToAppH(boa.ResultHandler{
		Failure: func(err error) {
			fmt.Printf("Error: %v\n", err)
		},
	})
}
