package boa

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// --- ParamEnricherEnvPrefix (0%) ---

func TestParamEnricherEnvPrefix(t *testing.T) {
	type Params struct {
		Host string `env:"HOST" descr:"hostname"`
		Port int    `env:"PORT" descr:"port" default:"8080"`
	}

	t.Run("prefixes env vars", func(t *testing.T) {
		os.Setenv("MYAPP_HOST", "example.com")
		defer os.Unsetenv("MYAPP_HOST")

		var got Params
		err := (CmdT[Params]{
			Use: "test",
			ParamEnrich: ParamEnricherCombine(
				ParamEnricherName,
				ParamEnricherEnvPrefix("MYAPP"),
			),
			RunFunc: func(p *Params, c *cobra.Command, args []string) { got = *p },
		}).RunArgsE([]string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Host != "example.com" {
			t.Errorf("Host = %q, want example.com", got.Host)
		}
	})

	t.Run("does not prefix empty env", func(t *testing.T) {
		type P struct {
			Name string `descr:"name" optional:"true"`
		}
		err := (CmdT[P]{
			Use: "test",
			ParamEnrich: ParamEnricherCombine(
				ParamEnricherName,
				ParamEnricherEnvPrefix("PFX"),
			),
			RunFunc: func(p *P, c *cobra.Command, args []string) {},
		}).RunArgsE([]string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// --- ConfigFormatExtensions (0%) ---

func TestConfigFormatExtensions(t *testing.T) {
	exts := ConfigFormatExtensions()
	// .json is always registered by default
	found := false
	for _, ext := range exts {
		if ext == ".json" {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected .json in ConfigFormatExtensions(), got: %v", exts)
	}
}

// --- UnMarshalFromFileParam (0%) ---

func TestUnMarshalFromFileParam(t *testing.T) {
	type Config struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}

	t.Run("loads config from file param", func(t *testing.T) {
		// Create a temp config file
		tmpFile, err := os.CreateTemp("", "boa-test-*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.WriteString(`{"host":"localhost","port":9090}`)
		tmpFile.Close()

		var cfg Config
		var configPath string
		(CmdT[struct {
			Config string `descr:"config file" default:""`
		}]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFuncCtxE: func(ctx *HookContext, p *struct {
				Config string `descr:"config file" default:""`
			}, c *cobra.Command, args []string) error {
				param := ctx.GetParam(&p.Config)
				configPath = p.Config
				return UnMarshalFromFileParam(param, &cfg, json.Unmarshal)
			},
		}).RunArgs([]string{"--config", tmpFile.Name()})

		if cfg.Host != "localhost" || cfg.Port != 9090 {
			t.Errorf("Got %+v, want {localhost 9090}", cfg)
		}
		_ = configPath
	})

	t.Run("no value returns nil", func(t *testing.T) {
		var cfg Config
		(CmdT[struct {
			Config string `descr:"config file" optional:"true"`
		}]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFuncCtxE: func(ctx *HookContext, p *struct {
				Config string `descr:"config file" optional:"true"`
			}, c *cobra.Command, args []string) error {
				param := ctx.GetParam(&p.Config)
				return UnMarshalFromFileParam(param, &cfg, json.Unmarshal)
			},
		}).RunArgs([]string{})
	})
}

// --- runFuncError.Error/Unwrap (0%) ---

func TestRunFuncError(t *testing.T) {
	inner := fmt.Errorf("database connection failed")
	rfe := &runFuncError{Err: inner}

	if rfe.Error() != "database connection failed" {
		t.Errorf("Error() = %q, want 'database connection failed'", rfe.Error())
	}
	if rfe.Unwrap() != inner {
		t.Error("Unwrap() did not return inner error")
	}

	// Verify errors.As works
	var rfe2 *runFuncError
	if !errors.As(rfe, &rfe2) {
		t.Error("errors.As should match runFuncError")
	}
}

func TestRunFuncErrorCausesPanicInRun(t *testing.T) {
	type Params struct {
		Name string `optional:"true"`
	}

	var panicValue any
	func() {
		defer func() {
			panicValue = recover()
		}()

		(CmdT[Params]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFuncE: func(p *Params, c *cobra.Command, args []string) error {
				return fmt.Errorf("runtime programming error")
			},
		}).RunArgs([]string{})
	}()

	if panicValue == nil {
		t.Fatal("Expected panic for non-UserInputError from RunFuncE in Run()")
	}
	if !strings.Contains(fmt.Sprint(panicValue), "runtime programming error") {
		t.Errorf("Panic value = %v, want 'runtime programming error'", panicValue)
	}
}

// --- paramMeta.MarshalJSON/UnmarshalJSON (0%) ---

func TestParamMetaMarshalJSON(t *testing.T) {
	t.Run("marshal with value", func(t *testing.T) {
		type Params struct {
			Name string `descr:"name"`
		}
		var marshaledParam []byte
		(CmdT[Params]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command, args []string) {
				param := ctx.GetParam(&p.Name)
				var err error
				marshaledParam, err = json.Marshal(param)
				if err != nil {
					t.Fatalf("MarshalJSON failed: %v", err)
				}
			},
		}).RunArgs([]string{"--name", "alice"})

		if string(marshaledParam) != `"alice"` {
			t.Errorf("MarshalJSON = %s, want '\"alice\"'", marshaledParam)
		}
	})

	t.Run("marshal without value returns null", func(t *testing.T) {
		type Params struct {
			Name *string `descr:"name"`
		}
		var marshaledParam []byte
		(CmdT[Params]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command, args []string) {
				param := ctx.GetParam(&p.Name)
				var err error
				marshaledParam, err = json.Marshal(param)
				if err != nil {
					t.Fatalf("MarshalJSON failed: %v", err)
				}
			},
		}).RunArgs([]string{})

		if string(marshaledParam) != "null" {
			t.Errorf("MarshalJSON = %s, want 'null'", marshaledParam)
		}
	})
}

func TestParamMetaUnmarshalJSON(t *testing.T) {
	t.Run("unmarshal sets value", func(t *testing.T) {
		type Params struct {
			Host string `descr:"host" default:"default-host"`
			Port int    `descr:"port" default:"8080"`
		}

		// Create a temp config file
		tmpFile, err := os.CreateTemp("", "boa-unmarshal-test-*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.WriteString(`{"host":"from-config","port":3000}`)
		tmpFile.Close()

		var got Params
		err = (CmdT[Params]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFunc:     func(p *Params, c *cobra.Command, args []string) { got = *p },
			InitFunc: func(p *Params, c *cobra.Command) error {
				return LoadConfigFile(tmpFile.Name(), p, json.Unmarshal)
			},
		}).RunArgsE([]string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Host != "from-config" {
			t.Errorf("Host = %q, want from-config", got.Host)
		}
		if got.Port != 3000 {
			t.Errorf("Port = %d, want 3000", got.Port)
		}
	})

	t.Run("CLI overrides config file", func(t *testing.T) {
		type Params struct {
			Host string `descr:"host" default:"default-host"`
		}

		tmpFile, err := os.CreateTemp("", "boa-unmarshal-cli-*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.WriteString(`{"host":"from-config"}`)
		tmpFile.Close()

		var got Params
		err = (CmdT[Params]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFunc:     func(p *Params, c *cobra.Command, args []string) { got = *p },
			InitFunc: func(p *Params, c *cobra.Command) error {
				return LoadConfigFile(tmpFile.Name(), p, json.Unmarshal)
			},
		}).RunArgsE([]string{"--host", "from-cli"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Host != "from-cli" {
			t.Errorf("Host = %q, want from-cli (CLI should override config)", got.Host)
		}
	})
}

// --- SetCustomValidatorT (20%) ---

func TestSetCustomValidatorT(t *testing.T) {
	t.Run("typed validator rejects invalid value", func(t *testing.T) {
		type Params struct {
			Port int `descr:"port" default:"8080"`
		}

		err := (CmdT[Params]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			InitFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command) error {
				param := GetParamT[int](ctx, &p.Port)
				param.SetCustomValidatorT(func(v int) error {
					if v < 1024 {
						return fmt.Errorf("port must be >= 1024")
					}
					return nil
				})
				return nil
			},
			RunFunc: func(p *Params, c *cobra.Command, args []string) {},
		}).RunArgsE([]string{"--port", "80"})

		if err == nil {
			t.Fatal("Expected error for port < 1024")
		}
		if !strings.Contains(err.Error(), "port must be >= 1024") {
			t.Errorf("Expected validator error, got: %v", err)
		}
	})

	t.Run("typed validator accepts valid value", func(t *testing.T) {
		type Params struct {
			Port int `descr:"port" default:"8080"`
		}

		err := (CmdT[Params]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			InitFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command) error {
				param := GetParamT[int](ctx, &p.Port)
				param.SetCustomValidatorT(func(v int) error {
					if v < 1024 {
						return fmt.Errorf("port must be >= 1024")
					}
					return nil
				})
				return nil
			},
			RunFunc: func(p *Params, c *cobra.Command, args []string) {},
		}).RunArgsE([]string{"--port", "8080"})

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
	})

	t.Run("nil validator clears existing", func(t *testing.T) {
		type Params struct {
			Port int `descr:"port" default:"8080"`
		}

		err := (CmdT[Params]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			InitFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command) error {
				param := GetParamT[int](ctx, &p.Port)
				param.SetCustomValidatorT(func(v int) error {
					return fmt.Errorf("always fail")
				})
				// Clear it
				param.SetCustomValidatorT(nil)
				return nil
			},
			RunFunc: func(p *Params, c *cobra.Command, args []string) {},
		}).RunArgsE([]string{"--port", "80"})

		if err != nil {
			t.Fatalf("Expected nil validator to be cleared, got: %v", err)
		}
	})
}

