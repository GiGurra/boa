package boa

import (
	"reflect"
	"slices"
	"testing"
	"unsafe"

	"github.com/spf13/cobra"
)

// TestSubstructReassignmentPreservesMirror exercises the scenario the user asked
// about directly:
//
//  1. An InitFuncCtx fetches the mirror for a field inside an optional substruct
//     pointer (preallocated by boa) and configures it with a dynamic alternatives
//     func.
//  2. The user reassigns the entire substruct pointer — the new pointee has a
//     different heap address, so `&params.DB.Host` now points into different memory.
//  3. The hook fetches the mirror again via the new field address and applies a
//     second, non-conflicting piece of configuration (a custom validator).
//
// The test verifies:
//   - GetParam returns the SAME mirror instance after reassignment.
//   - Both configurations (alternatives func + custom validator) are preserved on
//     the final mirror.
//   - CLI parsing still routes values into the (reassigned) substruct correctly.
//
// This is the load-bearing win of field-index keying: mirror state is owned by
// the path, not by the original heap address of the substruct.
func TestSubstructReassignmentPreservesMirror(t *testing.T) {
	type DB struct {
		Host string `descr:"host" default:"localhost"`
		Port int    `descr:"port" default:"5432"`
	}
	type Params struct {
		DB *DB
	}

	var (
		mirrorBeforeReassign Param
		mirrorAfterReassign  Param
		finalAltsFunc        func(cmd *cobra.Command, args []string, toComplete string) []string
		finalValidator       func(any) error
		finalHost            string
		finalPort            int
		validatorCalls       int
	)

	cmd := (CmdT[Params]{
		Use: "test",
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			// Step 1: mirror via the preallocated pointee — set a dynamic alternatives func.
			m1 := ctx.GetParam(&params.DB.Host)
			if m1 == nil {
				t.Fatal("mirror for DB.Host not found before reassignment")
			}
			m1.SetAlternativesFunc(func(cmd *cobra.Command, args []string, toComplete string) []string {
				return []string{"gamma", "delta"}
			})
			mirrorBeforeReassign = m1

			// Step 2: reassign the substruct pointer. The new pointee is at a different
			// heap address, so &params.DB.Host now points into different memory.
			params.DB = &DB{Host: "inline-default", Port: 9999}

			// Step 3: mirror via the NEW field address — must resolve to the same instance.
			m2 := ctx.GetParam(&params.DB.Host)
			if m2 == nil {
				t.Fatal("mirror for DB.Host not found after reassignment")
			}
			mirrorAfterReassign = m2

			// Non-conflicting configuration: a custom validator.
			m2.SetCustomValidator(func(v any) error {
				validatorCalls++
				return nil
			})
			return nil
		},
		RunFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command, args []string) {
			if p.DB != nil {
				finalHost = p.DB.Host
				finalPort = p.DB.Port
			}
			if m := ctx.GetParam(&p.DB.Host); m != nil {
				finalAltsFunc = m.GetAlternativesFunc()
				if pm, ok := m.(*paramMeta); ok {
					finalValidator = pm.customValidator
				}
			}
		},
	}).ToCobra()

	cmd.SetArgs([]string{"--db-host", "alpha"})
	if err := Execute(cmd); err != nil {
		t.Fatalf("execute: %v", err)
	}

	// --- Mirror identity across reassignment ---
	if mirrorBeforeReassign == nil || mirrorAfterReassign == nil {
		t.Fatalf("mirror nil: before=%v after=%v", mirrorBeforeReassign, mirrorAfterReassign)
	}
	if mirrorBeforeReassign != mirrorAfterReassign {
		t.Errorf("mirror identity changed across reassignment: before=%p after=%p",
			mirrorBeforeReassign, mirrorAfterReassign)
	}

	// --- Configuration combined on final mirror ---
	if finalAltsFunc == nil {
		t.Errorf("alternatives func set BEFORE reassignment was lost")
	} else {
		got := finalAltsFunc(cmd, nil, "")
		if !slices.Equal(got, []string{"gamma", "delta"}) {
			t.Errorf("alternatives func returned wrong result: %v", got)
		}
	}
	if finalValidator == nil {
		t.Errorf("custom validator set AFTER reassignment was lost")
	}
	if validatorCalls == 0 {
		t.Errorf("custom validator was never called during validation phase")
	}

	// --- CLI value reached the reassigned substruct ---
	if finalHost != "alpha" {
		t.Errorf("CLI value did not reach reassigned substruct: host=%q", finalHost)
	}
	// Port was set by reassignment (not CLI). If syncMirrors injected the non-zero
	// raw value, cleanupPreallocatedPtrs should see the struct as "set" and leave
	// it alive. Document the observed behavior rather than assume.
	if finalPort != 9999 {
		t.Logf("reassignment-set port value did not survive: port=%d (may be expected; see cleanupPreallocatedPtrs)", finalPort)
	}
}

