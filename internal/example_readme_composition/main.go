// Example matching the README "Composition" section.
// This demonstrates composing structs for complex parameter structures.
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
	"time"
)

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

type Combined struct {
	Base Base1
	Base2
	Baz  string
	FB   string    `optional:"true"`
	Time time.Time `optional:"true"`
}

func main() {
	boa.CmdT[Combined]{
		Use:   "hello-world",
		Short: "a generic cli tool",
		Long:  "A generic cli tool that has a longer description",
		RunFunc: func(params *Combined, cmd *cobra.Command, args []string) {
			fmt.Printf(
				"Hello world with params: %s, %d, %s, %s, %s, %v\n",
				params.Base.Foo,  // string
				params.Base.Bar,  // int
				params.Base.File, // string
				params.Baz,       // string
				params.FB,        // string
				params.Time,      // time.Time
			)
		},
	}.Run()
}
