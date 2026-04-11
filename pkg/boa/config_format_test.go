package boa

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

// fakeUnmarshal delegates to encoding/json so we can reuse JSON bytes on disk
// while still exercising the ConfigFormat plumbing under a non-".json"
// extension — this proves the extension lookup path is wired correctly.
func fakeUnmarshal(data []byte, target any) error {
	return json.Unmarshal(data, target)
}

// fakeKeyTree builds the KeyTree as a plain map[string]any. Matching the
// jsonKeyTree shape ensures the generic walker in markConfigKeysPresentInStruct
// is format-agnostic.
func fakeKeyTree(data []byte) (map[string]any, error) {
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func TestCustomConfigFormat_KeyTreeDetectsZeroValueWrite(t *testing.T) {
	RegisterConfigFormatFull(".fmtA", ConfigFormat{
		Unmarshal: fakeUnmarshal,
		KeyTree:   fakeKeyTree,
	})

	type Inner struct {
		Host string `descr:"host" default:"localhost"`
		Port int    `descr:"port" default:"5432"`
	}
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		DB         *Inner
	}

	// Write Host="" (zero value) AND Port=5432 (same as default).
	// Snapshot comparison alone cannot distinguish these from "never set",
	// so this is the acid test for KeyTree-based detection.
	raw := []byte(`{"DB":{"Host":"","Port":5432}}`)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "cfg.fmtA")
	if err := os.WriteFile(cfgPath, raw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var gotDB *Inner
	var dbSetByConfig bool
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			gotDB = p.DB
			if p.DB != nil {
				dbSetByConfig = ctx.HasValue(&p.DB.Host) && ctx.HasValue(&p.DB.Port)
			}
		},
	}).RunArgsE([]string{"--config-file", cfgPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotDB == nil {
		t.Fatal("expected DB pointer group to survive cleanup (zero-value + default writes should be detected via KeyTree)")
	}
	if !dbSetByConfig {
		t.Error("expected both Host and Port to report HasValue=true after KeyTree-based detection")
	}
}

func TestCustomConfigFormat_RegisteredFormatAppliesToCmd(t *testing.T) {
	// Use a dedicated extension with only Unmarshal — no KeyTree — and verify
	// the command still runs successfully (format resolution by extension,
	// graceful snapshot fallback for key-presence detection).
	RegisterConfigFormat(".fmtB", fakeUnmarshal)

	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Host       string `optional:"true"`
	}

	raw := []byte(`{"Host":"from-fmtB"}`)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "cfg.fmtB")
	if err := os.WriteFile(cfgPath, raw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var gotHost string
	err := (CmdT[Params]{
		Use: "test",
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotHost = p.Host
		},
	}).RunArgsE([]string{"--config-file", cfgPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHost != "from-fmtB" {
		t.Errorf("expected Host=from-fmtB, got %q", gotHost)
	}
}

func TestCustomConfigFormat_CmdConfigFormatOverridesExtension(t *testing.T) {
	// Register an extension-level handler that would mangle the data, then
	// use Cmd.ConfigFormat to override it per-command. The override must win.
	RegisterConfigFormatFull(".fmtC", ConfigFormat{
		Unmarshal: func(data []byte, target any) error {
			return fmt.Errorf("extension handler should not be called when Cmd.ConfigFormat is set")
		},
	})

	type Inner struct {
		Host string `descr:"host"`
	}
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		DB         *Inner
	}

	raw := []byte(`{"DB":{"Host":""}}`)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "cfg.fmtC")
	if err := os.WriteFile(cfgPath, raw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var gotDB *Inner
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		ConfigFormat: ConfigFormat{
			Unmarshal: fakeUnmarshal,
			KeyTree:   fakeKeyTree,
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotDB = p.DB
		},
	}).RunArgsE([]string{"--config-file", cfgPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The per-command override provides a KeyTree, so DB must survive cleanup
	// even though its only written field is the zero value.
	if gotDB == nil {
		t.Fatal("expected DB pointer group to be non-nil when Cmd.ConfigFormat with KeyTree is used")
	}
}

// TestCustomConfigFormat_MultipleFormatsOneBinary covers the headline scenario:
// a single compiled program registers a non-JSON format at init/startup and
// is then able to load EITHER a .json or a .fmtMulti file — picked per-run by
// --config-file extension — with no per-command override and no code changes
// between deployments.
func TestCustomConfigFormat_MultipleFormatsOneBinary(t *testing.T) {
	RegisterConfigFormatFull(".fmtMulti", ConfigFormat{
		Unmarshal: fakeUnmarshal,
		KeyTree:   fakeKeyTree,
	})

	type Inner struct {
		Host string `descr:"host" default:"localhost"`
		Port int    `descr:"port" default:"5432"`
	}
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		DB         *Inner
	}

	newCmd := func(captured **Inner, ctxCapture *bool, _ **HookContext) CmdT[Params] {
		return CmdT[Params]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
				*captured = p.DB
				if p.DB != nil {
					*ctxCapture = ctx.HasValue(&p.DB.Host) && ctx.HasValue(&p.DB.Port)
				}
			},
		}
	}

	dir := t.TempDir()

	// Pass 1: load a .json file. Uses the built-in JSON handler, including its KeyTree.
	jsonPath := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(jsonPath, []byte(`{"DB":{"Host":"","Port":5432}}`), 0o644); err != nil {
		t.Fatalf("write json: %v", err)
	}
	var gotDB *Inner
	var ctxHasBoth bool
	if err := newCmd(&gotDB, &ctxHasBoth, nil).RunArgsE([]string{"--config-file", jsonPath}); err != nil {
		t.Fatalf("json run: %v", err)
	}
	if gotDB == nil {
		t.Fatal("json run: expected DB pointer group to survive (KeyTree detects zero-value + default writes)")
	}
	if !ctxHasBoth {
		t.Error("json run: expected both DB.Host and DB.Port to report HasValue=true")
	}

	// Pass 2: SAME command struct, same process, different file extension.
	// Dispatch goes through the registered .fmtMulti handler — no per-command
	// override, no rebuild, just a different --config-file argument.
	altPath := filepath.Join(dir, "cfg.fmtMulti")
	if err := os.WriteFile(altPath, []byte(`{"DB":{"Host":"","Port":5432}}`), 0o644); err != nil {
		t.Fatalf("write alt: %v", err)
	}
	gotDB = nil
	ctxHasBoth = false
	if err := newCmd(&gotDB, &ctxHasBoth, nil).RunArgsE([]string{"--config-file", altPath}); err != nil {
		t.Fatalf("alt run: %v", err)
	}
	if gotDB == nil {
		t.Fatal("alt run: expected DB pointer group to survive under registered custom format")
	}
	if !ctxHasBoth {
		t.Error("alt run: expected both DB.Host and DB.Port to report HasValue=true via custom KeyTree")
	}
}

