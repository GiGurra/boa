package boa

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// --- boa:"noflag" tag ---

func TestNoFlag_NoCLIFlagRegistered(t *testing.T) {
	type Params struct {
		Name   string `descr:"public name"`
		Secret string `descr:"api token" boa:"noflag" env:"API_TOKEN" optional:"true"`
	}

	// --secret should be rejected as an unknown flag.
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--name", "alice", "--secret", "hunter2"})

	if err == nil {
		t.Fatal("expected unknown-flag error for --secret when tagged boa:\"noflag\"")
	}
	if !strings.Contains(err.Error(), "secret") {
		t.Errorf("expected error mentioning 'secret', got: %v", err)
	}

	// And the help output should not list --secret.
	usage := captureUsage(t, CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	})
	if strings.Contains(usage, "--secret") || strings.Contains(usage, "api token") {
		t.Errorf("noflag field should not appear in --help:\n%s", usage)
	}
}

func TestNoFlag_AliasNoCLI(t *testing.T) {
	type Params struct {
		Name   string `descr:"public name"`
		Secret string `descr:"api token" boa:"nocli" optional:"true"`
	}
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--name", "alice", "--secret", "hunter2"})
	if err == nil {
		t.Fatal("expected unknown-flag error for --secret (boa:\"nocli\" alias)")
	}
}

func TestNoFlag_StillReadsEnv(t *testing.T) {
	type Params struct {
		Name   string `descr:"public name"`
		Secret string `descr:"api token" boa:"noflag" env:"BOA_NOFLAG_TEST_TOKEN" optional:"true"`
	}

	t.Setenv("BOA_NOFLAG_TEST_TOKEN", "hunter2")

	var gotSecret string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotSecret = p.Secret
		},
	}).RunArgsE([]string{"--name", "alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotSecret != "hunter2" {
		t.Errorf("expected noflag field to be populated from env, got %q", gotSecret)
	}
}

func TestNoFlag_StillReadsConfigFile(t *testing.T) {
	type Params struct {
		ConfigFile string `configfile:"true" default:"" optional:"true"`
		Name       string `descr:"public name"`
		Secret     string `descr:"api token" boa:"noflag" optional:"true"`
	}

	cfgData, _ := json.Marshal(map[string]any{
		"Name":   "alice",
		"Secret": "from-config",
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	var gotSecret string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotSecret = p.Secret
		},
	}).RunArgsE([]string{"--config-file", cfgPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotSecret != "from-config" {
		t.Errorf("expected noflag field loaded from config, got %q", gotSecret)
	}
}

func TestNoFlag_ValidationStillRuns(t *testing.T) {
	// min/max/pattern should still apply to noflag fields.
	type Params struct {
		Name string `descr:"public name"`
		Port int    `boa:"noflag" env:"BOA_NOFLAG_TEST_PORT" min:"1" max:"65535" optional:"true"`
	}

	t.Setenv("BOA_NOFLAG_TEST_PORT", "99999")

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--name", "alice"})
	if err == nil {
		t.Fatal("expected max-value validation error for noflag field")
	}
	if !strings.Contains(err.Error(), "max") {
		t.Errorf("expected error mentioning 'max', got: %v", err)
	}
}

func TestNoFlag_DefaultValueApplies(t *testing.T) {
	type Params struct {
		Name    string `descr:"public name"`
		Timeout int    `boa:"noflag" default:"30" optional:"true"`
	}

	var gotTimeout int
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotTimeout = p.Timeout
		},
	}).RunArgsE([]string{"--name", "alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotTimeout != 30 {
		t.Errorf("expected default=30 to apply, got %d", gotTimeout)
	}
}

func TestNoFlag_RequiredAndMissingFails(t *testing.T) {
	// noflag + required + no env/config value → should fail required check.
	type Params struct {
		Secret string `boa:"noflag"` // no optional → required
	}
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{})
	if err == nil {
		t.Fatal("expected missing-required error for required noflag field")
	}
	if !strings.Contains(err.Error(), "missing required param") {
		t.Errorf("expected 'missing required param' in error, got: %v", err)
	}
}

