package boa

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

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

// --- Tests for DumpConfigBytes / DumpConfigFile ---

func TestDumpConfigBytes(t *testing.T) {
	type Params struct {
		Host string `json:"Host"`
		Port int    `json:"Port"`
	}
	sample := &Params{Host: "h", Port: 1234}

	t.Run("default is pretty-printed JSON", func(t *testing.T) {
		data, err := DumpConfigBytes(sample, "", nil)
		if err != nil {
			t.Fatalf("DumpConfigBytes: %v", err)
		}
		s := string(data)
		if !strings.Contains(s, "\n") {
			t.Errorf("expected multi-line pretty JSON, got %q", s)
		}
		if !strings.Contains(s, `"Host": "h"`) {
			t.Errorf("expected indented pair in output, got %q", s)
		}
		if !strings.HasSuffix(s, "\n") {
			t.Error("expected trailing newline in pretty JSON dump")
		}
	})

	t.Run("round-trips through LoadConfigBytes", func(t *testing.T) {
		data, err := DumpConfigBytes(sample, ".json", nil)
		if err != nil {
			t.Fatalf("DumpConfigBytes: %v", err)
		}
		var back Params
		if err := LoadConfigBytes(data, ".json", &back, nil); err != nil {
			t.Fatalf("LoadConfigBytes: %v", err)
		}
		if back != *sample {
			t.Errorf("round-trip mismatch: got %+v want %+v", back, *sample)
		}
	})

	t.Run("normalizes ext without leading dot", func(t *testing.T) {
		data, err := DumpConfigBytes(sample, "json", nil)
		if err != nil {
			t.Fatalf("DumpConfigBytes: %v", err)
		}
		if !strings.Contains(string(data), `"Host"`) {
			t.Errorf("expected JSON output, got %q", string(data))
		}
	})

	t.Run("marshalFunc override wins", func(t *testing.T) {
		called := false
		custom := func(v any) ([]byte, error) {
			called = true
			return []byte("CUSTOM"), nil
		}
		data, err := DumpConfigBytes(sample, ".json", custom)
		if err != nil {
			t.Fatalf("DumpConfigBytes: %v", err)
		}
		if !called {
			t.Error("expected custom marshalFunc to be called")
		}
		if string(data) != "CUSTOM" {
			t.Errorf("expected CUSTOM, got %q", string(data))
		}
	})

	t.Run("dispatches to registered format marshaler", func(t *testing.T) {
		fakeMarshal := func(v any) ([]byte, error) {
			out, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			return append([]byte("FAKE:"), out...), nil
		}
		registerFormatCleanup(t, ".fakedump", ConfigFormat{
			Unmarshal: fakeUnmarshal,
			Marshal:   fakeMarshal,
			KeyTree:   fakeKeyTree,
		})

		data, err := DumpConfigBytes(sample, ".fakedump", nil)
		if err != nil {
			t.Fatalf("DumpConfigBytes: %v", err)
		}
		if !strings.HasPrefix(string(data), "FAKE:") {
			t.Errorf("expected FAKE-prefixed output, got %q", string(data))
		}
	})

	t.Run("registered format without Marshal returns clear error", func(t *testing.T) {
		registerFormatCleanup(t, ".nomarsh", ConfigFormat{
			Unmarshal: fakeUnmarshal,
			KeyTree:   fakeKeyTree,
		})

		_, err := DumpConfigBytes(sample, ".nomarsh", nil)
		if err == nil {
			t.Fatal("expected error when dumping format with no marshaler")
		}
		if !strings.Contains(err.Error(), "RegisterConfigMarshaler") {
			t.Errorf("expected error to hint at RegisterConfigMarshaler, got: %v", err)
		}
	})

	t.Run("unknown extension falls back to JSON pretty", func(t *testing.T) {
		data, err := DumpConfigBytes(sample, ".nope", nil)
		if err != nil {
			t.Fatalf("DumpConfigBytes: %v", err)
		}
		if !strings.Contains(string(data), `"Host": "h"`) {
			t.Errorf("expected pretty JSON fallback, got %q", string(data))
		}
	})

	t.Run("marshalFunc error is surfaced", func(t *testing.T) {
		boom := errors.New("boom")
		_, err := DumpConfigBytes(sample, "", func(v any) ([]byte, error) {
			return nil, boom
		})
		if !errors.Is(err, boom) {
			t.Errorf("expected wrapped boom error, got: %v", err)
		}
	})
}

