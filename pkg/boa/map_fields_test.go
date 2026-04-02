package boa

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestMapField_StringToString_CLI(t *testing.T) {
	type Params struct {
		Labels map[string]string `descr:"key-value labels"`
	}

	var got map[string]string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = p.Labels
		},
	}).RunArgsE([]string{"--labels", "env=prod,tier=frontend"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil map")
	}
	if got["env"] != "prod" {
		t.Errorf("expected labels[env]='prod', got %q", got["env"])
	}
	if got["tier"] != "frontend" {
		t.Errorf("expected labels[tier]='frontend', got %q", got["tier"])
	}
}

func TestMapField_StringToString_NotProvided(t *testing.T) {
	type Params struct {
		Labels map[string]string `descr:"key-value labels" optional:"true"`
	}

	var got map[string]string
	err := (CmdT[Params]{
		Use: "test",
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = p.Labels
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil map when not provided, got %v", got)
	}
}

func TestMapField_StringToInt_CLI(t *testing.T) {
	type Params struct {
		Ports map[string]int `descr:"named ports"`
	}

	var got map[string]int
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = p.Ports
		},
	}).RunArgsE([]string{"--ports", "http=80,https=443"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil map")
	}
	if got["http"] != 80 {
		t.Errorf("expected ports[http]=80, got %d", got["http"])
	}
	if got["https"] != 443 {
		t.Errorf("expected ports[https]=443, got %d", got["https"])
	}
}

func TestMapField_EnvVar(t *testing.T) {
	type Params struct {
		Labels map[string]string `descr:"labels" env:"TEST_MAP_LABELS"`
	}

	_ = os.Setenv("TEST_MAP_LABELS", "env=staging,region=us-east")
	defer func() { _ = os.Unsetenv("TEST_MAP_LABELS") }()

	var got map[string]string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = p.Labels
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil map from env var")
	}
	if got["env"] != "staging" {
		t.Errorf("expected labels[env]='staging', got %q", got["env"])
	}
}

func TestMapField_ConfigFile(t *testing.T) {
	type Params struct {
		ConfigFile string            `configfile:"true" default:"" optional:"true"`
		Labels     map[string]string `descr:"labels" optional:"true"`
	}

	cfgData, _ := json.Marshal(map[string]any{
		"Labels": map[string]string{"app": "myapp", "version": "v2"},
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	var got map[string]string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = p.Labels
		},
	}).RunArgsE([]string{"--config-file", cfgPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil map from config file")
	}
	if got["app"] != "myapp" {
		t.Errorf("expected labels[app]='myapp', got %q", got["app"])
	}
}

func TestMapField_DefaultOptional(t *testing.T) {
	// Map fields should default to optional (like pointer fields — nil = not set)
	type Params struct {
		Labels map[string]string `descr:"labels"`
	}

	err := (CmdT[Params]{
		Use: "test",
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("map field should be optional by default, got error: %v", err)
	}
}
