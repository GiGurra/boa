package boa

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
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
//
// All direct mutations of configFormats go through configFormatsMu so the
// race detector stays happy alongside concurrent test runs.
func registerFormatCleanup(t *testing.T, ext string, f ConfigFormat) {
	t.Helper()
	configFormatsMu.RLock()
	prev, hadPrev := configFormats[ext]
	configFormatsMu.RUnlock()
	RegisterConfigFormatFull(ext, f)
	t.Cleanup(func() {
		configFormatsMu.Lock()
		defer configFormatsMu.Unlock()
		if hadPrev {
			configFormats[ext] = prev
			return
		}
		delete(configFormats, ext)
	})
}

// TestUniversalConfigFormat_SynthesizesKeyTree proves that the one-liner
// helper produces a ConfigFormat whose KeyTree decodes via the same unmarshal
// function — no closure required from the caller.
func TestUniversalConfigFormat_SynthesizesKeyTree(t *testing.T) {
	cf := UniversalConfigFormat(fakeUnmarshal)
	if cf.Unmarshal == nil {
		t.Fatal("Unmarshal should be non-nil")
	}
	if cf.KeyTree == nil {
		t.Fatal("KeyTree should be auto-synthesized")
	}
	tree, err := cf.KeyTree([]byte(`{"DB":{"Host":"","Port":5432}}`))
	if err != nil {
		t.Fatalf("KeyTree returned error: %v", err)
	}
	db, ok := tree["DB"].(map[string]any)
	if !ok {
		t.Fatalf("expected DB to be map[string]any, got %T", tree["DB"])
	}
	if _, ok := db["Host"]; !ok {
		t.Error("expected DB.Host key in synthesized tree")
	}
	if _, ok := db["Port"]; !ok {
		t.Error("expected DB.Port key in synthesized tree")
	}
}

func TestUniversalConfigFormat_NilUnmarshalPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected UniversalConfigFormat(nil) to panic")
		}
		if msg := fmt.Sprintf("%v", r); !strings.Contains(msg, "UniversalConfigFormat") {
			t.Errorf("panic message should mention the helper name; got: %s", msg)
		}
	}()
	UniversalConfigFormat(nil)
}