func TestDumpConfigFile(t *testing.T) {
	type Params struct {
		Host string
		Port int
	}
	sample := &Params{Host: "h", Port: 9090}

	t.Run("writes pretty JSON to disk and round-trips", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.json")
		if err := DumpConfigFile(path, sample, nil); err != nil {
			t.Fatalf("DumpConfigFile: %v", err)
		}
		var back Params
		if err := LoadConfigFile(path, &back, nil); err != nil {
			t.Fatalf("LoadConfigFile: %v", err)
		}
		if back != *sample {
			t.Errorf("round-trip mismatch: got %+v want %+v", back, *sample)
		}
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.json")
		if err := os.WriteFile(path, []byte("stale"), 0644); err != nil {
			t.Fatalf("seed: %v", err)
		}
		if err := DumpConfigFile(path, sample, nil); err != nil {
			t.Fatalf("DumpConfigFile: %v", err)
		}
		data, _ := os.ReadFile(path)
		if !strings.Contains(string(data), `"Host"`) {
			t.Errorf("expected fresh JSON, got %q", string(data))
		}
	})

	t.Run("empty filePath returns user input error", func(t *testing.T) {
		err := DumpConfigFile("", sample, nil)
		if err == nil {
			t.Fatal("expected error for empty filePath")
		}
	})

	t.Run("bad directory is surfaced as a write error", func(t *testing.T) {
		err := DumpConfigFile("/nonexistent/dir/out.json", sample, nil)
		if err == nil {
			t.Fatal("expected error for unwritable path")
		}
	})

	t.Run("marshalFunc override is honored", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "out.json")
		custom := func(v any) ([]byte, error) { return []byte("HELLO"), nil }
		if err := DumpConfigFile(path, sample, custom); err != nil {
			t.Fatalf("DumpConfigFile: %v", err)
		}
		data, _ := os.ReadFile(path)
		if string(data) != "HELLO" {
			t.Errorf("expected HELLO, got %q", string(data))
		}
	})
}