func TestNoFlag_WithPositionalIsError(t *testing.T) {
	// noflag + positional is nonsensical — should error at setup time.
	type Params struct {
		Arg string `boa:"noflag" positional:"true"`
	}
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"val"})
	if err == nil {
		t.Fatal("expected error for noflag+positional combo")
	}
	if !strings.Contains(err.Error(), "positional") {
		t.Errorf("expected error mentioning 'positional', got: %v", err)
	}
}

func TestNoFlag_ProgrammaticViaHook(t *testing.T) {
	// Set noflag programmatically via InitFuncCtx on a field that has no tag.
	type Params struct {
		Name   string `descr:"public name"`
		Secret string `descr:"api token" env:"BOA_NOFLAG_HOOK_TOKEN" optional:"true"`
	}

	t.Setenv("BOA_NOFLAG_HOOK_TOKEN", "env-value")

	var gotSecret string
	var usage string
	cmd := CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			GetParamT(ctx, &params.Secret).SetNoFlag(true)
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotSecret = p.Secret
		},
	}
	// CLI flag should no longer exist.
	err := cmd.RunArgsE([]string{"--name", "alice", "--secret", "overridden"})
	if err == nil {
		t.Fatal("expected unknown-flag error after programmatic SetNoFlag(true)")
	}

	// Env var should still populate.
	err = cmd.RunArgsE([]string{"--name", "alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotSecret != "env-value" {
		t.Errorf("expected env-value, got %q", gotSecret)
	}

	usage = captureUsage(t, cmd)
	if strings.Contains(usage, "--secret") {
		t.Errorf("--secret should not appear in help after SetNoFlag:\n%s", usage)
	}
}

// --- boa:"noenv" tag ---

func TestNoEnv_EnvIsSkipped(t *testing.T) {
	type Params struct {
		Name string `descr:"name" boa:"noenv" env:"BOA_NOENV_TEST_NAME" optional:"true"`
	}

	t.Setenv("BOA_NOENV_TEST_NAME", "from-env")

	var got string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = p.Name
		},
	}).RunArgsE([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected env to be ignored, got %q", got)
	}
}

func TestNoEnv_CLIStillWorks(t *testing.T) {
	type Params struct {
		Name string `descr:"name" boa:"noenv" optional:"true"`
	}
	var got string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = p.Name
		},
	}).RunArgsE([]string{"--name", "alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "alice" {
		t.Errorf("expected CLI value 'alice', got %q", got)
	}
}

func TestNoEnv_SuppressesAutoGeneratedEnvName(t *testing.T) {
	// The primary use case for `boa:"noenv"` is combining with ParamEnricherEnv:
	// the auto-generated env var binding should be suppressed for tagged fields.
	type Params struct {
		Name     string `descr:"public name" optional:"true"`
		Internal string `descr:"internal knob" boa:"noenv" optional:"true"`
	}

	// ParamEnricherEnv would normally map Internal → INTERNAL.
	t.Setenv("INTERNAL", "leaked-from-env")
	t.Setenv("NAME", "alice")

	var gotName, gotInternal string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherCombine(ParamEnricherName, ParamEnricherEnv),
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotName = p.Name
			gotInternal = p.Internal
		},
	}).RunArgsE([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName != "alice" {
		t.Errorf("ParamEnricherEnv should still populate Name from $NAME, got %q", gotName)
	}
	if gotInternal != "" {
		t.Errorf("boa:\"noenv\" should suppress $INTERNAL, got %q", gotInternal)
	}
}

func TestNoEnv_ProgrammaticViaHook(t *testing.T) {
	type Params struct {
		Name string `descr:"name" env:"BOA_NOENV_HOOK_NAME" optional:"true"`
	}
	t.Setenv("BOA_NOENV_HOOK_NAME", "from-env")

	var got string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			GetParamT(ctx, &params.Name).SetNoEnv(true)
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = p.Name
		},
	}).RunArgsE([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected env ignored after SetNoEnv, got %q", got)
	}
}