// TestRegisterConfigFormat_SimpleFormGivesFullDetection covers the headline
// DX improvement: a plain RegisterConfigFormat(ext, fn) call — no closure,
// no ConfigFormat literal — should give the same zero-value and
// same-as-default detection that previously required the verbose full form.
func TestRegisterConfigFormat_SimpleFormGivesFullDetection(t *testing.T) {
	registerSimpleFormatCleanup(t, ".fmtSimple", fakeUnmarshal)

	type Inner struct {
		Host string `descr:"host" default:"localhost"`
		Port int    `descr:"port" default:"5432"`
	}
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		DB         *Inner
	}

	raw := []byte(`{"DB":{"Host":"","Port":5432}}`)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "cfg.fmtSimple")
	if err := os.WriteFile(cfgPath, raw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var gotDB *Inner
	var dbBothSetViaCfg bool
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			gotDB = p.DB
			if p.DB != nil {
				dbBothSetViaCfg = ctx.HasValue(&p.DB.Host) && ctx.HasValue(&p.DB.Port)
			}
		},
	}).RunArgsE([]string{"--config-file", cfgPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotDB == nil {
		t.Fatal("expected DB pointer group to survive cleanup under simple RegisterConfigFormat call (KeyTree should be auto-synthesized)")
	}
	if !dbBothSetViaCfg {
		t.Error("expected both DB.Host and DB.Port to report HasValue=true via auto-synthesized KeyTree")
	}
}

// registerSimpleFormatCleanup is the simple-form counterpart to
// registerFormatCleanup: it routes through RegisterConfigFormat (not
// ...Full) and still restores any prior registry entry on test completion.
func registerSimpleFormatCleanup(t *testing.T, ext string, fn func([]byte, any) error) {
	t.Helper()
	prev, hadPrev := configFormats[ext]
	RegisterConfigFormat(ext, fn)
	t.Cleanup(func() {
		if hadPrev {
			configFormats[ext] = prev
			return
		}
		delete(configFormats, ext)
	})
}

// TestConfigFormatExtensions_Sorted proves ConfigFormatExtensions returns a
// stable sorted slice, not randomized map iteration order. boaviper's
// FindConfig relies on a deterministic probe order when the same search path
// could match several registered extensions; without the sort, which file
// wins would depend on goroutine state.
func TestConfigFormatExtensions_Sorted(t *testing.T) {
	registerFormatCleanup(t, ".bbb", ConfigFormat{Unmarshal: fakeUnmarshal})
	registerFormatCleanup(t, ".aaa", ConfigFormat{Unmarshal: fakeUnmarshal})
	registerFormatCleanup(t, ".zzz", ConfigFormat{Unmarshal: fakeUnmarshal})

	exts := ConfigFormatExtensions()
	if !sort.StringsAreSorted(exts) {
		t.Fatalf("ConfigFormatExtensions should return sorted slice, got %v", exts)
	}
	// Quick sanity: all three test extensions must be present and .json too.
	seen := map[string]bool{}
	for _, e := range exts {
		seen[e] = true
	}
	for _, want := range []string{".aaa", ".bbb", ".zzz", ".json"} {
		if !seen[want] {
			t.Errorf("expected %q in extensions, got %v", want, exts)
		}
	}
}

// TestSnapshotFallbackIsScopedPerLoad reproduces the bug CodeRabbit flagged:
// a sub-load whose format lacks a KeyTree used to trigger a whole-tree
// snapshot fallback, which over-marked fields in *other* subtrees whose
// root-level KeyTree load had already covered them precisely.
//
// Setup:
//
//   - Root config (.json, built-in KeyTree): writes DB.Host to a non-default
//     value. Precise KeyTree detection should mark DB.Host as set-by-config
//     and leave DB.Port alone.
//   - Substruct config (.nokt, no KeyTree): forces a fallback that, with the
//     old whole-tree behaviour, would blanket-mark everything inside the DB
//     pointer group — including DB.Port — because snapshot comparison sees
//     DB changed from the root's write.
//
// With the per-subtree scoping fix, the fallback is limited to the substruct's
// own subtree, so DB.Port correctly reports HasValue=false.
func TestSnapshotFallbackIsScopedPerLoad(t *testing.T) {
	// .nokt: valid unmarshal, but deliberately no KeyTree → forces fallback.
	registerFormatCleanup(t, ".nokt", ConfigFormat{Unmarshal: fakeUnmarshal})

	type DB struct {
		Host string `descr:"db host" default:"localhost"`
		Port int    `descr:"db port" default:"5432"`
	}
	type SubCfg struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Note       string `descr:"sub note" default:"defaultnote"`
	}
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		DB         *DB
		Sub        SubCfg
	}

	dir := t.TempDir()

	// Root config writes DB.Host to a non-default value. KeyTree works.
	rootPath := filepath.Join(dir, "root.json")
	if err := os.WriteFile(rootPath, []byte(`{"DB":{"Host":"rootval"}}`), 0o644); err != nil {
		t.Fatalf("write root cfg: %v", err)
	}

	// Sub config uses the .nokt format (no KeyTree). Writes Sub.Note to a
	// non-default value so the sub load actually does something.
	subPath := filepath.Join(dir, "sub.nokt")
	if err := os.WriteFile(subPath, []byte(`{"Note":"fromsub"}`), 0o644); err != nil {
		t.Fatalf("write sub cfg: %v", err)
	}

	// Check setByConfig directly (we're in-package) instead of HasValue(),
	// because HasValue() also returns true when a parameter has a default,
	// which would confound this test — both DB.Host and DB.Port have
	// defaults, so only the setByConfig flag distinguishes "the config
	// file wrote this" from "the parameter has a default".
	var gotDB *DB
	var hostSetByConfig, portSetByConfig bool
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			gotDB = p.DB
			if p.DB == nil {
				return
			}
			if pm, ok := ctx.GetParam(&p.DB.Host).(*paramMeta); ok {
				hostSetByConfig = pm.setByConfig
			}
			if pm, ok := ctx.GetParam(&p.DB.Port).(*paramMeta); ok {
				portSetByConfig = pm.setByConfig
			}
		},
	}).RunArgsE([]string{"--config-file", rootPath, "--sub-config-file", subPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotDB == nil {
		t.Fatal("expected DB pointer group to survive (root KeyTree marked Host)")
	}
	if gotDB.Host != "rootval" {
		t.Errorf("DB.Host = %q, want rootval", gotDB.Host)
	}
	if !hostSetByConfig {
		t.Error("DB.Host setByConfig should be true (root KeyTree marked it precisely)")
	}
	// The headline assertion: without per-subtree scoping, the sub-load's
	// fallback would over-mark DB.Port here. With the fix, DB.Port is
	// correctly reported as NOT set by config.
	if portSetByConfig {
		t.Error("DB.Port setByConfig should be false — the sub-load's snapshot fallback must not leak into the DB subtree covered by the root KeyTree")
	}
}

// TestRegisterConfigFormatFull_NormalizesDotlessExtension covers the DX
// that a forgotten leading dot doesn't silently break dispatch. Both "yaml"
// and ".yaml" forms end up at the same canonical key, and a load of
// config.yaml actually reaches the registered handler (instead of falling
// through to JSON).
func TestRegisterConfigFormatFull_NormalizesDotlessExtension(t *testing.T) {
	// A sentinel Unmarshal that records whether it was called.
	var called bool
	sentinel := func(data []byte, target any) error {
		called = true
		return fakeUnmarshal(data, target)
	}

	// Register WITHOUT a leading dot.
	registerFormatCleanup(t, "fmtNoDot", UniversalConfigFormat(sentinel))

	// The canonical key in the registry should be ".fmtNoDot".
	configFormatsMu.RLock()
	_, canonicalOK := configFormats[".fmtNoDot"]
	_, rawOK := configFormats["fmtNoDot"]
	configFormatsMu.RUnlock()
	if !canonicalOK {
		t.Error("registration should normalize 'fmtNoDot' to '.fmtNoDot' in the registry")
	}
	if rawOK {
		t.Error("the dot-less form should not appear as a separate registry key")
	}

	// ConfigFormatExtensions should report the canonical form only.
	exts := ConfigFormatExtensions()
	var sawCanonical, sawRaw bool
	for _, e := range exts {
		if e == ".fmtNoDot" {
			sawCanonical = true
		}
		if e == "fmtNoDot" {
			sawRaw = true
		}
	}
	if !sawCanonical {
		t.Error("ConfigFormatExtensions should list the canonical '.fmtNoDot'")
	}
	if sawRaw {
		t.Error("ConfigFormatExtensions should not list the dot-less 'fmtNoDot' (would break boaviper's path construction)")
	}

	// Most important: actually loading a file through the normalized
	// registration path should hit the sentinel unmarshaler. Before this
	// fix, registering with "fmtNoDot" left the handler unreachable.
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Host       string `optional:"true"`
	}
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "cfg.fmtNoDot")
	if err := os.WriteFile(cfgPath, []byte(`{"Host":"x"}`), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	var gotHost string
	if err := (CmdT[Params]{
		Use: "test",
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotHost = p.Host
		},
	}).RunArgsE([]string{"--config-file", cfgPath}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !called {
		t.Error("sentinel Unmarshal should have been invoked via normalized registry key")
	}
	if gotHost != "x" {
		t.Errorf("Host = %q, want x", gotHost)
	}
}

