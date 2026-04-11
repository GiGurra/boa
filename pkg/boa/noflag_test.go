package boa

import (
	"encoding/json"
	"fmt"
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

func TestNoFlag_CustomValidatorStillRuns(t *testing.T) {
	// Custom validators should still fire for noflag fields.
	type Params struct {
		Name  string `descr:"public name"`
		Token string `descr:"token" boa:"noflag" env:"BOA_NOFLAG_CV_TOKEN" optional:"true"`
	}
	t.Setenv("BOA_NOFLAG_CV_TOKEN", "short")

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			GetParamT(ctx, &params.Token).SetCustomValidatorT(func(v string) error {
				if len(v) < 10 {
					return fmt.Errorf("token must be at least 10 chars, got %d", len(v))
				}
				return nil
			})
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--name", "alice"})
	if err == nil {
		t.Fatal("expected custom validator to reject short token on noflag field")
	}
	if !strings.Contains(err.Error(), "at least 10") {
		t.Errorf("expected custom validator error, got: %v", err)
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

func TestSetIgnored_WithPositionalIsError(t *testing.T) {
	// Programmatically marking a positional field ignored must be rejected:
	// a positional needs argv consumption, which the ignored branch short-
	// circuits. Without this check the positional bookkeeping would hold
	// an entry with no binder attached.
	type Params struct {
		Target string `positional:"true"`
	}
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			GetParamT(ctx, &params.Target).SetIgnored(true)
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"hello"})
	if err == nil {
		t.Fatal("expected error for ignored+positional combo")
	}
	if !strings.Contains(err.Error(), "positional") {
		t.Errorf("expected error mentioning 'positional', got: %v", err)
	}
}

func TestSetNoFlag_WithPositionalProgrammaticIsError(t *testing.T) {
	// Same for flipping noflag on a positional field programmatically.
	type Params struct {
		Target string `positional:"true"`
	}
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			GetParamT(ctx, &params.Target).SetNoFlag(true)
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"hello"})
	if err == nil {
		t.Fatal("expected error for noflag+positional combo set programmatically")
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
		BoaNoenvPublic   string `descr:"public name" optional:"true"`
		BoaNoenvInternal string `descr:"internal knob" boa:"noenv" optional:"true"`
	}

	// ParamEnricherEnv would normally map these to BOA_NOENV_PUBLIC / BOA_NOENV_INTERNAL.
	t.Setenv("BOA_NOENV_PUBLIC", "alice")
	t.Setenv("BOA_NOENV_INTERNAL", "leaked-from-env")

	var gotName, gotInternal string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherCombine(ParamEnricherName, ParamEnricherEnv),
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotName = p.BoaNoenvPublic
			gotInternal = p.BoaNoenvInternal
		},
	}).RunArgsE([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName != "alice" {
		t.Errorf("ParamEnricherEnv should still populate Public from $BOA_NOENV_PUBLIC, got %q", gotName)
	}
	if gotInternal != "" {
		t.Errorf("boa:\"noenv\" should suppress $BOA_NOENV_INTERNAL, got %q", gotInternal)
	}
}

func TestNoEnv_SuppressesNamedStructAutoPrefix(t *testing.T) {
	// Named substruct fields auto-prefix their children (DB.Host → --db-host,
	// $DB_HOST). noenv on the child field should suppress the prefixed env
	// binding too, while leaving the CLI flag intact.
	type DB struct {
		Host string `descr:"db host" optional:"true"`
		Pass string `descr:"db password" boa:"noenv" optional:"true"`
	}
	type Params struct {
		DB DB
	}

	t.Setenv("DB_HOST", "db.example.com")
	t.Setenv("DB_PASS", "leaked-from-env")

	var gotHost, gotPass string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherCombine(ParamEnricherName, ParamEnricherEnv),
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotHost = p.DB.Host
			gotPass = p.DB.Pass
		},
	}).RunArgsE([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHost != "db.example.com" {
		t.Errorf("expected DB.Host from $DB_HOST, got %q", gotHost)
	}
	if gotPass != "" {
		t.Errorf("boa:\"noenv\" should suppress $DB_PASS for nested field, got %q", gotPass)
	}

	// And the CLI flag should still work.
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherCombine(ParamEnricherName, ParamEnricherEnv),
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotPass = p.DB.Pass
		},
	}).RunArgsE([]string{"--db-pass", "from-cli"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPass != "from-cli" {
		t.Errorf("CLI should still set noenv field, got %q", gotPass)
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
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			p := GetParamT(ctx, &params.Port)
			p.SetMinT(1)
			p.SetMaxT(100)
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

// --- SetMinT / SetMaxT / SetPattern type guards ---

func TestSetMinT_PanicsOnUnsupportedType(t *testing.T) {
	// bool isn't numeric and isn't length-based — SetMinT must panic with the
	// "numeric T required" message.
	type Params struct {
		Flag bool `optional:"true"`
	}
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected SetMinT on bool to panic")
		}
		if !strings.Contains(fmt.Sprint(r), "SetMinT") {
			t.Errorf("expected panic mentioning SetMinT, got: %v", r)
		}
	}()
	_ = (CmdT[Params]{
		Use: "test",
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			GetParamT(ctx, &params.Flag).SetMinT(true)
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{})
}

func TestSetMaxT_PanicsOnUnsupportedType(t *testing.T) {
	type Params struct {
		Flag bool `optional:"true"`
	}
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected SetMaxT on bool to panic")
		}
	}()
	_ = (CmdT[Params]{
		Use: "test",
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			GetParamT(ctx, &params.Flag).SetMaxT(true)
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{})
}

func TestSetMinT_PanicsOnStringField(t *testing.T) {
	// string is length-based — SetMinT must redirect to SetMinLen.
	type Params struct {
		Name string `optional:"true"`
	}
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected SetMinT on string to panic")
		}
		msg := fmt.Sprint(r)
		if !strings.Contains(msg, "SetMinLen") {
			t.Errorf("expected panic to recommend SetMinLen, got: %v", r)
		}
	}()
	_ = (CmdT[Params]{
		Use: "test",
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			GetParamT(ctx, &params.Name).SetMinT("abc")
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{})
}

