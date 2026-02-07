package boa

import (
	"testing"

	"github.com/spf13/cobra"
)

func resetGlobalConfig() {
	cfg = globalConfig{}
}

func TestInitWithDefaultOptional(t *testing.T) {
	defer resetGlobalConfig()

	type Params struct {
		Name string `descr:"a name"`
		Port int    `descr:"a port"`
	}

	Init(WithDefaultOptional())

	// With defaultOptional, raw fields should not be required
	err := CmdT[Params]{
		Use:   "test",
		Short: "test command",
		RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
		},
	}.RunArgsE([]string{})
	if err != nil {
		t.Fatalf("expected no error with optional defaults, got: %v", err)
	}
}

func TestDefaultBehaviorWithoutInit(t *testing.T) {
	defer resetGlobalConfig()

	type Params struct {
		Name string `descr:"a name"`
	}

	// Without Init, raw fields should be required (default behavior)
	err := CmdT[Params]{
		Use:   "test",
		Short: "test command",
		RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
			t.Fatal("RunFunc should not be called when required field is missing")
		},
	}.RunArgsE([]string{})
	if err == nil {
		t.Fatal("expected error for missing required field without Init")
	}
}

func TestExplicitRequiredTagOverridesDefaultOptional(t *testing.T) {
	defer resetGlobalConfig()

	type Params struct {
		Name string `descr:"a name" required:"true"`
	}

	Init(WithDefaultOptional())

	// Explicit required:"true" should override defaultOptional
	err := CmdT[Params]{
		Use:   "test",
		Short: "test command",
		RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
			t.Fatal("RunFunc should not be called when required field is missing")
		},
	}.RunArgsE([]string{})
	if err == nil {
		t.Fatal("expected error for explicitly required field even with defaultOptional")
	}
}

func TestExplicitOptionalTagOverridesDefault(t *testing.T) {
	defer resetGlobalConfig()

	type Params struct {
		Name string `descr:"a name" optional:"true"`
	}

	// No Init - default is required, but explicit optional:"true" overrides
	err := CmdT[Params]{
		Use:   "test",
		Short: "test command",
		RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
		},
	}.RunArgsE([]string{})
	if err != nil {
		t.Fatalf("expected no error for explicitly optional field, got: %v", err)
	}
}

func TestRequiredWrapperUnaffectedByDefaultOptional(t *testing.T) {
	defer resetGlobalConfig()

	type Params struct {
		Name Required[string] `descr:"a name"`
	}

	Init(WithDefaultOptional())

	// Required[T] wrapper should still be required regardless of defaultOptional
	err := CmdT[Params]{
		Use:   "test",
		Short: "test command",
		RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
			t.Fatal("RunFunc should not be called when required field is missing")
		},
	}.RunArgsE([]string{})
	if err == nil {
		t.Fatal("expected error for Required[T] field even with defaultOptional")
	}
}

func TestOptionalWrapperUnaffectedByDefault(t *testing.T) {
	defer resetGlobalConfig()

	type Params struct {
		Name Optional[string] `descr:"a name"`
	}

	// No Init - default is required, but Optional[T] should still be optional
	err := CmdT[Params]{
		Use:   "test",
		Short: "test command",
		RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
		},
	}.RunArgsE([]string{})
	if err != nil {
		t.Fatalf("expected no error for Optional[T] field, got: %v", err)
	}
}
