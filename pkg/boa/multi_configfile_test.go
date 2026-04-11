package boa

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// writeFile is a tiny helper for the overlay tests — keeps each test body
// focused on the overlay semantics rather than temp-file plumbing.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return p
}

// --- []string configfile tag: tag-driven overlay chain ---

func TestMultiConfigFile_StringSliceTag(t *testing.T) {
	type Params struct {
		ConfigFiles []string `configfile:"true" optional:"true"`
		Host        string   `optional:"true"`
		Port        int      `optional:"true"`
		Region      string   `optional:"true"`
		Tags        []string `optional:"true"`
	}

	t.Run("later file overlays earlier at the key level", func(t *testing.T) {
		dir := t.TempDir()
		base := writeFile(t, dir, "base.json", `{"Host":"base","Port":80,"Region":"us"}`)
		local := writeFile(t, dir, "local.json", `{"Port":8080,"Region":"eu"}`)

		var got Params
		CmdT[Params]{
			Use: "test",
			RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
				got = *p
			},
		}.RunArgs([]string{"--config-files", base + "," + local})

		if got.Host != "base" {
			t.Errorf("Host: expected 'base' (from base, not overridden), got %q", got.Host)
		}
		if got.Port != 8080 {
			t.Errorf("Port: expected 8080 (local overlays base), got %d", got.Port)
		}
		if got.Region != "eu" {
			t.Errorf("Region: expected 'eu' (local overlays base), got %q", got.Region)
		}
	})

	t.Run("slice field is fully replaced by the later file", func(t *testing.T) {
		dir := t.TempDir()
		base := writeFile(t, dir, "base.json", `{"Tags":["a","b"]}`)
		local := writeFile(t, dir, "local.json", `{"Tags":["c"]}`)

		var got Params
		CmdT[Params]{
			Use: "test",
			RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
				got = *p
			},
		}.RunArgs([]string{"--config-files", base + "," + local})

		if len(got.Tags) != 1 || got.Tags[0] != "c" {
			t.Errorf("Tags: expected [c] (full replace), got %v", got.Tags)
		}
	})

	t.Run("single-element list behaves like a plain string configfile", func(t *testing.T) {
		dir := t.TempDir()
		only := writeFile(t, dir, "only.json", `{"Host":"solo","Port":1}`)

		var got Params
		CmdT[Params]{
			Use: "test",
			RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
				got = *p
			},
		}.RunArgs([]string{"--config-files", only})

		if got.Host != "solo" || got.Port != 1 {
			t.Errorf("single-element: got %+v", got)
		}
	})

	t.Run("empty list is a no-op", func(t *testing.T) {
		var got Params
		CmdT[Params]{
			Use: "test",
			RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
				got = *p
			},
		}.RunArgs([]string{})

		if got.Host != "" || got.Port != 0 {
			t.Errorf("expected zero params with empty config-files, got %+v", got)
		}
	})

	t.Run("missing file in chain returns a clear error", func(t *testing.T) {
		dir := t.TempDir()
		base := writeFile(t, dir, "base.json", `{"Host":"base"}`)

		err := CmdT[Params]{
			Use: "test",
			RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
				t.Fatal("should not run")
			},
			RawArgs: []string{"--config-files", base + ",/nonexistent/overlay.json"},
		}.Validate()

		if err == nil {
			t.Fatal("expected error for missing file in chain")
		}
		if !strings.Contains(err.Error(), "/nonexistent/overlay.json") {
			t.Errorf("expected error to name the bad file, got: %v", err)
		}
	})

	t.Run("CLI overrides every file in the chain", func(t *testing.T) {
		dir := t.TempDir()
		base := writeFile(t, dir, "base.json", `{"Host":"base","Port":80}`)
		local := writeFile(t, dir, "local.json", `{"Port":8080}`)

		var got Params
		CmdT[Params]{
			Use: "test",
			RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
				got = *p
			},
		}.RunArgs([]string{
			"--config-files", base + "," + local,
			"--host", "from-cli",
			"--port", "9999",
		})

		if got.Host != "from-cli" {
			t.Errorf("expected Host=from-cli (CLI wins), got %q", got.Host)
		}
		if got.Port != 9999 {
			t.Errorf("expected Port=9999 (CLI wins), got %d", got.Port)
		}
	})
}

// --- HasValue tracking across overlay chains ---

func TestMultiConfigFile_HasValueAcrossChain(t *testing.T) {
	type Params struct {
		ConfigFiles []string `configfile:"true" optional:"true"`
		Host        string   `optional:"true"`
		Port        int      `optional:"true"`
		OnlyInBase  string   `optional:"true"`
	}

	dir := t.TempDir()
	base := writeFile(t, dir, "base.json", `{"Host":"base","OnlyInBase":"x"}`)
	local := writeFile(t, dir, "local.json", `{"Port":8080}`)

	var (
		hostHasValue   bool
		portHasValue   bool
		onlyHasValue   bool
	)
	CmdT[Params]{
		Use: "test",
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			hostHasValue = ctx.HasValue(&p.Host)
			portHasValue = ctx.HasValue(&p.Port)
			onlyHasValue = ctx.HasValue(&p.OnlyInBase)
		},
	}.RunArgs([]string{"--config-files", base + "," + local})

	if !hostHasValue {
		t.Error("expected HasValue(Host)=true (set by base)")
	}
	if !portHasValue {
		t.Error("expected HasValue(Port)=true (set by local)")
	}
	if !onlyHasValue {
		t.Error("expected HasValue(OnlyInBase)=true (set by base, unchanged by local)")
	}
}