func TestSetMinLen_PanicsOnNumericField(t *testing.T) {
	// int is numeric — SetMinLen must redirect to SetMinT.
	type Params struct {
		Port int `optional:"true"`
	}
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected SetMinLen on int to panic")
		}
		msg := fmt.Sprint(r)
		if !strings.Contains(msg, "SetMinT") {
			t.Errorf("expected panic to recommend SetMinT, got: %v", r)
		}
	}()
	_ = (CmdT[Params]{
		Use: "test",
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			GetParamT(ctx, &params.Port).SetMinLen(1)
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{})
}

func TestSetPattern_PanicsOnNonString(t *testing.T) {
	type Params struct {
		Port int `optional:"true"`
	}
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected SetPattern on int to panic")
		}
		if !strings.Contains(fmt.Sprint(r), "SetPattern") {
			t.Errorf("expected panic mentioning SetPattern, got: %v", r)
		}
	}()
	_ = (CmdT[Params]{
		Use: "test",
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			GetParamT(ctx, &params.Port).SetPattern("^[0-9]+$")
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{})
}

func TestClearMin_AllowedOnUnsupportedType(t *testing.T) {
	// ClearMin is always safe — it's a no-op when no bound is set, and the
	// type guard only applies to SetMin.
	type Params struct {
		Flag bool `optional:"true"`
	}
	err := (CmdT[Params]{
		Use: "test",
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			GetParamT(ctx, &params.Flag).ClearMin() // no-op, must not panic
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- programmatic SetMin/SetMax across the supported type spectrum ---

func TestSetMin_Programmatic_BelowIntMin(t *testing.T) {
	type Params struct {
		Port int `descr:"port" optional:"true"`
	}
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			p := GetParamT(ctx, &params.Port)
			p.SetMinT(10)
			p.SetMaxT(100)
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--port", "5"})
	if err == nil {
		t.Fatal("expected validation error from programmatic min")
	}
	if !strings.Contains(err.Error(), "min") {
		t.Errorf("expected error mentioning 'min', got: %v", err)
	}
}

func TestSetMinMax_Programmatic_Float(t *testing.T) {
	type Params struct {
		Rate float64 `descr:"rate" optional:"true"`
	}
	for _, tc := range []struct {
		name    string
		arg     string
		wantErr string // substring to look for; "" = expect success
	}{
		{"below_min", "-0.1", "min"},
		{"above_max", "1.5", "max"},
		{"in_range", "0.5", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := (CmdT[Params]{
				Use:         "test",
				ParamEnrich: ParamEnricherName,
				InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
					p := GetParamT(ctx, &params.Rate)
					p.SetMinT(0)
					p.SetMaxT(1)
					return nil
				},
				RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
			}).RunArgsE([]string{"--rate", tc.arg})
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tc.wantErr, err)
			}
		})
	}
}