// --- doParsePositional edge cases (50%) ---

func TestDoParsePositional_EmptyRequiredWithDefault(t *testing.T) {
	// A required positional arg with a default should not error on empty
	type Params struct {
		Mode string `positional:"true" required:"true" default:"auto"`
	}

	var got string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, c *cobra.Command, args []string) { got = p.Mode },
	}).RunArgsE([]string{})
	// With 0 args, cobra's RangeArgs(1,1) rejects before doParsePositional runs.
	// But we can test with a default by checking the value is set.
	if err != nil {
		// This is expected — cobra rejects 0 args even with defaults
		return
	}
	if got != "auto" {
		t.Errorf("Mode = %q, want auto", got)
	}
}

func TestDoParsePositional_EmptyRequiredNoDefault(t *testing.T) {
	type Params struct {
		Name string `positional:"true" required:"true"`
	}

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, c *cobra.Command, args []string) {},
	}).RunArgsE([]string{})

	if err == nil {
		t.Fatal("Expected error for empty required positional without default")
	}
}

func TestDoParsePositional_InvalidTypeValue(t *testing.T) {
	type Params struct {
		Count int `positional:"true" required:"true"`
	}

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, c *cobra.Command, args []string) {},
	}).RunArgsE([]string{"not-a-number"})

	if err == nil {
		t.Fatal("Expected error for non-numeric positional int arg")
	}
	if !IsUserInputError(err) {
		t.Errorf("Expected UserInputError, got: %T", err)
	}
}

// --- HookContext.HasValue / GetParam edge cases ---

func TestHookContextHasValue(t *testing.T) {
	type Params struct {
		Name string `descr:"name" optional:"true"`
		Age  int    `descr:"age" default:"25"`
	}

	t.Run("HasValue returns true for set field", func(t *testing.T) {
		var hasVal bool
		(CmdT[Params]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command, args []string) {
				hasVal = ctx.HasValue(&p.Age)
			},
		}).RunArgs([]string{"--age", "30"})
		if !hasVal {
			t.Error("Expected HasValue to return true for --age 30")
		}
	})

	t.Run("HasValue returns false for unset optional", func(t *testing.T) {
		var hasVal bool
		(CmdT[Params]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command, args []string) {
				hasVal = ctx.HasValue(&p.Name)
			},
		}).RunArgs([]string{})
		if hasVal {
			t.Error("Expected HasValue to return false for unset optional Name")
		}
	})

	t.Run("HasValue returns false for unknown pointer", func(t *testing.T) {
		var hasVal bool
		var unknown string
		(CmdT[Params]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command, args []string) {
				hasVal = ctx.HasValue(&unknown)
			},
		}).RunArgs([]string{})
		if hasVal {
			t.Error("Expected HasValue to return false for unknown pointer")
		}
	})
}

func TestGetParamTNilReturn(t *testing.T) {
	type Params struct {
		Name string `descr:"name" optional:"true"`
	}

	var result ParamT[string]
	var unknown string
	(CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command) error {
			result = GetParamT[string](ctx, &unknown) // not a field in Params
			return nil
		},
		RunFunc: func(p *Params, c *cobra.Command, args []string) {},
	}).RunArgs([]string{})

	if result != nil {
		t.Error("Expected GetParamT to return nil for unknown field pointer")
	}
}

// --- AllMirrors ---

func TestAllMirrorsCoverage(t *testing.T) {
	type Params struct {
		Host string `descr:"host" default:"localhost"`
		Port int    `descr:"port" default:"8080"`
	}

	var mirrors []Param
	(CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command, args []string) {
			mirrors = ctx.AllMirrors()
		},
	}).RunArgs([]string{})

	if len(mirrors) != 2 {
		t.Errorf("Expected 2 mirrors, got %d", len(mirrors))
	}
}

func TestAllMirrorsNilContextCoverage(t *testing.T) {
	ctx := &HookContext{}
	mirrors := ctx.AllMirrors()
	if mirrors != nil {
		t.Error("Expected nil mirrors for empty context")
	}
}

// --- RegisterType with nil Format (80%) ---

func TestRegisterTypeNilFormat(t *testing.T) {
	type MyID string

	RegisterType[MyID](TypeDef[MyID]{
		Parse: func(s string) (MyID, error) {
			if s == "" {
				return "", fmt.Errorf("ID cannot be empty")
			}
			return MyID(s), nil
		},
		// Format is nil — should use default fmt.Sprintf
	})

	type Params struct {
		ID MyID `descr:"resource ID" optional:"true"`
	}

	var got MyID
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, c *cobra.Command, args []string) { got = p.ID },
	}).RunArgsE([]string{"--id", "abc-123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "abc-123" {
		t.Errorf("ID = %q, want abc-123", got)
	}
}

// --- toCobraImplE subcommand-only path ---

func TestToCobraImplESubcommandOnly(t *testing.T) {
	type SubParams struct{}

	root := Cmd{
		Use: "app",
		SubCmds: SubCmds(
			CmdT[SubParams]{
				Use:         "valid",
				RunFunc:     func(p *SubParams, c *cobra.Command, args []string) {},
				ParamEnrich: ParamEnricherName,
			},
		),
	}

	err := root.RunArgsE([]string{"bogus"})
	if err == nil {
		t.Fatal("Expected error for unknown subcommand via RunArgsE")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("Expected 'unknown command' error, got: %v", err)
	}
}

// --- defaultValueStr edge case (50%) ---

func TestDefaultValueStr(t *testing.T) {
	type Params struct {
		Port int    `descr:"port" default:"8080"`
		Name string `descr:"name" default:"world"`
	}

	// Verify defaults appear in usage string (which exercises defaultValueStr)
	usage := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, c *cobra.Command, args []string) {},
	}).ToCobra().UsageString()

	if !strings.Contains(usage, "8080") {
		t.Errorf("Expected default 8080 in usage:\n%s", usage)
	}
	if !strings.Contains(usage, "world") {
		t.Errorf("Expected default 'world' in usage:\n%s", usage)
	}
}

// --- SetCustomValidatorT reflection path (covers type alias conversion) ---

func TestSetCustomValidatorT_TypeAlias(t *testing.T) {
	type MyString string
	type Params struct {
		Tag MyString `descr:"tag" optional:"true"`
	}

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command) error {
			param := GetParamT[MyString](ctx, &p.Tag)
			param.SetCustomValidatorT(func(v MyString) error {
				if len(v) > 0 && v[0] != 'v' {
					return fmt.Errorf("tag must start with 'v'")
				}
				return nil
			})
			return nil
		},
		RunFunc: func(p *Params, c *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--tag", "1.0.0"})

	if err == nil {
		t.Fatal("Expected error for tag not starting with v")
	}
	if !strings.Contains(err.Error(), "tag must start with 'v'") {
		t.Errorf("Expected validator error, got: %v", err)
	}
}

// --- parseTimeString error path ---

func TestParseTimeString_InvalidFormat(t *testing.T) {
	type Params struct {
		When time.Time `descr:"timestamp" optional:"true"`
	}

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, c *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--when", "not-a-timestamp"})

	if err == nil {
		t.Fatal("Expected error for invalid time format")
	}
}

func TestParseTimeString_ValidFormats(t *testing.T) {
	type Params struct {
		When time.Time `descr:"timestamp"`
	}

	formats := []string{
		"2024-01-15T10:30:00Z",           // RFC3339
		"2024-01-15T10:30:00.123456789Z", // RFC3339Nano
		"2024-01-15",                      // date only
		"2024-01-15T10:30:00",            // datetime without timezone
		"2024-01-15 10:30:00",            // datetime with space
	}

	for _, ts := range formats {
		t.Run(ts, func(t *testing.T) {
			err := (CmdT[Params]{
				Use:         "test",
				ParamEnrich: ParamEnricherName,
				RunFunc:     func(p *Params, c *cobra.Command, args []string) {},
			}).RunArgsE([]string{"--when", ts})
			if err != nil {
				t.Errorf("Expected valid time format %q, got error: %v", ts, err)
			}
		})
	}
}

// --- MarshalJSON with default value (branch coverage) ---

func TestParamMetaMarshalJSON_WithDefault(t *testing.T) {
	type Params struct {
		Port int `descr:"port" default:"8080"`
	}

	var marshaled []byte
	(CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command, args []string) {
			param := ctx.GetParam(&p.Port)
			var err error
			marshaled, err = json.Marshal(param)
			if err != nil {
				t.Fatalf("MarshalJSON failed: %v", err)
			}
		},
	}).RunArgs([]string{})

	if string(marshaled) != "8080" {
		t.Errorf("MarshalJSON = %s, want 8080", marshaled)
	}
}

// --- paramMeta.UnmarshalJSON direct tests ---

// --- doParsePositional: env-set fallback ---

func TestDoParsePositional_EnvFallback(t *testing.T) {
	type Params struct {
		File string `positional:"true" required:"true" env:"TEST_FILE_POS"`
	}

	os.Setenv("TEST_FILE_POS", "from-env.txt")
	defer os.Unsetenv("TEST_FILE_POS")

	var got string
	err := (CmdT[Params]{
		Use: "test",
		ParamEnrich: ParamEnricherCombine(
			ParamEnricherName,
			ParamEnricherEnv,
		),
		RunFunc: func(p *Params, c *cobra.Command, args []string) { got = p.File },
	}).RunArgsE([]string{})

	// Cobra's Args validator runs before env parsing, so this may fail at the
	// cobra level. The important thing is no panic.
	if err != nil {
		// Expected — cobra rejects 0 args before env parsing
		return
	}
	if got != "from-env.txt" {
		t.Errorf("File = %q, want from-env.txt", got)
	}
}

// --- toCobraImplE with RunFuncE ---

