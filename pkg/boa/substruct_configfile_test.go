package boa

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestSubStructConfigFile_Basic(t *testing.T) {
	// A substruct with its own configfile:"true" field loads from its own file
	type DBConfig struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Host       string `descr:"host" default:"localhost"`
		Port       int    `descr:"port" default:"5432"`
	}
	type Params struct {
		Name string   `descr:"app name"`
		DB   DBConfig
	}

	cfgData, _ := json.Marshal(map[string]any{
		"Host": "db-file-host",
		"Port": 3306,
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "db.json")
	os.WriteFile(cfgPath, cfgData, 0644)

	var gotHost string
	var gotPort int
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotHost = p.DB.Host
			gotPort = p.DB.Port
		},
	}).RunArgsE([]string{"--name", "myapp", "--db-config-file", cfgPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHost != "db-file-host" {
		t.Errorf("expected host='db-file-host', got %q", gotHost)
	}
	if gotPort != 3306 {
		t.Errorf("expected port=3306, got %d", gotPort)
	}
}

func TestSubStructConfigFile_RootOverridesInner(t *testing.T) {
	// Root config file should override substruct config file values.
	// Priority: root config > substruct config > defaults
	type DBConfig struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Host       string `descr:"host" default:"localhost"`
		Port       int    `descr:"port" default:"5432"`
	}
	type Params struct {
		ConfigFile string   `configfile:"true" optional:"true"`
		Name       string   `descr:"app name" default:"unnamed"`
		DB         DBConfig
	}

	tmpDir := t.TempDir()

	// Inner config: sets Host and Port
	dbCfg, _ := json.Marshal(map[string]any{
		"Host": "inner-host",
		"Port": 3306,
	})
	dbCfgPath := filepath.Join(tmpDir, "db.json")
	os.WriteFile(dbCfgPath, dbCfg, 0644)

	// Root config: overrides Host but not Port
	rootCfg, _ := json.Marshal(map[string]any{
		"Name": "from-root",
		"DB":   map[string]any{"Host": "root-override-host"},
	})
	rootCfgPath := filepath.Join(tmpDir, "app.json")
	os.WriteFile(rootCfgPath, rootCfg, 0644)

	var gotName, gotHost string
	var gotPort int
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotName = p.Name
			gotHost = p.DB.Host
			gotPort = p.DB.Port
		},
	}).RunArgsE([]string{"--config-file", rootCfgPath, "--db-config-file", dbCfgPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName != "from-root" {
		t.Errorf("expected name='from-root', got %q", gotName)
	}
	if gotHost != "root-override-host" {
		t.Errorf("expected host='root-override-host' (root overrides inner), got %q", gotHost)
	}
	if gotPort != 3306 {
		t.Errorf("expected port=3306 (from inner, not overridden by root), got %d", gotPort)
	}
}

func TestSubStructConfigFile_CLIOverridesBothConfigs(t *testing.T) {
	// CLI should override both root and substruct config values
	type DBConfig struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Host       string `descr:"host" default:"localhost"`
		Port       int    `descr:"port" default:"5432"`
	}
	type Params struct {
		ConfigFile string   `configfile:"true" optional:"true"`
		DB         DBConfig
	}

	tmpDir := t.TempDir()

	dbCfg, _ := json.Marshal(map[string]any{"Host": "inner-host", "Port": 3306})
	dbPath := filepath.Join(tmpDir, "db.json")
	os.WriteFile(dbPath, dbCfg, 0644)

	rootCfg, _ := json.Marshal(map[string]any{"DB": map[string]any{"Host": "root-host"}})
	rootPath := filepath.Join(tmpDir, "app.json")
	os.WriteFile(rootPath, rootCfg, 0644)

	var gotHost string
	var gotPort int
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotHost = p.DB.Host
			gotPort = p.DB.Port
		},
	}).RunArgsE([]string{
		"--config-file", rootPath,
		"--db-config-file", dbPath,
		"--db-host", "cli-wins",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHost != "cli-wins" {
		t.Errorf("expected host='cli-wins' (CLI overrides all), got %q", gotHost)
	}
	if gotPort != 3306 {
		t.Errorf("expected port=3306 (from inner config), got %d", gotPort)
	}
}

func TestSubStructConfigFile_InnerOnly(t *testing.T) {
	// Only inner config, no root config — should work fine
	type DBConfig struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Host       string `descr:"host" default:"localhost"`
		Port       int    `descr:"port" default:"5432"`
	}
	type Params struct {
		Name string   `descr:"name"`
		DB   DBConfig
	}

	cfgData, _ := json.Marshal(map[string]any{"Host": "inner-only", "Port": 9999})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "db.json")
	os.WriteFile(cfgPath, cfgData, 0644)

	var gotHost string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotHost = p.DB.Host
		},
	}).RunArgsE([]string{"--name", "x", "--db-config-file", cfgPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHost != "inner-only" {
		t.Errorf("expected host='inner-only', got %q", gotHost)
	}
}
