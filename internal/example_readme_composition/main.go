// Example matching the README "Composition" section.
// This demonstrates composing structs for complex parameter structures.
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
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
	boa.NewCmdT[Combined]("hello-world").
		WithShort("a generic cli tool").
		WithLong("A generic cli tool that has a longer description").
		WithRunFunc(func(params *Combined) {
			fmt.Printf(
				"Hello world with params: %s, %d, %s, %s, %s, %v\n",
				params.Base.Foo,  // string
				params.Base.Bar,  // int
				params.Base.File, // string
				params.Baz,       // string
				params.FB,        // string
				params.Time,      // time.Time
			)
		}).
		Run()
}
