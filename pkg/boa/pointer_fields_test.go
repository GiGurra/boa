package boa

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
)

func TestPointerField_OptionalString(t *testing.T) {
	type Params struct {
		Name *string `descr:"optional name"`
	}

	var got *string
	err := (CmdT[Params]{
		Use: "test",
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = p.Name
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil pointer when flag not provided, got %q", *got)
	}
}

func TestPointerField_StringWithValue(t *testing.T) {
	type Params struct {
		Name *string `descr:"optional name"`
	}

	var got *string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = p.Name
		},
	}).RunArgsE([]string{"--name", "alice"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil pointer when flag provided")
	}
	if *got != "alice" {
		t.Errorf("expected 'alice', got %q", *got)
	}
}

func TestPointerField_IntWithValue(t *testing.T) {
	type Params struct {
		Port *int `descr:"optional port"`
	}

	var got *int
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = p.Port
		},
	}).RunArgsE([]string{"--port", "8080"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil pointer when flag provided")
	}
	if *got != 8080 {
		t.Errorf("expected 8080, got %d", *got)
	}
}

func TestPointerField_IntNotProvided(t *testing.T) {
	type Params struct {
		Port *int `descr:"optional port"`
	}

	var got *int
	err := (CmdT[Params]{
		Use: "test",
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = p.Port
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil pointer when flag not provided, got %d", *got)
	}
}

func TestPointerField_BoolWithValue(t *testing.T) {
	type Params struct {
		Verbose *bool `descr:"optional verbose"`
	}

	var got *bool
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = p.Verbose
		},
	}).RunArgsE([]string{"--verbose"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil pointer when flag provided")
	}
	if !*got {
		t.Error("expected true, got false")
	}
}

func TestPointerField_DefaultOptional(t *testing.T) {
	// Pointer fields should always be optional, even when global default is required
	type Params struct {
		Name *string `descr:"optional name"`
	}

	err := (CmdT[Params]{
		Use: "test",
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("pointer field should be optional by default, got error: %v", err)
	}
}

func TestPointerField_ExplicitRequired(t *testing.T) {
	// Even pointer fields can be marked required
	type Params struct {
		Name *string `descr:"required name" required:"true"`
	}

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			t.Fatal("should not be called")
		},
	}).RunArgsE([]string{})

	if err == nil {
		t.Fatal("expected error for missing required pointer field")
	}
}

func TestPointerField_EnvVar(t *testing.T) {
	type Params struct {
		Name *string `descr:"optional name" env:"TEST_PTR_NAME"`
	}

	_ = os.Setenv("TEST_PTR_NAME", "from-env")
	defer func() { _ = os.Unsetenv("TEST_PTR_NAME") }()

	var got *string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = p.Name
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil pointer when env var set")
	}
	if *got != "from-env" {
		t.Errorf("expected 'from-env', got %q", *got)
	}
}

func TestPointerField_DefaultValue(t *testing.T) {
	type Params struct {
		Name *string `descr:"optional name" default:"fallback"`
	}

	var got *string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = p.Name
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil pointer when default is set")
	}
	if *got != "fallback" {
		t.Errorf("expected 'fallback', got %q", *got)
	}
}

func TestPointerField_MixedWithPlainFields(t *testing.T) {
	type Params struct {
		Name    string  `descr:"required name"`
		Port    *int    `descr:"optional port"`
		Verbose *bool   `descr:"optional verbose"`
	}

	var gotName string
	var gotPort *int
	var gotVerbose *bool
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotName = p.Name
			gotPort = p.Port
			gotVerbose = p.Verbose
		},
	}).RunArgsE([]string{"--name", "alice", "--port", "9090"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName != "alice" {
		t.Errorf("expected 'alice', got %q", gotName)
	}
	if gotPort == nil || *gotPort != 9090 {
		t.Error("expected port 9090")
	}
	if gotVerbose != nil {
		t.Error("expected nil verbose when not provided")
	}
}
