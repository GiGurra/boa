package boa

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// iniUnmarshal is a minimal INI-style deserializer (key=value per line).
// Supports string, int, and bool fields. For testing config format registry.
func iniUnmarshal(data []byte, target any) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("ini: target must be a pointer to struct")
	}
	v = v.Elem()
	t := v.Type()

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if strings.EqualFold(field.Name, key) {
				fv := v.Field(i)
				switch fv.Kind() {
				case reflect.String:
					fv.SetString(val)
				case reflect.Int, reflect.Int64:
					n, err := strconv.ParseInt(val, 10, 64)
					if err != nil {
						return fmt.Errorf("ini: invalid int for %s: %w", key, err)
					}
					fv.SetInt(n)
				case reflect.Bool:
					b, err := strconv.ParseBool(val)
					if err != nil {
						return fmt.Errorf("ini: invalid bool for %s: %w", key, err)
					}
					fv.SetBool(b)
				}
				break
			}
		}
	}
	return scanner.Err()
}

func TestMixedConfigFormats_JSONRoot_INISubstruct(t *testing.T) {
	// Root config is JSON, substruct config is INI.
	// Tests that RegisterConfigFormat + file extension detection works.
	RegisterConfigFormat(".ini", iniUnmarshal)
	defer delete(configFormats, ".ini") // clean up

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

	// INI file for DB substruct
	iniData := "Host = ini-db-host\nPort = 3307\n"
	iniPath := filepath.Join(tmpDir, "db.ini")
	os.WriteFile(iniPath, []byte(iniData), 0644)

	// JSON file for root (overrides DB.Host but not DB.Port)
	rootCfg, _ := json.Marshal(map[string]any{
		"Name": "from-json",
		"DB":   map[string]any{"Host": "json-override-host"},
	})
	jsonPath := filepath.Join(tmpDir, "app.json")
	os.WriteFile(jsonPath, rootCfg, 0644)

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
	}).RunArgsE([]string{"--config-file", jsonPath, "--db-config-file", iniPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName != "from-json" {
		t.Errorf("expected name='from-json', got %q", gotName)
	}
	if gotHost != "json-override-host" {
		t.Errorf("expected host='json-override-host' (root JSON overrides INI), got %q", gotHost)
	}
	if gotPort != 3307 {
		t.Errorf("expected port=3307 (from INI, not overridden by root), got %d", gotPort)
	}
}

func TestMixedConfigFormats_INIOnly(t *testing.T) {
	// Just INI, no root config — verify the format registry works standalone
	RegisterConfigFormat(".ini", iniUnmarshal)
	defer delete(configFormats, ".ini")

	type DBConfig struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Host       string `descr:"host" default:"localhost"`
		Port       int    `descr:"port" default:"5432"`
		Debug      bool   `descr:"debug" default:"false" optional:"true"`
	}
	type Params struct {
		Name string   `descr:"name"`
		DB   DBConfig
	}

	iniData := "# DB configuration\nHost = ini-host\nPort = 9999\nDebug = true\n"
	tmpDir := t.TempDir()
	iniPath := filepath.Join(tmpDir, "db.ini")
	os.WriteFile(iniPath, []byte(iniData), 0644)

	var gotHost string
	var gotPort int
	var gotDebug bool
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotHost = p.DB.Host
			gotPort = p.DB.Port
			gotDebug = p.DB.Debug
		},
	}).RunArgsE([]string{"--name", "myapp", "--db-config-file", iniPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHost != "ini-host" {
		t.Errorf("expected host='ini-host', got %q", gotHost)
	}
	if gotPort != 9999 {
		t.Errorf("expected port=9999, got %d", gotPort)
	}
	if !gotDebug {
		t.Error("expected debug=true from INI")
	}
}

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