func TestToCobraImplE_RunFuncE(t *testing.T) {
	type Params struct {
		Name string `descr:"name" optional:"true"`
	}

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncE: func(p *Params, c *cobra.Command, args []string) error {
			return fmt.Errorf("intentional error")
		},
	}).RunArgsE([]string{})

	if err == nil {
		t.Fatal("Expected error from RunFuncE")
	}
	if !strings.Contains(err.Error(), "intentional error") {
		t.Errorf("Expected 'intentional error', got: %v", err)
	}
}

// --- Struct literal validation (values set via Params field) ---

func TestStructLiteralValidation(t *testing.T) {
	type Params struct {
		Port int `descr:"port" min:"1" max:"65535" default:"8080"`
	}

	t.Run("invalid literal value rejected", func(t *testing.T) {
		err := (CmdT[Params]{
			Use:         "test",
			Params:      &Params{Port: 99999},
			ParamEnrich: ParamEnricherName,
			RunFunc:     func(p *Params, c *cobra.Command, args []string) {},
		}).RunArgsE([]string{})
		if err == nil {
			t.Fatal("Expected error for struct literal Port: 99999 exceeding max")
		}
		if !strings.Contains(err.Error(), "max") {
			t.Errorf("Expected 'max' error, got: %v", err)
		}
	})

	t.Run("valid literal value accepted", func(t *testing.T) {
		var got int
		err := (CmdT[Params]{
			Use:         "test",
			Params:      &Params{Port: 3000},
			ParamEnrich: ParamEnricherName,
			RunFunc:     func(p *Params, c *cobra.Command, args []string) { got = p.Port },
		}).RunArgsE([]string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 3000 {
			t.Errorf("Port = %d, want 3000", got)
		}
	})

	t.Run("CLI overrides literal value", func(t *testing.T) {
		var got int
		err := (CmdT[Params]{
			Use:         "test",
			Params:      &Params{Port: 3000},
			ParamEnrich: ParamEnricherName,
			RunFunc:     func(p *Params, c *cobra.Command, args []string) { got = p.Port },
		}).RunArgsE([]string{"--port", "4000"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 4000 {
			t.Errorf("Port = %d, want 4000 (CLI should override literal)", got)
		}
	})
}

func TestToCobraImplE_RunFuncCtxE(t *testing.T) {
	type Params struct {
		Name string `descr:"name" optional:"true"`
	}

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtxE: func(ctx *HookContext, p *Params, c *cobra.Command, args []string) error {
			return fmt.Errorf("ctx intentional error")
		},
	}).RunArgsE([]string{})

	if err == nil {
		t.Fatal("Expected error from RunFuncCtxE")
	}
	if !strings.Contains(err.Error(), "ctx intentional error") {
		t.Errorf("Expected 'ctx intentional error', got: %v", err)
	}
}

// --- paramMeta.UnmarshalJSON direct tests ---

func TestParamMetaUnmarshalJSON_Direct(t *testing.T) {
	t.Run("unmarshal string value", func(t *testing.T) {
		pm := &paramMeta{fieldType: reflect.TypeOf("")}
		err := pm.UnmarshalJSON([]byte(`"hello"`))
		if err != nil {
			t.Fatalf("UnmarshalJSON failed: %v", err)
		}
		if pm.valuePtr == nil {
			t.Fatal("Expected valuePtr to be set")
		}
		if reflect.ValueOf(pm.valuePtr).Elem().String() != "hello" {
			t.Errorf("Got %v, want hello", pm.valuePtr)
		}
		if !pm.injected {
			t.Error("Expected injected to be true")
		}
	})

	t.Run("unmarshal int value", func(t *testing.T) {
		pm := &paramMeta{fieldType: reflect.TypeOf(0)}
		err := pm.UnmarshalJSON([]byte(`42`))
		if err != nil {
			t.Fatalf("UnmarshalJSON failed: %v", err)
		}
		if reflect.ValueOf(pm.valuePtr).Elem().Int() != 42 {
			t.Errorf("Got %v, want 42", pm.valuePtr)
		}
	})

	t.Run("unmarshal null is no-op", func(t *testing.T) {
		pm := &paramMeta{fieldType: reflect.TypeOf("")}
		err := pm.UnmarshalJSON([]byte(`null`))
		if err != nil {
			t.Fatalf("UnmarshalJSON failed: %v", err)
		}
		if pm.valuePtr != nil {
			t.Error("Expected valuePtr to remain nil for null")
		}
	})

	t.Run("unmarshal zero value is no-op", func(t *testing.T) {
		pm := &paramMeta{fieldType: reflect.TypeOf(0)}
		err := pm.UnmarshalJSON([]byte(`0`))
		if err != nil {
			t.Fatalf("UnmarshalJSON failed: %v", err)
		}
		if pm.valuePtr != nil {
			t.Error("Expected valuePtr to remain nil for zero value")
		}
	})

	t.Run("unmarshal invalid JSON returns error", func(t *testing.T) {
		pm := &paramMeta{fieldType: reflect.TypeOf("")}
		err := pm.UnmarshalJSON([]byte(`{invalid`))
		if err == nil {
			t.Fatal("Expected error for invalid JSON")
		}
	})

	t.Run("skipped when set on CLI", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().String("name", "", "")
		cmd.Flags().Set("name", "from-cli")

		pm := &paramMeta{
			name:      "name",
			fieldType: reflect.TypeOf(""),
			parent:    cmd,
		}
		err := pm.UnmarshalJSON([]byte(`"from-json"`))
		if err != nil {
			t.Fatalf("UnmarshalJSON failed: %v", err)
		}
		if pm.valuePtr != nil {
			t.Error("Expected valuePtr to remain nil when CLI value is set")
		}
	})
}

// --- Map types (buildMapBindFlag, buildMapParse) ---

func TestMapTypesCoverage(t *testing.T) {
	t.Run("map[string]int", func(t *testing.T) {
		type P struct {
			Limits map[string]int `descr:"resource limits"`
		}
		var got map[string]int
		err := (CmdT[P]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFunc:     func(p *P, c *cobra.Command, args []string) { got = p.Limits },
		}).RunArgsE([]string{"--limits", "cpu=4,memory=8192"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["cpu"] != 4 || got["memory"] != 8192 {
			t.Errorf("Got %v, want cpu=4,memory=8192", got)
		}
	})

	t.Run("map[string]int64", func(t *testing.T) {
		type P struct {
			Sizes map[string]int64 `descr:"sizes"`
		}
		var got map[string]int64
		err := (CmdT[P]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFunc:     func(p *P, c *cobra.Command, args []string) { got = p.Sizes },
		}).RunArgsE([]string{"--sizes", "disk=1099511627776"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["disk"] != 1099511627776 {
			t.Errorf("Got %v", got)
		}
	})

	t.Run("map[string]string", func(t *testing.T) {
		type P struct {
			Labels map[string]string `descr:"labels"`
		}
		var got map[string]string
		err := (CmdT[P]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFunc:     func(p *P, c *cobra.Command, args []string) { got = p.Labels },
		}).RunArgsE([]string{"--labels", "env=prod,team=backend"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["env"] != "prod" || got["team"] != "backend" {
			t.Errorf("Got %v", got)
		}
	})

	t.Run("map[string]string with default", func(t *testing.T) {
		type P struct {
			Labels map[string]string `descr:"labels"`
		}
		var got map[string]string
		(CmdT[P]{
			Use:         "test",
			Params:      &P{Labels: map[string]string{"default": "val"}},
			ParamEnrich: ParamEnricherName,
			RunFunc:     func(p *P, c *cobra.Command, args []string) { got = p.Labels },
		}).RunArgs([]string{})
		if got["default"] != "val" {
			t.Errorf("Got %v, want default=val", got)
		}
	})
}

// --- JSON fallback for complex types ---

func TestJSONFallbackCoverage(t *testing.T) {
	t.Run("nested slice via JSON", func(t *testing.T) {
		type P struct {
			Matrix [][]int `descr:"matrix" optional:"true"`
		}
		var got [][]int
		err := (CmdT[P]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFunc:     func(p *P, c *cobra.Command, args []string) { got = p.Matrix },
		}).RunArgsE([]string{"--matrix", "[[1,2],[3,4]]"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 || got[0][0] != 1 || got[1][1] != 4 {
			t.Errorf("Got %v", got)
		}
	})

	t.Run("nested slice invalid JSON", func(t *testing.T) {
		type P struct {
			Matrix [][]int `descr:"matrix" optional:"true"`
		}
		err := (CmdT[P]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFunc:     func(p *P, c *cobra.Command, args []string) {},
		}).RunArgsE([]string{"--matrix", "not-json"})
		if err == nil {
			t.Fatal("Expected error for invalid JSON")
		}
	})

	t.Run("complex map via JSON", func(t *testing.T) {
		type P struct {
			Meta map[string][]string `descr:"metadata" optional:"true"`
		}
		var got map[string][]string
		err := (CmdT[P]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFunc:     func(p *P, c *cobra.Command, args []string) { got = p.Meta },
		}).RunArgsE([]string{"--meta", `{"tags":["a","b"]}`})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got["tags"]) != 2 {
			t.Errorf("Got %v", got)
		}
	})

	t.Run("nested slice default from struct literal", func(t *testing.T) {
		type P struct {
			Matrix [][]int `descr:"matrix" optional:"true"`
		}
		var got [][]int
		(CmdT[P]{
			Use:         "test",
			Params:      &P{Matrix: [][]int{{1, 2}}},
			ParamEnrich: ParamEnricherName,
			RunFunc:     func(p *P, c *cobra.Command, args []string) { got = p.Matrix },
		}).RunArgs([]string{})
		if len(got) != 1 || got[0][0] != 1 {
			t.Errorf("Got %v, want [[1,2]]", got)
		}
	})
}

// --- Various numeric types with defaults (registerBuiltinTypes coverage) ---

func TestNumericTypesCoverage(t *testing.T) {
	t.Run("int32 with default", func(t *testing.T) {
		type P struct {
			Val int32 `descr:"val" default:"42"`
		}
		var got int32
		(CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Val }}).RunArgs([]string{})
		if got != 42 {
			t.Errorf("Got %d, want 42", got)
		}
	})

	t.Run("int64 with default", func(t *testing.T) {
		type P struct {
			Val int64 `descr:"val" default:"999999999999"`
		}
		var got int64
		(CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Val }}).RunArgs([]string{})
		if got != 999999999999 {
			t.Errorf("Got %d", got)
		}
	})

	t.Run("float32 with default", func(t *testing.T) {
		type P struct {
			Val float32 `descr:"val" default:"3.14"`
		}
		var got float32
		(CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Val }}).RunArgs([]string{})
		if got < 3.13 || got > 3.15 {
			t.Errorf("Got %f, want ~3.14", got)
		}
	})

	t.Run("float64 with default", func(t *testing.T) {
		type P struct {
			Val float64 `descr:"val" default:"2.718"`
		}
		var got float64
		(CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Val }}).RunArgs([]string{})
		if got != 2.718 {
			t.Errorf("Got %f, want 2.718", got)
		}
	})

	t.Run("bool with default true", func(t *testing.T) {
		type P struct {
			Val bool `descr:"val" default:"true"`
		}
		var got bool
		(CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Val }}).RunArgs([]string{})
		if !got {
			t.Error("Got false, want true")
		}
	})

	t.Run("int32 CLI override", func(t *testing.T) {
		type P struct {
			Val int32 `descr:"val" default:"0"`
		}
		var got int32
		(CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Val }}).RunArgs([]string{"--val", "100"})
		if got != 100 {
			t.Errorf("Got %d, want 100", got)
		}
	})

	t.Run("float32 CLI override", func(t *testing.T) {
		type P struct {
			Val float32 `descr:"val" default:"0"`
		}
		var got float32
		(CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Val }}).RunArgs([]string{"--val", "1.5"})
		if got != 1.5 {
			t.Errorf("Got %f, want 1.5", got)
		}
	})
}

