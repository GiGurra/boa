package boa

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"testing"
)

type TestStruct struct {
	Flag1 Required[string]
	Flag2 Required[int]
}

func TestTyped1(t *testing.T) {

	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	os.Args = []string{"test", "--flag1", "value1", "--flag2", "42"}

	builder :=
		NewCmdBuilder[TestStruct]("test").
			WithRunFunc(func(params *TestStruct) {
				fmt.Printf("params: %+v\n", params)
				if params.Flag1.Value() != "value1" {
					t.Fatalf("expected value1 but got %s", params.Flag1.Value())
				}
				if params.Flag2.Value() != 42 {
					t.Fatalf("expected 42 but got %d", params.Flag2.Value())
				}
			})

	if builder.Params.Flag1.HasValue() {
		t.Errorf("Flag1 should not have value")
	}

	if builder.Params.Flag2.HasValue() {
		t.Errorf("Flag2 should not have value")
	}

	builder.Run()

	if !builder.Params.Flag1.HasValue() {
		t.Errorf("Flag1 should have value")
	}

	if !builder.Params.Flag2.HasValue() {
		t.Errorf("Flag2 should have value")
	}
}

func TestTyped2(t *testing.T) {

	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	os.Args = []string{"test", "--flag1", "value1", "--flag2", "42"}

	builder :=
		NewCmdBuilder2("test", &TestStruct{}).
			WithRunFunc3(func(params *TestStruct, cmd *cobra.Command, args []string) {
				fmt.Printf("params: %+v\n", params)
				if params.Flag1.Value() != "value1" {
					t.Fatalf("expected value1 but got %s", params.Flag1.Value())
				}
				if params.Flag2.Value() != 42 {
					t.Fatalf("expected 42 but got %d", params.Flag2.Value())
				}
			})

	if builder.Params.Flag1.HasValue() {
		t.Errorf("Flag1 should not have value")
	}

	if builder.Params.Flag2.HasValue() {
		t.Errorf("Flag2 should not have value")
	}

	builder.Run()

	if !builder.Params.Flag1.HasValue() {
		t.Errorf("Flag1 should have value")
	}

	if !builder.Params.Flag2.HasValue() {
		t.Errorf("Flag2 should have value")
	}
}

func TestTypedWithInitFunc(t *testing.T) {

	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	os.Args = []string{"test", "--flag1", "value1"}

	builder :=
		NewCmdBuilder[TestStruct]("test").
			WithInitFunc(func(params *TestStruct) { params.Flag2.Default = Default(42) }).
			WithRunFunc(func(params *TestStruct) {
				fmt.Printf("params: %+v\n", params)
				if params.Flag1.Value() != "value1" {
					t.Fatalf("expected value1 but got %s", params.Flag1.Value())
				}
				if params.Flag2.Value() != 42 {
					t.Fatalf("expected 42 but got %d", params.Flag2.Value())
				}
			})

	if builder.Params.Flag1.HasValue() {
		t.Errorf("Flag1 should not have value")
	}

	if builder.Params.Flag2.HasValue() {
		t.Errorf("Flag2 should not have value")
	}

	builder.Run()

	if !builder.Params.Flag1.HasValue() {
		t.Errorf("Flag1 should have value")
	}

	if !builder.Params.Flag2.HasValue() {
		t.Errorf("Flag2 should have value")
	}
}

func TestNoParams(t *testing.T) {

	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	os.Args = []string{"test"}

	builder :=
		NewCmdBuilder[NoParamsT]("test").
			WithRunFunc(func(_ *NoParamsT) {
			})
	builderCpy := builder

	builder.Run()
	if builderCpy.Validate() != nil {
		t.Fatalf("expected no error but got %v", builder.Validate())
	}
}