func TestHookContext_DumpBytes_SourceAware(t *testing.T) {
	type DB struct {
		Host string `optional:"true"`
		Port int    `optional:"true"`
	}
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Name       string `optional:"true"`
		Age        int    `optional:"true" default:"42"`
		Unset      string `optional:"true"`
		Silent     bool   `optional:"true"`
		DB         DB
	}

	parseDump := func(t *testing.T, data []byte) map[string]any {
		t.Helper()
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatalf("unmarshal dump: %v\nraw: %s", err, string(data))
		}
		return m
	}

	t.Run("omits unset fields, keeps defaults and CLI values", func(t *testing.T) {
		var captured []byte
		CmdT[Params]{
			Use: "test",
			RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
				data, err := ctx.DumpBytes("", nil)
				if err != nil {
					t.Fatalf("DumpBytes: %v", err)
				}
				captured = data
			},
		}.RunArgs([]string{"--name", "alice", "--db-host", "h1"})

		m := parseDump(t, captured)
		if m["Name"] != "alice" {
			t.Errorf("expected Name=alice, got %v", m["Name"])
		}
		// Age has a default → should appear even without CLI set
		if v, ok := m["Age"]; !ok {
			t.Error("expected Age in dump (default should be pinned)")
		} else if int(v.(float64)) != 42 {
			t.Errorf("expected Age=42, got %v", v)
		}
		if _, ok := m["Unset"]; ok {
			t.Error("expected Unset to be omitted")
		}
		if _, ok := m["Silent"]; ok {
			t.Error("expected Silent to be omitted (never set, no default)")
		}
		db, ok := m["DB"].(map[string]any)
		if !ok {
			t.Fatalf("expected DB in dump, got %v", m["DB"])
		}
		if db["Host"] != "h1" {
			t.Errorf("expected DB.Host=h1, got %v", db["Host"])
		}
		if _, ok := db["Port"]; ok {
			t.Error("expected DB.Port to be omitted (never set)")
		}
	})

	t.Run("skips the configfile field itself", func(t *testing.T) {
		cfgPath := writeTestConfigFile(t, `{"Name":"bob"}`)

		var captured []byte
		CmdT[Params]{
			Use: "test",
			RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
				data, err := ctx.DumpBytes("", nil)
				if err != nil {
					t.Fatalf("DumpBytes: %v", err)
				}
				captured = data
			},
		}.RunArgs([]string{"--config-file", cfgPath})

		m := parseDump(t, captured)
		if _, ok := m["ConfigFile"]; ok {
			t.Error("expected configfile path to be omitted from dump")
		}
		if m["Name"] != "bob" {
			t.Errorf("expected Name=bob (from config file), got %v", m["Name"])
		}
	})

	t.Run("prunes nested struct when no descendant is set", func(t *testing.T) {
		var captured []byte
		CmdT[Params]{
			Use: "test",
			RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
				data, err := ctx.DumpBytes("", nil)
				if err != nil {
					t.Fatalf("DumpBytes: %v", err)
				}
				captured = data
			},
		}.RunArgs([]string{"--name", "solo"})

		m := parseDump(t, captured)
		if _, ok := m["DB"]; ok {
			t.Error("expected DB to be omitted when no descendant is set")
		}
	})

	t.Run("dump round-trips through LoadConfigBytes", func(t *testing.T) {
		// First run: set --name and --db-host, capture the dump.
		var dumped []byte
		CmdT[Params]{
			Use: "test",
			RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
				data, err := ctx.DumpBytes("", nil)
				if err != nil {
					t.Fatalf("DumpBytes: %v", err)
				}
				dumped = data
			},
		}.RunArgs([]string{"--name", "alice", "--db-host", "h1"})

		// Second run: no CLI args, load from dumped bytes in PreValidate.
		// Values should be recovered exactly, including pinned Age default.
		var seen Params
		CmdT[Params]{
			Use: "test",
			PreValidateFunc: func(p *Params, cmd *cobra.Command, args []string) error {
				return LoadConfigBytes(dumped, "", p, nil)
			},
			RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
				seen = *p
			},
		}.RunArgs([]string{})

		if seen.Name != "alice" {
			t.Errorf("round-trip Name: got %q want alice", seen.Name)
		}
		if seen.Age != 42 {
			t.Errorf("round-trip Age: got %d want 42", seen.Age)
		}
		if seen.DB.Host != "h1" {
			t.Errorf("round-trip DB.Host: got %q want h1", seen.DB.Host)
		}
		if seen.DB.Port != 0 {
			t.Errorf("round-trip DB.Port: got %d want 0 (not dumped)", seen.DB.Port)
		}
	})

	t.Run("DumpFile writes source-aware output and round-trips", func(t *testing.T) {
		dir := t.TempDir()
		outPath := filepath.Join(dir, "out.json")

		CmdT[Params]{
			Use: "test",
			RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
				if err := ctx.DumpFile(outPath, nil); err != nil {
					t.Fatalf("DumpFile: %v", err)
				}
			},
		}.RunArgs([]string{"--name", "carol"})

		data, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("read dump: %v", err)
		}
		m := parseDump(t, data)
		if m["Name"] != "carol" {
			t.Errorf("expected Name=carol in dump file, got %v", m["Name"])
		}
		if _, ok := m["Unset"]; ok {
			t.Error("expected Unset to be omitted")
		}
	})
}

