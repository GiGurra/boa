package boa

import (
	"os"
	"slices"
	"strconv"
	"testing"

	"github.com/spf13/cobra"
)

type RawConfig struct {
	Host   Required[string] `long:"host" env:"HOST"`
	Port   int              `long:"port" env:"PORT" default:"8080"`
	Extra1 string           `long:"extra1" env:"EXTRA1" required:"false"`
	Extra2 string           `long:"extra2" env:"EXTRA2" optional:"true"`
	Extra3 string           `long:"extra3" env:"EXTRA3" required:"true" default:"blah"`
	Extra4 string           `long:"extra4" env:"EXTRA4" required:"true"`
	Extra5 string           `long:"extra5" env:"EXTRA5" required:"true"`
	Extra6 string           `long:"extra6" env:"EXTRA6" required:"true" default:"error"`
}

func TestRawConfig(t *testing.T) {

	expected := RawConfig{
		Host:   Req("someHost"),
		Port:   12345,
		Extra1: "ex1",
		Extra2: "ex2",
		Extra3: "blah",          // default is used
		Extra4: "from-file",     // config file value is used
		Extra5: "not-from-file", // config file value is overridden by cli arg
		Extra6: "from-env",      // env var is used
	}

	config := RawConfig{}

	err := os.Setenv("EXTRA6", "from-env")
	if err != nil {
		t.Fatalf("Error setting env var: %v", err)
	}
	defer func() { _ = os.Unsetenv("EXTRA6") }()

	err = NewCmdT2("root", &config).
		WithPreValidateFunc(func(params *RawConfig, cmd *cobra.Command, args []string) {
			params.Extra4 = "from-file"
			params.Extra5 = "from-file"
		}).
		WithRawArgs([]string{
			"--host", expected.Host.Value(),
			"--port", strconv.Itoa(expected.Port),
			"--extra1", expected.Extra1,
			"--extra2", expected.Extra2,
			"--extra5", "not-from-file",
		}).
		Validate()

	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	if config.Host.Value() != expected.Host.Value() {
		t.Fatalf("Expected Host: %v, got: %v", expected.Host.Value(), config.Host.Value())
	}

	if config.Port != expected.Port {
		t.Fatalf("Expected Port: %v, got: %v", expected.Port, config.Port)
	}

	if config.Extra1 != expected.Extra1 {
		t.Fatalf("Expected Extra1: %v, got: %v", expected.Extra1, config.Extra1)
	}

	if config.Extra2 != expected.Extra2 {
		t.Fatalf("Expected Extra2: %v, got: %v", expected.Extra2, config.Extra2)
	}

	if config.Extra3 != expected.Extra3 {
		t.Fatalf("Expected Extra3: %v, got: %v", expected.Extra3, config.Extra3)
	}

	if config.Extra4 != expected.Extra4 {
		t.Fatalf("Expected Extra4: %v, got: %v", expected.Extra4, config.Extra4)
	}

	if config.Extra5 != expected.Extra5 {
		t.Fatalf("Expected Extra5: %v, got: %v", expected.Extra5, config.Extra5)
	}

	if config.Extra6 != expected.Extra6 {
		t.Fatalf("Expected Extra6: %v, got: %v", expected.Extra6, config.Extra6)
	}
}

// Tests for context-aware hooks and GetParam

type RawParamsWithCtx struct {
	Name    string `optional:"true"` // raw field
	Count   int    `optional:"true"` // raw field
	Verbose bool   `optional:"true"` // raw field
}

func TestInitFuncCtx_SetDefault(t *testing.T) {
	ran := false
	config := RawParamsWithCtx{}

	NewCmdT2("test", &config).
		WithInitFuncCtx(func(ctx *HookContext, params *RawParamsWithCtx, cmd *cobra.Command) error {
			// Use GetParam to set default on raw field
			nameParam := ctx.GetParam(&params.Name)
			if nameParam == nil {
				t.Fatal("expected to get param for Name")
			}
			nameParam.SetDefault(Default("default-name"))

			countParam := ctx.GetParam(&params.Count)
			countParam.SetDefault(Default(42))

			return nil
		}).
		WithRunFunc(func(params *RawParamsWithCtx) {
			ran = true
			if params.Name != "default-name" {
				t.Fatalf("expected Name to be 'default-name' but got '%s'", params.Name)
			}
			if params.Count != 42 {
				t.Fatalf("expected Count to be 42 but got %d", params.Count)
			}
		}).
		WithParamEnrich(ParamEnricherCombine(ParamEnricherBool, ParamEnricherName)).
		RunArgs([]string{})

	if !ran {
		t.Fatal("expected command to run")
	}
}

