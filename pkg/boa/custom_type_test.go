package boa

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// SemVer is a custom type for testing RegisterType
type SemVer struct {
	Major, Minor, Patch int
}

func (v SemVer) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func parseSemVer(s string) (SemVer, error) {
	parts := strings.SplitN(s, ".", 3)
	if len(parts) != 3 {
		return SemVer{}, fmt.Errorf("expected MAJOR.MINOR.PATCH, got %q", s)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid major: %w", err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid minor: %w", err)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid patch: %w", err)
	}
	return SemVer{Major: major, Minor: minor, Patch: patch}, nil
}

func TestCustomType_CLI(t *testing.T) {
	RegisterType[SemVer](TypeDef[SemVer]{
		Parse:  parseSemVer,
		Format: func(v SemVer) string { return v.String() },
	})

	type Params struct {
		Version SemVer `descr:"app version"`
	}

	var got SemVer
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = p.Version
		},
	}).RunArgsE([]string{"--version", "1.2.3"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Major != 1 || got.Minor != 2 || got.Patch != 3 {
		t.Errorf("expected 1.2.3, got %v", got)
	}
}

func TestCustomType_InvalidValue(t *testing.T) {
	RegisterType[SemVer](TypeDef[SemVer]{
		Parse:  parseSemVer,
		Format: func(v SemVer) string { return v.String() },
	})

	type Params struct {
		Version SemVer `descr:"app version"`
	}

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--version", "not-a-version"})

	if err == nil {
		t.Fatal("expected error for invalid semver")
	}
}

func TestCustomType_Default(t *testing.T) {
	RegisterType[SemVer](TypeDef[SemVer]{
		Parse:  parseSemVer,
		Format: func(v SemVer) string { return v.String() },
	})

	type Params struct {
		Version SemVer `descr:"app version" default:"0.1.0"`
	}

	var got SemVer
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = p.Version
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Major != 0 || got.Minor != 1 || got.Patch != 0 {
		t.Errorf("expected 0.1.0, got %v", got)
	}
}

func TestCustomType_EnvVar(t *testing.T) {
	RegisterType[SemVer](TypeDef[SemVer]{
		Parse:  parseSemVer,
		Format: func(v SemVer) string { return v.String() },
	})

	type Params struct {
		Version SemVer `descr:"app version" env:"APP_VERSION"`
	}

	os.Setenv("APP_VERSION", "2.0.0")
	defer os.Unsetenv("APP_VERSION")

	var got SemVer
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = p.Version
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Major != 2 {
		t.Errorf("expected 2.0.0, got %v", got)
	}
}

func TestCustomType_Optional(t *testing.T) {
	RegisterType[SemVer](TypeDef[SemVer]{
		Parse:  parseSemVer,
		Format: func(v SemVer) string { return v.String() },
	})

	type Params struct {
		Version *SemVer `descr:"app version"`
	}

	// Not provided — should be nil
	var got *SemVer
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			got = p.Version
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for unset optional custom type, got %v", got)
	}
}
