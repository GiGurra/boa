package boa

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestNestedSlice_ConfigFile(t *testing.T) {
	type Params struct {
		ConfigFile string     `configfile:"true" default:"" optional:"true"`
		Matrix     [][]string `descr:"matrix of values" optional:"true"`
	}

	cfgData, _ := json.Marshal(map[string]any{
		"Matrix": [][]string{{"a", "b"}, {"c", "d"}},
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	os.WriteFile(cfgPath, cfgData, 0644)

	var got [][]string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = p.Matrix
		},
	}).RunArgsE([]string{"--config-file", cfgPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(got))
	}
	if got[0][0] != "a" || got[0][1] != "b" {
		t.Errorf("expected row 0 = [a,b], got %v", got[0])
	}
	if got[1][0] != "c" || got[1][1] != "d" {
		t.Errorf("expected row 1 = [c,d], got %v", got[1])
	}
}

func TestNestedSlice_IntMatrix_ConfigFile(t *testing.T) {
	type Params struct {
		ConfigFile string  `configfile:"true" default:"" optional:"true"`
		Matrix     [][]int `descr:"int matrix" optional:"true"`
	}

	cfgData, _ := json.Marshal(map[string]any{
		"Matrix": [][]int{{1, 2}, {3, 4}},
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	os.WriteFile(cfgPath, cfgData, 0644)

	var got [][]int
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = p.Matrix
		},
	}).RunArgsE([]string{"--config-file", cfgPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(got))
	}
	if got[0][0] != 1 || got[0][1] != 2 {
		t.Errorf("expected row 0 = [1,2], got %v", got[0])
	}
}

func TestNestedSlice_DefaultOptional(t *testing.T) {
	// Nested slices should default to optional (nil = not set)
	type Params struct {
		Matrix [][]string `descr:"matrix"`
	}

	err := (CmdT[Params]{
		Use: "test",
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("nested slice field should be optional by default, got error: %v", err)
	}
}