func TestSetMinMaxLen_Programmatic_StringLength(t *testing.T) {
	type Params struct {
		Name string `descr:"name" optional:"true"`
	}
	makeCmd := func() CmdT[Params] {
		return CmdT[Params]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
				p := GetParamT(ctx, &params.Name)
				p.SetMinLen(3)
				p.SetMaxLen(10)
				return nil
			},
			RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
		}
	}
	if err := makeCmd().RunArgsE([]string{"--name", "ab"}); err == nil || !strings.Contains(err.Error(), "min") {
		t.Errorf("expected min error for short name, got: %v", err)
	}
	if err := makeCmd().RunArgsE([]string{"--name", "abcdefghijk"}); err == nil || !strings.Contains(err.Error(), "max") {
		t.Errorf("expected max error for long name, got: %v", err)
	}
	if err := makeCmd().RunArgsE([]string{"--name", "okay"}); err != nil {
		t.Errorf("unexpected error for valid-length name: %v", err)
	}
}

func TestSetMinMaxLen_Programmatic_SliceLength(t *testing.T) {
	type Params struct {
		Tags []string `descr:"tags" optional:"true"`
	}
	makeCmd := func() CmdT[Params] {
		return CmdT[Params]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
				p := GetParamT(ctx, &params.Tags)
				p.SetMinLen(2)
				p.SetMaxLen(3)
				return nil
			},
			RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
		}
	}
	if err := makeCmd().RunArgsE([]string{"--tags", "a"}); err == nil || !strings.Contains(err.Error(), "min") {
		t.Errorf("expected min error for 1-tag input, got: %v", err)
	}
	if err := makeCmd().RunArgsE([]string{"--tags", "a,b,c,d"}); err == nil || !strings.Contains(err.Error(), "max") {
		t.Errorf("expected max error for 4-tag input, got: %v", err)
	}
	if err := makeCmd().RunArgsE([]string{"--tags", "a,b"}); err != nil {
		t.Errorf("unexpected error for 2-tag input: %v", err)
	}
}

func TestClearMinMax_Programmatic_RemovesBound(t *testing.T) {
	// Set a bound and then clear it — the validation must not run.
	type Params struct {
		Port int `descr:"port" optional:"true"`
	}
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			p := GetParamT(ctx, &params.Port)
			p.SetMinT(10)
			p.SetMaxT(20)
			p.ClearMin()
			p.ClearMax()
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--port", "999"})
	if err != nil {
		t.Fatalf("expected no validation error after ClearMin/ClearMax, got: %v", err)
	}
}

func TestSetMinMax_Programmatic_GetRoundTrip(t *testing.T) {
	// SetMinT / GetMin should round-trip at full int precision, and
	// ClearMin should reset to nil. GetMin / GetMax live on the non-generic
	// Param interface (the typed view only exposes setters), so we read them
	// via ctx.GetParam. Int fields store as *int64.
	type Params struct {
		Port int `descr:"port" optional:"true"`
	}
	var sawMinSet, sawMinCleared any
	var sawMaxSet, sawMaxCleared any
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			tp := GetParamT(ctx, &params.Port)
			tp.SetMinT(42)
			tp.SetMaxT(99)

			raw := ctx.GetParam(&params.Port)
			sawMinSet = raw.GetMin()
			sawMaxSet = raw.GetMax()

			tp.ClearMin()
			tp.ClearMax()
			sawMinCleared = raw.GetMin()
			sawMaxCleared = raw.GetMax()
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--port", "100"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	minP, ok := sawMinSet.(*int64)
	if !ok || minP == nil || *minP != 42 {
		t.Errorf("expected GetMin to return *int64(42), got: %T %v", sawMinSet, sawMinSet)
	}
	maxP, ok := sawMaxSet.(*int64)
	if !ok || maxP == nil || *maxP != 99 {
		t.Errorf("expected GetMax to return *int64(99), got: %T %v", sawMaxSet, sawMaxSet)
	}
	if sawMinCleared != nil {
		t.Errorf("expected GetMin to return nil after ClearMin, got: %v", sawMinCleared)
	}
	if sawMaxCleared != nil {
		t.Errorf("expected GetMax to return nil after ClearMax, got: %v", sawMaxCleared)
	}
}