// --- Programmatic SetConfigFile on []string ---

func TestMultiConfigFile_ProgrammaticSetConfigFile(t *testing.T) {
	type Params struct {
		ConfigFiles []string `optional:"true"`
		Host        string   `optional:"true"`
		Port        int      `optional:"true"`
	}

	dir := t.TempDir()
	base := writeFile(t, dir, "base.json", `{"Host":"base","Port":80}`)
	local := writeFile(t, dir, "local.json", `{"Port":8080}`)

	var got Params
	CmdT[Params]{
		Use: "test",
		InitFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command) error {
			ctx.GetParam(&p.ConfigFiles).SetConfigFile(true)
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = *p
		},
	}.RunArgs([]string{"--config-files", base + "," + local})

	if got.Host != "base" {
		t.Errorf("Host: expected 'base', got %q", got.Host)
	}
	if got.Port != 8080 {
		t.Errorf("Port: expected 8080, got %d", got.Port)
	}
}

// --- Substruct overlay chain interacting with root ---

func TestMultiConfigFile_SubstructAndRoot(t *testing.T) {
	type DB struct {
		ConfigFiles []string `configfile:"true" optional:"true"`
		Host        string   `optional:"true"`
		Port        int      `optional:"true"`
	}
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Name       string `optional:"true"`
		DB         DB
	}

	dir := t.TempDir()
	dbBase := writeFile(t, dir, "db-base.json", `{"Host":"db-base","Port":5432}`)
	dbLocal := writeFile(t, dir, "db-local.json", `{"Host":"db-local"}`)
	root := writeFile(t, dir, "root.json", `{"Name":"app","DB":{"Port":6000}}`)

	var got Params
	CmdT[Params]{
		Use: "test",
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = *p
		},
	}.RunArgs([]string{
		"--config-file", root,
		"--db-config-files", dbBase + "," + dbLocal,
	})

	// Substruct chain loads first: dbBase then dbLocal → Host=db-local, Port=5432
	// Root loads last, overriding DB.Port → 6000. DB.Host remains db-local
	// because root.json doesn't mention it.
	if got.Name != "app" {
		t.Errorf("Name: expected 'app', got %q", got.Name)
	}
	if got.DB.Host != "db-local" {
		t.Errorf("DB.Host: expected 'db-local' (substruct overlay result), got %q", got.DB.Host)
	}
	if got.DB.Port != 6000 {
		t.Errorf("DB.Port: expected 6000 (root overrides substruct), got %d", got.DB.Port)
	}
}

// --- Public LoadConfigFiles helper (for hook-based users) ---

func TestLoadConfigFiles_PublicHelper(t *testing.T) {
	type Params struct {
		Host string `optional:"true"`
		Port int    `optional:"true"`
	}

	t.Run("overlays in order", func(t *testing.T) {
		dir := t.TempDir()
		base := writeFile(t, dir, "base.json", `{"Host":"base","Port":80}`)
		local := writeFile(t, dir, "local.json", `{"Port":8080}`)

		var p Params
		if err := LoadConfigFiles([]string{base, local}, &p, nil); err != nil {
			t.Fatalf("LoadConfigFiles: %v", err)
		}
		if p.Host != "base" {
			t.Errorf("Host: expected 'base', got %q", p.Host)
		}
		if p.Port != 8080 {
			t.Errorf("Port: expected 8080, got %d", p.Port)
		}
	})

	t.Run("empty-string entries are skipped silently", func(t *testing.T) {
		dir := t.TempDir()
		base := writeFile(t, dir, "base.json", `{"Host":"base"}`)

		var p Params
		if err := LoadConfigFiles([]string{"", base, ""}, &p, nil); err != nil {
			t.Fatalf("LoadConfigFiles: %v", err)
		}
		if p.Host != "base" {
			t.Errorf("expected Host=base, got %q", p.Host)
		}
	})

	t.Run("nil and empty lists are no-ops", func(t *testing.T) {
		var p Params
		if err := LoadConfigFiles(nil, &p, nil); err != nil {
			t.Fatalf("nil: %v", err)
		}
		if err := LoadConfigFiles([]string{}, &p, nil); err != nil {
			t.Fatalf("empty: %v", err)
		}
	})

	t.Run("stops at the first missing file", func(t *testing.T) {
		dir := t.TempDir()
		base := writeFile(t, dir, "base.json", `{"Host":"base"}`)

		var p Params
		err := LoadConfigFiles([]string{base, "/nonexistent/overlay.json"}, &p, nil)
		if err == nil {
			t.Fatal("expected error for missing file")
		}
		if p.Host != "base" {
			t.Errorf("expected first file to have loaded before the error, got %q", p.Host)
		}
	})

	t.Run("works inside PreValidateFunc", func(t *testing.T) {
		dir := t.TempDir()
		base := writeFile(t, dir, "base.json", `{"Host":"base","Port":80}`)
		local := writeFile(t, dir, "local.json", `{"Port":8080}`)

		var got Params
		CmdT[Params]{
			Use: "test",
			PreValidateFunc: func(p *Params, cmd *cobra.Command, args []string) error {
				return LoadConfigFiles([]string{base, local}, p, nil)
			},
			RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
				got = *p
			},
		}.RunArgs([]string{})

		if got.Host != "base" || got.Port != 8080 {
			t.Errorf("PreValidate overlay: got %+v", got)
		}
	})
}