// TestRegisterConfigFormatFull_DottedExtensionUnchanged confirms that the
// already-dotted form still works (no double-dot normalization bug).
func TestRegisterConfigFormatFull_DottedExtensionUnchanged(t *testing.T) {
	registerFormatCleanup(t, ".fmtDotted", UniversalConfigFormat(fakeUnmarshal))
	configFormatsMu.RLock()
	_, hasDotted := configFormats[".fmtDotted"]
	_, hasDoubleDot := configFormats["..fmtDotted"]
	configFormatsMu.RUnlock()
	if !hasDotted {
		t.Error("already-dotted form should be stored under '.fmtDotted'")
	}
	if hasDoubleDot {
		t.Error("normalization should not prepend a second dot when one is already present")
	}
}

// TestRegisterConfigFormatFull_EmptyExtPanics is the one case we still reject:
// an outright empty string is unambiguously a programming mistake.
func TestRegisterConfigFormatFull_EmptyExtPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected RegisterConfigFormatFull('') to panic")
		}
		if msg := fmt.Sprintf("%v", r); !strings.Contains(msg, "empty") {
			t.Errorf("panic message should mention emptiness; got: %s", msg)
		}
	}()
	RegisterConfigFormatFull("", ConfigFormat{Unmarshal: fakeUnmarshal})
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

// --- Mini "kvp" config format: a hand-rolled `key: value` parser used to
// prove the format-aware field-tag path end-to-end without pulling in a
// real YAML/TOML/HCL dependency. Supports:
//
//   - Flat scalar lines:           name: value
//   - Dot-nested keys:              svc.port: 8080
//   - String, int, and bool leaves
//   - Per-field renames via `kvp:"name"` struct tag, with `kvp:"-"`
//     to skip and `kvp:"name,opt,opt"` option-list handling
//   - `#` line comments and blank lines
//
// It deliberately does not support arrays / maps / multiline values —
// the failing test doesn't need those and the parser stays ~80 lines.
// ---

func miniKVUnmarshal(data []byte, target any) error {
	tree, err := miniKVKeyTree(data)
	if err != nil {
		return err
	}
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("mini-kv: target must be a non-nil pointer")
	}
	return miniKVAssign(v.Elem(), tree)
}

func miniKVKeyTree(data []byte) (map[string]any, error) {
	out := map[string]any{}
	for i, line := range strings.Split(string(data), "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		k, v, ok := strings.Cut(s, ":")
		if !ok {
			return nil, fmt.Errorf("mini-kv line %d: missing ':'", i+1)
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		// Support dot-nested keys: foo.bar.baz: v
		parts := strings.Split(k, ".")
		m := out
		for _, p := range parts[:len(parts)-1] {
			sub, ok := m[p].(map[string]any)
			if !ok {
				sub = map[string]any{}
				m[p] = sub
			}
			m = sub
		}
		m[parts[len(parts)-1]] = v
	}
	return out, nil
}

func miniKVAssign(v reflect.Value, tree map[string]any) error {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}
		key := sf.Name
		if tv := sf.Tag.Get("kvp"); tv != "" {
			name := strings.SplitN(tv, ",", 2)[0]
			switch name {
			case "-":
				continue
			case "":
				// bare options, e.g. `kvp:",omitempty"` → fall back to field name
			default:
				key = name
			}
		}
		// Case-insensitive raw-key lookup, matching encoding/json behaviour.
		raw, ok := tree[key]
		if !ok {
			for k, val := range tree {
				if strings.EqualFold(k, key) {
					raw, ok = val, true
					break
				}
			}
		}
		if !ok {
			continue
		}
		fv := v.Field(i)
		// Nested struct / *struct: recurse when the raw value is a map.
		ft := sf.Type
		inner := fv
		if ft.Kind() == reflect.Ptr {
			if fv.IsNil() {
				fv.Set(reflect.New(ft.Elem()))
			}
			inner = fv.Elem()
			ft = ft.Elem()
		}
		if sub, isMap := raw.(map[string]any); isMap && ft.Kind() == reflect.Struct {
			if err := miniKVAssign(inner, sub); err != nil {
				return err
			}
			continue
		}
		// Leaf — the raw value is a string scalar. For slice and map
		// fields we decode a mini CSV / key=value shape inline so the
		// complex-struct tests can exercise those kinds without pulling
		// in a real YAML/TOML parser.
		s, _ := raw.(string)
		if err := miniKVAssignScalar(inner, s, sf.Name); err != nil {
			return err
		}
	}
	return nil
}

func miniKVAssignScalar(inner reflect.Value, s, fieldName string) error {
	switch inner.Kind() {
	case reflect.String:
		inner.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return fmt.Errorf("mini-kv %s: %w", fieldName, err)
		}
		inner.SetInt(n)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return fmt.Errorf("mini-kv %s: %w", fieldName, err)
		}
		inner.SetBool(b)
	case reflect.Slice:
		// Comma-separated scalar slice (e.g. "a,b,c" or "1,2,3").
		// Empty string means an empty (but non-nil) slice so the
		// mirror sees "something was written here".
		parts := []string{}
		if s != "" {
			parts = strings.Split(s, ",")
		}
		sl := reflect.MakeSlice(inner.Type(), len(parts), len(parts))
		for i, p := range parts {
			if err := miniKVAssignScalar(sl.Index(i), strings.TrimSpace(p), fieldName); err != nil {
				return err
			}
		}
		inner.Set(sl)
	case reflect.Map:
		// `k1=v1;k2=v2` inline map. Value parsing delegates back to
		// miniKVAssignScalar so map[string]int etc. Just Work™.
		if inner.IsNil() {
			inner.Set(reflect.MakeMapWithSize(inner.Type(), 0))
		}
		if s == "" {
			return nil
		}
		for _, pair := range strings.Split(s, ";") {
			k, v, ok := strings.Cut(pair, "=")
			if !ok {
				return fmt.Errorf("mini-kv %s: map entry %q missing '='", fieldName, pair)
			}
			keyVal := reflect.New(inner.Type().Key()).Elem()
			if err := miniKVAssignScalar(keyVal, strings.TrimSpace(k), fieldName); err != nil {
				return err
			}
			valVal := reflect.New(inner.Type().Elem()).Elem()
			if err := miniKVAssignScalar(valVal, strings.TrimSpace(v), fieldName); err != nil {
				return err
			}
			inner.SetMapIndex(keyVal, valVal)
		}
	}
	return nil
}