// --- boa:"configonly" new semantics (not an ignore alias) ---

func TestConfigOnly_NoFlagNoEnvButValidationRuns(t *testing.T) {
	// Under the new semantics, `boa:"configonly"` is `noflag + noenv` with
	// the mirror preserved, so min/max/pattern tags (and custom validators)
	// still run on the config-loaded value.
	type Params struct {
		ConfigFile string `configfile:"true" default:"" optional:"true"`
		Name       string `descr:"public name" optional:"true"`
		Port       int    `descr:"db port" boa:"configonly" min:"1" max:"65535" optional:"true"`
	}

	// First: the CLI flag must not exist
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--port", "8080"})
	if err == nil {
		t.Fatal("expected --port to be unknown under boa:\"configonly\"")
	}

	// Second: the env binding must be ignored
	t.Setenv("PORT", "leaked")
	var gotPort int
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherCombine(ParamEnricherName, ParamEnricherEnv),
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotPort = p.Port
		},
	}).RunArgsE([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPort != 0 {
		t.Errorf("expected env to be ignored for configonly field, got %d", gotPort)
	}

	// Third: a config file value above max should still trigger validation
	cfgData, _ := json.Marshal(map[string]any{"Port": 99999})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--config-file", cfgPath})
	if err == nil {
		t.Fatal("expected max validation to fire on configonly field loaded from config")
	}
	if !strings.Contains(err.Error(), "max") {
		t.Errorf("expected max error, got: %v", err)
	}

	// Fourth: a valid config value should load correctly
	cfgDataOK, _ := json.Marshal(map[string]any{"Port": 8080})
	_ = os.WriteFile(cfgPath, cfgDataOK, 0644)

	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotPort = p.Port
		},
	}).RunArgsE([]string{"--config-file", cfgPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPort != 8080 {
		t.Errorf("expected configonly field loaded from config, got %d", gotPort)
	}
}

// --- SetIgnored vs config-file leaf setByConfig (reviewer issue #1) ---

func TestSetIgnored_ConfigDoesNotMarkSetByConfig(t *testing.T) {
	// Programmatically ignored mirrors must not be marked setByConfig by the
	// config-key-presence probe. Two observable consequences:
	//   (a) the field's HasValue() stays false even when the config mentions
	//       it, which matches the semantics of the `boa:"ignore"` tag where
	//       the mirror never exists in the first place; and
	//   (b) this guarantees that an ignored child can't reach the validate
	//       or required-check paths via setByConfig later.
	type Params struct {
		ConfigFile string `configfile:"true" default:"" optional:"true"`
		Visible    string `descr:"visible" optional:"true"`
		Hidden     string `descr:"hidden" optional:"true"`
	}

	cfgData, _ := json.Marshal(map[string]any{
		"Visible": "from-config",
		"Hidden":  "also-from-config",
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	var hiddenHasValue bool
	var hiddenRaw string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			GetParamT(ctx, &params.Hidden).SetIgnored(true)
			return nil
		},
		PreExecuteFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command, args []string) error {
			// By the time PreExecute runs, the config file has been loaded
			// and setByConfig has been applied. The ignored mirror must NOT
			// have been marked.
			hidden := ctx.GetParam(&params.Hidden)
			hiddenHasValue = hidden.HasValue()
			hiddenRaw = params.Hidden
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--config-file", cfgPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hiddenHasValue {
		t.Error("ignored field's mirror.HasValue() should be false even though the config mentioned it")
	}
	// The raw struct field IS written by json.Unmarshal (config loading
	// bypasses the mirror layer) — that's by design and matches the
	// `boa:"ignore"` tag behavior.
	if hiddenRaw != "also-from-config" {
		t.Errorf("raw struct field should still be written by json.Unmarshal, got %q", hiddenRaw)
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
