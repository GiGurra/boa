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

	cmd :=
		NewCmdT[TestStruct]("test").
			WithRunFunc(func(params *TestStruct) {
				fmt.Printf("params: %+v\n", params)
				if params.Flag1.Value() != "value1" {
					t.Fatalf("expected value1 but got %s", params.Flag1.Value())
				}
				if params.Flag2.Value() != 42 {
					t.Fatalf("expected 42 but got %d", params.Flag2.Value())
				}
			})

	if cmd.Params.Flag1.HasValue() {
		t.Errorf("Flag1 should not have value")
	}

	if cmd.Params.Flag2.HasValue() {
		t.Errorf("Flag2 should not have value")
	}

	cmd.Run()

	if !cmd.Params.Flag1.HasValue() {
		t.Errorf("Flag1 should have value")
	}

	if !cmd.Params.Flag2.HasValue() {
		t.Errorf("Flag2 should have value")
	}
}

func TestTyped2(t *testing.T) {

	cmd :=
		NewCmdT2("test", &TestStruct{}).
			WithRawArgs([]string{"--flag1", "value1", "--flag2", "42"}).
			WithRunFunc3(func(params *TestStruct, cmd *cobra.Command, args []string) {
				fmt.Printf("params: %+v\n", params)
				if params.Flag1.Value() != "value1" {
					t.Fatalf("expected value1 but got %s", params.Flag1.Value())
				}
				if params.Flag2.Value() != 42 {
					t.Fatalf("expected 42 but got %d", params.Flag2.Value())
				}
			})

	if cmd.Params.Flag1.HasValue() {
		t.Errorf("Flag1 should not have value")
	}

	if cmd.Params.Flag2.HasValue() {
		t.Errorf("Flag2 should not have value")
	}

	cmd.Run()

	if !cmd.Params.Flag1.HasValue() {
		t.Errorf("Flag1 should have value")
	}

	if !cmd.Params.Flag2.HasValue() {
		t.Errorf("Flag2 should have value")
	}
}

func TestTypedWithInitFunc(t *testing.T) {

	cmd :=
		NewCmdT[TestStruct]("test").
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

	if cmd.Params.Flag1.HasValue() {
		t.Errorf("Flag1 should not have value")
	}

	if cmd.Params.Flag2.HasValue() {
		t.Errorf("Flag2 should not have value")
	}

	cmd.WithRawArgs([]string{"--flag1", "value1"}).Run()

	if !cmd.Params.Flag1.HasValue() {
		t.Errorf("Flag1 should have value")
	}

	if !cmd.Params.Flag2.HasValue() {
		t.Errorf("Flag2 should have value")
	}
}

func TestNoParams(t *testing.T) {

	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	os.Args = []string{"test"}

	cmd :=
		NewCmdT[NoParams]("test").
			WithRunFunc(func(_ *NoParams) {
			})
	cmdCpy := cmd

	cmd.Run()
	if cmdCpy.Validate() != nil {
		t.Fatalf("expected no error but got %v", cmd.Validate())
	}
}

func TestCmdTree(t *testing.T) {
	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	os.Args = []string{"test", "subcmd1", "--flag1", "value1", "--flag2", "42"}

	ranInnerCommand := false
	NewCmdT[NoParams]("test").WithSubCmds(
		NewCmdT[TestStruct]("subcmd1").WithRunFunc(func(params *TestStruct) {
			fmt.Printf("params: %+v\n", params)
			if params.Flag1.Value() != "value1" {
				t.Fatalf("expected value1 but got %s", params.Flag1.Value())
			}
			ranInnerCommand = true
		}),
	).Run()

	if !ranInnerCommand {
		t.Fatalf("expected inner command to run but it didn't")
	}
}

func TestGolangUpcastBullShit(t *testing.T) {
	ran := false
	type Args struct {
	}
	CmdT[Args]{
		Use: "test",
		RunFunc: func(params *Args, cmd *cobra.Command, args []string) {
			ran = true
		},
	}.RunArgs([]string{})

	if !ran {
		t.Fatalf("expected inner command to run but it didn't")
	}
}

func TestTypedManual(t *testing.T) {
	ran := false
	type Args struct {
		MyInt int
	}
	CmdT[Args]{
		Use:    "test",
		Params: &Args{},
		RunFunc: func(params *Args, cmd *cobra.Command, args []string) {
			ran = true
			if params.MyInt != 42 {
				t.Fatalf("expected 42 but got %d", params.MyInt)
			}
		},
	}.RunArgs([]string{"--my-int", "42"})

	if !ran {
		t.Fatalf("expected inner command to run but it didn't")
	}
}

func TestAutoGenerateParamsFieldWhenOmitted(t *testing.T) {
	ran := false
	type Args struct {
		MyInt int
	}
	CmdT[Args]{
		Use: "test",
		RunFunc: func(params *Args, cmd *cobra.Command, args []string) {
			ran = true
			if params.MyInt != 42 {
				t.Fatalf("expected 42 but got %d", params.MyInt)
			}
		},
	}.RunArgs([]string{"--my-int", "42"})

	if !ran {
		t.Fatalf("expected inner command to run but it didn't")
	}
}

func TestCmdList(t *testing.T) {
	ran := false
	type Args struct {
		MyInt int
	}
	Cmd{
		RunFunc: func(cmd *cobra.Command, args []string) {
			t.Fatalf("expected to not run")
		},
		SubCmds: SubCmds(
			NewCmdT[NoParams]("123").WithRunFunc(func(params *NoParams) {}),
			CmdT[NoParams]{Use: "subcmd1"},
			Cmd{Use: "subcmd2"},
			CmdT[Args]{Use: "subcmd3", RunFunc: func(params *Args, cmd *cobra.Command, args []string) {
				ran = true
				if params.MyInt != 42 {
					t.Fatalf("expected 42 but got %d", params.MyInt)
				}
			}},
		),
	}.RunArgs([]string{"subcmd3", "--my-int", "42"})

	if !ran {
		t.Fatalf("expected inner command to run but it didn't")
	}
}