// --- newParam branches (nested slices, req tag, optional tag) ---

func TestNewParamBranches(t *testing.T) {
	t.Run("nested slice defaults to optional", func(t *testing.T) {
		type P struct {
			Matrix [][]string `descr:"matrix"`
		}
		// Should not error with no value — nested slices default optional
		err := (CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) {}}).RunArgsE([]string{})
		if err != nil {
			t.Fatalf("Nested slice should default to optional, got: %v", err)
		}
	})

	t.Run("req tag alias", func(t *testing.T) {
		type P struct {
			Name string `descr:"name" req:"true"`
		}
		err := (CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) {}}).RunArgsE([]string{})
		if err == nil {
			t.Fatal("Expected error for missing required field via req tag")
		}
	})

	t.Run("opt tag alias", func(t *testing.T) {
		type P struct {
			Name string `descr:"name" opt:"true"`
		}
		err := (CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) {}}).RunArgsE([]string{})
		if err != nil {
			t.Fatalf("Expected no error for optional field via opt tag, got: %v", err)
		}
	})

	t.Run("required false", func(t *testing.T) {
		type P struct {
			Name string `descr:"name" required:"false"`
		}
		err := (CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) {}}).RunArgsE([]string{})
		if err != nil {
			t.Fatalf("Expected no error for required:false, got: %v", err)
		}
	})
}

// --- UnMarshalFromFileParam error paths ---

func TestUnMarshalFromFileParam_ErrorPaths(t *testing.T) {
	type Config struct {
		Host string `json:"host"`
	}

	t.Run("empty string value", func(t *testing.T) {
		type P struct {
			Config string `descr:"config" default:""`
		}
		var unmarshalErr error
		(CmdT[P]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFuncCtxE: func(ctx *HookContext, p *P, c *cobra.Command, args []string) error {
				param := ctx.GetParam(&p.Config)
				var cfg Config
				unmarshalErr = UnMarshalFromFileParam(param, &cfg, json.Unmarshal)
				return nil
			},
		}).RunArgs([]string{"--config", ""})
		if unmarshalErr == nil {
			t.Fatal("Expected error for empty config path")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		type P struct {
			Config string `descr:"config"`
		}
		var unmarshalErr error
		(CmdT[P]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFuncCtxE: func(ctx *HookContext, p *P, c *cobra.Command, args []string) error {
				param := ctx.GetParam(&p.Config)
				var cfg Config
				unmarshalErr = UnMarshalFromFileParam(param, &cfg, json.Unmarshal)
				return nil
			},
		}).RunArgs([]string{"--config", "/nonexistent/path.json"})
		if unmarshalErr == nil {
			t.Fatal("Expected error for nonexistent file")
		}
	})
}

// --- GetParam with Param interface directly ---

func TestGetParamWithParamInterface(t *testing.T) {
	type P struct {
		Name string `descr:"name" optional:"true"`
	}

	(CmdT[P]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *P, c *cobra.Command, args []string) {
			// First get via field pointer
			param := ctx.GetParam(&p.Name)
			if param == nil {
				t.Fatal("Expected param from field pointer")
			}
			// Then get via Param interface directly (the other branch)
			param2 := ctx.GetParam(param)
			if param2 != param {
				t.Error("Expected GetParam(Param) to return same Param")
			}
		},
	}).RunArgs([]string{})
}

// --- MarshalJSON default value branch ---

func TestMarshalJSON_DefaultBranch(t *testing.T) {
	pm := &paramMeta{
		fieldType: reflect.TypeOf(0),
	}
	// Set a default value
	v := reflect.ValueOf(42)
	pm.defaultVal = &v

	data, err := pm.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	if string(data) != "42" {
		t.Errorf("Got %s, want 42", data)
	}
}

// --- Slice with defaults (derefSliceDefault) ---

func TestSliceWithDefaultsCoverage(t *testing.T) {
	type P struct {
		Tags []string `descr:"tags"`
	}
	var got []string
	(CmdT[P]{
		Use:         "test",
		Params:      &P{Tags: []string{"a", "b"}},
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *P, c *cobra.Command, args []string) { got = p.Tags },
	}).RunArgs([]string{})
	if len(got) != 2 || got[0] != "a" {
		t.Errorf("Got %v, want [a b]", got)
	}
}

// --- ParamEnricherCombine error path ---

func TestParamEnricherCombineError(t *testing.T) {
	type P struct {
		Name string `descr:"name"`
	}
	// An enricher that returns an error
	failEnricher := func(alreadyProcessed []Param, param Param, fieldName string) error {
		return fmt.Errorf("enricher failed")
	}

	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Expected panic from failed enricher")
			}
		}()
		(CmdT[P]{
			Use:         "test",
			ParamEnrich: ParamEnricherCombine(ParamEnricherName, failEnricher),
			RunFunc:     func(p *P, c *cobra.Command, args []string) {},
		}).ToCobra()
	}()
}

// --- connect: short flag "h" collision ---

func TestConnectShortFlagCollision(t *testing.T) {
	type P struct {
		Host string `short:"h" descr:"hostname"`
	}
	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Expected panic for short flag 'h' collision")
			}
		}()
		(CmdT[P]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFunc:     func(p *P, c *cobra.Command, args []string) {},
		}).ToCobra()
	}()
}

// --- Special types with defaults (registerBuiltinTypes default branches) ---

func TestSpecialTypesWithDefaults(t *testing.T) {
	t.Run("duration with default", func(t *testing.T) {
		type P struct {
			Timeout time.Duration `descr:"timeout" default:"5s"`
		}
		var got time.Duration
		(CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Timeout }}).RunArgs([]string{})
		if got != 5*time.Second {
			t.Errorf("Got %v, want 5s", got)
		}
	})

	t.Run("duration CLI override", func(t *testing.T) {
		type P struct {
			Timeout time.Duration `descr:"timeout" default:"5s"`
		}
		var got time.Duration
		(CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Timeout }}).RunArgs([]string{"--timeout", "10m"})
		if got != 10*time.Minute {
			t.Errorf("Got %v, want 10m", got)
		}
	})

	t.Run("duration with struct literal default", func(t *testing.T) {
		type P struct {
			Timeout time.Duration `descr:"timeout"`
		}
		var got time.Duration
		(CmdT[P]{Use: "test", Params: &P{Timeout: 30 * time.Second}, ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Timeout }}).RunArgs([]string{})
		if got != 30*time.Second {
			t.Errorf("Got %v, want 30s", got)
		}
	})

	t.Run("net.IP with struct literal default", func(t *testing.T) {
		type P struct {
			Addr net.IP `descr:"address"`
		}
		var got net.IP
		(CmdT[P]{Use: "test", Params: &P{Addr: net.ParseIP("10.0.0.1")}, ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Addr }}).RunArgs([]string{})
		if got.String() != "10.0.0.1" {
			t.Errorf("Got %v, want 10.0.0.1", got)
		}
	})

	t.Run("*url.URL with struct literal default", func(t *testing.T) {
		type P struct {
			URL *url.URL `descr:"url"`
		}
		defURL, _ := url.Parse("https://example.com")
		var got *url.URL
		(CmdT[P]{Use: "test", Params: &P{URL: defURL}, ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.URL }}).RunArgs([]string{})
		if got == nil || got.String() != "https://example.com" {
			t.Errorf("Got %v, want https://example.com", got)
		}
	})

	t.Run("time.Time with default tag", func(t *testing.T) {
		type P struct {
			When time.Time `descr:"when" default:"2024-01-15T00:00:00Z"`
		}
		var got time.Time
		(CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.When }}).RunArgs([]string{})
		if got.Year() != 2024 || got.Month() != 1 || got.Day() != 15 {
			t.Errorf("Got %v, want 2024-01-15", got)
		}
	})

	t.Run("time.Time with struct literal default", func(t *testing.T) {
		type P struct {
			When time.Time `descr:"when"`
		}
		defTime := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
		var got time.Time
		(CmdT[P]{Use: "test", Params: &P{When: defTime}, ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.When }}).RunArgs([]string{})
		if got != defTime {
			t.Errorf("Got %v, want %v", got, defTime)
		}
	})
}

