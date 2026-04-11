package boa

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/spf13/cobra"
)

// reloadTestParams covers the main bits we care about on reload: plain
// scalars, a field with a default, a field only ever set via config, and a
// configfile path.
type reloadTestParams struct {
	ConfigFile string `configfile:"true" optional:"true"`
	Host       string `optional:"true"`
	Port       int    `optional:"true" default:"8080"`
	Region     string `optional:"true"`
}

func writeReloadFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return p
}

// --- Core reload semantics ---

func TestReload_PicksUpConfigFileEdits(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeReloadFile(t, dir, "cfg.json", `{"Host":"initial","Region":"us"}`)

	var (
		firstHost   string
		firstRegion string
		secondHost  string
		secondRegion string
	)

	CmdT[reloadTestParams]{
		Use: "test",
		RunFuncCtx: func(ctx *HookContext, p *reloadTestParams, cmd *cobra.Command, args []string) {
			firstHost = p.Host
			firstRegion = p.Region

			// Edit the file on disk and reload.
			if err := os.WriteFile(cfgPath, []byte(`{"Host":"updated","Region":"eu"}`), 0644); err != nil {
				t.Fatalf("rewrite: %v", err)
			}
			fresh, err := Reload[reloadTestParams](ctx)
			if err != nil {
				t.Fatalf("Reload: %v", err)
			}
			secondHost = fresh.Host
			secondRegion = fresh.Region

			// Original struct must be unchanged — reload returns a fresh allocation.
			if p.Host != "initial" {
				t.Errorf("old struct mutated: Host=%q want 'initial'", p.Host)
			}
			if p.Region != "us" {
				t.Errorf("old struct mutated: Region=%q want 'us'", p.Region)
			}
			// Fresh struct pointer must differ from the original.
			if fresh == p {
				t.Error("expected fresh allocation, got same pointer as original params")
			}
		},
	}.RunArgs([]string{"--config-file", cfgPath})

	if firstHost != "initial" || firstRegion != "us" {
		t.Errorf("first read unexpected: host=%q region=%q", firstHost, firstRegion)
	}
	if secondHost != "updated" || secondRegion != "eu" {
		t.Errorf("reload failed to pick up edits: host=%q region=%q", secondHost, secondRegion)
	}
}

func TestReload_CLIStillWinsAfterReload(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeReloadFile(t, dir, "cfg.json", `{"Host":"from-file","Region":"us"}`)

	var (
		firstHost  string
		secondHost string
	)

	CmdT[reloadTestParams]{
		Use: "test",
		RunFuncCtx: func(ctx *HookContext, p *reloadTestParams, cmd *cobra.Command, args []string) {
			firstHost = p.Host // "from-cli" should win at startup

			// File change shouldn't affect Host — CLI still wins on reload.
			if err := os.WriteFile(cfgPath, []byte(`{"Host":"changed-in-file","Region":"eu"}`), 0644); err != nil {
				t.Fatalf("rewrite: %v", err)
			}
			fresh, err := Reload[reloadTestParams](ctx)
			if err != nil {
				t.Fatalf("Reload: %v", err)
			}
			secondHost = fresh.Host
			if fresh.Region != "eu" {
				t.Errorf("expected Region=eu from reloaded file, got %q", fresh.Region)
			}
		},
	}.RunArgs([]string{"--config-file", cfgPath, "--host", "from-cli"})

	if firstHost != "from-cli" {
		t.Errorf("first read: expected CLI host, got %q", firstHost)
	}
	if secondHost != "from-cli" {
		t.Errorf("reload must preserve CLI Host: got %q want 'from-cli'", secondHost)
	}
}

func TestReload_DefaultsArePinned(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeReloadFile(t, dir, "cfg.json", `{"Host":"h"}`)

	CmdT[reloadTestParams]{
		Use: "test",
		RunFuncCtx: func(ctx *HookContext, p *reloadTestParams, cmd *cobra.Command, args []string) {
			if p.Port != 8080 {
				t.Errorf("startup Port: got %d want 8080 (default)", p.Port)
			}
			fresh, err := Reload[reloadTestParams](ctx)
			if err != nil {
				t.Fatalf("Reload: %v", err)
			}
			if fresh.Port != 8080 {
				t.Errorf("reload Port: got %d want 8080 (default re-applied)", fresh.Port)
			}
		},
	}.RunArgs([]string{"--config-file", cfgPath})
}

// --- Validation failure semantics ---

