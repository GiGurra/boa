// Example matching the README "Array/slice parameters" section.
// This demonstrates array/slice types with proper parsing.
package main

import (
	"fmt"
	"github.com/GiGurra/boa/pkg/boa"
)

type Params struct {
	Numbers []int    `descr:"list of numbers"`
	Tags    []string `descr:"tags" default:"[a,b,c]"`
	Ports   []int64  `descr:"ports" default:"[8080,8081,8082]"`
}

func main() {
	boa.NewCmdT[Params]("hello-world").
		WithShort("a generic cli tool").
		WithRunFunc(func(params *Params) {
			fmt.Printf("Numbers: %v\n", params.Numbers)
			fmt.Printf("Tags: %v\n", params.Tags)
			fmt.Printf("Ports: %v\n", params.Ports)
		}).
		Run()
}