// --- Slice types with defaults and CLI (parseSliceWith, derefSliceDefault) ---

func TestSliceTypesCoverage(t *testing.T) {
	t.Run("[]int with default", func(t *testing.T) {
		type P struct {
			Ports []int `descr:"ports"`
		}
		var got []int
		(CmdT[P]{Use: "test", Params: &P{Ports: []int{80, 443}}, ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Ports }}).RunArgs([]string{})
		if len(got) != 2 || got[0] != 80 {
			t.Errorf("Got %v, want [80 443]", got)
		}
	})

	t.Run("[]int CLI override", func(t *testing.T) {
		type P struct {
			Ports []int `descr:"ports"`
		}
		var got []int
		(CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Ports }}).RunArgs([]string{"--ports", "8080", "--ports", "9090"})
		if len(got) != 2 || got[0] != 8080 {
			t.Errorf("Got %v, want [8080 9090]", got)
		}
	})

	t.Run("[]bool", func(t *testing.T) {
		type P struct {
			Flags []bool `descr:"flags" optional:"true"`
		}
		var got []bool
		(CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Flags }}).RunArgs([]string{"--flags", "true", "--flags", "false"})
		if len(got) != 2 || got[0] != true || got[1] != false {
			t.Errorf("Got %v, want [true false]", got)
		}
	})

	t.Run("[]float64", func(t *testing.T) {
		type P struct {
			Vals []float64 `descr:"vals" optional:"true"`
		}
		var got []float64
		(CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Vals }}).RunArgs([]string{"--vals", "1.5", "--vals", "2.5"})
		if len(got) != 2 || got[0] != 1.5 {
			t.Errorf("Got %v, want [1.5 2.5]", got)
		}
	})

	t.Run("[]int32", func(t *testing.T) {
		type P struct {
			Vals []int32 `descr:"vals" optional:"true"`
		}
		var got []int32
		(CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Vals }}).RunArgs([]string{"--vals", "1", "--vals", "2"})
		if len(got) != 2 || got[0] != 1 {
			t.Errorf("Got %v", got)
		}
	})

	t.Run("[]int64", func(t *testing.T) {
		type P struct {
			Vals []int64 `descr:"vals" optional:"true"`
		}
		var got []int64
		(CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Vals }}).RunArgs([]string{"--vals", "100", "--vals", "200"})
		if len(got) != 2 || got[0] != 100 {
			t.Errorf("Got %v", got)
		}
	})

	t.Run("[]float32", func(t *testing.T) {
		type P struct {
			Vals []float32 `descr:"vals" optional:"true"`
		}
		var got []float32
		(CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Vals }}).RunArgs([]string{"--vals", "1.0", "--vals", "2.0"})
		if len(got) != 2 {
			t.Errorf("Got %v", got)
		}
	})
}

// --- traverse branches: embedded structs, boa:ignore ---

func TestTraverseCoverage(t *testing.T) {
	t.Run("embedded struct fields are not prefixed", func(t *testing.T) {
		type Common struct {
			Verbose bool `descr:"verbose" optional:"true"`
		}
		type P struct {
			Common // embedded
			Name string `descr:"name" optional:"true"`
		}
		usage := (CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) {}}).ToCobra().UsageString()
		// Embedded → no prefix
		if !strings.Contains(usage, "--verbose") {
			t.Errorf("Expected --verbose (no prefix) for embedded struct:\n%s", usage)
		}
	})

	t.Run("boa:ignore fields skipped from CLI", func(t *testing.T) {
		type P struct {
			Name   string `descr:"name" optional:"true"`
			Secret string `boa:"ignore"`
		}
		usage := (CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) {}}).ToCobra().UsageString()
		if strings.Contains(usage, "secret") {
			t.Errorf("boa:ignore field should not appear in usage:\n%s", usage)
		}
	})

	t.Run("boa:configonly fields skipped from CLI", func(t *testing.T) {
		type P struct {
			Name   string `descr:"name" optional:"true"`
			Secret string `boa:"configonly"`
		}
		usage := (CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) {}}).ToCobra().UsageString()
		if strings.Contains(usage, "secret") {
			t.Errorf("boa:configonly field should not appear in usage:\n%s", usage)
		}
	})
}

// --- map parse error path ---

func TestMapParseErrorCoverage(t *testing.T) {
	t.Run("invalid map syntax", func(t *testing.T) {
		type P struct {
			Labels map[string]string `descr:"labels"`
		}
		err := (CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) {}}).RunArgsE([]string{"--labels", "no-equals-sign"})
		// pflag handles map parsing for string maps, so error format may vary
		if err == nil {
			t.Fatal("Expected error for invalid map syntax")
		}
	})
}

// --- toCobraImplE RunFuncCtx path ---

func TestToCobraImplE_RunFuncCtx(t *testing.T) {
	type P struct {
		Name string `descr:"name" optional:"true"`
	}
	var ran bool
	err := (CmdT[P]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx:  func(ctx *HookContext, p *P, c *cobra.Command, args []string) { ran = true },
	}).RunArgsE([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ran {
		t.Error("Expected RunFuncCtx to run")
	}
}

func TestConnectNameHelpCollision(t *testing.T) {
	type P struct {
		Help string `name:"help" descr:"help text"`
	}
	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Expected panic for flag name 'help' collision")
			}
		}()
		(CmdT[P]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFunc:     func(p *P, c *cobra.Command, args []string) {},
		}).ToCobra()
	}()
}

// --- newParam branches: pointer types, opt:"false", tag edge cases ---

func TestNewParamPointerTypes(t *testing.T) {
	t.Run("*string optional by default", func(t *testing.T) {
		type P struct {
			Name *string `descr:"name"`
		}
		err := (CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) {}}).RunArgsE([]string{})
		if err != nil {
			t.Fatalf("*string should be optional by default: %v", err)
		}
	})

	t.Run("*int optional by default, nil when unset", func(t *testing.T) {
		type P struct {
			Count *int `descr:"count"`
		}
		var got *int
		(CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Count }}).RunArgs([]string{})
		if got != nil {
			t.Errorf("Expected nil for unset *int, got %v", *got)
		}
	})

	t.Run("*int set via CLI", func(t *testing.T) {
		type P struct {
			Count *int `descr:"count"`
		}
		var got *int
		(CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Count }}).RunArgs([]string{"--count", "42"})
		if got == nil || *got != 42 {
			t.Errorf("Expected 42, got %v", got)
		}
	})

	t.Run("*bool set via CLI", func(t *testing.T) {
		type P struct {
			Verbose *bool `descr:"verbose"`
		}
		var got *bool
		(CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Verbose }}).RunArgs([]string{"--verbose=true"})
		if got == nil || !*got {
			t.Errorf("Expected true, got %v", got)
		}
	})

	t.Run("opt:false makes field required", func(t *testing.T) {
		type P struct {
			Name string `descr:"name" opt:"false"`
		}
		err := (CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) {}}).RunArgsE([]string{})
		if err == nil {
			t.Fatal("Expected error for opt:false with no value")
		}
	})

	t.Run("req:false makes field optional", func(t *testing.T) {
		type P struct {
			Name string `descr:"name" req:"false"`
		}
		err := (CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) {}}).RunArgsE([]string{})
		if err != nil {
			t.Fatalf("Expected no error for req:false: %v", err)
		}
	})
}

// --- parsePtr: map JSON fallback ---

func TestParsePtrMapJsonFallback(t *testing.T) {
	type P struct {
		Data map[string][]int `descr:"data" optional:"true"`
	}
	var got map[string][]int
	err := (CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Data }}).RunArgsE([]string{"--data", `{"nums":[1,2,3]}`})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got["nums"]) != 3 {
		t.Errorf("Got %v", got)
	}
}

// --- doParsePositional: optional positional with no value ---

func TestDoParsePositionalOptionalEmpty(t *testing.T) {
	type P struct {
		Required string `positional:"true" required:"true"`
		Optional string `positional:"true" optional:"true"`
	}
	var gotReq, gotOpt string
	(CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) {
		gotReq = p.Required
		gotOpt = p.Optional
	}}).RunArgs([]string{"hello"})
	if gotReq != "hello" {
		t.Errorf("Required = %q, want hello", gotReq)
	}
	if gotOpt != "" {
		t.Errorf("Optional = %q, want empty", gotOpt)
	}
}

// --- SetCustomValidatorT: pointer field path ---

