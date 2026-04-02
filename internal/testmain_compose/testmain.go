package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
	"time"
)

func main() {
	type Base1 struct {
		Foo  string
		Bar  int
		File string
	}

	type Base2 struct {
		Foo2  string
		Bar2  int
		File2 string
	}

	type Base3 struct {
		Foo3  string
		Bar3  int
		File3 string
	}

	type Base4 struct {
		Foo24  string
		Bar24  int
		File24 string
	}

	var params struct {
		Base Base1
		Base2
		Base3
		Base4
		Baz  string
		FB   string    `optional:"true"`
		Time time.Time `optional:"true"`
	}

	if err := (boa.Cmd{
		Use:    "hello-world",
		Short:  "a generic cli tool",
		Long:   `A generic cli tool that has a longer description. See the README.MD for more information`,
		Params: &params,
		RunFunc: func(cmd *cobra.Command, args []string) {
			fmt.Printf(
				"Hello world from subcommand1 with params: %s, %d, %s, %s, %q, %v\n",
				params.Base.Foo,  // string
				params.Base.Bar,  // int
				params.Base.File, // string
				params.Baz,       // string
				params.FB,        // string
				params.Time,      // time.Time
			)
		},
	}.RunE()); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