// TestConfigFile_FormatAwareFieldTag_MiniKV reproduces — and, after the
// fix, exercises — the behaviour that config-file set-detection honours
// the format-appropriate struct tag. Before the fix, the walker hard-
// coded a `json` tag lookup so a field renamed via ANY other tag
// (`yaml`, `toml`, `hcl`, or a custom `kvp` here) was never marked as
// set-by-config. After the fix, tag resolution lives on the dump/load
// shared helper (structTagForExt) and defaults to the file extension
// minus its leading dot, so registering `.kvp` is all that's needed —
// no extra plumbing, no ConfigFormat field.
func TestConfigFile_FormatAwareFieldTag_MiniKV(t *testing.T) {
	registerFormatCleanup(t, ".kvp", ConfigFormat{
		Unmarshal: miniKVUnmarshal,
		KeyTree:   miniKVKeyTree,
	})

	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		// kvp tag renames the field. No json tag — the walker must not
		// fall back to the Go field name here; it must consult `kvp`
		// because the file extension is ".kvp".
		Retries int `descr:"retries" kvp:"retry_count" optional:"true"`
	}

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "cfg.kvp")
	if err := os.WriteFile(cfgPath, []byte("retry_count: 0\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	// Check setByConfig directly — HasValue would also be true if the
	// parameter had a default, which would confound this test. Here we
	// want: "the config file mentioned this key, therefore setByConfig".
	var gotRetries int
	var retriesSetByConfig bool
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			gotRetries = p.Retries
			if pm, ok := ctx.GetParam(&p.Retries).(*paramMeta); ok {
				retriesSetByConfig = pm.setByConfig
			}
		},
	}).RunArgsE([]string{"--config-file", cfgPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The mini parser itself is correct (it reads the kvp tag), so value
	// loading works regardless of the bug — this assertion guards the
	// test's own fixture.
	if gotRetries != 0 {
		t.Errorf("Retries = %d, want 0 from config", gotRetries)
	}
	// The headline assertion: the walker must recognise the kvp tag so
	// it marks Retries as set-by-config, even though the value happens
	// to be the zero value.
	if !retriesSetByConfig {
		t.Error("Retries.setByConfig should be true — kvp:\"retry_count\" is present in the config, but set-detection is only consulting the json tag")
	}
}

// TestConfigFile_FormatAwareFieldTag_MiniKV_NonZeroValue is the
// complement to the zero-value test — with a non-zero value the
// snapshot fallback can also catch the change, so this test pins the
// "happy path" and guards against a regression where the fix stopped
// running the key-presence walker for non-zero writes.
func TestConfigFile_FormatAwareFieldTag_MiniKV_NonZeroValue(t *testing.T) {
	registerFormatCleanup(t, ".kvp", ConfigFormat{
		Unmarshal: miniKVUnmarshal,
		KeyTree:   miniKVKeyTree,
	})

	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Retries    int    `descr:"retries" kvp:"retry_count" optional:"true"`
	}

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "cfg.kvp")
	if err := os.WriteFile(cfgPath, []byte("retry_count: 7\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	var gotRetries int
	var retriesSetByConfig bool
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			gotRetries = p.Retries
			if pm, ok := ctx.GetParam(&p.Retries).(*paramMeta); ok {
				retriesSetByConfig = pm.setByConfig
			}
		},
	}).RunArgsE([]string{"--config-file", cfgPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotRetries != 7 {
		t.Errorf("Retries = %d, want 7", gotRetries)
	}
	if !retriesSetByConfig {
		t.Error("Retries.setByConfig should be true for a non-zero write via kvp:\"retry_count\"")
	}
}

// TestConfigFile_FormatAwareFieldTag_MiniKV_NestedPointerGroup is the
// most realistic shape: an optional struct-pointer group (e.g. a
// database section) whose child fields are renamed via kvp tags, and
// whose values in the config happen to be zero. Without the fix, the
// walker never matches the renamed keys, cleanup nils out the group,
// and the user's "DB section was explicitly configured" signal is lost.
func TestConfigFile_FormatAwareFieldTag_MiniKV_NestedPointerGroup(t *testing.T) {
	registerFormatCleanup(t, ".kvp", ConfigFormat{
		Unmarshal: miniKVUnmarshal,
		KeyTree:   miniKVKeyTree,
	})

	type DB struct {
		Host string `descr:"db host" kvp:"host_name" default:"localhost"`
		Port int    `descr:"db port" kvp:"listen_port" default:"5432"`
	}
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		DB         *DB
	}

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "cfg.kvp")
	// Both leaves are the zero value for their Go type, so only a
	// working KeyTree + canonicalisation can keep the DB pointer alive
	// past cleanup.
	raw := []byte("db.host_name: \ndb.listen_port: 0\n")
	if err := os.WriteFile(cfgPath, raw, 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	var gotDB *DB
	var hostSet, portSet bool
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			gotDB = p.DB
			if p.DB != nil {
				if pm, ok := ctx.GetParam(&p.DB.Host).(*paramMeta); ok {
					hostSet = pm.setByConfig
				}
				if pm, ok := ctx.GetParam(&p.DB.Port).(*paramMeta); ok {
					portSet = pm.setByConfig
				}
			}
		},
	}).RunArgsE([]string{"--config-file", cfgPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotDB == nil {
		t.Fatal("DB pointer group should survive cleanup when children are mentioned in config via kvp tags")
	}
	if !hostSet {
		t.Error("DB.Host.setByConfig should be true — kvp:\"host_name\" present in the config")
	}
	if !portSet {
		t.Error("DB.Port.setByConfig should be true — kvp:\"listen_port\" present in the config")
	}
}

// TestConfigFile_FormatAwareFieldTag_MiniKV_TagDashSkipsField pins the
// tag-value-"-" contract: a format that explicitly excludes a field
// (e.g. `kvp:"-"`) must not have boa mark that field as set-by-config,
// even if a same-named key happens to appear in the config file.
// This matches how encoding/json handles `json:"-"`.
func TestConfigFile_FormatAwareFieldTag_MiniKV_TagDashSkipsField(t *testing.T) {
	registerFormatCleanup(t, ".kvp", ConfigFormat{
		Unmarshal: miniKVUnmarshal,
		KeyTree:   miniKVKeyTree,
	})

	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Secret     string `descr:"secret" kvp:"-" optional:"true"`
	}

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "cfg.kvp")
	// A stray line mentioning "Secret" — the kvp:"-" tag means the
	// format owner declared this field off-limits for kvp files,
	// so set-detection must not pick it up.
	if err := os.WriteFile(cfgPath, []byte("Secret: leaked\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	var secretSet bool
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			if pm, ok := ctx.GetParam(&p.Secret).(*paramMeta); ok {
				secretSet = pm.setByConfig
			}
		},
	}).RunArgsE([]string{"--config-file", cfgPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secretSet {
		t.Error("Secret should NOT be marked setByConfig — kvp:\"-\" means the format excludes this field")
	}
}

// TestConfigFile_FormatAwareFieldTag_MiniKV_TagWithOptions covers the
// `name,opt1,opt2` tag value shape that encoding/json popularised and
// every mainstream format parser inherits. Only the first comma-
// separated segment is the raw key name; the rest are format-specific
// options (omitempty, inline, …) that boa doesn't care about.
func TestConfigFile_FormatAwareFieldTag_MiniKV_TagWithOptions(t *testing.T) {
	registerFormatCleanup(t, ".kvp", ConfigFormat{
		Unmarshal: miniKVUnmarshal,
		KeyTree:   miniKVKeyTree,
	})

	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Name       string `descr:"name" kvp:"display_name,omitempty,extra" optional:"true"`
	}

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "cfg.kvp")
	if err := os.WriteFile(cfgPath, []byte("display_name: boa\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	var gotName string
	var nameSet bool
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			gotName = p.Name
			if pm, ok := ctx.GetParam(&p.Name).(*paramMeta); ok {
				nameSet = pm.setByConfig
			}
		},
	}).RunArgsE([]string{"--config-file", cfgPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName != "boa" {
		t.Errorf("Name = %q, want boa", gotName)
	}
	if !nameSet {
		t.Error("Name.setByConfig should be true — tag options after the first comma must not break key matching")
	}
}

// TestConfigFile_FormatAwareFieldTag_MiniKV_CaseInsensitiveLookup
// locks in encoding/json's case-insensitive matching rule for non-
// JSON formats too, since the canonicalization step reuses the same
// configKeyLookup helper. A config file that writes `DISPLAY_NAME`
// (upper-cased) must still match a field tagged `kvp:"display_name"`.
func TestConfigFile_FormatAwareFieldTag_MiniKV_CaseInsensitiveLookup(t *testing.T) {
	registerFormatCleanup(t, ".kvp", ConfigFormat{
		Unmarshal: miniKVUnmarshal,
		KeyTree:   miniKVKeyTree,
	})

	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Name       string `descr:"name" kvp:"display_name" optional:"true"`
	}

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "cfg.kvp")
	if err := os.WriteFile(cfgPath, []byte("DISPLAY_NAME: boa\n"), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	var nameSet bool
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			if pm, ok := ctx.GetParam(&p.Name).(*paramMeta); ok {
				nameSet = pm.setByConfig
			}
		},
	}).RunArgsE([]string{"--config-file", cfgPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !nameSet {
		t.Error("Name.setByConfig should be true — case-insensitive raw-key match must still work with a custom tag")
	}
}

// TestConfigFile_FormatAwareFieldTag_YmlExtensionUsesYamlTag verifies
// the one hand-wired special case in structTagForExt: `.yml` files map
// to the `yaml` struct tag, because yaml parsers (yaml.v2, yaml.v3,
// goccy/go-yaml, …) all consult the yaml tag regardless of which file
// extension the user wrote. Without this special case, a `.yml` file
// would try the `yml` tag, which nobody uses.
func TestConfigFile_FormatAwareFieldTag_YmlExtensionUsesYamlTag(t *testing.T) {
	// Fake yaml: reuse fakeUnmarshal (JSON bytes) and synthesize a
	// KeyTree that uses yaml-style key names. Registering `.yml`
	// directly pins the special case to the load path.
	registerFormatCleanup(t, ".yml", ConfigFormat{
		Unmarshal: fakeUnmarshal,
		KeyTree: func(data []byte) (map[string]any, error) {
			return map[string]any{"retry_count": 0}, nil
		},
	})

	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Retries    int    `descr:"retries" yaml:"retry_count" optional:"true"`
	}

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "cfg.yml")
	if err := os.WriteFile(cfgPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	var retriesSet bool
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			if pm, ok := ctx.GetParam(&p.Retries).(*paramMeta); ok {
				retriesSet = pm.setByConfig
			}
		},
	}).RunArgsE([]string{"--config-file", cfgPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !retriesSet {
		t.Error(".yml files should consult the `yaml` struct tag (structTagForExt special case)")
	}
}

// TestConfigFile_FormatAwareFieldTag_UnknownExtDefaultsToExtMinusDot
// pins the "ext minus leading dot" fallback in structTagForExt: any
// registered extension that isn't in the explicit special-case list
// picks up the extension name as its struct tag by default. So
// registering `.bespoke` automatically consults the `bespoke` tag
// with no extra configuration.
func TestConfigFile_FormatAwareFieldTag_UnknownExtDefaultsToExtMinusDot(t *testing.T) {
	registerFormatCleanup(t, ".bespoke", ConfigFormat{
		Unmarshal: fakeUnmarshal,
		KeyTree: func(data []byte) (map[string]any, error) {
			return map[string]any{"magic_field": 0}, nil
		},
	})

	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Magic      int    `descr:"magic" bespoke:"magic_field" optional:"true"`
	}

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "cfg.bespoke")
	if err := os.WriteFile(cfgPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	var magicSet bool
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			if pm, ok := ctx.GetParam(&p.Magic).(*paramMeta); ok {
				magicSet = pm.setByConfig
			}
		},
	}).RunArgsE([]string{"--config-file", cfgPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !magicSet {
		t.Error("Magic.setByConfig should be true — `.bespoke` should auto-resolve to the `bespoke` struct tag")
	}
}

// TestConfigFile_FormatAwareFieldTag_ZeroIntFlatJsonNoTag regresses the
// original user report: a JSON config file writes an integer field to
// 0, the struct has no rename tag, and the field sits at the top level
// (flat — no optional pointer group). Before the fix, set-by-config
// detection was gated on `len(PreallocatedPtrs) > 0`, so a flat struct
// with a zero-value write was never marked. After the fix, the walker
// runs for every root load, so the zero write is correctly detected.
func TestConfigFile_FormatAwareFieldTag_ZeroIntFlatJsonNoTag(t *testing.T) {
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Retries    int    `descr:"retries" optional:"true"`
	}

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(cfgPath, []byte(`{"Retries": 0}`), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	var gotRetries int
	var retriesSet bool
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			gotRetries = p.Retries
			if pm, ok := ctx.GetParam(&p.Retries).(*paramMeta); ok {
				retriesSet = pm.setByConfig
			}
		},
	}).RunArgsE([]string{"--config-file", cfgPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotRetries != 0 {
		t.Errorf("Retries = %d, want 0", gotRetries)
	}
	if !retriesSet {
		t.Error("Retries.setByConfig should be true for a flat zero-value JSON write — this was the original bug report")
	}
}

// TestConfigFile_FormatAwareFieldTag_MixedFormatsInOneBinary proves
// that per-format tag resolution is fully per-load: the SAME struct
// type can be loaded from two different files in the same process,
// each honouring its own rename convention. `.json` uses `json` tags,
// `.kvp` uses `kvp` tags, no cross-contamination.
func TestConfigFile_FormatAwareFieldTag_MixedFormatsInOneBinary(t *testing.T) {
	registerFormatCleanup(t, ".kvp", ConfigFormat{
		Unmarshal: miniKVUnmarshal,
		KeyTree:   miniKVKeyTree,
	})

	// Two different tag names on the same field — the format-aware
	// walker must pick the one matching each file's extension.
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Retries    int    `descr:"retries" json:"json_retries" kvp:"kvp_retries" optional:"true"`
	}

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(jsonPath, []byte(`{"json_retries": 3}`), 0o644); err != nil {
		t.Fatalf("write json: %v", err)
	}
	kvpPath := filepath.Join(dir, "cfg.kvp")
	if err := os.WriteFile(kvpPath, []byte("kvp_retries: 5\n"), 0o644); err != nil {
		t.Fatalf("write kvp: %v", err)
	}

	run := func(cfgPath string) (int, bool) {
		var gotRetries int
		var retriesSet bool
		err := (CmdT[Params]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
				gotRetries = p.Retries
				if pm, ok := ctx.GetParam(&p.Retries).(*paramMeta); ok {
					retriesSet = pm.setByConfig
				}
			},
		}).RunArgsE([]string{"--config-file", cfgPath})
		if err != nil {
			t.Fatalf("RunArgsE(%s): %v", cfgPath, err)
		}
		return gotRetries, retriesSet
	}

	jsonRetries, jsonSet := run(jsonPath)
	if jsonRetries != 3 {
		t.Errorf(".json load: Retries = %d, want 3 (via json tag)", jsonRetries)
	}
	if !jsonSet {
		t.Error(".json load: Retries.setByConfig should be true")
	}

	kvpRetries, kvpSet := run(kvpPath)
	if kvpRetries != 5 {
		t.Errorf(".kvp load: Retries = %d, want 5 (via kvp tag)", kvpRetries)
	}
	if !kvpSet {
		t.Error(".kvp load: Retries.setByConfig should be true")
	}
}

