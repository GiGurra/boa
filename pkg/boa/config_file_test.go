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
		os.WriteFile(cfgPath, []byte(`{"Host":"custom-format","Port":1234}`), 0644)

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

		os.Setenv("TEST_CFG_HOST", "from-env")
		defer os.Unsetenv("TEST_CFG_HOST")

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
