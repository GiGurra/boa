package boa

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func writeTestConfigFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}
	return path
}

// --- Tests for the explicit LoadConfigFile pattern ---

func TestConfigFile_ExplicitPattern(t *testing.T) {
	type Params struct {
		ConfigFile string `optional:"true" descr:"path to config file"`
		Host       string `optional:"true"`
		Port       int    `optional:"true"`
	}

	t.Run("loads from config file", func(t *testing.T) {
		cfgPath := writeTestConfigFile(t, `{"Host":"from-file","Port":3000}`)

		ran := false
		CmdT[Params]{
			Use: "test",
			PreValidateFunc: func(params *Params, cmd *cobra.Command, args []string) error {
				return LoadConfigFile(params.ConfigFile, params, nil)
			},
			RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
				ran = true
				if params.Host != "from-file" {
					t.Errorf("expected Host='from-file', got %q", params.Host)
				}
				if params.Port != 3000 {
					t.Errorf("expected Port=3000, got %d", params.Port)
				}
			},
		}.RunArgs([]string{"--config-file", cfgPath})

		if !ran {
			t.Fatal("command did not run")
		}
	})

	t.Run("CLI overrides config file", func(t *testing.T) {
		cfgPath := writeTestConfigFile(t, `{"Host":"from-file","Port":3000}`)

		ran := false
		CmdT[Params]{
			Use: "test",
			PreValidateFunc: func(params *Params, cmd *cobra.Command, args []string) error {
				return LoadConfigFile(params.ConfigFile, params, nil)
			},
			RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
				ran = true
				if params.Host != "from-cli" {
					t.Errorf("expected Host='from-cli', got %q", params.Host)
				}
				if params.Port != 3000 {
					t.Errorf("expected Port=3000 (from file), got %d", params.Port)
				}
			},
		}.RunArgs([]string{"--config-file", cfgPath, "--host", "from-cli"})

		if !ran {
			t.Fatal("command did not run")
		}
	})

	t.Run("no config file is fine", func(t *testing.T) {
		ran := false
		CmdT[Params]{
			Use: "test",
			PreValidateFunc: func(params *Params, cmd *cobra.Command, args []string) error {
				return LoadConfigFile(params.ConfigFile, params, nil)
			},
			RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
				ran = true
			},
		}.RunArgs([]string{"--host", "direct", "--port", "80"})

		if !ran {
			t.Fatal("command did not run")
		}
	})

	t.Run("default config path", func(t *testing.T) {
		cfgPath := writeTestConfigFile(t, `{"Host":"default-file","Port":5000}`)

		type ParamsWithDefault struct {
			ConfigFile string `optional:"true" descr:"path to config file"`
			Host       string `optional:"true"`
			Port       int    `optional:"true"`
		}

		ran := false
		CmdT[ParamsWithDefault]{
			Use: "test",
			InitFuncCtx: func(ctx *HookContext, params *ParamsWithDefault, cmd *cobra.Command) error {
				ctx.GetParam(&params.ConfigFile).SetDefault(Default(cfgPath))
				return nil
			},
			PreValidateFunc: func(params *ParamsWithDefault, cmd *cobra.Command, args []string) error {
				return LoadConfigFile(params.ConfigFile, params, nil)
			},
			RunFunc: func(params *ParamsWithDefault, cmd *cobra.Command, args []string) {
				ran = true
				if params.Host != "default-file" {
					t.Errorf("expected Host='default-file', got %q", params.Host)
				}
			},
		}.RunArgs([]string{})

		if !ran {
			t.Fatal("command did not run")
		}
	})
}

// --- Tests for the configfile struct tag shorthand ---