func TestHookContext_DumpBytes_AdvancedTypes(t *testing.T) {
	// Embedded fields inside the top-level params exercise the "no named
	// prefix" path: children appear flat in the output rather than nested
	// under a wrapper key.
	type Embedded struct {
		EmbeddedField string `optional:"true"`
	}
	type Nested struct {
		Inner string `optional:"true"`
		Deep  struct {
			Value int `optional:"true"`
		}
	}
	type Params struct {
		Embedded
		StartAt time.Time     `optional:"true"`
		Timeout time.Duration `optional:"true"`
		Listen  net.IP        `optional:"true"`
		Tags    []string      `optional:"true"`
		Scores  []int         `optional:"true"`
		Labels  map[string]string
		Nested  Nested
	}

	// Run once with a rich set of values supplied via CLI.
	var dumped []byte
	CmdT[Params]{
		Use: "test",
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			data, err := ctx.DumpBytes("", nil)
			if err != nil {
				t.Fatalf("DumpBytes: %v", err)
			}
			dumped = data
		},
	}.RunArgs([]string{
		"--embedded-field", "emb",
		"--start-at", "2024-05-06T07:08:09Z",
		"--timeout", "1h30m",
		"--listen", "10.0.0.5",
		"--tags", "a,b,c",
		"--scores", "1,2,3",
		"--labels", "env=prod,tier=api",
		"--nested-inner", "n1",
		"--nested-deep-value", "99",
	})

	t.Logf("dumped:\n%s", string(dumped))

	// Parse into a plain map first so we can assert structure.
	var tree map[string]any
	if err := json.Unmarshal(dumped, &tree); err != nil {
		t.Fatalf("parse dump: %v", err)
	}

	// Embedded field should be flat at the top level, not under "Embedded".
	if tree["EmbeddedField"] != "emb" {
		t.Errorf("expected EmbeddedField=emb at top level, got %v (full=%v)", tree["EmbeddedField"], tree)
	}
	if _, stillWrapped := tree["Embedded"]; stillWrapped {
		t.Error("expected embedded fields to be flat, not under 'Embedded' key")
	}

	// time.Time serializes as RFC3339 string and round-trips through JSON.
	// net.IP implements TextMarshaler, so it round-trips as a plain string.
	// time.Duration serializes as int64 nanoseconds — works bidirectionally
	// when the load path runs through boa's custom handler, but stdlib
	// encoding/json will see it as a number (which json.Unmarshal cannot
	// convert back to time.Duration directly). We verify the full
	// round-trip via LoadConfigFile below, which runs boa's own
	// post-unmarshal sync and therefore handles the special types
	// correctly.

	// Nested struct with deeper sub-struct.
	nested, ok := tree["Nested"].(map[string]any)
	if !ok {
		t.Fatalf("expected Nested object, got %v", tree["Nested"])
	}
	if nested["Inner"] != "n1" {
		t.Errorf("expected Nested.Inner=n1, got %v", nested["Inner"])
	}
	deep, ok := nested["Deep"].(map[string]any)
	if !ok {
		t.Fatalf("expected Nested.Deep object, got %v", nested["Deep"])
	}
	if int(deep["Value"].(float64)) != 99 {
		t.Errorf("expected Nested.Deep.Value=99, got %v", deep["Value"])
	}

	// Slices
	tags, ok := tree["Tags"].([]any)
	if !ok || len(tags) != 3 {
		t.Errorf("expected Tags=[a b c], got %v", tree["Tags"])
	}
	scores, ok := tree["Scores"].([]any)
	if !ok || len(scores) != 3 {
		t.Errorf("expected Scores=[1 2 3], got %v", tree["Scores"])
	}

	// Map
	labels, ok := tree["Labels"].(map[string]any)
	if !ok || labels["env"] != "prod" || labels["tier"] != "api" {
		t.Errorf("expected Labels={env:prod tier:api}, got %v", tree["Labels"])
	}

	// Full boa round-trip via a file load: run a second command that loads
	// the dumped bytes via a configfile field, which exercises boa's own
	// sync path for special types (time.Time, time.Duration, net.IP).
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dump.json")
	if err := os.WriteFile(cfgPath, dumped, 0644); err != nil {
		t.Fatalf("write dump: %v", err)
	}

	type LoadParams struct {
		ConfigFile string `configfile:"true" optional:"true"`
		// Re-declare exactly as above so boa treats the dumped bytes as its
		// own config. (We can't reuse Params because of the embedded field's
		// cross-test-function visibility constraints.)
		EmbeddedField string            `optional:"true"`
		StartAt       time.Time         `optional:"true"`
		Timeout       time.Duration     `optional:"true"`
		Listen        net.IP            `optional:"true"`
		Tags          []string          `optional:"true"`
		Scores        []int             `optional:"true"`
		Labels        map[string]string `optional:"true"`
	}

	var got LoadParams
	CmdT[LoadParams]{
		Use: "test2",
		RunFunc: func(p *LoadParams, cmd *cobra.Command, args []string) {
			got = *p
		},
	}.RunArgs([]string{"--config-file", cfgPath})

	if got.EmbeddedField != "emb" {
		t.Errorf("round-trip EmbeddedField: got %q want emb", got.EmbeddedField)
	}
	wantTime, _ := time.Parse(time.RFC3339, "2024-05-06T07:08:09Z")
	if !got.StartAt.Equal(wantTime) {
		t.Errorf("round-trip StartAt: got %v want %v", got.StartAt, wantTime)
	}
	if got.Timeout != 90*time.Minute {
		t.Errorf("round-trip Timeout: got %v want 1h30m", got.Timeout)
	}
	if !got.Listen.Equal(net.ParseIP("10.0.0.5")) {
		t.Errorf("round-trip Listen: got %v want 10.0.0.5", got.Listen)
	}
	if !reflect.DeepEqual(got.Tags, []string{"a", "b", "c"}) {
		t.Errorf("round-trip Tags: got %v want [a b c]", got.Tags)
	}
	if !reflect.DeepEqual(got.Scores, []int{1, 2, 3}) {
		t.Errorf("round-trip Scores: got %v want [1 2 3]", got.Scores)
	}
	if got.Labels["env"] != "prod" || got.Labels["tier"] != "api" {
		t.Errorf("round-trip Labels: got %v want env=prod tier=api", got.Labels)
	}
}

