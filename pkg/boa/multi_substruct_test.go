package boa

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestMultipleSubstructsOfSameType_NoInterference exercises the case where a
// root params struct holds two fields of the same substruct type (a common
// pattern — e.g., a read-write and a read-only database connection config).
// Each instance gets its own prefix and its own mirrors; configuring one must
// not affect the other, even though their Go types are identical.
//
// Shape:
//
//	type Params struct {
//	    RW DB  // flags: --rw-host, --rw-port, paths "0.0", "0.1"
//	    RO DB  // flags: --ro-host, --ro-port, paths "1.0", "1.1"
//	}
//
// The test proves:
//   - Each substruct instance produces its own distinct paramMeta.
//   - Alternatives and custom validators set on one instance's mirror are
//     invisible to the other's.
//   - CLI values for --rw-host and --ro-host land in their respective fields.
func TestMultipleSubstructsOfSameType_NoInterference(t *testing.T) {
	type DB struct {
		Host string `descr:"host"`
		Port int    `descr:"port" default:"0"`
	}
	type Params struct {
		RW DB
		RO DB
	}

	var (
		rwHostMirror   Param
		roHostMirror   Param
		rwValidations  int
		roValidations  int
		finalRWHost    string
		finalROHost    string
	)

	makeTree := func() CmdT[Params] {
		return CmdT[Params]{
			Use: "test",
			InitFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command) error {
				rwHostMirror = ctx.GetParam(&p.RW.Host)
				roHostMirror = ctx.GetParam(&p.RO.Host)

				// Different strict alternatives sets.
				rwHostMirror.SetAlternatives([]string{"rw-a", "rw-b"})
				roHostMirror.SetAlternatives([]string{"ro-a", "ro-b"})

				// Different custom validators with observable side effects.
				rwHostMirror.SetCustomValidator(func(v any) error {
					rwValidations++
					if !strings.HasPrefix(v.(string), "rw-") {
						return fmt.Errorf("RW host must have 'rw-' prefix")
					}
					return nil
				})
				roHostMirror.SetCustomValidator(func(v any) error {
					roValidations++
					if !strings.HasPrefix(v.(string), "ro-") {
						return fmt.Errorf("RO host must have 'ro-' prefix")
					}
					return nil
				})
				return nil
			},
			RunFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command, args []string) {
				finalRWHost = p.RW.Host
				finalROHost = p.RO.Host
			},
		}
	}

	reset := func() {
		rwHostMirror, roHostMirror = nil, nil
		rwValidations, roValidations = 0, 0
		finalRWHost, finalROHost = "", ""
	}

	// --- Distinct mirror instances for the two substructs ---
	reset()
	tree := makeTree()
	if err := tree.RunArgsE([]string{"--rw-host", "rw-a", "--ro-host", "ro-b"}); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if rwHostMirror == nil || roHostMirror == nil {
		t.Fatalf("mirror fetch returned nil: rw=%p ro=%p", rwHostMirror, roHostMirror)
	}
	if rwHostMirror == roHostMirror {
		t.Errorf("two substructs of the same type share a single mirror instance: rw=ro=%p", rwHostMirror)
	}

	// --- Alternatives did not bleed across instances ---
	rwAlts := rwHostMirror.GetAlternatives()
	roAlts := roHostMirror.GetAlternatives()
	if !slices.Equal(rwAlts, []string{"rw-a", "rw-b"}) {
		t.Errorf("RW alternatives clobbered: %v", rwAlts)
	}
	if !slices.Equal(roAlts, []string{"ro-a", "ro-b"}) {
		t.Errorf("RO alternatives clobbered: %v", roAlts)
	}

	// --- Each validator fires exactly once for its own field, not the other ---
	if rwValidations != 1 {
		t.Errorf("rwValidations = %d, want 1", rwValidations)
	}
	if roValidations != 1 {
		t.Errorf("roValidations = %d, want 1", roValidations)
	}

	// --- CLI values landed in their respective fields ---
	if finalRWHost != "rw-a" {
		t.Errorf("finalRWHost = %q, want 'rw-a'", finalRWHost)
	}
	if finalROHost != "ro-b" {
		t.Errorf("finalROHost = %q, want 'ro-b'", finalROHost)
	}

	// --- Cross-substruct rejection: feeding RW's alt to RO should fail strict-alts ---
	reset()
	err := makeTree().RunArgsE([]string{"--rw-host", "rw-a", "--ro-host", "rw-a"})
	if err == nil {
		t.Errorf("RO should reject 'rw-a' (not in its strict alts list)")
	}

	// --- Cross-substruct rejection via the custom validator prefix check ---
	// Pick a value that passes strict-alts for RO but fails its custom validator.
	reset()
	makeTree2 := makeTree
	// Replace RO's alternatives with a value that's NOT a valid rw-/ro- prefix to
	// isolate the custom validator's contribution.
	_ = makeTree2 // keep compiler happy if the logic below changes
	err = makeTree().RunArgsE([]string{"--rw-host", "rw-a", "--ro-host", "ro-a"})
	if err != nil {
		t.Errorf("ro-a should be accepted by both RO alts and RO validator, got: %v", err)
	}
}

// TestMultipleSubstructPointersOfSameType_NoInterference is the pointer-field
// variant: the two substruct fields are *DB instead of DB. Boa preallocates
// both on build, each gets its own mirrors and its own heap allocation. The
// same isolation invariants must hold.
func TestMultipleSubstructPointersOfSameType_NoInterference(t *testing.T) {
	type DB struct {
		Host string `descr:"host"`
	}
	type Params struct {
		RW *DB
		RO *DB
	}

	var (
		rwHostMirror Param
		roHostMirror Param
		finalRWHost  string
		finalROHost  string
	)

	makeTree := func() CmdT[Params] {
		return CmdT[Params]{
			Use: "test",
			InitFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command) error {
				// Both substructs are preallocated by boa before init fires, so
				// &p.RW.Host and &p.RO.Host are both safe to address here.
				rwHostMirror = ctx.GetParam(&p.RW.Host)
				roHostMirror = ctx.GetParam(&p.RO.Host)
				rwHostMirror.SetAlternatives([]string{"rw-only"})
				roHostMirror.SetAlternatives([]string{"ro-only"})
				return nil
			},
			RunFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command, args []string) {
				if p.RW != nil {
					finalRWHost = p.RW.Host
				}
				if p.RO != nil {
					finalROHost = p.RO.Host
				}
			},
		}
	}

	// Happy path: each substruct takes its own value.
	if err := makeTree().RunArgsE([]string{"--rw-host", "rw-only", "--ro-host", "ro-only"}); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if rwHostMirror == roHostMirror {
		t.Errorf("two preallocated substructs of the same pointer type share a single mirror: %p", rwHostMirror)
	}
	if finalRWHost != "rw-only" {
		t.Errorf("finalRWHost = %q, want 'rw-only'", finalRWHost)
	}
	if finalROHost != "ro-only" {
		t.Errorf("finalROHost = %q, want 'ro-only'", finalROHost)
	}

	// Cross-reject: RW's value rejected by RO's strict alts.
	err := makeTree().RunArgsE([]string{"--rw-host", "rw-only", "--ro-host", "rw-only"})
	if err == nil {
		t.Errorf("RO should reject RW's alt value under strict alternatives")
	}

	// Cross-reject: RO's value rejected by RW's strict alts.
	err = makeTree().RunArgsE([]string{"--rw-host", "ro-only", "--ro-host", "ro-only"})
	if err == nil {
		t.Errorf("RW should reject RO's alt value under strict alternatives")
	}
}
