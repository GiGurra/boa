package boa

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// --- min/max for numeric types ---

func TestValidationTag_MinMax_Int(t *testing.T) {
	type Params struct {
		Port int `descr:"port" min:"1" max:"65535"`
	}

	// Valid value
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--port", "8080"})
	if err != nil {
		t.Fatalf("expected no error for valid port, got: %v", err)
	}

	// Below min
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--port", "0"})
	if err == nil {
		t.Fatal("expected error for port below min")
	}
	if !strings.Contains(err.Error(), "min") {
		t.Errorf("expected error about min, got: %v", err)
	}

	// Above max
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--port", "70000"})
	if err == nil {
		t.Fatal("expected error for port above max")
	}
	if !strings.Contains(err.Error(), "max") {
		t.Errorf("expected error about max, got: %v", err)
	}
}

func TestValidationTag_MinOnly(t *testing.T) {
	type Params struct {
		Count int `descr:"count" min:"0"`
	}

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--count", "-1"})
	if err == nil {
		t.Fatal("expected error for count below min")
	}
}

func TestValidationTag_MaxOnly(t *testing.T) {
	type Params struct {
		Retries int `descr:"retries" max:"10"`
	}

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--retries", "11"})
	if err == nil {
		t.Fatal("expected error for retries above max")
	}
}

func TestValidationTag_MinMax_Float(t *testing.T) {
	type Params struct {
		Rate float64 `descr:"rate" min:"0.0" max:"1.0"`
	}

	// Valid
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--rate", "0.5"})
	if err != nil {
		t.Fatalf("expected no error for valid rate, got: %v", err)
	}

	// Above max
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--rate", "1.5"})
	if err == nil {
		t.Fatal("expected error for rate above max")
	}
}

// --- pattern for string types ---

func TestValidationTag_Pattern(t *testing.T) {
	type Params struct {
		Name string `descr:"name" pattern:"^[a-z][a-z0-9-]*$"`
	}

	// Valid
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--name", "my-app-123"})
	if err != nil {
		t.Fatalf("expected no error for valid name, got: %v", err)
	}

	// Invalid — starts with uppercase
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--name", "MyApp"})
	if err == nil {
		t.Fatal("expected error for name not matching pattern")
	}
	if !strings.Contains(err.Error(), "pattern") {
		t.Errorf("expected error about pattern, got: %v", err)
	}
}

func TestValidationTag_Pattern_Optional_NotSet(t *testing.T) {
	// Pattern should not trigger when the field is optional and not set
	type Params struct {
		Name *string `descr:"name" pattern:"^[a-z]+$"`
	}

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{})
	if err != nil {
		t.Fatalf("expected no error when optional pattern field not set, got: %v", err)
	}
}

// --- min/max for string length ---

func TestValidationTag_MinMax_Pointer_Set(t *testing.T) {
	// min/max should validate pointer fields when a value is provided
	type Params struct {
		Port *int `descr:"port" min:"1" max:"65535"`
	}

	// Valid
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--port", "8080"})
	if err != nil {
		t.Fatalf("expected no error for valid port, got: %v", err)
	}

	// Below min
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--port", "0"})
	if err == nil {
		t.Fatal("expected error for port below min")
	}
}

func TestValidationTag_MinMax_Pointer_NotSet(t *testing.T) {
	// min/max should NOT trigger when pointer field is not set (nil)
	type Params struct {
		Port *int `descr:"port" min:"1" max:"65535"`
	}

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{})
	if err != nil {
		t.Fatalf("expected no error when optional min/max field not set, got: %v", err)
	}
}

func TestValidationTag_Pattern_Pointer_Set(t *testing.T) {
	// pattern should validate pointer fields when a value is provided
	type Params struct {
		Tag *string `descr:"tag" pattern:"^v[0-9]+\\.[0-9]+\\.[0-9]+$"`
	}

	// Valid
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--tag", "v1.2.3"})
	if err != nil {
		t.Fatalf("expected no error for valid tag, got: %v", err)
	}

	// Invalid
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--tag", "latest"})
	if err == nil {
		t.Fatal("expected error for tag not matching pattern")
	}
}

func TestValidationTag_MinMax_StringLength(t *testing.T) {
	type Params struct {
		Name string `descr:"name" min:"3" max:"20"`
	}

	// Valid
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--name", "alice"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Too short
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--name", "ab"})
	if err == nil {
		t.Fatal("expected error for name too short")
	}

	// Too long
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--name", "a-very-long-name-that-exceeds-twenty"})
	if err == nil {
		t.Fatal("expected error for name too long")
	}
}