// TestDeepNestedReassignment_FullTree exercises the full-tree-reassignment case
// for deeply nested pointer substructs: the user replaces the outermost pointer,
// which transitively replaces every inner pointer and leaf. Every intermediate
// heap address changes. GetParam must still resolve mirror identity through the
// new tree.
func TestDeepNestedReassignment_FullTree(t *testing.T) {
	type Leaf struct {
		Value string `descr:"value"`
	}
	type Middle struct {
		Leaf *Leaf
	}
	type Outer struct {
		Middle *Middle
	}
	type Params struct {
		Outer *Outer
	}

	var (
		mBefore    Param
		mAfter     Param
		finalValue string
	)

	cmd := (CmdT[Params]{
		Use: "test",
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			mBefore = ctx.GetParam(&params.Outer.Middle.Leaf.Value)
			if mBefore == nil {
				t.Fatal("mirror for Outer.Middle.Leaf.Value not found before reassignment")
			}
			mBefore.SetDefault(Default("initial-default"))

			// Replace the entire tree — every heap address below params.Outer changes.
			params.Outer = &Outer{Middle: &Middle{Leaf: &Leaf{Value: ""}}}

			mAfter = ctx.GetParam(&params.Outer.Middle.Leaf.Value)
			if mAfter == nil {
				t.Fatal("mirror for Outer.Middle.Leaf.Value not found after reassignment")
			}
			return nil
		},
		RunFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command, args []string) {
			if p.Outer != nil && p.Outer.Middle != nil && p.Outer.Middle.Leaf != nil {
				finalValue = p.Outer.Middle.Leaf.Value
			}
		},
	}).ToCobra()

	cmd.SetArgs([]string{"--outer-middle-leaf-value", "cli-supplied"})
	if err := Execute(cmd); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if mBefore != mAfter {
		t.Errorf("mirror identity changed across full-tree reassignment: before=%p after=%p", mBefore, mAfter)
	}
	if finalValue != "cli-supplied" {
		t.Errorf("CLI value did not reach the deeply nested reassigned field: got %q", finalValue)
	}
}