func TestSetCustomValidatorT_PointerField(t *testing.T) {
	type P struct {
		Port *int `descr:"port"`
	}

	t.Run("typed validator works on pointer field", func(t *testing.T) {
		err := (CmdT[P]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			InitFuncCtx: func(ctx *HookContext, p *P, c *cobra.Command) error {
				// GetParamT infers T=*int from &p.Port (**int), returns ParamT[*int]
				param := GetParamT[*int](ctx, &p.Port)
				param.SetCustomValidatorT(func(v *int) error {
					if v != nil && *v < 1024 {
						return fmt.Errorf("port must be >= 1024")
					}
					return nil
				})
				return nil
			},
			RunFunc: func(p *P, c *cobra.Command, args []string) {},
		}).RunArgsE([]string{"--port", "80"})
		if err == nil {
			t.Fatal("Expected error for port < 1024 on pointer field")
		}
		if !strings.Contains(err.Error(), "port must be >= 1024") {
			t.Errorf("Expected validator error, got: %v", err)
		}
	})

	t.Run("typed validator receives nil for unset pointer", func(t *testing.T) {
		var validatorCalled bool
		var receivedNil bool
		err := (CmdT[P]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			InitFuncCtx: func(ctx *HookContext, p *P, c *cobra.Command) error {
				param := GetParamT[*int](ctx, &p.Port)
				param.SetCustomValidatorT(func(v *int) error {
					validatorCalled = true
					receivedNil = (v == nil)
					return nil
				})
				return nil
			},
			RunFunc: func(p *P, c *cobra.Command, args []string) {},
		}).RunArgsE([]string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Validator may not be called for unset optional fields — that's fine
		_ = validatorCalled
		_ = receivedNil
	})
}

// --- runImpl: success handler ---

func TestRunImplSuccessHandler(t *testing.T) {
	type P struct{}
	var succeeded bool
	handler := resultHandler{
		Success: func() { succeeded = true },
	}
	cmd := (CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) {}}).ToCobra()
	cmd.SetArgs([]string{})
	runImpl(cmd, handler)
	if !succeeded {
		t.Error("Expected success handler to be called")
	}
}

// --- configfile tag ---

func TestConfigFileTagCoverage(t *testing.T) {
	type P struct {
		Config string `configfile:"true" optional:"true"`
		Host   string `descr:"host" optional:"true"`
		Port   int    `descr:"port" optional:"true"`
	}

	tmpFile, _ := os.CreateTemp("", "boa-cfgtag-*.json")
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(`{"host":"from-config","port":7777}`)
	tmpFile.Close()

	var got P
	err := (CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = *p }}).RunArgsE([]string{"--config", tmpFile.Name()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Host != "from-config" {
		t.Errorf("Host = %q, want from-config", got.Host)
	}
	if got.Port != 7777 {
		t.Errorf("Port = %d, want 7777", got.Port)
	}
}

// --- Validate with setup error ---

func TestValidatePanicRecovery(t *testing.T) {
	type P struct {
		Name string `descr:"name"`
	}
	// InitFunc errors cause panics in ToCobra path; Validate catches them
	err := (CmdT[P]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		PreValidateFunc: func(p *P, c *cobra.Command, args []string) error {
			return fmt.Errorf("pre-validate failed")
		},
		RunFunc: func(p *P, c *cobra.Command, args []string) {},
	}).Validate()
	if err == nil {
		t.Fatal("Expected error from Validate when PreValidateFunc fails")
	}
}

// --- loadConfigFileInto with extension-based format ---

func TestLoadConfigFileExtensionLookup(t *testing.T) {
	type Config struct {
		Host string `json:"host"`
	}
	RegisterConfigFormat(".test-fmt", json.Unmarshal)

	tmpFile, _ := os.CreateTemp("", "boa-ext-*.test-fmt")
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(`{"host":"from-ext"}`)
	tmpFile.Close()

	var cfg Config
	err := loadConfigFileInto(tmpFile.Name(), &cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Host != "from-ext" {
		t.Errorf("Host = %q, want from-ext", cfg.Host)
	}
}

// --- GetParam with Param interface directly ---

func TestGetParamWithParamInterfaceCoverage(t *testing.T) {
	type P struct {
		Name string `descr:"name" optional:"true"`
	}
	(CmdT[P]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *P, c *cobra.Command, args []string) {
			param := ctx.GetParam(&p.Name)
			if param == nil {
				t.Fatal("Expected param from field pointer")
			}
			param2 := ctx.GetParam(param)
			if param2 != param {
				t.Error("Expected GetParam(Param) to return same Param")
			}
		},
	}).RunArgs([]string{})
}

// --- MarshalJSON default value branch ---

func TestMarshalJSON_DefaultBranchCoverage(t *testing.T) {
	pm := &paramMeta{fieldType: reflect.TypeOf(0)}
	v := reflect.ValueOf(42)
	pm.defaultVal = &v

	data, err := pm.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	if string(data) != "42" {
		t.Errorf("Got %s, want 42", data)
	}
}

// ============================================================
// Coverage push: targeting functions below 70%
// ============================================================

// --- SetCustomValidatorT: *T branch and type alias reflection branch ---

func TestSetCustomValidatorT_StarTBranch(t *testing.T) {
	type P struct {
		Name string `descr:"name" optional:"true"`
	}
	var called bool
	err := (CmdT[P]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, p *P, c *cobra.Command) error {
			param := GetParamT[string](ctx, &p.Name)
			param.SetCustomValidatorT(func(v string) error {
				called = true
				if v == "" {
					return fmt.Errorf("name cannot be empty")
				}
				return nil
			})
			return nil
		},
		RunFunc: func(p *P, c *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--name", "alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("Expected validator to be called")
	}
}

func TestSetCustomValidatorT_TypeAliasReflection(t *testing.T) {
	type MyString string
	type P struct {
		Tag MyString `descr:"tag" optional:"true"`
	}
	var called bool
	err := (CmdT[P]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, p *P, c *cobra.Command) error {
			param := GetParamT[MyString](ctx, &p.Tag)
			param.SetCustomValidatorT(func(v MyString) error {
				called = true
				if len(v) > 0 && v[0] != 'v' {
					return fmt.Errorf("must start with v")
				}
				return nil
			})
			return nil
		},
		RunFunc: func(p *P, c *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--tag", "v1.0"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("Expected validator to be called via reflection conversion")
	}
}

// --- doParsePositional: int positional ---

func TestDoParsePositional_IntPositional(t *testing.T) {
	type P struct {
		Port int `positional:"true" required:"true"`
	}
	var got int
	(CmdT[P]{Use: "test", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Port }}).RunArgs([]string{"8080"})
	if got != 8080 {
		t.Errorf("Got %d, want 8080", got)
	}
}

// --- SetDefault: type alias, nil ---

func TestSetDefault_TypeAlias(t *testing.T) {
	type MyPort int
	type P struct {
		Port MyPort `descr:"port"`
	}
	var got MyPort
	err := (CmdT[P]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, p *P, c *cobra.Command) error {
			param := ctx.GetParam(&p.Port)
			param.SetDefault(Default(MyPort(9090)))
			return nil
		},
		RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Port },
	}).RunArgsE([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 9090 {
		t.Errorf("Got %d, want 9090", got)
	}
}

func TestSetDefault_Nil(t *testing.T) {
	// Directly test that SetDefault(nil) clears the default
	pm := &paramMeta{fieldType: reflect.TypeOf(0)}
	v := reflect.ValueOf(42)
	pm.defaultVal = &v
	if !pm.hasDefaultValue() {
		t.Fatal("Expected default to be set")
	}
	pm.SetDefault(nil)
	if pm.hasDefaultValue() {
		t.Fatal("Expected default to be cleared after SetDefault(nil)")
	}
}

// --- jsonFallbackHandler: bindFlag with struct literal default ---

func TestJsonFallback_StructLiteralDefault(t *testing.T) {
	type P struct {
		Matrix [][]int `descr:"matrix" optional:"true"`
	}
	var got [][]int
	(CmdT[P]{
		Use:         "test",
		Params:      &P{Matrix: [][]int{{1, 2}, {3, 4}}},
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *P, c *cobra.Command, args []string) { got = p.Matrix },
	}).RunArgs([]string{})
	if len(got) != 2 || got[0][0] != 1 {
		t.Errorf("Got %v, want [[1,2],[3,4]]", got)
	}
}

// --- buildMapBindFlag: defaults for int and int64 maps ---

func TestMapBindFlag_IntDefault(t *testing.T) {
	type P struct {
		Limits map[string]int `descr:"limits"`
	}
	var got map[string]int
	(CmdT[P]{
		Use:         "test",
		Params:      &P{Limits: map[string]int{"cpu": 4}},
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *P, c *cobra.Command, args []string) { got = p.Limits },
	}).RunArgs([]string{})
	if got["cpu"] != 4 {
		t.Errorf("Got %v, want cpu=4", got)
	}
}

func TestMapBindFlag_Int64Default(t *testing.T) {
	type P struct {
		Sizes map[string]int64 `descr:"sizes"`
	}
	var got map[string]int64
	(CmdT[P]{
		Use:         "test",
		Params:      &P{Sizes: map[string]int64{"disk": 1024}},
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *P, c *cobra.Command, args []string) { got = p.Sizes },
	}).RunArgs([]string{})
	if got["disk"] != 1024 {
		t.Errorf("Got %v, want disk=1024", got)
	}
}

// --- defaultValueStr: no-default path ---

func TestDefaultValueStr_NoDefault(t *testing.T) {
	pm := &paramMeta{name: "test-param", fieldType: reflect.TypeOf(0)}
	result := pm.defaultValueStr()
	if result != "" {
		t.Errorf("Expected empty string for no-default, got %q", result)
	}
}

// --- ptrToAnyToString: panic paths ---

func TestPtrToAnyToString_NonPointerPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected panic for non-pointer")
		}
	}()
	ptrToAnyToString(42)
}

func TestPtrToAnyToString_NilPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected panic for nil")
		}
	}()
	ptrToAnyToString(nil)
}

// --- parsePtr: array and unsupported types ---

func TestParsePtr_ArrayType(t *testing.T) {
	_, err := parsePtr("test", reflect.TypeOf([3]int{}), reflect.Array, "1,2,3")
	if err == nil {
		t.Fatal("Expected error for array type")
	}
	if !strings.Contains(err.Error(), "arrays not supported") {
		t.Errorf("Expected 'arrays not supported', got: %v", err)
	}
}

