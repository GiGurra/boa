package boa

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestPersistentFlag_InheritedThroughTwoLevels(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "before subcommands", args: []string{"--db", "before", "middle", "leaf"}},
		{name: "after subcommands", args: []string{"middle", "leaf", "--db", "after"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type RootParams struct {
				DB string `long:"db" persistent:"true" optional:"true"`
			}

			var got string
			leaf := &cobra.Command{
				Use: "leaf",
				Run: func(cmd *cobra.Command, args []string) {
					got = cmd.Root().PersistentFlags().Lookup("db").Value.String()
				},
			}
			middle := &cobra.Command{Use: "middle"}
			middle.AddCommand(leaf)

			params := RootParams{}
			root := (CmdT[RootParams]{
				Use:     "root",
				Params:  &params,
				SubCmds: []*cobra.Command{middle},
			}).ToCobra()
			root.SetArgs(tt.args)

			if err := root.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			want := tt.args[1]
			if tt.name == "after subcommands" {
				want = tt.args[3]
			}
			if params.DB != want {
				t.Errorf("params.DB = %q, want %q", params.DB, want)
			}
			if got != want {
				t.Errorf("flag value at leaf = %q, want %q", got, want)
			}
		})
	}
}

func TestPersistentFlag_RootValidationRunsForLeaf(t *testing.T) {
	type RootParams struct {
		DB string `long:"db" persistent:"true"`
	}

	leafRan := false
	leaf := &cobra.Command{Use: "leaf", Run: func(cmd *cobra.Command, args []string) { leafRan = true }}
	root := (CmdT[RootParams]{
		Use:     "root",
		SubCmds: []*cobra.Command{leaf},
	}).ToCobra()
	root.SetArgs([]string{"leaf"})

	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "missing required param 'db'") {
		t.Fatalf("Execute() error = %v, want missing root db error", err)
	}
	if leafRan {
		t.Error("leaf ran despite failed root parameter validation")
	}
}