func TestConfigFile_TagShorthand(t *testing.T) {
	t.Run("loads via configfile tag", func(t *testing.T) {
		cfgPath := writeTestConfigFile(t, `{"Host":"tag-file","Port":7000}`)

		type Params struct {
			ConfigFile string `configfile:"true" optional:"true"`
			Host       string `optional:"true"`
			Port       int    `optional:"true"`
		}

		ran := false
		CmdT[Params]{
			Use: "test",
			RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
				ran = true
				if params.Host != "tag-file" {
					t.Errorf("expected Host='tag-file', got %q", params.Host)
				}
				if params.Port != 7000 {
					t.Errorf("expected Port=7000, got %d", params.Port)
				}
			},
		}.RunArgs([]string{"--config-file", cfgPath})

		if !ran {
			t.Fatal("command did not run")
		}
	})

	t.Run("CLI overrides configfile tag", func(t *testing.T) {
		cfgPath := writeTestConfigFile(t, `{"Host":"tag-file","Port":7000}`)

		type Params struct {
			ConfigFile string `configfile:"true" optional:"true"`
			Host       string `optional:"true"`
			Port       int    `optional:"true"`
		}

		ran := false
		CmdT[Params]{
			Use: "test",
			RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
				ran = true
				if params.Host != "from-cli" {
					t.Errorf("expected Host='from-cli', got %q", params.Host)
				}
				if params.Port != 7000 {
					t.Errorf("expected Port=7000 (from file), got %d", params.Port)
				}
			},
		}.RunArgs([]string{"--config-file", cfgPath, "--host", "from-cli"})

		if !ran {
			t.Fatal("command did not run")
		}
	})

	t.Run("no config file with tag is fine", func(t *testing.T) {
		type Params struct {
			ConfigFile string `configfile:"true" optional:"true"`
			Host       string `optional:"true"`
		}

		ran := false
		CmdT[Params]{
			Use: "test",
			RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
				ran = true
			},
		}.RunArgs([]string{"--host", "direct"})

		if !ran {
			t.Fatal("command did not run")
		}
	})

	t.Run("configfile tag with default path", func(t *testing.T) {
		cfgPath := writeTestConfigFile(t, `{"Host":"default-tag","Port":9000}`)

		type Params struct {
			ConfigFile string `configfile:"true" optional:"true"`
			Host       string `optional:"true"`
			Port       int    `optional:"true"`
		}

		ran := false
		CmdT[Params]{
			Use: "test",
			InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
				ctx.GetParam(&params.ConfigFile).SetDefault(Default(cfgPath))
				return nil
			},
			RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
				ran = true
				if params.Host != "default-tag" {
					t.Errorf("expected Host='default-tag', got %q", params.Host)
				}
				if params.Port != 9000 {
					t.Errorf("expected Port=9000, got %d", params.Port)
				}
			},
		}.RunArgs([]string{})

		if !ran {
			t.Fatal("command did not run")
		}
	})

	t.Run("configfile tag with custom unmarshal", func(t *testing.T) {
		// Use a custom format: line-separated key=value
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.custom")
		_ = os.WriteFile(cfgPath, []byte(`{"Host":"custom-format","Port":1234}`), 0644)

		type Params struct {
			ConfigFile string `configfile:"true" optional:"true"`
			Host       string `optional:"true"`
			Port       int    `optional:"true"`
		}

		ran := false
		CmdT[Params]{
			Use:             "test",
			ConfigUnmarshal: json.Unmarshal, // explicit, same as default
			RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
				ran = true
				if params.Host != "custom-format" {
					t.Errorf("expected Host='custom-format', got %q", params.Host)
				}
			},
		}.RunArgs([]string{"--config-file", cfgPath})

		if !ran {
			t.Fatal("command did not run")
		}
	})

	t.Run("env overrides config file", func(t *testing.T) {
		cfgPath := writeTestConfigFile(t, `{"Host":"from-file","Port":3000}`)

		type Params struct {
			ConfigFile string `configfile:"true" optional:"true"`
			Host       string `optional:"true" env:"TEST_CFG_HOST"`
			Port       int    `optional:"true"`
		}

		_ = os.Setenv("TEST_CFG_HOST", "from-env")
		defer func() { _ = os.Unsetenv("TEST_CFG_HOST") }()

		ran := false
		CmdT[Params]{
			Use: "test",
			RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
				ran = true
				if params.Host != "from-env" {
					t.Errorf("expected Host='from-env', got %q", params.Host)
				}
				if params.Port != 3000 {
					t.Errorf("expected Port=3000 (from file), got %d", params.Port)
				}
			},
		}.RunArgs([]string{"--config-file", cfgPath})

		if !ran {
			t.Fatal("command did not run")
		}
	})

	t.Run("missing config file returns error", func(t *testing.T) {
		type Params struct {
			ConfigFile string `configfile:"true" optional:"true"`
			Host       string `optional:"true"`
		}

		err := CmdT[Params]{
			Use: "test",
			RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
				t.Fatal("should not run")
			},
			RawArgs: []string{"--config-file", "/nonexistent/path.json"},
		}.Validate()

		if err == nil {
			t.Fatal("expected error for missing config file")
		}
	})
}

// --- Tests for LoadConfigBytes ---

