package boa

import (
	"fmt"
	"os"
	"testing"

	"github.com/spf13/cobra"
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

type CustomStringType string
type CustomIntType int

type TestStructCustType struct {
	Flag1 Required[CustomStringType]
	Flag2 Required[CustomIntType]
}

func TestTypedCustType1(t *testing.T) {

	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	os.Args = []string{"test", "--flag1", "value1", "--flag2", "42"}

	cmd :=
		NewCmdT[TestStructCustType]("test").
			WithRunFunc(func(params *TestStructCustType) {
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

type TestStructCustTypeOptional struct {
	Flag1 Optional[CustomStringType]
	Flag2 Optional[CustomIntType]
}

func TestTypedCustTypeOptional(t *testing.T) {

	prevArgs := os.Args
	defer func() {
		os.Args = prevArgs
	}()

	os.Args = []string{"test", "--flag1", "value1", "--flag2", "42"}

	cmd :=
		NewCmdT[TestStructCustTypeOptional]("test").
			WithRunFunc(func(params *TestStructCustTypeOptional) {
				fmt.Printf("params: %+v\n", params)
				if params.Flag1.Value() == nil {
					t.Fatalf("expected Flag1 to have value")
				}
				if *params.Flag1.Value() != "value1" {
					t.Fatalf("expected value1 but got %s", *params.Flag1.Value())
				}
				if params.Flag2.Value() == nil {
					t.Fatalf("expected Flag2 to have value")
				}
				if *params.Flag2.Value() != 42 {
					t.Fatalf("expected 42 but got %d", *params.Flag2.Value())
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

func TestSliceFlagAlts(t *testing.T) {
	ran := false
	type Args struct {
		Types []string `alts:"file, dir, all" default:"all,dir"`
	}
	CmdT[Args]{
		RunFunc: func(args *Args, cmd *cobra.Command, rawArgs []string) {
			if len(args.Types) != 2 {
				t.Fatalf("expected 2 types but got %d", len(args.Types))
			}
			if args.Types[0] != "all" {
				t.Fatalf("expected first type to be 'all' but got '%s'", args.Types[0])
			}
			if args.Types[1] != "dir" {
				t.Fatalf("expected second type to be 'dir' but got '%s'", args.Types[1])
			}
			ran = true
		},
	}.RunArgs([]string{})

	if !ran {
		t.Fatalf("expected inner command to run but it didn't")
	}
}

func TestPositionalArgs(t *testing.T) {
	ran := false
	type Args struct {
		MyInt       int    `positional:"true"`
		MyString    string `pos:"true"`
		MyStringOpt string `pos:"true" optional:"true"`
	}

	CmdT[Args]{
		RunFunc: func(params *Args, cmd *cobra.Command, args []string) {
			ran = true
			if params.MyInt != 42 {
				t.Fatalf("expected 42 but got %d", params.MyInt)
			}
			if params.MyString != "hello" {
				t.Fatalf("expected 'hello' but got '%s'", params.MyString)
			}
			if params.MyStringOpt != "" {
				t.Fatalf("expected '' but got '%s'", params.MyStringOpt)
			}
		},
	}.RunArgs([]string{"42", "hello"})

	if !ran {
		t.Fatalf("expected inner command to run but it didn't")
	}
}

func TestSlicePositionalArgs(t *testing.T) {
	ran := false
	type Args struct {
		Names []string `positional:"true"`
	}

	CmdT[Args]{
		RunFunc: func(params *Args, cmd *cobra.Command, args []string) {
			ran = true
			if len(params.Names) != 3 {
				t.Fatalf("expected 3 names but got %d", len(params.Names))
			}
			if params.Names[0] != "alice" {
				t.Fatalf("expected first name to be 'alice' but got '%s'", params.Names[0])
			}
			if params.Names[1] != "bob" {
				t.Fatalf("expected second name to be 'bob' but got '%s'", params.Names[1])
			}
			if params.Names[2] != "carol" {
				t.Fatalf("expected third name to be 'carol' but got '%s'", params.Names[2])
			}
		},
	}.RunArgs([]string{"alice", "bob", "carol"})

	if !ran {
		t.Fatalf("expected inner command to run but it didn't")
	}
}

func TestCommandAliases(t *testing.T) {
	cmd := NewCmdT[NoParams]("server").
		WithAliases("srv", "s").
		WithShort("Start the server").
		WithRunFunc(func(_ *NoParams) {})

	cobraCmd := cmd.ToCobra()
	if len(cobraCmd.Aliases) != 2 {
		t.Fatalf("expected 2 aliases but got %d", len(cobraCmd.Aliases))
	}
	if cobraCmd.Aliases[0] != "srv" {
		t.Fatalf("expected first alias to be 'srv' but got '%s'", cobraCmd.Aliases[0])
	}
	if cobraCmd.Aliases[1] != "s" {
		t.Fatalf("expected second alias to be 's' but got '%s'", cobraCmd.Aliases[1])
	}
}

func TestCommandAliasesNonGeneric(t *testing.T) {
	cmd := Cmd{
		Use:   "server",
		Short: "Start the server",
	}.WithAliases("srv", "s")

	cobraCmd := cmd.ToCobra()
	if len(cobraCmd.Aliases) != 2 {
		t.Fatalf("expected 2 aliases but got %d", len(cobraCmd.Aliases))
	}
	if cobraCmd.Aliases[0] != "srv" {
		t.Fatalf("expected first alias to be 'srv' but got '%s'", cobraCmd.Aliases[0])
	}
	if cobraCmd.Aliases[1] != "s" {
		t.Fatalf("expected second alias to be 's' but got '%s'", cobraCmd.Aliases[1])
	}
}

func TestCommandGroups(t *testing.T) {
	ran := false

	// Groups are auto-generated from subcommand GroupIDs
	root := NewCmdT[NoParams]("app").
		WithSubCmds(
			NewCmdT[NoParams]("start").
				WithGroupID("core").
				WithRunFunc(func(_ *NoParams) { ran = true }),
			NewCmdT[NoParams]("stop").
				WithGroupID("core").
				WithRunFunc(func(_ *NoParams) {}),
			NewCmdT[NoParams]("debug").
				WithGroupID("extra").
				WithRunFunc(func(_ *NoParams) {}),
		)

	cobraCmd := root.ToCobra()

	// Check that groups are auto-generated
	if len(cobraCmd.Groups()) != 2 {
		t.Fatalf("expected 2 groups but got %d", len(cobraCmd.Groups()))
	}

	// Check that subcommands have correct GroupID
	for _, sub := range cobraCmd.Commands() {
		switch sub.Use {
		case "start", "stop":
			if sub.GroupID != "core" {
				t.Fatalf("expected subcommand '%s' to have GroupID 'core' but got '%s'", sub.Use, sub.GroupID)
			}
		case "debug":
			if sub.GroupID != "extra" {
				t.Fatalf("expected subcommand '%s' to have GroupID 'extra' but got '%s'", sub.Use, sub.GroupID)
			}
		}
	}

	// Run the start command
	root.RunArgs([]string{"start"})
	if !ran {
		t.Fatalf("expected start command to run but it didn't")
	}
}

func TestCommandGroupsNonGeneric(t *testing.T) {
	// Groups are auto-generated from subcommand GroupIDs
	root := Cmd{
		Use: "app",
	}.WithSubCmds(
		Cmd{Use: "start"}.WithGroupID("core"),
	)

	cobraCmd := root.ToCobra()

	// Check that group is auto-generated
	if len(cobraCmd.Groups()) != 1 {
		t.Fatalf("expected 1 group but got %d", len(cobraCmd.Groups()))
	}
	if cobraCmd.Groups()[0].ID != "core" {
		t.Fatalf("expected group ID 'core' but got '%s'", cobraCmd.Groups()[0].ID)
	}
	// Auto-generated title should be "ID:"
	if cobraCmd.Groups()[0].Title != "core:" {
		t.Fatalf("expected group title 'core:' but got '%s'", cobraCmd.Groups()[0].Title)
	}

	// Check that subcommand has correct GroupID
	for _, sub := range cobraCmd.Commands() {
		if sub.Use == "start" && sub.GroupID != "core" {
			t.Fatalf("expected subcommand 'start' to have GroupID 'core' but got '%s'", sub.GroupID)
		}
	}
}

func TestAliasExecution(t *testing.T) {
	ran := false

	root := NewCmdT[NoParams]("app").
		WithSubCmds(
			NewCmdT[NoParams]("server").
				WithAliases("srv", "s").
				WithRunFunc(func(_ *NoParams) { ran = true }),
		)

	// Execute using alias
	root.RunArgs([]string{"srv"})
	if !ran {
		t.Fatalf("expected command to run via alias 'srv' but it didn't")
	}

	// Reset and test with shorter alias
	ran = false
	root.RunArgs([]string{"s"})
	if !ran {
		t.Fatalf("expected command to run via alias 's' but it didn't")
	}
}

func TestMixedExplicitAndAutoGeneratedGroups(t *testing.T) {
	// "core" has explicit custom title, "extra" will be auto-generated
	root := NewCmdT[NoParams]("app").
		WithGroups(
			&cobra.Group{ID: "core", Title: "Core Commands:"},
		).
		WithSubCmds(
			NewCmdT[NoParams]("start").WithGroupID("core"),
			NewCmdT[NoParams]("debug").WithGroupID("extra"),
		)

	cobraCmd := root.ToCobra()

	// Check that both groups exist
	if len(cobraCmd.Groups()) != 2 {
		t.Fatalf("expected 2 groups but got %d", len(cobraCmd.Groups()))
	}

	// Find each group and verify titles
	var coreGroup, extraGroup *cobra.Group
	for _, g := range cobraCmd.Groups() {
		switch g.ID {
		case "core":
			coreGroup = g
		case "extra":
			extraGroup = g
		}
	}

	if coreGroup == nil {
		t.Fatal("expected to find 'core' group")
	}
	if coreGroup.Title != "Core Commands:" {
		t.Fatalf("expected core group title 'Core Commands:' but got '%s'", coreGroup.Title)
	}

	if extraGroup == nil {
		t.Fatal("expected to find 'extra' group")
	}
	if extraGroup.Title != "extra:" {
		t.Fatalf("expected auto-generated extra group title 'extra:' but got '%s'", extraGroup.Title)
	}
}