func TestReload_ValidationFailureLeavesOldStateIntact(t *testing.T) {
	type StrictParams struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Port       int    `optional:"true" min:"1" max:"65535"`
	}
	dir := t.TempDir()
	cfgPath := writeReloadFile(t, dir, "cfg.json", `{"Port":3000}`)

	CmdT[StrictParams]{
		Use: "test",
		RunFuncCtx: func(ctx *HookContext, p *StrictParams, cmd *cobra.Command, args []string) {
			if p.Port != 3000 {
				t.Errorf("startup: expected Port=3000, got %d", p.Port)
			}

			// Rewrite the file with an out-of-range port.
			if err := os.WriteFile(cfgPath, []byte(`{"Port":70000}`), 0644); err != nil {
				t.Fatalf("rewrite: %v", err)
			}
			fresh, err := Reload[StrictParams](ctx)
			if err == nil {
				t.Fatalf("expected validation error on reload, got fresh=%+v", fresh)
			}
			if fresh != nil {
				t.Errorf("expected nil fresh params on error, got %+v", fresh)
			}
			// Old state must be untouched.
			if p.Port != 3000 {
				t.Errorf("old struct mutated after failed reload: Port=%d want 3000", p.Port)
			}
		},
	}.RunArgs([]string{"--config-file", cfgPath})
}

func TestReload_MalformedFileReturnsErrorAndPreservesOldState(t *testing.T) {
	// A partially-written file (e.g. the editor/deploy process caught
	// mid-write) must produce a parse error, NOT a silent fallback to
	// defaults. The caller's existing struct stays intact.
	dir := t.TempDir()
	cfgPath := writeReloadFile(t, dir, "cfg.json", `{"Host":"good","Region":"us"}`)

	CmdT[reloadTestParams]{
		Use: "test",
		RunFuncCtx: func(ctx *HookContext, p *reloadTestParams, cmd *cobra.Command, args []string) {
			if p.Host != "good" {
				t.Fatalf("startup: Host=%q want 'good'", p.Host)
			}

			// Corrupt the file — truncated JSON mid-key.
			if err := os.WriteFile(cfgPath, []byte(`{"Host":"partial`), 0644); err != nil {
				t.Fatal(err)
			}
			fresh, err := Reload[reloadTestParams](ctx)
			if err == nil {
				t.Fatalf("expected parse error on malformed JSON, got fresh=%+v", fresh)
			}
			if fresh != nil {
				t.Errorf("expected nil fresh on parse error, got %+v", fresh)
			}
			// Error should name the file so the operator can see what went wrong.
			if !strings.Contains(err.Error(), cfgPath) {
				t.Errorf("expected error to name the bad file, got: %v", err)
			}
			// Old state must be untouched — reload is transactional.
			if p.Host != "good" || p.Region != "us" {
				t.Errorf("old struct mutated after failed reload: %+v", *p)
			}
		},
	}.RunArgs([]string{"--config-file", cfgPath})
}

func TestReload_MissingFileDuringReloadReturnsError(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeReloadFile(t, dir, "cfg.json", `{"Host":"h"}`)

	CmdT[reloadTestParams]{
		Use: "test",
		RunFuncCtx: func(ctx *HookContext, p *reloadTestParams, cmd *cobra.Command, args []string) {
			if err := os.Remove(cfgPath); err != nil {
				t.Fatalf("rm: %v", err)
			}
			_, err := Reload[reloadTestParams](ctx)
			if err == nil {
				t.Fatal("expected error when reload target no longer exists")
			}
		},
	}.RunArgs([]string{"--config-file", cfgPath})
}

// --- Multi-file overlay reload ---

func TestReload_MultiFileOverlay(t *testing.T) {
	type Params struct {
		ConfigFiles []string `configfile:"true" optional:"true"`
		Host        string   `optional:"true"`
		Port        int      `optional:"true"`
		Region      string   `optional:"true"`
	}

	dir := t.TempDir()
	base := writeReloadFile(t, dir, "base.json", `{"Host":"base","Port":80,"Region":"us"}`)
	local := writeReloadFile(t, dir, "local.json", `{"Port":8080}`)

	CmdT[Params]{
		Use: "test",
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			if p.Host != "base" || p.Port != 8080 || p.Region != "us" {
				t.Errorf("startup: got %+v", *p)
			}

			// Rewrite both files. local now changes Region too; base changes Host.
			if err := os.WriteFile(base, []byte(`{"Host":"base2","Port":70,"Region":"us2"}`), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(local, []byte(`{"Port":9090,"Region":"eu"}`), 0644); err != nil {
				t.Fatal(err)
			}

			fresh, err := Reload[Params](ctx)
			if err != nil {
				t.Fatalf("Reload: %v", err)
			}
			if fresh.Host != "base2" {
				t.Errorf("reload Host: got %q want 'base2' (only base sets it)", fresh.Host)
			}
			if fresh.Port != 9090 {
				t.Errorf("reload Port: got %d want 9090 (local wins)", fresh.Port)
			}
			if fresh.Region != "eu" {
				t.Errorf("reload Region: got %q want 'eu' (local wins)", fresh.Region)
			}
		},
	}.RunArgs([]string{"--config-files", base + "," + local})
}

