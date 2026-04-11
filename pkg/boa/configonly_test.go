package boa

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestBoaIgnore_StillLoadedFromConfigFile(t *testing.T) {
	// boa:"ignore" should skip CLI/env registration but json.Unmarshal
	// still writes to the field since it doesn't look at boa tags
	type Params struct {
		ConfigFile string            `configfile:"true" default:"" optional:"true"`
		Name       string            `descr:"name"`
		Metadata   map[string]string `boa:"ignore"`
	}

	cfgData, _ := json.Marshal(map[string]any{
		"Name":     "from-config",
		"Metadata": map[string]string{"env": "prod", "region": "us-east"},
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	var gotName string
	var gotMeta map[string]string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotName = p.Name
			gotMeta = p.Metadata
		},
	}).RunArgsE([]string{"--config-file", cfgPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName != "from-config" {
		t.Errorf("expected name 'from-config', got %q", gotName)
	}
	if gotMeta == nil {
		t.Fatal("expected metadata to be loaded from config file despite boa:ignore")
	}
	if gotMeta["env"] != "prod" {
		t.Errorf("expected metadata[env]='prod', got %q", gotMeta["env"])
	}
}

func TestBoaConfigOnly_LoadedFromConfigFile(t *testing.T) {
	// boa:"configonly" is now a noflag+noenv shorthand (NOT an ignore alias).
	// The mirror still exists — validation runs — but the field is hidden
	// from CLI and env, so config files are the only remaining write path.
	type Params struct {
		ConfigFile string            `configfile:"true" default:"" optional:"true"`
		Name       string            `descr:"name"`
		Metadata   map[string]string `boa:"configonly"`
	}

	cfgData, _ := json.Marshal(map[string]any{
		"Name":     "from-config",
		"Metadata": map[string]string{"env": "prod"},
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	var gotMeta map[string]string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotMeta = p.Metadata
		},
	}).RunArgsE([]string{"--config-file", cfgPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMeta == nil || gotMeta["env"] != "prod" {
		t.Errorf("expected metadata loaded from config, got %v", gotMeta)
	}
}

func TestBoaConfigOnly_NoCLIFlag(t *testing.T) {
	// boa:"configonly" should not create CLI flags
	type Params struct {
		Name     string            `descr:"name"`
		Metadata map[string]string `boa:"configonly"`
	}

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--name", "alice", "--metadata", "key=val"})

	if err == nil {
		t.Fatal("expected error for unknown flag --metadata")
	}
}

func TestBoaIgnore_NoCLIFlag(t *testing.T) {
	// boa:"ignore" fields should NOT create CLI flags
	type Params struct {
		Name     string            `descr:"name"`
		Metadata map[string]string `boa:"ignore"`
	}

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
		},
	}).RunArgsE([]string{"--name", "alice", "--metadata", "key=val"})

	if err == nil {
		t.Fatal("expected error for unknown flag --metadata (boa:ignore should skip it)")
	}
}

func TestSubStruct_NotFlattenedWhenIgnored(t *testing.T) {
	// A sub-struct with boa:"ignore" should not have its fields become CLI flags,
	// but should still be populated from config file
	type DBConfig struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}
	type Params struct {
		ConfigFile string   `configfile:"true" default:"" optional:"true"`
		Name       string   `descr:"app name"`
		DB         DBConfig `boa:"ignore"`
	}

	cfgData, _ := json.Marshal(map[string]any{
		"Name": "myapp",
		"DB":   map[string]any{"host": "localhost", "port": 5432},
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	var gotDB DBConfig
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
	if gotDB.Host != "localhost" {
		t.Errorf("expected DB.Host='localhost', got %q", gotDB.Host)
	}
	if gotDB.Port != 5432 {
		t.Errorf("expected DB.Port=5432, got %d", gotDB.Port)
	}
}

func TestSubStruct_FlattenedByDefault(t *testing.T) {
	// Without boa:"ignore", named sub-struct fields ARE flattened into CLI flags
	// with the parent field name as prefix: DB.Host → --db-host
	type DBCfg struct {
		Host string `descr:"database host"`
		Port int    `descr:"database port" default:"5432"`
	}
	type Params struct {
		Name string `descr:"app name"`
		DB   DBCfg  // named field — children become --db-host, --db-port
	}

	var gotHost string
	var gotPort int
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotHost = p.DB.Host
			gotPort = p.DB.Port
		},
	}).RunArgsE([]string{"--name", "myapp", "--db-host", "db.example.com"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHost != "db.example.com" {
		t.Errorf("expected DB.Host='db.example.com', got %q", gotHost)
	}
	if gotPort != 5432 {
		t.Errorf("expected DB.Port=5432, got %d", gotPort)
	}
}