func TestHookContext_DumpBytes_HonorsJSONTags(t *testing.T) {
	type Params struct {
		Host    string `json:"hostname" optional:"true"`
		Port    int    `json:"port,omitempty" optional:"true"`
		Private string `json:"-" optional:"true"`
		Plain   string `optional:"true"`
	}

	var dumped []byte
	CmdT[Params]{
		Use: "test",
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			data, err := ctx.DumpBytes("", nil)
			if err != nil {
				t.Fatalf("DumpBytes: %v", err)
			}
			dumped = data
		},
	// CLI flag names come from the Go field name, not the json tag — the
	// json tag only affects how the dump/load cycle serializes and parses
	// the value.
	}.RunArgs([]string{
		"--host", "h",
		"--port", "9000",
		"--private", "secret",
		"--plain", "p",
	})

	var m map[string]any
	if err := json.Unmarshal(dumped, &m); err != nil {
		t.Fatalf("parse: %v\nraw: %s", err, string(dumped))
	}

	// Tag value renames Host → hostname.
	if m["hostname"] != "h" {
		t.Errorf("expected hostname=h, got %v", m["hostname"])
	}
	if _, ok := m["Host"]; ok {
		t.Error("expected Host (Go field name) to be absent — json tag rename")
	}
	// omitempty suffix is ignored; we still emit the field because it's set.
	if int(m["port"].(float64)) != 9000 {
		t.Errorf("expected port=9000, got %v", m["port"])
	}
	// "-" tag means "skip entirely".
	if _, ok := m["Private"]; ok {
		t.Error("expected Private to be absent (json:\"-\")")
	}
	if _, ok := m["private"]; ok {
		t.Error("expected private to be absent (json:\"-\")")
	}
	// Untagged field falls back to Go field name.
	if m["Plain"] != "p" {
		t.Errorf("expected Plain=p (field-name fallback), got %v", m["Plain"])
	}

	// Full round-trip: the dumped bytes should re-load cleanly into the
	// same struct because json.Unmarshal honours json:"hostname" on the
	// target field too.
	var back Params
	if err := LoadConfigBytes(dumped, "", &back, nil); err != nil {
		t.Fatalf("LoadConfigBytes: %v", err)
	}
	if back.Host != "h" {
		t.Errorf("round-trip Host: got %q want h", back.Host)
	}
	if back.Port != 9000 {
		t.Errorf("round-trip Port: got %d want 9000", back.Port)
	}
	if back.Plain != "p" {
		t.Errorf("round-trip Plain: got %q want p", back.Plain)
	}
	if back.Private != "" {
		t.Errorf("round-trip Private: got %q want empty (json:\"-\" means no data)", back.Private)
	}
}