func TestInitFuncCtx_SetAlternatives(t *testing.T) {
	ran := false
	config := RawParamsWithCtx{}

	NewCmdT2("test", &config).
		WithInitFuncCtx(func(ctx *HookContext, params *RawParamsWithCtx, cmd *cobra.Command) error {
			nameParam := ctx.GetParam(&params.Name)
			nameParam.SetAlternatives([]string{"alice", "bob", "carol"})
			nameParam.SetStrictAlts(true)
			return nil
		}).
		WithRunFunc(func(params *RawParamsWithCtx) {
			ran = true
			if params.Name != "bob" {
				t.Fatalf("expected Name to be 'bob' but got '%s'", params.Name)
			}
		}).
		RunArgs([]string{"--name", "bob"})

	if !ran {
		t.Fatal("expected command to run")
	}
}

func TestInitFuncCtx_SetAlternatives_Invalid(t *testing.T) {
	config := RawParamsWithCtx{}

	err := NewCmdT2("test", &config).
		WithInitFuncCtx(func(ctx *HookContext, params *RawParamsWithCtx, cmd *cobra.Command) error {
			nameParam := ctx.GetParam(&params.Name)
			nameParam.SetAlternatives([]string{"alice", "bob", "carol"})
			nameParam.SetStrictAlts(true)
			return nil
		}).
		WithRunFunc(func(params *RawParamsWithCtx) {
			t.Fatal("should not run with invalid alternative")
		}).
		WithRawArgs([]string{"--name", "invalid"}).
		Validate()

	if err == nil {
		t.Fatal("expected validation error for invalid alternative")
	}
}

func TestInitFuncCtx_SetEnv(t *testing.T) {
	ran := false
	config := RawParamsWithCtx{}

	err := os.Setenv("TEST_CUSTOM_NAME", "from-env")
	if err != nil {
		t.Fatalf("Error setting env var: %v", err)
	}
	defer func() { _ = os.Unsetenv("TEST_CUSTOM_NAME") }()

	NewCmdT2("test", &config).
		WithInitFuncCtx(func(ctx *HookContext, params *RawParamsWithCtx, cmd *cobra.Command) error {
			nameParam := ctx.GetParam(&params.Name)
			nameParam.SetEnv("TEST_CUSTOM_NAME")
			return nil
		}).
		WithRunFunc(func(params *RawParamsWithCtx) {
			ran = true
			if params.Name != "from-env" {
				t.Fatalf("expected Name to be 'from-env' but got '%s'", params.Name)
			}
		}).
		RunArgs([]string{})

	if !ran {
		t.Fatal("expected command to run")
	}
}