// TestDeepNestedReassignment_MiddleOnly exercises the partial-reassignment case:
// only the middle pointer is replaced, leaving the outer pointer identity intact.
// Path resolution must walk through the unchanged outer and the new middle.
func TestDeepNestedReassignment_MiddleOnly(t *testing.T) {
	type Leaf struct {
		Value string `descr:"value"`
	}
	type Middle struct {
		Leaf *Leaf
	}
	type Outer struct {
		Middle *Middle
	}
	type Params struct {
		Outer *Outer
	}

	var (
		mBefore    Param
		mAfter     Param
		finalValue string
	)

	cmd := (CmdT[Params]{
		Use: "test",
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			mBefore = ctx.GetParam(&params.Outer.Middle.Leaf.Value)
			if mBefore == nil {
				t.Fatal("mirror not found before middle reassignment")
			}

			// Replace only Middle (Outer stays the same preallocated instance).
			params.Outer.Middle = &Middle{Leaf: &Leaf{Value: ""}}

			mAfter = ctx.GetParam(&params.Outer.Middle.Leaf.Value)
			if mAfter == nil {
				t.Fatal("mirror not found after middle reassignment")
			}
			return nil
		},
		RunFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command, args []string) {
			if p.Outer != nil && p.Outer.Middle != nil && p.Outer.Middle.Leaf != nil {
				finalValue = p.Outer.Middle.Leaf.Value
			}
		},
	}).ToCobra()

	cmd.SetArgs([]string{"--outer-middle-leaf-value", "partial"})
	if err := Execute(cmd); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if mBefore != mAfter {
		t.Errorf("mirror identity changed across middle reassignment: before=%p after=%p", mBefore, mAfter)
	}
	if finalValue != "partial" {
		t.Errorf("CLI value did not reach reassigned middle: got %q", finalValue)
	}
}

// TestReassignToNilAndBack exercises the case where a user clears a substruct
// pointer (sets it to nil) and then sets it back to a fresh non-nil pointee.
// GetParam must return nil while the pointer is nil, then resolve the mirror
// again once the pointer is restored.
func TestReassignToNilAndBack(t *testing.T) {
	type DB struct {
		Host string `descr:"host"`
	}
	type Params struct {
		DB *DB
	}

	var (
		mBefore   Param
		mDuringNil Param
		mAfter    Param
		finalHost string
	)

	cmd := (CmdT[Params]{
		Use: "test",
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			mBefore = ctx.GetParam(&params.DB.Host)
			if mBefore == nil {
				t.Fatal("mirror not found initially")
			}

			// Nil the substruct — while nil, the path cannot be resolved.
			params.DB = nil
			// We cannot call &params.DB.Host here (it would segfault on the nil
			// dereference), so we intentionally reach into the internal mirrorByPath
			// to verify the invariant this test exists to prove: the mirror stays in
			// the authoritative store even while the pointer that used to address it
			// is nil. This is a deliberate test-only coupling to internal state —
			// the mirror store is the single source of truth, and field-address
			// resolution is just a convenience layered on top.
			if ctx != nil && ctx.ctx != nil {
				if m, ok := ctx.ctx.mirrorByPath["0.0"]; ok {
					mDuringNil = m
				}
			}

			// Restore to a fresh pointee at a new heap address.
			params.DB = &DB{Host: ""}
			mAfter = ctx.GetParam(&params.DB.Host)
			if mAfter == nil {
				t.Fatal("mirror not found after restoration")
			}
			return nil
		},
		RunFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command, args []string) {
			if p.DB != nil {
				finalHost = p.DB.Host
			}
		},
	}).ToCobra()

	cmd.SetArgs([]string{"--db-host", "restored"})
	if err := Execute(cmd); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if mBefore != mAfter {
		t.Errorf("mirror identity changed across nil-and-back: before=%p after=%p", mBefore, mAfter)
	}
	if mDuringNil != mBefore {
		t.Errorf("mirror dropped from authoritative store while pointer was nil: during=%p before=%p", mDuringNil, mBefore)
	}
	if finalHost != "restored" {
		t.Errorf("CLI value did not reach restored pointee: got %q", finalHost)
	}
}