// --- Deep / complex config-file tests ---
//
// These tests exercise the whole config-file machinery on a realistic
// three-level shape (app → service → network), with mixed scalars,
// slices, maps, renamed fields, and optional struct-pointer groups.
// Each test picks a single config file format (JSON or the custom
// kvp format), writes either a partial or a full payload, and asserts
// two things end-to-end:
//
//   1. Every field that the config mentioned loaded to the expected
//      value.
//   2. Every mentioned field reports setByConfig=true, and fields that
//      the config did NOT mention report setByConfig=false, so the
//      "was this written by the user?" signal stays trustworthy even
//      when a written value equals the Go zero value or the default.
//
// Both tests share the same struct shape so we can compare apples to
// apples across formats.

type deepNet struct {
	// Leaf scalars, a slice, and a map — the four kinds the walker
	// treats as terminal nodes.
	Host    string            `descr:"host" json:"host_name" kvp:"host_name" default:"localhost"`
	Port    int               `descr:"port" json:"listen_port" kvp:"listen_port" default:"8080"`
	TLS     bool              `descr:"tls" json:"tls_enabled" kvp:"tls_enabled" optional:"true"`
	Origins []string          `descr:"allowed origins" json:"allowed_origins" kvp:"allowed_origins" optional:"true"`
	Labels  map[string]string `descr:"labels" json:"labels" kvp:"labels" optional:"true"`
}