func TestGetParam_WorksForBothRawAndWrapped(t *testing.T) {
	type MixedParams struct {
		RawName     string         // raw field
		WrappedAge  Required[int]  // wrapped field
		WrappedFlag Optional[bool] // wrapped optional field
	}

	ran := false
	config := MixedParams{}

	NewCmdT2("test", &config).
		WithInitFuncCtx(func(ctx *HookContext, params *MixedParams, cmd *cobra.Command) error {
			// GetParam should work for raw fields
			rawParam := ctx.GetParam(&params.RawName)
			if rawParam == nil {
				t.Fatal("expected to get param for RawName")
			}
			rawParam.SetDefault(Default("raw-default"))

			// GetParam should work for wrapped fields too
			wrappedParam := ctx.GetParam(&params.WrappedAge)
			if wrappedParam == nil {
				t.Fatal("expected to get param for WrappedAge")
			}
			wrappedParam.SetDefault(Default(25))

			// GetParam should work for optional wrapped fields
			optionalParam := ctx.GetParam(&params.WrappedFlag)
			if optionalParam == nil {
				t.Fatal("expected to get param for WrappedFlag")
			}
			optionalParam.SetDefault(Default(true))

			return nil
		}).
		WithRunFunc(func(params *MixedParams) {
			ran = true
			if params.RawName != "raw-default" {
				t.Fatalf("expected RawName to be 'raw-default' but got '%s'", params.RawName)
			}
			if params.WrappedAge.Value() != 25 {
				t.Fatalf("expected WrappedAge to be 25 but got %d", params.WrappedAge.Value())
			}
			if !params.WrappedFlag.HasValue() || *params.WrappedFlag.Value() != true {
				t.Fatalf("expected WrappedFlag to be true")
			}
		}).
		RunArgs([]string{})

	if !ran {
		t.Fatal("expected command to run")
	}
}

func TestAllMirrors(t *testing.T) {
	type MultiRawParams struct {
		Field1 string `optional:"true"`
		Field2 int    `optional:"true"`
		Field3 bool   `optional:"true"`
	}

	config := MultiRawParams{}
	var mirrorCount int

	NewCmdT2("test", &config).
		WithInitFuncCtx(func(ctx *HookContext, params *MultiRawParams, cmd *cobra.Command) error {
			mirrors := ctx.AllMirrors()
			mirrorCount = len(mirrors)
			return nil
		}).
		WithRunFunc(func(params *MultiRawParams) {}).
		RunArgs([]string{})

	if mirrorCount != 3 {
		t.Fatalf("expected 3 mirrors but got %d", mirrorCount)
	}
}

// Interface-based context hooks

type ParamsWithInitCtx struct {
	Name string `optional:"true"`
}

func (p *ParamsWithInitCtx) InitCtx(ctx *HookContext) error {
	param := ctx.GetParam(&p.Name)
	param.SetDefault(Default("interface-default"))
	param.SetAlternatives([]string{"interface-default", "other"})
	return nil
}

func TestCfgStructInitCtx(t *testing.T) {
	ran := false
	config := ParamsWithInitCtx{}

	NewCmdT2("test", &config).
		WithRunFunc(func(params *ParamsWithInitCtx) {
			ran = true
			if params.Name != "interface-default" {
				t.Fatalf("expected Name to be 'interface-default' but got '%s'", params.Name)
			}
		}).
		WithParamEnrich(ParamEnricherName).
		RunArgs([]string{})

	if !ran {
		t.Fatal("expected command to run")
	}
}

type ParamsWithPreValidateCtx struct {
	Name  string `optional:"true"`
	Count int    `optional:"true"`
}

func (p *ParamsWithPreValidateCtx) PreValidateCtx(ctx *HookContext) error {
	// Can inspect/modify params after parsing but before validation
	if p.Name == "" {
		p.Name = "pre-validate-default"
	}
	return nil
}

func TestCfgStructPreValidateCtx(t *testing.T) {
	ran := false
	config := ParamsWithPreValidateCtx{}

	NewCmdT2("test", &config).
		WithRunFunc(func(params *ParamsWithPreValidateCtx) {
			ran = true
			if params.Name != "pre-validate-default" {
				t.Fatalf("expected Name to be 'pre-validate-default' but got '%s'", params.Name)
			}
			if params.Count != 100 {
				t.Fatalf("expected Count to be 100 but got %d", params.Count)
			}
		}).
		WithParamEnrich(ParamEnricherName).
		RunArgs([]string{"--count", "100"})

	if !ran {
		t.Fatal("expected command to run")
	}
}

type ParamsWithPreExecuteCtx struct {
	Input  string `optional:"true"`
	Output string `optional:"true"`
}

func (p *ParamsWithPreExecuteCtx) PreExecuteCtx(ctx *HookContext) error {
	// Can set derived values after validation
	if p.Output == "" {
		p.Output = p.Input + "-processed"
	}
	return nil
}