func TestLoadConfigBytes(t *testing.T) {
	type Params struct {
		Host string `optional:"true"`
		Port int    `optional:"true"`
	}

	t.Run("loads JSON bytes with default format", func(t *testing.T) {
		var p Params
		if err := LoadConfigBytes([]byte(`{"Host":"h","Port":1234}`), "", &p, nil); err != nil {
			t.Fatalf("LoadConfigBytes: %v", err)
		}
		if p.Host != "h" || p.Port != 1234 {
			t.Errorf("unexpected parsed values: %+v", p)
		}
	})

	t.Run("loads JSON bytes with explicit .json ext", func(t *testing.T) {
		var p Params
		if err := LoadConfigBytes([]byte(`{"Host":"h"}`), ".json", &p, nil); err != nil {
			t.Fatalf("LoadConfigBytes: %v", err)
		}
		if p.Host != "h" {
			t.Errorf("expected Host=h, got %q", p.Host)
		}
	})

	t.Run("normalizes ext without leading dot", func(t *testing.T) {
		var p Params
		if err := LoadConfigBytes([]byte(`{"Host":"h"}`), "json", &p, nil); err != nil {
			t.Fatalf("LoadConfigBytes: %v", err)
		}
		if p.Host != "h" {
			t.Errorf("expected Host=h, got %q", p.Host)
		}
	})

	t.Run("dispatches to registered format by extension", func(t *testing.T) {
		registerFormatCleanup(t, ".fake", UniversalConfigFormat(fakeUnmarshal))

		var p Params
		if err := LoadConfigBytes([]byte(`{"Host":"from-fake","Port":9}`), ".fake", &p, nil); err != nil {
			t.Fatalf("LoadConfigBytes: %v", err)
		}
		if p.Host != "from-fake" || p.Port != 9 {
			t.Errorf("unexpected parsed values: %+v", p)
		}
	})

	t.Run("unmarshalFunc overrides ext", func(t *testing.T) {
		called := false
		custom := func(data []byte, target any) error {
			called = true
			return json.Unmarshal(data, target)
		}
		var p Params
		if err := LoadConfigBytes([]byte(`{"Host":"x"}`), ".yaml", &p, custom); err != nil {
			t.Fatalf("LoadConfigBytes: %v", err)
		}
		if !called {
			t.Error("expected custom unmarshal func to be called")
		}
		if p.Host != "x" {
			t.Errorf("expected Host=x, got %q", p.Host)
		}
	})

	t.Run("empty bytes is a no-op", func(t *testing.T) {
		var p Params
		if err := LoadConfigBytes(nil, "", &p, nil); err != nil {
			t.Fatalf("nil: %v", err)
		}
		if err := LoadConfigBytes([]byte{}, "", &p, nil); err != nil {
			t.Fatalf("empty: %v", err)
		}
		if p.Host != "" || p.Port != 0 {
			t.Errorf("expected zero params, got %+v", p)
		}
	})

	t.Run("malformed bytes return error", func(t *testing.T) {
		var p Params
		err := LoadConfigBytes([]byte(`{not json`), "", &p, nil)
		if err == nil {
			t.Fatal("expected error for malformed bytes")
		}
	})

	t.Run("works inside PreValidateFunc with embed-like input", func(t *testing.T) {
		// Simulates //go:embed or stdin pipes feeding bytes into a hook.
		type CmdParams struct {
			Host string `optional:"true"`
			Port int    `optional:"true"`
		}
		embedded := []byte(`{"Host":"embedded","Port":4242}`)

		ran := false
		CmdT[CmdParams]{
			Use: "test",
			PreValidateFunc: func(params *CmdParams, cmd *cobra.Command, args []string) error {
				return LoadConfigBytes(embedded, ".json", params, nil)
			},
			RunFunc: func(params *CmdParams, cmd *cobra.Command, args []string) {
				ran = true
				if params.Host != "embedded" {
					t.Errorf("expected Host='embedded', got %q", params.Host)
				}
				if params.Port != 4242 {
					t.Errorf("expected Port=4242, got %d", params.Port)
				}
			},
		}.RunArgs([]string{})

		if !ran {
			t.Fatal("command did not run")
		}
	})

	t.Run("CLI overrides bytes config in PreValidateFunc", func(t *testing.T) {
		type CmdParams struct {
			Host string `optional:"true"`
			Port int    `optional:"true"`
		}
		embedded := []byte(`{"Host":"from-bytes","Port":3000}`)

		ran := false
		CmdT[CmdParams]{
			Use: "test",
			PreValidateFunc: func(params *CmdParams, cmd *cobra.Command, args []string) error {
				return LoadConfigBytes(embedded, "", params, nil)
			},
			RunFunc: func(params *CmdParams, cmd *cobra.Command, args []string) {
				ran = true
				if params.Host != "from-cli" {
					t.Errorf("expected Host='from-cli' (CLI wins), got %q", params.Host)
				}
				if params.Port != 3000 {
					t.Errorf("expected Port=3000 (from bytes), got %d", params.Port)
				}
			},
		}.RunArgs([]string{"--host", "from-cli"})

		if !ran {
			t.Fatal("command did not run")
		}
	})
}