type deepSvc struct {
	Name    string `descr:"service name" json:"svc_name" kvp:"svc_name" optional:"true"`
	Retries int    `descr:"retries" json:"retry_count" kvp:"retry_count" default:"3"`
	// Nested optional pointer group — the "level 3" target.
	Net *deepNet `json:"net" kvp:"net"`
}

type deepApp struct {
	ConfigFile string `configfile:"true" optional:"true"`
	AppName    string `descr:"app name" json:"app_name" kvp:"app_name" default:"boa-app"`
	// Nested optional pointer group — the "level 2" target. deepSvc
	// contains a further pointer group so key-presence detection has
	// to survive two boundaries to reach the deepest leaves.
	Svc *deepSvc `json:"svc" kvp:"svc"`
}

func runDeepApp(t *testing.T, cfgPath string) (*deepApp, map[string]bool) {
	t.Helper()
	var got deepApp
	marked := map[string]bool{}
	// Record setByConfig for every mirror we care about. Using direct
	// paramMeta access keeps the assertions precise — HasValue is
	// polluted by defaults and would give spurious passes.
	err := (CmdT[deepApp]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *deepApp, cmd *cobra.Command, args []string) {
			got = *p
			record := func(label string, addr any) {
				if pm, ok := ctx.GetParam(addr).(*paramMeta); ok {
					marked[label] = pm.setByConfig
				} else {
					marked[label] = false
				}
			}
			record("app_name", &p.AppName)
			if p.Svc != nil {
				record("svc_name", &p.Svc.Name)
				record("retries", &p.Svc.Retries)
				if p.Svc.Net != nil {
					record("host", &p.Svc.Net.Host)
					record("port", &p.Svc.Net.Port)
					record("tls", &p.Svc.Net.TLS)
					record("origins", &p.Svc.Net.Origins)
					record("labels", &p.Svc.Net.Labels)
				}
			}
		},
	}).RunArgsE([]string{"--config-file", cfgPath})
	if err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	return &got, marked
}