func TestCfgStructPreExecuteCtx(t *testing.T) {
	ran := false
	config := ParamsWithPreExecuteCtx{}

	NewCmdT2("test", &config).
		WithRunFunc(func(params *ParamsWithPreExecuteCtx) {
			ran = true
			if params.Input != "test-input" {
				t.Fatalf("expected Input to be 'test-input' but got '%s'", params.Input)
			}
			if params.Output != "test-input-processed" {
				t.Fatalf("expected Output to be 'test-input-processed' but got '%s'", params.Output)
			}
		}).
		RunArgs([]string{"--input", "test-input"})

	if !ran {
		t.Fatal("expected command to run")
	}
}

type ParamsWithPostCreate struct {
	Name       string `optional:"true"`
	WasCalled  bool
	FlagExists bool
}

func (p *ParamsWithPostCreate) PostCreate() error {
	p.WasCalled = true
	return nil
}

func TestCfgStructPostCreate(t *testing.T) {
	ran := false
	config := ParamsWithPostCreate{}

	NewCmdT2("test", &config).
		WithRunFunc(func(params *ParamsWithPostCreate) {
			ran = true
			if !params.WasCalled {
				t.Fatal("expected PostCreate to be called")
			}
		}).
		RunArgs([]string{})

	if !ran {
		t.Fatal("expected command to run")
	}
}

type ParamsWithPostCreateCtx struct {
	Name       string `optional:"true"`
	FlagExists bool
}

func (p *ParamsWithPostCreateCtx) PostCreateCtx(ctx *HookContext) error {
	// Verify we can access param mirrors after flags are created
	param := ctx.GetParam(&p.Name)
	p.FlagExists = param != nil && param.GetName() == "name"
	return nil
}

func TestCfgStructPostCreateCtx(t *testing.T) {
	ran := false
	config := ParamsWithPostCreateCtx{}

	NewCmdT2("test", &config).
		WithRunFunc(func(params *ParamsWithPostCreateCtx) {
			ran = true
			if !params.FlagExists {
				t.Fatal("expected PostCreateCtx to find the 'name' param")
			}
		}).
		RunArgs([]string{})

	if !ran {
		t.Fatal("expected command to run")
	}
}

func TestPreValidateFuncCtx(t *testing.T) {
	ran := false
	config := RawParamsWithCtx{}

	NewCmdT2("test", &config).
		WithPreValidateFuncCtx(func(ctx *HookContext, params *RawParamsWithCtx, cmd *cobra.Command, args []string) error {
			// Inspect mirrors after parsing
			nameParam := ctx.GetParam(&params.Name)
			if nameParam == nil {
				t.Fatal("expected to get param for Name in PreValidateFuncCtx")
			}
			// Can check if value was set
			if !nameParam.HasValue() {
				params.Name = "pre-validate-fallback"
			}
			return nil
		}).
		WithParamEnrich(ParamEnricherName).
		WithRunFunc(func(params *RawParamsWithCtx) {
			ran = true
			if params.Name != "pre-validate-fallback" {
				t.Fatalf("expected Name to be 'pre-validate-fallback' but got '%s'", params.Name)
			}
		}).
		RunArgs([]string{})

	if !ran {
		t.Fatal("expected command to run")
	}
}

func TestPreExecuteFuncCtx(t *testing.T) {
	ran := false
	config := RawParamsWithCtx{}

	NewCmdT2("test", &config).
		WithInitFuncCtx(func(ctx *HookContext, params *RawParamsWithCtx, cmd *cobra.Command) error {
			ctx.GetParam(&params.Name).SetDefault(Default("initial"))
			return nil
		}).
		WithPreExecuteFuncCtx(func(ctx *HookContext, params *RawParamsWithCtx, cmd *cobra.Command, args []string) error {
			// Can access mirrors after validation
			nameParam := ctx.GetParam(&params.Name)
			alts := nameParam.GetAlternatives()
			// Verify we can read param state
			if nameParam.GetName() != "name" {
				t.Fatalf("expected param name to be 'name' but got '%s'", nameParam.GetName())
			}
			// alts should be nil since we didn't set any
			if alts != nil {
				t.Fatalf("expected no alternatives but got %v", alts)
			}
			return nil
		}).
		WithRunFunc(func(params *RawParamsWithCtx) {
			ran = true
		}).
		RunArgs([]string{})

	if !ran {
		t.Fatal("expected command to run")
	}
}

