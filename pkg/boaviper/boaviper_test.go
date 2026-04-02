package boaviper

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

func TestFindConfig_CurrentDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create myapp.json in "current dir"
	cfgPath := filepath.Join(tmpDir, "myapp.json")
	os.WriteFile(cfgPath, []byte(`{"port": 9090}`), 0644)

	found := FindConfig("myapp", tmpDir)
	if found != cfgPath {
		t.Errorf("expected %q, got %q", cfgPath, found)
	}
}

func TestFindConfig_ConfigSubdir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config dir with config.json
	configDir := filepath.Join(tmpDir, "myapp")
	os.MkdirAll(configDir, 0755)
	cfgPath := filepath.Join(configDir, "config.json")
	os.WriteFile(cfgPath, []byte(`{"port": 9090}`), 0644)

	// Search: first path has nothing, second path has it
	emptyDir := filepath.Join(tmpDir, "empty")
	os.MkdirAll(emptyDir, 0755)
	found := FindConfig("myapp", emptyDir, configDir)
	if found != cfgPath {
		t.Errorf("expected %q, got %q", cfgPath, found)
	}
}

func TestFindConfig_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	found := FindConfig("myapp", tmpDir)
	if found != "" {
		t.Errorf("expected empty string, got %q", found)
	}
}

func TestFindConfig_PriorityOrder(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files in two locations — first path should win
	dir1 := filepath.Join(tmpDir, "first")
	dir2 := filepath.Join(tmpDir, "second")
	os.MkdirAll(dir1, 0755)
	os.MkdirAll(dir2, 0755)

	path1 := filepath.Join(dir1, "myapp.json")
	path2 := filepath.Join(dir2, "config.json")
	os.WriteFile(path1, []byte(`{"port": 1}`), 0644)
	os.WriteFile(path2, []byte(`{"port": 2}`), 0644)

	found := FindConfig("myapp", dir1, dir2)
	if found != path1 {
		t.Errorf("expected first path %q to win, got %q", path1, found)
	}
}

func TestAutoConfig_SetsPath(t *testing.T) {
	tmpDir := t.TempDir()

	cfgData, _ := json.Marshal(map[string]any{"Port": 9090})
	cfgPath := filepath.Join(tmpDir, "myapp.json")
	os.WriteFile(cfgPath, cfgData, 0644)

	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Port       int    `descr:"port" default:"8080"`
	}

	var gotPort int
	err := (boa.CmdT[Params]{
		Use:         "myapp",
		ParamEnrich: boa.ParamEnricherName,
		InitFunc:    AutoConfig[Params]("myapp", tmpDir),
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotPort = p.Port
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPort != 9090 {
		t.Errorf("expected port=9090 (from auto-discovered config), got %d", gotPort)
	}
}

func TestAutoConfig_CLIOverridesAutoDiscovery(t *testing.T) {
	tmpDir := t.TempDir()

	// Auto-discoverable config
	autoData, _ := json.Marshal(map[string]any{"Port": 1111})
	autoPath := filepath.Join(tmpDir, "myapp.json")
	os.WriteFile(autoPath, autoData, 0644)

	// Explicit config
	explicitDir := filepath.Join(tmpDir, "explicit")
	os.MkdirAll(explicitDir, 0755)
	explicitData, _ := json.Marshal(map[string]any{"Port": 2222})
	explicitPath := filepath.Join(explicitDir, "custom.json")
	os.WriteFile(explicitPath, explicitData, 0644)

	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Port       int    `descr:"port" default:"8080"`
	}

	var gotPort int
	err := (boa.CmdT[Params]{
		Use:         "myapp",
		ParamEnrich: boa.ParamEnricherName,
		InitFunc:    AutoConfig[Params]("myapp", tmpDir),
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotPort = p.Port
		},
	}).RunArgsE([]string{"--config-file", explicitPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPort != 2222 {
		t.Errorf("expected port=2222 (from explicit config), got %d", gotPort)
	}
}

func TestAutoConfig_NoConfigFileFound(t *testing.T) {
	tmpDir := t.TempDir()

	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Port       int    `descr:"port" default:"8080"`
	}

	var gotPort int
	err := (boa.CmdT[Params]{
		Use:         "myapp",
		ParamEnrich: boa.ParamEnricherName,
		InitFunc:    AutoConfig[Params]("myapp", tmpDir),
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotPort = p.Port
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPort != 8080 {
		t.Errorf("expected port=8080 (default, no config found), got %d", gotPort)
	}
}

func TestSetEnvPrefix(t *testing.T) {
	type Params struct {
		Port int `descr:"port" default:"8080"`
	}

	os.Setenv("MYAPP_PORT", "3000")
	defer os.Unsetenv("MYAPP_PORT")

	var gotPort int
	err := (boa.CmdT[Params]{
		Use: "myapp",
		ParamEnrich: boa.ParamEnricherCombine(
			boa.ParamEnricherDefault,
			SetEnvPrefix("MYAPP"),
		),
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotPort = p.Port
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPort != 3000 {
		t.Errorf("expected port=3000 (from MYAPP_PORT env), got %d", gotPort)
	}
}
