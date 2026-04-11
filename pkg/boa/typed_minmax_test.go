package boa

import (
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// --- int64 precision: the whole point of the typed-storage refactor ---

// TestValidationTag_Int64_PrecisionBug exercises the specific correctness gap
// the pre-refactor float64 pipeline had. With the old code:
//
//	max := float64(1 << 53)     // = 9007199254740992
//	val := float64(1<<53 + 1)   // rounds to 9007199254740992 (!!)
//	val > max                   // false → value (1<<53 + 1) passed validation
//
// Post-refactor the bound is stored as *int64 and v.Int() returns int64, so
// the comparison is lossless. This test fails loudly on any regression.
func TestValidationTag_Int64_PrecisionBug(t *testing.T) {
	// 2^53 is exactly representable in float64; 2^53+1 is not (it rounds to
	// 2^53). So if the bound is 2^53 and the input is 2^53+1, the old
	// float64-based comparison said "equal" and let it through.
	const maxStr = "9007199254740992"          // 2^53
	const cheatingVal = "9007199254740993"     // 2^53 + 1 — must be rejected

	type Params struct {
		N int64 `descr:"n" max:"9007199254740992"`
	}

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--n", maxStr})
	if err != nil {
		t.Fatalf("value == max (2^53) should pass, got: %v", err)
	}

	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--n", cheatingVal})
	if err == nil {
		t.Fatalf("value = 2^53+1 with max=2^53 must be rejected (this was the precision bug)")
	}
	if !strings.Contains(err.Error(), "max") {
		t.Errorf("expected error mentioning 'max', got: %v", err)
	}
}

// TestSetMaxT_Int64_PrecisionBug does the same check via the typed
// programmatic API.
func TestSetMaxT_Int64_PrecisionBug(t *testing.T) {
	const max = int64(1 << 53)     // 9007199254740992, exactly representable in f64
	const cheating = max + 1       // 9007199254740993, rounds down in f64

	type Params struct {
		N int64 `descr:"n" optional:"true"`
	}

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			GetParamT(ctx, &params.N).SetMaxT(max)
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--n", strconv.FormatInt(cheating, 10)})
	if err == nil || !strings.Contains(err.Error(), "max") {
		t.Fatalf("value 2^53+1 with max=2^53 must be rejected, got: %v", err)
	}
}

// TestValidationTag_Int64_MaxBoundary validates that math.MaxInt64 itself is
// parseable as a tag and is enforced exactly.
func TestValidationTag_Int64_MaxBoundary(t *testing.T) {
	type Params struct {
		N int64 `descr:"n" max:"9223372036854775807"` // math.MaxInt64
	}

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--n", "9223372036854775807"})
	if err != nil {
		t.Fatalf("math.MaxInt64 should pass == max, got: %v", err)
	}
}

// Note on uint / int8 / int16 fields: boa's type_handler.go only registers
// CLI handlers for Int, Int32, Int64, Float32, Float64. Unsigned and narrow
// signed kinds aren't valid field types today — the field traversal rejects
// them — so we don't test min/max against them. The boundKind classifier
// still has a signedIntBound / unsignedIntBound split as forward compat for
// when those kinds get first-class handler support.

// --- map length bound (previously not supported; CLAUDE.md mentioned maps
//     but supportsMinMax never returned true for reflect.Map) ---

func TestValidationTag_MapLength(t *testing.T) {
	type Params struct {
		Labels map[string]string `descr:"labels" min:"2" max:"3" optional:"true"`
	}
	// 1 entry → below min
	if err := (CmdT[Params]{
		Use: "test", ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--labels", "a=1"}); err == nil || !strings.Contains(err.Error(), "min") {
		t.Errorf("expected min error for 1-entry map, got: %v", err)
	}
	// 4 entries → above max
	if err := (CmdT[Params]{
		Use: "test", ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--labels", "a=1,b=2,c=3,d=4"}); err == nil || !strings.Contains(err.Error(), "max") {
		t.Errorf("expected max error for 4-entry map, got: %v", err)
	}
	// 2 entries → valid
	if err := (CmdT[Params]{
		Use: "test", ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--labels", "a=1,b=2"}); err != nil {
		t.Errorf("expected no error for 2-entry map, got: %v", err)
	}
}

func TestSetMinMaxLen_Programmatic_Map(t *testing.T) {
	type Params struct {
		Labels map[string]string `descr:"labels" optional:"true"`
	}
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			p := GetParamT(ctx, &params.Labels)
			p.SetMinLen(2)
			p.SetMaxLen(3)
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--labels", "a=1"})
	if err == nil || !strings.Contains(err.Error(), "min") {
		t.Fatalf("expected min error for 1-entry map, got: %v", err)
	}
}

// --- type-alias fields behave like their underlying kind ---

func TestSetMinMaxT_TypeAlias(t *testing.T) {
	type Port int32
	type Params struct {
		P Port `descr:"p" optional:"true"`
	}
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			p := GetParamT(ctx, &params.P)
			p.SetMinT(Port(1))
			p.SetMaxT(Port(100))
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--p", "500"})
	if err == nil || !strings.Contains(err.Error(), "max") {
		t.Fatalf("expected max error for type-aliased int32, got: %v", err)
	}
}

// --- tag parser rejects garbage per-kind ---

func TestValidationTag_MinMax_RejectsFloatOnIntField(t *testing.T) {
	type Params struct {
		N int `descr:"n" min:"1.5"`
	}
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--n", "5"})
	if err == nil || !strings.Contains(err.Error(), "invalid min") {
		t.Errorf("expected tag parse error, got: %v", err)
	}
}

func TestValidationTag_MinMax_RejectsNegativeLengthOnSlice(t *testing.T) {
	type Params struct {
		Xs []string `descr:"xs" min:"-1" optional:"true"`
	}
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{})
	if err == nil || !strings.Contains(err.Error(), "invalid min") {
		t.Errorf("expected tag parse error on negative length, got: %v", err)
	}
}

// --- programmatic setter rejects float bound on int field ---

func TestSetMin_RejectsFloatBoundOnIntField(t *testing.T) {
	type Params struct {
		N int `descr:"n" optional:"true"`
	}
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when setting float bound on int field")
		}
		if !strings.Contains(strings.ToLower(strings.ReplaceAll(
			// safe stringify
			toString(r), "\n", " ")), "float") {
			t.Errorf("expected panic to mention 'float', got: %v", r)
		}
	}()
	_ = (CmdT[Params]{
		Use: "test",
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			ctx.GetParam(&params.N).SetMin(1.5)
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{})
}

// toString is a tiny helper to avoid importing fmt just for panic stringify.
func toString(v any) string {
	if s, ok := v.(interface{ Error() string }); ok {
		return s.Error()
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// --- non-generic Param.SetMin accepts any numeric ---

func TestParamSetMin_AcceptsAnyNumericOnIntField(t *testing.T) {
	type Params struct {
		N int `descr:"n" optional:"true"`
	}
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			p := ctx.GetParam(&params.N)
			// An int8 value should be coerced into the *int64 storage.
			p.SetMin(int8(5))
			p.SetMax(int16(10))
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--n", "3"})
	if err == nil || !strings.Contains(err.Error(), "min") {
		t.Fatalf("expected min error, got: %v", err)
	}
}