func TestPersistentFlag_DeclaredAtSubtreeNode(t *testing.T) {
	type MiddleParams struct {
		Region string `long:"region" persistent:"true" optional:"true"`
	}

	params := MiddleParams{}
	leaf := &cobra.Command{Use: "leaf", Run: func(cmd *cobra.Command, args []string) {}}
	middle := (CmdT[MiddleParams]{
		Use:     "middle",
		Params:  &params,
		SubCmds: []*cobra.Command{leaf},
	}).ToCobra()
	root := &cobra.Command{Use: "root"}
	root.AddCommand(middle)
	root.SetArgs([]string{"middle", "leaf", "--region", "eu-north"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if params.Region != "eu-north" {
		t.Errorf("params.Region = %q, want eu-north", params.Region)
	}
}

func TestPersistentFlag_ComposesRootAndIntermediatePipelines(t *testing.T) {
	type RootParams struct {
		DB string `long:"db" persistent:"true"`
	}
	type MiddleParams struct {
		Region string `long:"region" persistent:"true"`
	}

	rootParams := RootParams{}
	middleParams := MiddleParams{}
	var pipelineOrder []string
	leafRan := false
	leaf := &cobra.Command{
		Use: "leaf",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			pipelineOrder = append(pipelineOrder, "leaf")
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) { leafRan = true },
	}
	middle := (CmdT[MiddleParams]{
		Use:     "middle",
		Params:  &middleParams,
		SubCmds: []*cobra.Command{leaf},
		PreValidateFunc: func(params *MiddleParams, cmd *cobra.Command, args []string) error {
			pipelineOrder = append(pipelineOrder, "middle")
			return nil
		},
	}).ToCobra()
	root := (CmdT[RootParams]{
		Use:     "root",
		Params:  &rootParams,
		SubCmds: []*cobra.Command{middle},
		PreValidateFunc: func(params *RootParams, cmd *cobra.Command, args []string) error {
			pipelineOrder = append(pipelineOrder, "root")
			return nil
		},
	}).ToCobra()
	root.SetArgs([]string{"middle", "leaf", "--db", "app.db", "--region", "eu-north"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !leafRan {
		t.Error("leaf did not run")
	}
	if rootParams.DB != "app.db" || middleParams.Region != "eu-north" {
		t.Errorf("params = {DB:%q Region:%q}, want {DB:app.db Region:eu-north}", rootParams.DB, middleParams.Region)
	}
	if got := strings.Join(pipelineOrder, ","); got != "root,middle,leaf" {
		t.Errorf("pipeline order = %q, want root,middle,leaf", got)
	}
}

func TestPersistentFlag_ChildLocalFlagShadowsParent(t *testing.T) {
	type RootParams struct {
		Value string `long:"value" persistent:"true" optional:"true"`
	}
	type ChildParams struct {
		Value string `long:"value" optional:"true"`
	}

	rootParams := RootParams{}
	childParams := ChildParams{}
	child := (CmdT[ChildParams]{
		Use:     "child",
		Params:  &childParams,
		RunFunc: func(params *ChildParams, cmd *cobra.Command, args []string) {},
	}).ToCobra()
	root := (CmdT[RootParams]{
		Use:     "root",
		Params:  &rootParams,
		SubCmds: []*cobra.Command{child},
	}).ToCobra()
	root.SetArgs([]string{"child", "--value", "child-value"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if childParams.Value != "child-value" {
		t.Errorf("child value = %q, want child-value", childParams.Value)
	}
	if rootParams.Value != "" {
		t.Errorf("shadowed root value = %q, want empty", rootParams.Value)
	}
}

func TestPersistentFlag_AppearsInLeafHelp(t *testing.T) {
	type RootParams struct {
		DB string `long:"db" persistent:"true" optional:"true" descr:"database path"`
	}

	leaf := &cobra.Command{Use: "leaf", Run: func(cmd *cobra.Command, args []string) {}}
	root := (CmdT[RootParams]{Use: "root", SubCmds: []*cobra.Command{leaf}}).ToCobra()
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetErr(&output)
	root.SetArgs([]string{"leaf", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	help := output.String()
	if !strings.Contains(help, "Global Flags:") || !strings.Contains(help, "--db") || !strings.Contains(help, "database path") {
		t.Errorf("leaf help does not show inherited db flag in Global Flags:\n%s", help)
	}
}

func TestPersistentFlag_RejectsPositional(t *testing.T) {
	type Params struct {
		Target string `positional:"true" persistent:"true" optional:"true"`
	}

	_, err := (CmdT[Params]{Use: "cmd"}).ToCobraE()
	if err == nil || !strings.Contains(err.Error(), "cannot be both positional and persistent") {
		t.Fatalf("ToCobraE() error = %v, want positional/persistent error", err)
	}
}

func TestSetPersistent_Programmatic(t *testing.T) {
	type Params struct {
		DB string `long:"db" optional:"true"`
	}

	params := Params{}
	leafRan := false
	leaf := &cobra.Command{Use: "leaf", Run: func(cmd *cobra.Command, args []string) { leafRan = true }}
	root := (CmdT[Params]{
		Use:     "root",
		Params:  &params,
		SubCmds: []*cobra.Command{leaf},
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			param := GetParamT(ctx, &params.DB)
			param.SetPersistent(true)
			if !param.Param().IsPersistent() {
				t.Error("IsPersistent() = false after SetPersistent(true)")
			}
			return nil
		},
	}).ToCobra()
	root.SetArgs([]string{"leaf", "--db", "programmatic"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !leafRan || params.DB != "programmatic" {
		t.Errorf("leafRan = %v, params.DB = %q", leafRan, params.DB)
	}
}

func TestPersistentFlag_InvalidTagValue(t *testing.T) {
	type Params struct {
		DB string `persistent:"yes" optional:"true"`
	}

	_, err := (CmdT[Params]{Use: "cmd"}).ToCobraE()
	if err == nil || !strings.Contains(err.Error(), "invalid persistent value") {
		t.Fatalf("ToCobraE() error = %v, want invalid persistent value error", err)
	}
}

func TestPersistentFlag_AutoShortDoesNotCollideWithDescendant(t *testing.T) {
	type RootParams struct {
		DB string `long:"db" persistent:"true" optional:"true"`
	}
	type ChildParams struct {
		Description string `long:"description" optional:"true"`
	}

	rootParams := RootParams{}
	childParams := ChildParams{}
	child := (CmdT[ChildParams]{
		Use:     "create",
		Params:  &childParams,
		RunFunc: func(params *ChildParams, cmd *cobra.Command, args []string) {},
	}).ToCobra()
	root, err := (CmdT[RootParams]{
		Use:     "root",
		Params:  &rootParams,
		SubCmds: []*cobra.Command{child},
	}).ToCobraE()
	if err != nil {
		t.Fatalf("ToCobraE() error = %v", err)
	}

	if got := root.PersistentFlags().Lookup("db").Shorthand; got != "" {
		t.Errorf("persistent --db shorthand = %q, want none", got)
	}
	if got := child.Flags().Lookup("description").Shorthand; got != "d" {
		t.Errorf("local --description shorthand = %q, want d", got)
	}

	root.SetArgs([]string{"create", "-d", "created through shorthand"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if childParams.Description != "created through shorthand" {
		t.Errorf("child description = %q", childParams.Description)
	}
}

func TestPersistentFlag_AutoShortRetainedWithoutConflict(t *testing.T) {
	type RootParams struct {
		DB string `long:"db" persistent:"true" optional:"true"`
	}
	type ChildParams struct {
		Name string `long:"name" optional:"true"`
	}

	rootParams := RootParams{}
	child := (CmdT[ChildParams]{
		Use:     "create",
		RunFunc: func(params *ChildParams, cmd *cobra.Command, args []string) {},
	}).ToCobra()
	root, err := (CmdT[RootParams]{
		Use:     "root",
		Params:  &rootParams,
		SubCmds: []*cobra.Command{child},
	}).ToCobraE()
	if err != nil {
		t.Fatalf("ToCobraE() error = %v", err)
	}
	if got := root.PersistentFlags().Lookup("db").Shorthand; got != "d" {
		t.Fatalf("persistent --db shorthand = %q, want d", got)
	}

	root.SetArgs([]string{"create", "-d", "app.db"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if rootParams.DB != "app.db" {
		t.Errorf("root DB = %q, want app.db", rootParams.DB)
	}
}

func TestPersistentFlag_ExplicitShortCollisionIsConstructionError(t *testing.T) {
	type RootParams struct {
		DB string `long:"db" short:"d" persistent:"true" optional:"true"`
	}
	type ChildParams struct {
		Description string `long:"description" optional:"true"`
	}

	child := (CmdT[ChildParams]{
		Use:     "create",
		RunFunc: func(params *ChildParams, cmd *cobra.Command, args []string) {},
	}).ToCobra()
	_, err := (CmdT[RootParams]{
		Use:     "root",
		SubCmds: []*cobra.Command{child},
	}).ToCobraE()
	if err == nil {
		t.Fatal("ToCobraE() error = nil, want shorthand collision")
	}
	for _, want := range []string{"shorthand -d", "--description", "--db", `command "root create"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("ToCobraE() error = %q, want substring %q", err, want)
		}
	}
}

func TestPersistentFlag_ExplicitShortAllowsLongNameShadow(t *testing.T) {
	type RootParams struct {
		DB string `long:"db" short:"d" persistent:"true" optional:"true"`
	}
	type ChildParams struct {
		DB string `long:"db" short:"d" optional:"true"`
	}

	child := (CmdT[ChildParams]{
		Use:     "create",
		RunFunc: func(params *ChildParams, cmd *cobra.Command, args []string) {},
	}).ToCobra()
	if _, err := (CmdT[RootParams]{
		Use:     "root",
		SubCmds: []*cobra.Command{child},
	}).ToCobraE(); err != nil {
		t.Fatalf("ToCobraE() error = %v, want Cobra long-name shadowing", err)
	}
}