// --- Programmatic SetIgnored / SetDescription / SetMin / SetMax / SetPattern / SetRequired ---

func TestSetIgnored_ProgrammaticSkipsEverything(t *testing.T) {
	type Params struct {
		Name     string `descr:"name" optional:"true"`
		Internal string `descr:"internal" env:"BOA_IGNORED_TEST" optional:"true"`
	}
	t.Setenv("BOA_IGNORED_TEST", "from-env")

	var gotInternal string
	cmd := CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			GetParamT(ctx, &params.Internal).SetIgnored(true)
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotInternal = p.Internal
		},
	}
	// Ignored: --internal should not exist as a flag
	err := cmd.RunArgsE([]string{"--internal", "x"})
	if err == nil {
		t.Fatal("expected unknown-flag error for --internal after SetIgnored")
	}
	// Ignored: env should also be ignored
	err = cmd.RunArgsE([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotInternal != "" {
		t.Errorf("expected env ignored, got %q", gotInternal)
	}
}

func TestSetDescription_Programmatic(t *testing.T) {
	type Params struct {
		Name string `descr:"original" optional:"true"`
	}
	usage := captureUsage(t, CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			GetParamT(ctx, &params.Name).SetDescription("overridden help text")
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	})
	if !strings.Contains(usage, "overridden help text") {
		t.Errorf("expected 'overridden help text' in usage:\n%s", usage)
	}
	if strings.Contains(usage, "original") {
		t.Errorf("expected original description replaced:\n%s", usage)
	}
}

func TestSetMinMax_Programmatic(t *testing.T) {
	type Params struct {
		Port int `descr:"port" optional:"true"`
	}
	minV := float64(1)
	maxV := float64(100)
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			p := GetParamT(ctx, &params.Port)
			p.SetMin(&minV)
			p.SetMax(&maxV)
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--port", "500"})
	if err == nil {
		t.Fatal("expected validation error from programmatic max")
	}
	if !strings.Contains(err.Error(), "max") {
		t.Errorf("expected error mentioning 'max', got: %v", err)
	}
}

func TestSetPattern_Programmatic(t *testing.T) {
	type Params struct {
		Name string `optional:"true"`
	}
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			GetParamT(ctx, &params.Name).SetPattern(`^[a-z]+$`)
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--name", "Has-Digits-123"})
	if err == nil {
		t.Fatal("expected pattern validation failure")
	}
	if !strings.Contains(err.Error(), "pattern") {
		t.Errorf("expected error mentioning 'pattern', got: %v", err)
	}
}

func TestSetRequired_ProgrammaticTrue(t *testing.T) {
	// Start as optional via tag, then flip required at runtime.
	type Params struct {
		Name string `optional:"true"`
	}
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			GetParamT(ctx, &params.Name).SetRequired(true)
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{})
	if err == nil {
		t.Fatal("expected required-check failure after SetRequired(true)")
	}
}

func TestSetPositional_Programmatic(t *testing.T) {
	// Field without positional tag — flip it programmatically.
	type Params struct {
		Target string `optional:"true"`
	}
	var got string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			GetParamT(ctx, &params.Target).SetPositional(true)
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = p.Target
		},
	}).RunArgsE([]string{"hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello" {
		t.Errorf("expected positional value 'hello', got %q", got)
	}
}

func TestSetRequired_ProgrammaticFalse(t *testing.T) {
	// Start required (plain string), then flip optional at runtime.
	type Params struct {
		Name string
	}
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			GetParamT(ctx, &params.Name).SetRequired(false)
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- helpers ---

func captureUsage[P any](t *testing.T, spec CmdT[P]) string {
	t.Helper()
	cmd, err := spec.ToCobraE()
	if err != nil {
		t.Fatalf("ToCobraE failed: %v", err)
	}
	return cmd.UsageString()
}