// --- WatchedConfigFiles + WatchConfigFile registration ---

func TestWatchedConfigFiles_AutoTracksConfigfileTag(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeReloadFile(t, dir, "cfg.json", `{"Host":"h"}`)

	var watched []string
	CmdT[reloadTestParams]{
		Use: "test",
		RunFuncCtx: func(ctx *HookContext, p *reloadTestParams, cmd *cobra.Command, args []string) {
			watched = ctx.WatchedConfigFiles()
		},
	}.RunArgs([]string{"--config-file", cfgPath})

	if len(watched) != 1 || watched[0] != cfgPath {
		t.Errorf("expected WatchedConfigFiles=[%q], got %v", cfgPath, watched)
	}
}

func TestWatchedConfigFiles_IncludesMultiFileChain(t *testing.T) {
	type Params struct {
		ConfigFiles []string `configfile:"true" optional:"true"`
		Host        string   `optional:"true"`
	}
	dir := t.TempDir()
	base := writeReloadFile(t, dir, "base.json", `{"Host":"a"}`)
	local := writeReloadFile(t, dir, "local.json", `{"Host":"b"}`)

	var watched []string
	CmdT[Params]{
		Use: "test",
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			watched = ctx.WatchedConfigFiles()
		},
	}.RunArgs([]string{"--config-files", base + "," + local})

	if len(watched) != 2 {
		t.Fatalf("expected 2 watched files, got %v", watched)
	}
	if watched[0] != base || watched[1] != local {
		t.Errorf("expected [%q, %q], got %v", base, local, watched)
	}
}

func TestWatchedConfigFiles_ExtraRegistrationFromHook(t *testing.T) {
	type Params struct {
		Host string `optional:"true"`
	}
	dir := t.TempDir()
	extra := writeReloadFile(t, dir, "extra.json", `{"Host":"from-hook"}`)

	var watched []string
	CmdT[Params]{
		Use: "test",
		PreValidateFunc: func(p *Params, cmd *cobra.Command, args []string) error {
			return LoadConfigFile(extra, p, nil)
		},
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			// User manually wires the watch since LoadConfigFile can't auto-track.
			ctx.WatchConfigFile(extra)
			watched = ctx.WatchedConfigFiles()
		},
	}.RunArgs([]string{})

	if len(watched) != 1 || watched[0] != extra {
		t.Errorf("expected [%q], got %v", extra, watched)
	}
}

func TestWatchedConfigFiles_EmptyRegistrationIgnored(t *testing.T) {
	var watched []string
	CmdT[reloadTestParams]{
		Use: "test",
		RunFuncCtx: func(ctx *HookContext, p *reloadTestParams, cmd *cobra.Command, args []string) {
			ctx.WatchConfigFile("")
			ctx.WatchConfigFile("")
			watched = ctx.WatchedConfigFiles()
		},
	}.RunArgs([]string{})

	if len(watched) != 0 {
		t.Errorf("expected empty, got %v", watched)
	}
}

// --- Safety / concurrency contract ---

func TestReload_FreshAllocationPerCall(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeReloadFile(t, dir, "cfg.json", `{"Host":"a"}`)

	CmdT[reloadTestParams]{
		Use: "test",
		RunFuncCtx: func(ctx *HookContext, p *reloadTestParams, cmd *cobra.Command, args []string) {
			first, err := Reload[reloadTestParams](ctx)
			if err != nil {
				t.Fatalf("first reload: %v", err)
			}
			second, err := Reload[reloadTestParams](ctx)
			if err != nil {
				t.Fatalf("second reload: %v", err)
			}
			if first == second {
				t.Error("expected distinct pointers across reloads")
			}
			if first == p || second == p {
				t.Error("reload must not return the original params pointer")
			}
		},
	}.RunArgs([]string{"--config-file", cfgPath})
}

func TestReload_AtomicSwapPattern(t *testing.T) {
	// Demonstrates the idiomatic way to use Reload in a long-running program:
	// keep an atomic.Pointer and swap on successful reload. Readers never
	// see a half-updated struct.
	dir := t.TempDir()
	cfgPath := writeReloadFile(t, dir, "cfg.json", `{"Host":"a","Region":"us"}`)

	var active atomic.Pointer[reloadTestParams]

	CmdT[reloadTestParams]{
		Use: "test",
		RunFuncCtx: func(ctx *HookContext, p *reloadTestParams, cmd *cobra.Command, args []string) {
			active.Store(p)

			if err := os.WriteFile(cfgPath, []byte(`{"Host":"b","Region":"eu"}`), 0644); err != nil {
				t.Fatal(err)
			}
			fresh, err := Reload[reloadTestParams](ctx)
			if err != nil {
				t.Fatalf("Reload: %v", err)
			}
			active.Store(fresh)
		},
	}.RunArgs([]string{"--config-file", cfgPath})

	cur := active.Load()
	if cur == nil {
		t.Fatal("no active params")
	}
	if cur.Host != "b" || cur.Region != "eu" {
		t.Errorf("active params after swap: got %+v", *cur)
	}
}