// TestGetParam_AutoRepairsCacheAfterReassignment verifies that when the
// addrToPath cache is stale (because a substruct was reassigned after init),
// the fallback walk in GetParam not only finds the mirror but also *repairs
// the cache* so subsequent lookups for the same field address are O(1) again.
//
// This is a white-box test — it reaches into the internal addrToPath map to
// verify the repair — because the optimization is not observable through the
// public API except by microbenchmark. Verifying the map state is the direct
// evidence that the repair code path ran.
func TestGetParam_AutoRepairsCacheAfterReassignment(t *testing.T) {
	type DB struct {
		Host string `descr:"host" default:"localhost"`
	}
	type Params struct {
		DB *DB
	}

	var (
		newHostAddr           unsafe.Pointer
		cacheHadNewAddrBefore bool
		cacheHadNewAddrAfter  bool
	)

	cmd := (CmdT[Params]{
		Use: "test",
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			// Step 1: prime the cache with the preallocated pointee's addresses.
			if m := ctx.GetParam(&params.DB.Host); m == nil {
				t.Fatal("initial lookup returned nil")
			}

			// Step 2: reassign the substruct to a fresh pointee at a new heap
			// address. Old cache entries are now stale; the new addresses have
			// never been seen by addrToPath.
			params.DB = &DB{Host: ""}
			newHostAddr = reflect.ValueOf(&params.DB.Host).UnsafePointer()

			// Check: the new address should NOT be in the cache yet.
			if _, ok := ctx.ctx.addrToPath[newHostAddr]; ok {
				cacheHadNewAddrBefore = true
			}

			// Step 3: do the lookup through the new address. The cache lookup
			// will miss, the fallback walk will succeed, and (per the repair
			// code) the new address should then be written into addrToPath.
			m := ctx.GetParam(&params.DB.Host)
			if m == nil {
				t.Fatal("post-reassignment lookup returned nil")
			}

			// Check: after the lookup, the new address must now be in the cache.
			if _, ok := ctx.ctx.addrToPath[newHostAddr]; ok {
				cacheHadNewAddrAfter = true
			}
			return nil
		},
		RunFunc: func(p *Params, c *cobra.Command, args []string) {},
	}).ToCobra()

	cmd.SetArgs([]string{"--db-host", "x"})
	if err := Execute(cmd); err != nil {
		t.Fatalf("execute: %v", err)
	}

	// Pre-condition: the new address was not cached before the fallback ran.
	// (If this fires, the test setup is broken — we accidentally pre-populated
	// the cache somehow.)
	if cacheHadNewAddrBefore {
		t.Errorf("precondition violated: new address was already cached before the fallback lookup")
	}

	// Post-condition: the fallback walk repaired the cache.
	if !cacheHadNewAddrAfter {
		t.Errorf("cache repair did not happen: new address is still not in addrToPath after fallback lookup")
	}
}

// TestEmbeddedPointerStructReassignment covers anonymous (embedded) pointer
// struct fields, which get no flag-name prefix per boa's promotion rules. After
// reassigning the embedded pointer, the mirror for a promoted field must still
// be reachable via `&outer.Promoted`.
func TestEmbeddedPointerStructReassignment(t *testing.T) {
	type Inner struct {
		Thing string `descr:"thing"`
	}
	type Params struct {
		*Inner
	}

	var (
		mBefore    Param
		mAfter     Param
		finalThing string
	)

	cmd := (CmdT[Params]{
		Use: "test",
		InitFuncCtx: func(ctx *HookContext, params *Params, cmd *cobra.Command) error {
			mBefore = ctx.GetParam(&params.Thing)
			if mBefore == nil {
				t.Fatal("mirror for embedded field not found before reassignment")
			}

			params.Inner = &Inner{Thing: ""}

			mAfter = ctx.GetParam(&params.Thing)
			if mAfter == nil {
				t.Fatal("mirror for embedded field not found after reassignment")
			}
			return nil
		},
		RunFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command, args []string) {
			if p.Inner != nil {
				finalThing = p.Thing
			}
		},
	}).ToCobra()

	// Embedded → no prefix, so the flag is just --thing.
	cmd.SetArgs([]string{"--thing", "embedded-value"})
	if err := Execute(cmd); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if mBefore != mAfter {
		t.Errorf("mirror identity changed across embedded reassignment: before=%p after=%p", mBefore, mAfter)
	}
	if finalThing != "embedded-value" {
		t.Errorf("CLI value did not reach reassigned embedded struct: got %q", finalThing)
	}
}