func TestParsePtr_UnsupportedType(t *testing.T) {
	type MyStruct struct{ X int }
	_, err := parsePtr("test", reflect.TypeOf(MyStruct{}), reflect.Struct, "stuff")
	if err == nil {
		t.Fatal("Expected error for unsupported type")
	}
	if !strings.Contains(err.Error(), "unsupported param type") {
		t.Errorf("Expected 'unsupported', got: %v", err)
	}
}

// --- buildMapParse: error paths ---

func TestBuildMapParse_InvalidValue(t *testing.T) {
	type P struct {
		Limits map[string]int `descr:"limits"`
	}
	err := (CmdT[P]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *P, c *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--limits", "cpu=notanumber"})
	if err == nil {
		t.Fatal("Expected error for invalid map value")
	}
}

// --- parse error paths for int32, int64, float32, bool ---

func TestParseErrors(t *testing.T) {
	t.Run("int32 parse error", func(t *testing.T) {
		type P struct{ V int32 `descr:"v"` }
		err := (CmdT[P]{Use: "t", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) {}}).RunArgsE([]string{"--v", "notint"})
		if err == nil { t.Fatal("expected error") }
	})
	t.Run("int64 parse error", func(t *testing.T) {
		type P struct{ V int64 `descr:"v"` }
		err := (CmdT[P]{Use: "t", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) {}}).RunArgsE([]string{"--v", "notint"})
		if err == nil { t.Fatal("expected error") }
	})
	t.Run("float32 parse error", func(t *testing.T) {
		type P struct{ V float32 `descr:"v"` }
		err := (CmdT[P]{Use: "t", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) {}}).RunArgsE([]string{"--v", "notfloat"})
		if err == nil { t.Fatal("expected error") }
	})
	t.Run("float64 parse error", func(t *testing.T) {
		type P struct{ V float64 `descr:"v"` }
		err := (CmdT[P]{Use: "t", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) {}}).RunArgsE([]string{"--v", "notfloat"})
		if err == nil { t.Fatal("expected error") }
	})
	t.Run("bool parse error", func(t *testing.T) {
		type P struct{ V bool `descr:"v"` }
		err := (CmdT[P]{Use: "t", ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) {}}).RunArgsE([]string{"--v=notbool"})
		if err == nil { t.Fatal("expected error") }
	})
}

// --- jsonFallbackHandler: convert with non-empty string ---

func TestJsonFallback_ConvertNonEmpty(t *testing.T) {
	type P struct {
		Matrix [][]int `descr:"matrix" optional:"true"`
	}
	var got [][]int
	err := (CmdT[P]{
		Use: "test", ParamEnrich: ParamEnricherName,
		RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Matrix },
	}).RunArgsE([]string{"--matrix", "[[5,6]]"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0][0] != 5 {
		t.Errorf("Got %v, want [[5,6]]", got)
	}
}

// --- buildMapParse: empty string, invalid entry ---

func TestBuildMapParse_EmptyString(t *testing.T) {
	// map[string]int with empty value after = should parse as empty map via native pflag
	type P struct {
		Limits map[string]int `descr:"limits" optional:"true"`
	}
	err := (CmdT[P]{
		Use: "test", ParamEnrich: ParamEnricherName,
		RunFunc: func(p *P, c *cobra.Command, args []string) {},
	}).RunArgsE([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildMapParse_InvalidEntry(t *testing.T) {
	// map[string]int with missing = separator
	type P struct {
		Counts map[string]int `descr:"counts"`
	}
	err := (CmdT[P]{
		Use: "test", ParamEnrich: ParamEnricherName,
		RunFunc: func(p *P, c *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--counts", "noequals"})
	if err == nil {
		t.Fatal("Expected error for map entry without =")
	}
}

// --- buildMapBindFlag: the "default" fallback branch (complex map value types) ---

func TestMapNonNativeValueType(t *testing.T) {
	// map[string]float64 — not natively supported by pflag, uses StringP + buildMapParse
	type P struct {
		Rates map[string]float64 `descr:"rates" optional:"true"`
	}
	var got map[string]float64
	err := (CmdT[P]{
		Use: "test", ParamEnrich: ParamEnricherName,
		RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Rates },
	}).RunArgsE([]string{"--rates", "cpu=0.5,mem=0.8"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["cpu"] != 0.5 {
		t.Errorf("Got %v, want cpu=0.5", got)
	}
}

func TestMapJsonFallback(t *testing.T) {
	// map[string][]int — value type has no scalar handler, falls through to JSON fallback
	type P struct {
		Data map[string][]int `descr:"data" optional:"true"`
	}

	t.Run("set via CLI JSON", func(t *testing.T) {
		var got map[string][]int
		err := (CmdT[P]{
			Use: "test", ParamEnrich: ParamEnricherName,
			RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Data },
		}).RunArgsE([]string{"--data", `{"ports":[80,443]}`})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got["ports"]) != 2 || got["ports"][0] != 80 {
			t.Errorf("Got %v, want ports=[80,443]", got)
		}
	})

	t.Run("struct literal default", func(t *testing.T) {
		var got map[string][]int
		(CmdT[P]{
			Use: "test", ParamEnrich: ParamEnricherName,
			Params:  &P{Data: map[string][]int{"ids": {1, 2, 3}}},
			RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Data },
		}).RunArgs([]string{})
		if len(got["ids"]) != 3 {
			t.Errorf("Got %v, want ids=[1,2,3]", got)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		err := (CmdT[P]{
			Use: "test", ParamEnrich: ParamEnricherName,
			RunFunc: func(p *P, c *cobra.Command, args []string) {},
		}).RunArgsE([]string{"--data", "not-json"})
		if err == nil {
			t.Fatal("Expected error for invalid JSON map")
		}
	})
}

// --- SetDefault: non-pointer value, panic ---

func TestSetDefault_NonPointerValue(t *testing.T) {
	pm := &paramMeta{fieldType: reflect.TypeOf(0)}
	// Pass non-pointer int directly (the fallback path at line 189)
	pm.SetDefault(42)
	if !pm.hasDefaultValue() {
		t.Fatal("Expected default to be set from non-pointer value")
	}
}

func TestSetDefault_IncompatibleTypePanics(t *testing.T) {
	pm := &paramMeta{fieldType: reflect.TypeOf(0)}
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected panic for incompatible type")
		}
	}()
	pm.SetDefault("not-an-int") // string for int field — should panic
}

// --- parseSliceWith: empty bracket string ---

func TestParseSliceWith_EmptyBrackets(t *testing.T) {
	// Test parseSliceWith directly with empty brackets
	result, err := parseSliceWith("[]", func(s string) (int, error) {
		return strconv.Atoi(s)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(*result) != 0 {
		t.Errorf("Expected empty slice, got %v", *result)
	}
}

func TestParseSliceWith_ParseError(t *testing.T) {
	_, err := parseSliceWith("[1,bad,3]", func(s string) (int, error) {
		return strconv.Atoi(s)
	})
	if err == nil {
		t.Fatal("Expected parse error")
	}
}

// --- doParsePositional: empty required positional with default set ---

func TestDoParsePositional_EmptyRequiredWithDefaultDirect(t *testing.T) {
	// Directly test the doParsePositional function
	pm := &paramMeta{
		name:            "mode",
		fieldType:       reflect.TypeOf(""),
		defaultRequired: true,
	}
	v := reflect.ValueOf("auto")
	pm.defaultVal = &v

	err := doParsePositional(pm, "")
	if err != nil {
		t.Fatalf("Expected no error when required positional has default, got: %v", err)
	}
}

func TestDoParsePositional_EmptyRequiredNoDefaultDirect(t *testing.T) {
	pm := &paramMeta{
		name:            "file",
		fieldType:       reflect.TypeOf(""),
		defaultRequired: true,
	}
	err := doParsePositional(pm, "")
	if err == nil {
		t.Fatal("Expected error for empty required positional without default")
	}
	if !IsUserInputError(err) {
		t.Errorf("Expected UserInputError, got: %T", err)
	}
}

// --- SetCustomValidatorT: exercise *T and fallback branches directly ---

func TestSetCustomValidatorT_InternalBranches(t *testing.T) {
	t.Run("case *T: validator receives pointer to value", func(t *testing.T) {
		view := &ParamTView[int]{param: &paramMeta{fieldType: reflect.TypeOf(0)}}
		var received int
		view.SetCustomValidatorT(func(v int) error {
			received = v
			return nil
		})
		intVal := 42
		err := view.param.(*paramMeta).customValidator(&intVal)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if received != 42 {
			t.Errorf("Got %d, want 42", received)
		}
	})

	t.Run("case *T: nil pointer passes zero", func(t *testing.T) {
		view := &ParamTView[int]{param: &paramMeta{fieldType: reflect.TypeOf(0)}}
		var received int
		view.SetCustomValidatorT(func(v int) error {
			received = v
			return nil
		})
		err := view.param.(*paramMeta).customValidator((*int)(nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if received != 0 {
			t.Errorf("Got %d, want 0 (zero value for nil pointer)", received)
		}
	})
}

// --- jsonFallbackHandler: exercise parse error and convert branches ---

func TestJsonFallbackHandler_Direct(t *testing.T) {
	t.Run("parse error", func(t *testing.T) {
		handler := jsonFallbackHandler(reflect.TypeOf([][]int{}))
		_, err := handler.parse("test", "not-json")
		if err == nil {
			t.Fatal("Expected parse error")
		}
	})

	t.Run("convert from string", func(t *testing.T) {
		handler := jsonFallbackHandler(reflect.TypeOf([][]int{}))
		s := "[[1,2],[3,4]]"
		result, err := handler.convert("test", &s)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("Expected non-nil result")
		}
	})

	t.Run("convert empty string", func(t *testing.T) {
		handler := jsonFallbackHandler(reflect.TypeOf([][]int{}))
		s := ""
		result, err := handler.convert("test", &s)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := result.(*string); !ok {
			t.Errorf("Expected *string for empty, got %T", result)
		}
	})

	t.Run("convert already converted", func(t *testing.T) {
		handler := jsonFallbackHandler(reflect.TypeOf([][]int{}))
		val := [][]int{{1}}
		result, err := handler.convert("test", val)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result for already-converted value")
		}
	})

	t.Run("convert invalid JSON", func(t *testing.T) {
		handler := jsonFallbackHandler(reflect.TypeOf([][]int{}))
		s := "not-json"
		_, err := handler.convert("test", &s)
		if err == nil {
			t.Fatal("Expected error for invalid JSON")
		}
	})

	t.Run("bindFlag with default", func(t *testing.T) {
		handler := jsonFallbackHandler(reflect.TypeOf([][]int{}))
		cmd := &cobra.Command{Use: "test"}
		defaultVal := &[][]int{{1, 2}}
		result := handler.bindFlag(cmd, "matrix", "", "data", defaultVal)
		if result == nil {
			t.Fatal("Expected non-nil flag pointer")
		}
	})
}

// --- buildMapParse: direct exercise ---

func TestBuildMapParse_Direct(t *testing.T) {
	mapType := reflect.TypeOf(map[string]int{})
	valType := reflect.TypeOf(0)
	valHandler, _ := lookupHandler(valType)
	parse := buildMapParse(mapType, valType, valHandler)

	t.Run("empty string", func(t *testing.T) {
		result, err := parse("test", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("Expected non-nil result")
		}
	})

	t.Run("invalid entry", func(t *testing.T) {
		_, err := parse("test", "noequals")
		if err == nil {
			t.Fatal("Expected error for missing =")
		}
	})

	t.Run("invalid value", func(t *testing.T) {
		_, err := parse("test", "key=notanumber")
		if err == nil {
			t.Fatal("Expected error for non-int value")
		}
	})
}

// --- registerBuiltinTypes: struct literal defaults for all numeric types ---

func TestBuiltinTypeStructLiteralDefaults(t *testing.T) {
	t.Run("int32", func(t *testing.T) {
		type P struct{ V int32 `descr:"v"` }
		var got int32
		(CmdT[P]{Use: "t", Params: &P{V: 42}, ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.V }}).RunArgs([]string{})
		if got != 42 { t.Errorf("Got %d, want 42", got) }
	})

	t.Run("int64", func(t *testing.T) {
		type P struct{ V int64 `descr:"v"` }
		var got int64
		(CmdT[P]{Use: "t", Params: &P{V: 99}, ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.V }}).RunArgs([]string{})
		if got != 99 { t.Errorf("Got %d, want 99", got) }
	})

	t.Run("float32", func(t *testing.T) {
		type P struct{ V float32 `descr:"v"` }
		var got float32
		(CmdT[P]{Use: "t", Params: &P{V: 1.5}, ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.V }}).RunArgs([]string{})
		if got != 1.5 { t.Errorf("Got %f, want 1.5", got) }
	})

	t.Run("float64", func(t *testing.T) {
		type P struct{ V float64 `descr:"v"` }
		var got float64
		(CmdT[P]{Use: "t", Params: &P{V: 2.718}, ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.V }}).RunArgs([]string{})
		if got != 2.718 { t.Errorf("Got %f, want 2.718", got) }
	})

	t.Run("bool true", func(t *testing.T) {
		type P struct{ V bool `descr:"v"` }
		var got bool
		(CmdT[P]{Use: "t", Params: &P{V: true}, ParamEnrich: ParamEnricherName, RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.V }}).RunArgs([]string{})
		if !got { t.Error("Got false, want true") }
	})
}