func assertMarked(t *testing.T, marked map[string]bool, wantSet, wantUnset []string) {
	t.Helper()
	for _, k := range wantSet {
		if !marked[k] {
			t.Errorf("expected %q setByConfig=true, got false", k)
		}
	}
	for _, k := range wantUnset {
		if marked[k] {
			t.Errorf("expected %q setByConfig=false, got true", k)
		}
	}
}

// TestConfigFile_Deep_Json_FullPayload exercises the full three-level
// tree in JSON, with every leaf kind populated, including zero-value
// writes and same-as-default writes that would defeat a snapshot-only
// detector.
func TestConfigFile_Deep_Json_FullPayload(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "cfg.json")
	// Deliberate mix: host="" (zero), port=8080 (same-as-default),
	// tls=false (zero), origins=[] (empty slice), labels={} (empty
	// map), retry_count=3 (same-as-default). Only KeyTree-based
	// detection can mark all of these.
	raw := []byte(`{
		"app_name": "boa-app",
		"svc": {
			"svc_name": "api",
			"retry_count": 3,
			"net": {
				"host_name": "",
				"listen_port": 8080,
				"tls_enabled": false,
				"allowed_origins": [],
				"labels": {}
			}
		}
	}`)
	if err := os.WriteFile(cfgPath, raw, 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	got, marked := runDeepApp(t, cfgPath)

	// Values.
	if got.AppName != "boa-app" {
		t.Errorf("AppName = %q, want boa-app", got.AppName)
	}
	if got.Svc == nil {
		t.Fatal("Svc pointer group should survive cleanup (config mentioned it)")
	}
	if got.Svc.Name != "api" {
		t.Errorf("Svc.Name = %q, want api", got.Svc.Name)
	}
	if got.Svc.Retries != 3 {
		t.Errorf("Svc.Retries = %d, want 3", got.Svc.Retries)
	}
	if got.Svc.Net == nil {
		t.Fatal("Svc.Net pointer group should survive cleanup")
	}
	if got.Svc.Net.Host != "" {
		t.Errorf("Svc.Net.Host = %q, want empty", got.Svc.Net.Host)
	}
	if got.Svc.Net.Port != 8080 {
		t.Errorf("Svc.Net.Port = %d, want 8080", got.Svc.Net.Port)
	}
	if got.Svc.Net.TLS != false {
		t.Errorf("Svc.Net.TLS = %v, want false", got.Svc.Net.TLS)
	}

	// Every leaf the config mentioned must be reported as set.
	assertMarked(t, marked,
		[]string{"app_name", "svc_name", "retries", "host", "port", "tls", "origins", "labels"},
		nil,
	)
}