func TestCustomConfigFormat_KeyTreeHandlesMapAnyAny(t *testing.T) {
	// yaml.v2 and some other parsers produce map[any]any for nested mappings.
	// The walker's asKeyMap helper must coerce these transparently.
	RegisterConfigFormatFull(".fmtD", ConfigFormat{
		Unmarshal: fakeUnmarshal,
		KeyTree: func(data []byte) (map[string]any, error) {
			// Mimic yaml.v2: nested mappings come back as map[any]any.
			return map[string]any{
				"DB": map[any]any{
					"Host": "",
					"Port": 5432,
				},
			}, nil
		},
	})

	type Inner struct {
		Host string `descr:"host" default:"localhost"`
		Port int    `descr:"port" default:"5432"`
	}
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		DB         *Inner
	}

	// Bytes are JSON-parseable so fakeUnmarshal can populate the struct; the
	// KeyTree deliberately returns map[any]any to exercise coercion.
	raw := []byte(`{"DB":{"Host":"","Port":5432}}`)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "cfg.fmtD")
	if err := os.WriteFile(cfgPath, raw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var gotDB *Inner
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotDB = p.DB
		},
	}).RunArgsE([]string{"--config-file", cfgPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotDB == nil {
		t.Fatal("expected DB pointer group to survive when KeyTree returns map[any]any for nested mappings")
	}
}