// --- min/max for slice length ---

func TestValidationTag_MinMax_Slice(t *testing.T) {
	type Params struct {
		Tags []string `descr:"tags" min:"1" max:"3"`
	}

	// Valid: 2 items
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--tags", "a", "--tags", "b"})
	if err != nil {
		t.Fatalf("expected no error for valid slice, got: %v", err)
	}

	// Below min: 1 item when min is 1 (need at least 1 tag provided)
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{})
	if err == nil {
		t.Fatal("expected error for slice below min (required with 0 items)")
	}

	// Above max: 4 items
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--tags", "a", "--tags", "b", "--tags", "c", "--tags", "d"})
	if err == nil {
		t.Fatal("expected error for slice above max")
	}
	if !strings.Contains(err.Error(), "max") {
		t.Errorf("expected error about max, got: %v", err)
	}
}

func TestValidationTag_MinOnly_Slice(t *testing.T) {
	type Params struct {
		Files []string `descr:"files" min:"2"`
	}

	// Valid: exactly 2
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--files", "a", "--files", "b"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Below min: 1 item
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--files", "a"})
	if err == nil {
		t.Fatal("expected error for slice below min")
	}
}

func TestValidationTag_MaxOnly_Slice(t *testing.T) {
	type Params struct {
		Items []string `descr:"items" max:"2" optional:"true"`
	}

	// Valid: 0 items
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{})
	if err != nil {
		t.Fatalf("expected no error for empty slice, got: %v", err)
	}

	// Valid: 2 items
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--items", "a", "--items", "b"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Above max: 3 items
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--items", "a", "--items", "b", "--items", "c"})
	if err == nil {
		t.Fatal("expected error for slice above max")
	}
}

func TestValidationTag_MinMax_Slice_Positional(t *testing.T) {
	type Params struct {
		Files []string `positional:"true" min:"2" max:"4"`
	}

	// Valid: 3 items
	var got []string
	(CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) { got = p.Files },
	}).RunArgs([]string{"a", "b", "c"})
	if len(got) != 3 {
		t.Fatalf("expected 3 files, got %d", len(got))
	}

	// Below min: 1 item
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"a"})
	if err == nil {
		t.Fatal("expected error for positional slice below min")
	}
	if !strings.Contains(err.Error(), "min") {
		t.Errorf("expected error about min, got: %v", err)
	}

	// Above max: 5 items
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"a", "b", "c", "d", "e"})
	if err == nil {
		t.Fatal("expected error for positional slice above max")
	}
	if !strings.Contains(err.Error(), "max") {
		t.Errorf("expected error about max, got: %v", err)
	}
}

func TestValidationTag_MinMax_IntSlice(t *testing.T) {
	type Params struct {
		Ports []int `descr:"ports" min:"1" max:"3"`
	}

	// Valid: 2 items
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--ports", "80", "--ports", "443"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Above max: 4 items
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--ports", "80", "--ports", "443", "--ports", "8080", "--ports", "9090"})
	if err == nil {
		t.Fatal("expected error for int slice above max")
	}
}

func TestValidationTag_MinMax_RequiredSliceFlag(t *testing.T) {
	type Params struct {
		Tags []string `descr:"tags" min:"2" required:"true"`
	}

	// 0 items: required error fires first
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{})
	if err == nil {
		t.Fatal("expected error for 0 items on required slice with min:2")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("expected 'required' error for 0 items, got: %v", err)
	}

	// 1 item: min validation fires
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--tags", "a"})
	if err == nil {
		t.Fatal("expected error for 1 item with min:2")
	}
	if !strings.Contains(err.Error(), "min") {
		t.Errorf("expected 'min' error for 1 item, got: %v", err)
	}

	// 2 items: passes
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--tags", "a", "--tags", "b"})
	if err != nil {
		t.Fatalf("expected no error for 2 items, got: %v", err)
	}
}

func TestValidationTag_MinMax_RequiredSlicePositional(t *testing.T) {
	type Params struct {
		Files []string `positional:"true" min:"2" required:"true"`
	}

	// 0 items: cobra args validator fires first
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{})
	if err == nil {
		t.Fatal("expected error for 0 positional args with min:2")
	}

	// 1 item: min validation fires
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"a"})
	if err == nil {
		t.Fatal("expected error for 1 positional arg with min:2")
	}
	if !strings.Contains(err.Error(), "min") {
		t.Errorf("expected 'min' error, got: %v", err)
	}

	// 2 items: passes
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"a", "b"})
	if err != nil {
		t.Fatalf("expected no error for 2 items, got: %v", err)
	}
}