// TestConfigFile_Deep_Json_PartialPayload covers the "partial config"
// case: some leaves are present, others are absent. The present leaves
// must report setByConfig=true, the absent ones must report false —
// even when the absent fields have defaults that make HasValue true.
// This is the reason the test reads setByConfig directly instead of
// relying on HasValue.
func TestConfigFile_Deep_Json_PartialPayload(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "cfg.json")
	// Only two leaves written. Everything else must come from
	// defaults (AppName="boa-app", Retries=3, Port=8080, Host="localhost")
	// but defaults must NOT leak into setByConfig.
	raw := []byte(`{
		"svc": {
			"net": {
				"tls_enabled": true,
				"allowed_origins": ["https://a", "https://b"]
			}
		}
	}`)
	if err := os.WriteFile(cfgPath, raw, 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	got, marked := runDeepApp(t, cfgPath)

	if got.AppName != "boa-app" {
		t.Errorf("AppName = %q, want default boa-app", got.AppName)
	}
	if got.Svc == nil || got.Svc.Net == nil {
		t.Fatal("Svc/Svc.Net should survive — config mentioned them")
	}
	if got.Svc.Net.TLS != true {
		t.Errorf("Svc.Net.TLS = %v, want true", got.Svc.Net.TLS)
	}
	if len(got.Svc.Net.Origins) != 2 || got.Svc.Net.Origins[0] != "https://a" {
		t.Errorf("Svc.Net.Origins = %v, want [https://a https://b]", got.Svc.Net.Origins)
	}

	// Exactly the written keys should be marked; the rest must not be.
	assertMarked(t, marked,
		[]string{"tls", "origins"},
		[]string{"app_name", "svc_name", "retries", "host", "port", "labels"},
	)
}

// TestConfigFile_Deep_MiniKV_FullPayload mirrors the JSON full-payload
// test with the custom kvp format, so the three-level walk, the slice
// and map leaves, the zero-value / same-as-default writes, and the
// per-field renames are all exercised together under a non-JSON
// format's struct tag.
func TestConfigFile_Deep_MiniKV_FullPayload(t *testing.T) {
	registerFormatCleanup(t, ".kvp", ConfigFormat{
		Unmarshal: miniKVUnmarshal,
		KeyTree:   miniKVKeyTree,
	})

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "cfg.kvp")
	// Dot-nested keys walk into nested structs; slice is comma
	// separated, map is `k=v;k=v`.
	raw := []byte(strings.Join([]string{
		"app_name: boa-app",
		"svc.svc_name: api",
		"svc.retry_count: 3",
		"svc.net.host_name: ",
		"svc.net.listen_port: 8080",
		"svc.net.tls_enabled: false",
		"svc.net.allowed_origins: https://a,https://b",
		"svc.net.labels: team=core;tier=gold",
		"",
	}, "\n"))
	if err := os.WriteFile(cfgPath, raw, 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	got, marked := runDeepApp(t, cfgPath)

	if got.Svc == nil || got.Svc.Net == nil {
		t.Fatal("Svc/Svc.Net should survive — kvp config mentioned both")
	}
	if got.Svc.Name != "api" {
		t.Errorf("Svc.Name = %q, want api", got.Svc.Name)
	}
	if got.Svc.Retries != 3 {
		t.Errorf("Svc.Retries = %d, want 3", got.Svc.Retries)
	}
	if got.Svc.Net.Host != "" {
		t.Errorf("Svc.Net.Host = %q, want empty (zero-value write)", got.Svc.Net.Host)
	}
	if got.Svc.Net.Port != 8080 {
		t.Errorf("Svc.Net.Port = %d, want 8080", got.Svc.Net.Port)
	}
	if len(got.Svc.Net.Origins) != 2 || got.Svc.Net.Origins[0] != "https://a" {
		t.Errorf("Svc.Net.Origins = %v, want [https://a https://b]", got.Svc.Net.Origins)
	}
	if got.Svc.Net.Labels["team"] != "core" || got.Svc.Net.Labels["tier"] != "gold" {
		t.Errorf("Svc.Net.Labels = %v, want team=core tier=gold", got.Svc.Net.Labels)
	}

	assertMarked(t, marked,
		[]string{"app_name", "svc_name", "retries", "host", "port", "tls", "origins", "labels"},
		nil,
	)
}

// TestConfigFile_Deep_MiniKV_PartialPayload is the kvp counterpart to
// TestConfigFile_Deep_Json_PartialPayload: a partial config that only
// mentions a couple of deep leaves. Un-mentioned fields must still
// report setByConfig=false, and the optional pointer groups above a
// mentioned leaf must stay alive through cleanup.
func TestConfigFile_Deep_MiniKV_PartialPayload(t *testing.T) {
	registerFormatCleanup(t, ".kvp", ConfigFormat{
		Unmarshal: miniKVUnmarshal,
		KeyTree:   miniKVKeyTree,
	})

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "cfg.kvp")
	raw := []byte("svc.net.listen_port: 0\nsvc.net.tls_enabled: true\n")
	if err := os.WriteFile(cfgPath, raw, 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	got, marked := runDeepApp(t, cfgPath)

	if got.AppName != "boa-app" {
		t.Errorf("AppName = %q, want default boa-app", got.AppName)
	}
	if got.Svc == nil || got.Svc.Net == nil {
		t.Fatal("Svc/Svc.Net pointer groups should survive — config reached into them")
	}
	if got.Svc.Net.Port != 0 {
		t.Errorf("Svc.Net.Port = %d, want 0 (zero-value kvp write)", got.Svc.Net.Port)
	}
	if !got.Svc.Net.TLS {
		t.Errorf("Svc.Net.TLS = %v, want true", got.Svc.Net.TLS)
	}

	assertMarked(t, marked,
		[]string{"port", "tls"},
		[]string{"app_name", "svc_name", "retries", "host", "origins", "labels"},
	)
}