func TestPostCreateFuncCtx(t *testing.T) {
	ran := false
	config := RawParamsWithCtx{}
	var flagExists bool

	NewCmdT2("test", &config).
		WithPostCreateFuncCtx(func(ctx *HookContext, params *RawParamsWithCtx, cmd *cobra.Command) error {
			// At this point cobra flags have been created
			flag := cmd.Flags().Lookup("name")
			flagExists = flag != nil
			return nil
		}).
		WithRunFunc(func(params *RawParamsWithCtx) {
			ran = true
		}).
		RunArgs([]string{})

	if !ran {
		t.Fatal("expected command to run")
	}
	if !flagExists {
		t.Fatal("expected 'name' flag to exist in PostCreateFuncCtx")
	}
}

func TestGetParam_ReturnsNilForUnknownField(t *testing.T) {
	type Params struct {
		Name string `optional:"true"`
	}

	config := Params{}
	var unknownResult Param

	NewCmdT2("test", &config).
		WithInitFuncCtx(func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			// Try to get param for a local variable (not a field)
			localVar := "test"
			unknownResult = ctx.GetParam(&localVar)
			return nil
		}).
		WithRunFunc(func(params *Params) {}).
		RunArgs([]string{})

	if unknownResult != nil {
		t.Fatal("expected GetParam to return nil for unknown field")
	}
}

func TestGetParam_ParamMethods(t *testing.T) {
	type Params struct {
		Name string `short:"n" env:"TEST_NAME" descr:"The name" optional:"true"`
	}

	config := Params{}

	// Use PostCreateFuncCtx where names have been enriched
	NewCmdT2("test", &config).
		WithPostCreateFuncCtx(func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			param := ctx.GetParam(&params.Name)

			// Test various getter methods (available after enrichment)
			if param.GetName() != "name" {
				t.Fatalf("expected name 'name' but got '%s'", param.GetName())
			}
			if param.GetShort() != "n" {
				t.Fatalf("expected short 'n' but got '%s'", param.GetShort())
			}
			if param.GetEnv() != "TEST_NAME" {
				t.Fatalf("expected env 'TEST_NAME' but got '%s'", param.GetEnv())
			}
			// Note: optional:"true" makes it not required
			if param.IsRequired() != false {
				t.Fatal("expected IsRequired to be false for optional raw field")
			}

			// Test setters
			param.SetAlternatives([]string{"a", "b", "c"})
			alts := param.GetAlternatives()
			if !slices.Equal(alts, []string{"a", "b", "c"}) {
				t.Fatalf("expected alternatives [a b c] but got %v", alts)
			}

			return nil
		}).
		WithRunFunc(func(params *Params) {}).
		RunArgs([]string{"--name", "a"})
}

// Tests for RunFuncCtx and HookContext.HasValue

func TestRunFuncCtx_Basic(t *testing.T) {
	type Params struct {
		Name string `optional:"true" default:"default-name"`
		Port int    `optional:"true" default:"8080"`
	}

	ran := false
	config := Params{}

	NewCmdT2("test", &config).
		WithRunFuncCtx(func(ctx *HookContext, params *Params) {
			ran = true
			if params.Name != "custom-name" {
				t.Fatalf("expected Name to be 'custom-name' but got '%s'", params.Name)
			}
			if params.Port != 8080 {
				t.Fatalf("expected Port to be 8080 but got %d", params.Port)
			}
		}).
		RunArgs([]string{"--name", "custom-name"})

	if !ran {
		t.Fatal("expected command to run")
	}
}