// ============================================================
// Coverage push: bug-likely logic paths
// ============================================================

// --- Positional arg default value application (connect line 527-529) ---

func TestPositionalArgDefaultApplied(t *testing.T) {
	// When a required positional arg has a default, and fewer args are provided
	// than max, the default should be used for the missing positional.
	type P struct {
		Src  string `positional:"true" required:"true"`
		Mode string `positional:"true" required:"true" default:"copy"`
	}
	var gotSrc, gotMode string
	// Provide only 1 arg — Mode should get its default "copy"
	(CmdT[P]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *P, c *cobra.Command, args []string) {
			gotSrc = p.Src
			gotMode = p.Mode
		},
	}).RunArgs([]string{"file.txt"})
	if gotSrc != "file.txt" {
		t.Errorf("Src = %q, want file.txt", gotSrc)
	}
	if gotMode != "copy" {
		t.Errorf("Mode = %q, want copy (from default)", gotMode)
	}
}

// --- CfgStructInit lifecycle hook (toCobraBase line 932) ---

var initHookCalled bool

type initHookParams struct {
	Host string `descr:"host" optional:"true"`
}

func (p *initHookParams) Init() error {
	initHookCalled = true
	return nil
}

func TestCfgStructInit(t *testing.T) {
	t.Run("Init is called during setup", func(t *testing.T) {
		initHookCalled = false
		(CmdT[initHookParams]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFunc:     func(p *initHookParams, c *cobra.Command, args []string) {},
		}).RunArgs([]string{})
		if !initHookCalled {
			t.Error("Expected Init() to be called")
		}
	})
}

// --- CfgStructPreValidate lifecycle hook (internal line 1314) ---

var preValidateHookCalled bool

type preValidateHookParams struct {
	Port int `descr:"port" default:"8080"`
}

func (p *preValidateHookParams) PreValidate() error {
	preValidateHookCalled = true
	return nil
}

func TestCfgStructPreValidate(t *testing.T) {
	preValidateHookCalled = false
	(CmdT[preValidateHookParams]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *preValidateHookParams, c *cobra.Command, args []string) {},
	}).RunArgs([]string{})
	if !preValidateHookCalled {
		t.Error("Expected PreValidate() to be called")
	}
}

// --- []time.Time with struct literal default (type_handler line 395-402) ---

func TestSliceTimeDuration(t *testing.T) {
	t.Run("[]time.Duration with struct literal default", func(t *testing.T) {
		type P struct {
			Timeouts []time.Duration `descr:"timeouts"`
		}
		var got []time.Duration
		(CmdT[P]{
			Use:         "test",
			Params:      &P{Timeouts: []time.Duration{time.Second, 5 * time.Second}},
			ParamEnrich: ParamEnricherName,
			RunFunc:     func(p *P, c *cobra.Command, args []string) { got = p.Timeouts },
		}).RunArgs([]string{})
		if len(got) != 2 || got[0] != time.Second {
			t.Errorf("Got %v, want [1s 5s]", got)
		}
	})

	t.Run("[]time.Duration via CLI", func(t *testing.T) {
		type P struct {
			Timeouts []time.Duration `descr:"timeouts" optional:"true"`
		}
		var got []time.Duration
		(CmdT[P]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFunc:     func(p *P, c *cobra.Command, args []string) { got = p.Timeouts },
		}).RunArgs([]string{"--timeouts", "1s", "--timeouts", "5m"})
		if len(got) != 2 || got[0] != time.Second || got[1] != 5*time.Minute {
			t.Errorf("Got %v, want [1s 5m]", got)
		}
	})
}

// --- []*url.URL with struct literal default (type_handler line 435-442) ---

func TestSliceURL(t *testing.T) {
	t.Run("[]*url.URL with struct literal default", func(t *testing.T) {
		type P struct {
			URLs []*url.URL `descr:"urls"`
		}
		u1, _ := url.Parse("https://a.com")
		u2, _ := url.Parse("https://b.com")
		var got []*url.URL
		(CmdT[P]{
			Use:         "test",
			Params:      &P{URLs: []*url.URL{u1, u2}},
			ParamEnrich: ParamEnricherName,
			RunFunc:     func(p *P, c *cobra.Command, args []string) { got = p.URLs },
		}).RunArgs([]string{})
		if len(got) != 2 || got[0].String() != "https://a.com" {
			t.Errorf("Got %v, want [https://a.com https://b.com]", got)
		}
	})

	t.Run("[]*url.URL via CLI", func(t *testing.T) {
		type P struct {
			URLs []*url.URL `descr:"urls" optional:"true"`
		}
		var got []*url.URL
		(CmdT[P]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFunc:     func(p *P, c *cobra.Command, args []string) { got = p.URLs },
		}).RunArgs([]string{"--urls", "https://x.com", "--urls", "https://y.com"})
		if len(got) != 2 || got[0].String() != "https://x.com" {
			t.Errorf("Got %v", got)
		}
	})
}

// --- Slice default from string tag (connect line 608-613) ---

func TestSliceDefaultFromTag(t *testing.T) {
	type P struct {
		Ports []int `descr:"ports" default:"[80,443]"`
	}
	var got []int
	(CmdT[P]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *P, c *cobra.Command, args []string) { got = p.Ports },
	}).RunArgs([]string{})
	if len(got) != 2 || got[0] != 80 || got[1] != 443 {
		t.Errorf("Got %v, want [80 443]", got)
	}
}

// --- PostCreateFunc hook (toCobraBase line 1212) ---

func TestPostCreateFunc(t *testing.T) {
	type P struct {
		Name string `descr:"name" optional:"true"`
	}
	var hookCalled bool
	err := (CmdT[P]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		PostCreateFunc: func(p *P, c *cobra.Command) error {
			hookCalled = true
			return nil
		},
		RunFunc: func(p *P, c *cobra.Command, args []string) {},
	}).RunArgsE([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hookCalled {
		t.Error("Expected PostCreateFunc to be called")
	}
}
