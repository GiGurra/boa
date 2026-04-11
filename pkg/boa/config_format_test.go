package boa

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