func TestRunFuncCtx4_FullSignature(t *testing.T) {
	type Params struct {
		Name string `optional:"true"`
	}

	ran := false
	config := Params{}

	NewCmdT2("test", &config).
		WithRunFuncCtx4(func(ctx *HookContext, params *Params, cmd *cobra.Command, args []string) {
			ran = true
			if cmd == nil {
				t.Fatal("expected cmd to not be nil")
			}
			if params.Name != "test-name" {
				t.Fatalf("expected Name to be 'test-name' but got '%s'", params.Name)
			}
		}).
		RunArgs([]string{"--name", "test-name"})

	if !ran {
		t.Fatal("expected command to run")
	}
}

func TestRunFuncCtx_HasValue_SetByCli(t *testing.T) {
	type Params struct {
		Name string `optional:"true" default:"default-name"`
		Port int    `optional:"true" default:"8080"`
	}

	ran := false
	config := Params{}

	NewCmdT2("test", &config).
		WithRunFuncCtx(func(ctx *HookContext, params *Params) {
			ran = true

			// Name was set via CLI
			if !ctx.HasValue(&params.Name) {
				t.Fatal("expected HasValue to return true for Name (set via CLI)")
			}

			// Port was not set via CLI but has default
			if !ctx.HasValue(&params.Port) {
				t.Fatal("expected HasValue to return true for Port (has default)")
			}
		}).
		RunArgs([]string{"--name", "custom-name"})

	if !ran {
		t.Fatal("expected command to run")
	}
}

func TestRunFuncCtx_HasValue_SetByEnv(t *testing.T) {
	type Params struct {
		Name string `optional:"true" env:"TEST_RUN_NAME"`
	}

	ran := false
	config := Params{}

	err := os.Setenv("TEST_RUN_NAME", "from-env")
	if err != nil {
		t.Fatalf("Error setting env var: %v", err)
	}
	defer func() { _ = os.Unsetenv("TEST_RUN_NAME") }()

	NewCmdT2("test", &config).
		WithRunFuncCtx(func(ctx *HookContext, params *Params) {
			ran = true

			if !ctx.HasValue(&params.Name) {
				t.Fatal("expected HasValue to return true for Name (set via env)")
			}
			if params.Name != "from-env" {
				t.Fatalf("expected Name to be 'from-env' but got '%s'", params.Name)
			}
		}).
		RunArgs([]string{})

	if !ran {
		t.Fatal("expected command to run")
	}
}

func TestRunFuncCtx_HasValue_NoValueSet(t *testing.T) {
	type Params struct {
		Name string `optional:"true"`
	}

	ran := false
	config := Params{}

	NewCmdT2("test", &config).
		WithRunFuncCtx(func(ctx *HookContext, params *Params) {
			ran = true

			// Name has no default and was not set
			if ctx.HasValue(&params.Name) {
				t.Fatal("expected HasValue to return false for Name (no value set)")
			}
		}).
		WithParamEnrich(ParamEnricherName).
		RunArgs([]string{})

	if !ran {
		t.Fatal("expected command to run")
	}
}

func TestRunFuncCtx_HasValue_WithWrappedParams(t *testing.T) {
	type Params struct {
		Name Required[string]
		Port Optional[int]
	}

	ran := false
	config := Params{}

	NewCmdT2("test", &config).
		WithRunFuncCtx(func(ctx *HookContext, params *Params) {
			ran = true

			// Name was set via CLI
			if !ctx.HasValue(&params.Name) {
				t.Fatal("expected HasValue to return true for Name (set via CLI)")
			}

			// Port was not set
			if ctx.HasValue(&params.Port) {
				t.Fatal("expected HasValue to return false for Port (not set)")
			}
		}).
		RunArgs([]string{"--name", "test"})

	if !ran {
		t.Fatal("expected command to run")
	}
}

func TestRunFuncCtx_PanicsWhenBothRunFuncsSet(t *testing.T) {
	type Params struct {
		Name string `optional:"true"`
	}

	config := Params{}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when both RunFunc and RunFuncCtx are set")
		}
	}()

	// This should panic because we're setting both RunFunc and RunFuncCtx
	NewCmdT2("test", &config).
		WithRunFunc(func(params *Params) {}).
		WithRunFuncCtx(func(ctx *HookContext, params *Params) {}).
		RunArgs([]string{})
}
