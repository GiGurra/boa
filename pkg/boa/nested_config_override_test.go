package boa

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestNestedConfigFile_WithCLIOverride(t *testing.T) {
	// Config file uses natural JSON nesting (no prefixing).
	// CLI args use prefixed names (--db-host).
	// CLI should override config file values.
	type DBConfig struct {
		Host string `descr:"host" default:"localhost"`
		Port int    `descr:"port" default:"5432"`
	}
	type Params struct {
		ConfigFile string   `configfile:"true" default:"" optional:"true"`
		DB         DBConfig
	}

	cfgData, _ := json.Marshal(map[string]any{
		"DB": map[string]any{"Host": "config-host", "Port": 3306},
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	var gotHost string
	var gotPort int
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotHost = p.DB.Host
			gotPort = p.DB.Port
		},
		// CLI overrides host, port comes from config
	}).RunArgsE([]string{"--config-file", cfgPath, "--db-host", "cli-host"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHost != "cli-host" {
		t.Errorf("expected host='cli-host' (CLI override), got %q", gotHost)
	}
	if gotPort != 3306 {
		t.Errorf("expected port=3306 (from config file), got %d", gotPort)
	}
}

func TestNestedConfigFile_NoCliOverride(t *testing.T) {
	// Config file values should flow through when no CLI override
	type DBConfig struct {
		Host string `descr:"host" default:"localhost"`
		Port int    `descr:"port" default:"5432"`
	}
	type Params struct {
		ConfigFile string   `configfile:"true" default:"" optional:"true"`
		DB         DBConfig
	}

	cfgData, _ := json.Marshal(map[string]any{
		"DB": map[string]any{"Host": "config-host", "Port": 3306},
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	var gotHost string
	var gotPort int
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotHost = p.DB.Host
			gotPort = p.DB.Port
		},
	}).RunArgsE([]string{"--config-file", cfgPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHost != "config-host" {
		t.Errorf("expected host='config-host' (from config), got %q", gotHost)
	}
	if gotPort != 3306 {
		t.Errorf("expected port=3306 (from config), got %d", gotPort)
	}
}

func TestNestedConfigFile_EnvOverridesConfig(t *testing.T) {
	// Env vars (prefixed) should override config file values
	type DBConfig struct {
		Host string `descr:"host" default:"localhost"`
		Port int    `descr:"port" default:"5432"`
	}
	type Params struct {
		ConfigFile string   `configfile:"true" default:"" optional:"true"`
		DB         DBConfig
	}

	cfgData, _ := json.Marshal(map[string]any{
		"DB": map[string]any{"Host": "config-host", "Port": 3306},
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	_ = os.Setenv("DB_HOST", "env-host")
	defer func() { _ = os.Unsetenv("DB_HOST") }()

	var gotHost string
	var gotPort int
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherCombine(ParamEnricherDefault, ParamEnricherEnv),
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotHost = p.DB.Host
			gotPort = p.DB.Port
		},
	}).RunArgsE([]string{"--config-file", cfgPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHost != "env-host" {
		t.Errorf("expected host='env-host' (env override), got %q", gotHost)
	}
	if gotPort != 3306 {
		t.Errorf("expected port=3306 (from config), got %d", gotPort)
	}
}

func TestNestedConfigFile_DeepNesting(t *testing.T) {
	// 3 levels deep: config file JSON nests naturally, CLI uses prefixed flags
	type ConnConfig struct {
		Host string `descr:"host" default:"localhost"`
	}
	type ClusterConfig struct {
		Primary ConnConfig
	}
	type Params struct {
		ConfigFile string        `configfile:"true" default:"" optional:"true"`
		Infra      ClusterConfig
	}

	cfgData, _ := json.Marshal(map[string]any{
		"Infra": map[string]any{
			"Primary": map[string]any{"Host": "deep-config-host"},
		},
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	var gotHost string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotHost = p.Infra.Primary.Host
		},
	}).RunArgsE([]string{"--config-file", cfgPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHost != "deep-config-host" {
		t.Errorf("expected host='deep-config-host', got %q", gotHost)
	}
}
