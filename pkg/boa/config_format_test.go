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

// registerFormatCleanup registers a format and schedules restoration of the
// previous registry entry (if any) on test completion. Snapshot + restore
// keeps the suite hermetic even when a test overrides a built-in or
// previously-registered format — a blind delete would silently drop those.
func registerFormatCleanup(t *testing.T, ext string, f ConfigFormat) {
	t.Helper()
	prev, hadPrev := configFormats[ext]
	RegisterConfigFormatFull(ext, f)
	t.Cleanup(func() {
		if hadPrev {
			configFormats[ext] = prev
			return
		}
		delete(configFormats, ext)
	})
}

func TestRegisterConfigFormatFull_NilUnmarshalPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected RegisterConfigFormatFull to panic when Unmarshal is nil")
		}
		// Also assert the panic message mentions the extension so users can
		// trace the source without a stack walk.
		if msg := fmt.Sprintf("%v", r); !strings.Contains(msg, ".fmtBroken") {
			t.Errorf("panic message should mention the bad extension; got: %s", msg)
		}
	}()
	RegisterConfigFormatFull(".fmtBroken", ConfigFormat{
		// Unmarshal deliberately left nil — a user who forgot to wire it up.
		KeyTree: fakeKeyTree,
	})
}

func TestCustomConfigFormat_KeyTreeDetectsZeroValueWrite(t *testing.T) {
	registerFormatCleanup(t, ".fmtA", ConfigFormat{
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
	registerFormatCleanup(t, ".fmtB", ConfigFormat{Unmarshal: fakeUnmarshal})

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
	registerFormatCleanup(t, ".fmtC", ConfigFormat{
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
	registerFormatCleanup(t, ".fmtMulti", ConfigFormat{
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

// TestCustomConfigFormat_DeepNestingUnderCustomFormat proves that the KeyTree
// walker is format-agnostic and descends arbitrarily deep. Three levels of
// optional struct-pointer groups, plus a non-pointer substruct in the middle
// for good measure — every leaf value is written as either the zero value or
// the parameter's default, so only a working KeyTree-based detector can keep
// the pointer chain alive after cleanup.
func TestCustomConfigFormat_DeepNestingUnderCustomFormat(t *testing.T) {
	registerFormatCleanup(t, ".fmtDeep", ConfigFormat{
		Unmarshal: fakeUnmarshal,
		KeyTree:   fakeKeyTree,
	})

	type Leaf struct {
		Host string `descr:"leaf host" default:"localhost"`
		Port int    `descr:"leaf port" default:"5432"`
	}
	type Middle struct {
		// Non-pointer substruct inside a pointer group.
		Region string `descr:"middle region" default:"us-east-1"`
		Deepest *Leaf
	}
	type Top struct {
		Name   string `descr:"top name" default:"primary"`
		Middle *Middle
	}
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Top        *Top
	}

	// Every written value either equals the zero value or equals the default.
	// Snapshot comparison alone would say "nothing changed" and nil all three
	// pointers out during cleanup.
	raw := []byte(`{
		"Top": {
			"Name": "primary",
			"Middle": {
				"Region": "us-east-1",
				"Deepest": {
					"Host": "",
					"Port": 5432
				}
			}
		}
	}`)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "deep.fmtDeep")
	if err := os.WriteFile(cfgPath, raw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var gotTop *Top
	var deepestHostSet, deepestPortSet bool
	var middleRegionSet, topNameSet bool
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			gotTop = p.Top
			if p.Top != nil {
				topNameSet = ctx.HasValue(&p.Top.Name)
				if p.Top.Middle != nil {
					middleRegionSet = ctx.HasValue(&p.Top.Middle.Region)
					if p.Top.Middle.Deepest != nil {
						deepestHostSet = ctx.HasValue(&p.Top.Middle.Deepest.Host)
						deepestPortSet = ctx.HasValue(&p.Top.Middle.Deepest.Port)
					}
				}
			}
		},
	}).RunArgsE([]string{"--config-file", cfgPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotTop == nil {
		t.Fatal("expected Top pointer group to survive cleanup (level 1)")
	}
	if gotTop.Middle == nil {
		t.Fatal("expected Top.Middle pointer group to survive cleanup (level 2)")
	}
	if gotTop.Middle.Deepest == nil {
		t.Fatal("expected Top.Middle.Deepest pointer group to survive cleanup (level 3)")
	}
	if !topNameSet {
		t.Error("level 1 leaf: expected Top.Name to report HasValue=true (same-as-default write)")
	}
	if !middleRegionSet {
		t.Error("level 2 leaf (inside pointer group): expected Top.Middle.Region to report HasValue=true")
	}
	if !deepestHostSet {
		t.Error("level 3 leaf (pointer inside pointer): expected Top.Middle.Deepest.Host to report HasValue=true (zero-value write)")
	}
	if !deepestPortSet {
		t.Error("level 3 leaf (pointer inside pointer): expected Top.Middle.Deepest.Port to report HasValue=true (same-as-default write)")
	}
}

func TestCustomConfigFormat_KeyTreeHandlesMapAnyAny(t *testing.T) {
	// yaml.v2 and some other parsers produce map[any]any for nested mappings.
	// The walker's asKeyMap helper must coerce these transparently.
	registerFormatCleanup(t, ".fmtD", ConfigFormat{
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