func TestHookContext_DumpBytes_OptionalPointerStruct(t *testing.T) {
	// Exercise the "pointer struct acts as optional parameter group" path:
	// when the user doesn't touch DB at all, DB should be completely absent
	// from the dump (not an empty {} and not a null field).
	type DB struct {
		Host string `optional:"true"`
		Port int    `optional:"true"`
	}
	type Params struct {
		Name string `optional:"true"`
		DB   *DB
	}

	t.Run("set → emitted, unset → omitted", func(t *testing.T) {
		var dumpWithDB []byte
		CmdT[Params]{
			Use: "test",
			RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
				data, _ := ctx.DumpBytes("", nil)
				dumpWithDB = data
			},
		}.RunArgs([]string{"--name", "n", "--db-host", "h"})

		var m1 map[string]any
		_ = json.Unmarshal(dumpWithDB, &m1)
		if _, ok := m1["DB"]; !ok {
			t.Error("expected DB in dump when at least one subfield set")
		}

		var dumpNoDB []byte
		CmdT[Params]{
			Use: "test",
			RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
				data, _ := ctx.DumpBytes("", nil)
				dumpNoDB = data
			},
		}.RunArgs([]string{"--name", "n"})

		var m2 map[string]any
		_ = json.Unmarshal(dumpNoDB, &m2)
		if _, ok := m2["DB"]; ok {
			t.Error("expected DB to be absent from dump when no subfield set")
		}
	})
}

func TestRegisterConfigMarshaler(t *testing.T) {
	t.Run("attaches marshaler to existing format", func(t *testing.T) {
		registerFormatCleanup(t, ".rw1", UniversalConfigFormat(fakeUnmarshal))

		marshalCalled := false
		RegisterConfigMarshaler(".rw1", func(v any) ([]byte, error) {
			marshalCalled = true
			return json.Marshal(v)
		})
		// Undo the marshaler via cleanup: registerFormatCleanup re-registers
		// the original entry, which has no Marshal, on test end.

		type P struct{ X int }
		if _, err := DumpConfigBytes(&P{X: 1}, ".rw1", nil); err != nil {
			t.Fatalf("DumpConfigBytes: %v", err)
		}
		if !marshalCalled {
			t.Error("expected marshaler to be called")
		}
	})

	t.Run("panics on nil", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("expected panic on nil marshalFunc")
			}
		}()
		RegisterConfigMarshaler(".rw2", nil)
	})
}