// --- Error messaging ---

func TestReload_ClearErrorWhenFactoryMissing(t *testing.T) {
	// Build a HookContext without the reload factory (i.e. one constructed
	// directly via Cmd, not CmdT[T]). This exercises the "no factory" error
	// path so users don't get a cryptic nil-pointer panic.
	ctx := &HookContext{ctx: &processingContext{}}
	_, err := Reload[reloadTestParams](ctx)
	if err == nil {
		t.Fatal("expected error when reloadFactory is nil")
	}
	if !strings.Contains(err.Error(), "no reload factory") {
		t.Errorf("expected 'no reload factory' error, got: %v", err)
	}
}

func TestReload_SequentialReloadsSeeProgressiveEdits(t *testing.T) {
	// Simulates a file being edited N times; each reload must see the
	// latest state. Catches "reload reads from a stale snapshot" bugs
	// where we might accidentally cache the first load's bytes.
	dir := t.TempDir()
	cfgPath := writeReloadFile(t, dir, "cfg.json", `{"Host":"v0"}`)

	var seen []string
	CmdT[reloadTestParams]{
		Use: "test",
		RunFuncCtx: func(ctx *HookContext, p *reloadTestParams, cmd *cobra.Command, args []string) {
			seen = append(seen, p.Host)
			for i := 1; i <= 3; i++ {
				if err := os.WriteFile(cfgPath, []byte(`{"Host":"v`+string(rune('0'+i))+`"}`), 0644); err != nil {
					t.Fatal(err)
				}
				fresh, err := Reload[reloadTestParams](ctx)
				if err != nil {
					t.Fatalf("reload %d: %v", i, err)
				}
				seen = append(seen, fresh.Host)
			}
		},
	}.RunArgs([]string{"--config-file", cfgPath})

	want := []string{"v0", "v1", "v2", "v3"}
	if len(seen) != len(want) {
		t.Fatalf("expected %v, got %v", want, seen)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Errorf("step %d: got %q want %q", i, seen[i], want[i])
		}
	}
}

func TestWatchedConfigFiles_ExtraRegistrationSurvivesReload(t *testing.T) {
	// A PreValidateFunc that manually loads a file and registers it for
	// watching needs the registration to persist across reload — the
	// hook re-runs during the replay, so it re-registers cleanly.
	type Params struct {
		Host string `optional:"true"`
	}
	dir := t.TempDir()
	extra := writeReloadFile(t, dir, "extra.json", `{"Host":"v0"}`)

	var (
		firstWatched  []string
		secondWatched []string
	)
	CmdT[Params]{
		Use: "test",
		PreValidateFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) error {
			if err := LoadConfigFile(extra, p, nil); err != nil {
				return err
			}
			ctx.WatchConfigFile(extra)
			return nil
		},
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			firstWatched = ctx.WatchedConfigFiles()

			if err := os.WriteFile(extra, []byte(`{"Host":"v1"}`), 0644); err != nil {
				t.Fatal(err)
			}
			fresh, err := Reload[Params](ctx)
			if err != nil {
				t.Fatalf("Reload: %v", err)
			}
			if fresh.Host != "v1" {
				t.Errorf("reload didn't pick up PreValidate-loaded file edit: got %q", fresh.Host)
			}
			secondWatched = ctx.WatchedConfigFiles()
		},
	}.RunArgs([]string{})

	if len(firstWatched) != 1 || firstWatched[0] != extra {
		t.Errorf("first watched: got %v", firstWatched)
	}
	// The outer ctx still belongs to the original command — its registry
	// was NOT touched by the inner reload (which ran against a fresh ctx),
	// so it should still report the single extra path.
	if len(secondWatched) != 1 || secondWatched[0] != extra {
		t.Errorf("second watched: got %v", secondWatched)
	}
}

func TestReload_TypeMismatchProducesClearError(t *testing.T) {
	type Params struct {
		Host string `optional:"true"`
	}
	type OtherParams struct {
		Host string `optional:"true"`
	}

	var reloadErr error
	CmdT[Params]{
		Use: "test",
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			// Ask for the wrong type.
			_, err := Reload[OtherParams](ctx)
			reloadErr = err
		},
	}.RunArgs([]string{})

	if reloadErr == nil {
		t.Fatal("expected type mismatch error")
	}
	if !errors.Is(reloadErr, reloadErr) { // sanity: non-nil
		t.Fatal("unreachable")
	}
	if !strings.Contains(reloadErr.Error(), "not *") {
		t.Errorf("expected type mismatch hint, got: %v", reloadErr)
	}
}
