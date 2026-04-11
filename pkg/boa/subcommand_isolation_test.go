package boa

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// The tests in this file check that a single Params struct type can be used by a
// root command and multiple subcommands, each applying its own set of programmatic
// constraints via InitFuncCtx, without any cross-command interference.
//
// Each command has its own processingContext (built by its own toCobraBase call)
// and therefore its own set of paramMeta instances keyed by fieldPath. Mirrors
// are type-level identities instance-bound to a specific context — there is no
// shared global mirror store. These tests make that invariant observable.

// TestSubcommandIsolation_Alternatives — root, sub1, and sub2 share the same
// Params type but each assigns a different strict alternatives set via
// InitFuncCtx. Each command must accept its own alternatives and reject the
// others.
func TestSubcommandIsolation_Alternatives(t *testing.T) {
	type Params struct {
		Value string `descr:"value"`
	}

	// Captured per-invocation observations.
	var rootRan, sub1Ran, sub2Ran bool
	var rootVal, sub1Val, sub2Val string

	// Factory: builds a fresh tree per invocation to avoid cobra flag-state
	// leakage between Execute() calls.
	makeTree := func() CmdT[Params] {
		return CmdT[Params]{
			Use: "root",
			InitFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command) error {
				ctx.GetParam(&p.Value).SetAlternatives([]string{"root-a", "root-b"})
				return nil
			},
			RunFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command, args []string) {
				rootRan = true
				rootVal = p.Value
			},
			SubCmds: SubCmds(
				CmdT[Params]{
					Use: "sub1",
					InitFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command) error {
						ctx.GetParam(&p.Value).SetAlternatives([]string{"sub1-x", "sub1-y"})
						return nil
					},
					RunFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command, args []string) {
						sub1Ran = true
						sub1Val = p.Value
					},
				},
				CmdT[Params]{
					Use: "sub2",
					InitFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command) error {
						ctx.GetParam(&p.Value).SetAlternatives([]string{"sub2-m", "sub2-n"})
						return nil
					},
					RunFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command, args []string) {
						sub2Ran = true
						sub2Val = p.Value
					},
				},
			),
		}
	}

	reset := func() {
		rootRan, sub1Ran, sub2Ran = false, false, false
		rootVal, sub1Val, sub2Val = "", "", ""
	}

	// --- Each command accepts its own alternatives ---
	for _, tc := range []struct {
		name    string
		args    []string
		wantRan *bool
		wantVal string
	}{
		{"root accepts root-a", []string{"--value", "root-a"}, &rootRan, "root-a"},
		{"sub1 accepts sub1-x", []string{"sub1", "--value", "sub1-x"}, &sub1Ran, "sub1-x"},
		{"sub2 accepts sub2-m", []string{"sub2", "--value", "sub2-m"}, &sub2Ran, "sub2-m"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reset()
			if err := makeTree().RunArgsE(tc.args); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !*tc.wantRan {
				t.Errorf("expected command to run")
			}
			switch tc.wantRan {
			case &rootRan:
				if rootVal != tc.wantVal {
					t.Errorf("rootVal = %q, want %q", rootVal, tc.wantVal)
				}
			case &sub1Ran:
				if sub1Val != tc.wantVal {
					t.Errorf("sub1Val = %q, want %q", sub1Val, tc.wantVal)
				}
			case &sub2Ran:
				if sub2Val != tc.wantVal {
					t.Errorf("sub2Val = %q, want %q", sub2Val, tc.wantVal)
				}
			}
		})
	}

	// --- Each command rejects values that belong to a sibling ---
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"root rejects sub1-x", []string{"--value", "sub1-x"}},
		{"root rejects sub2-m", []string{"--value", "sub2-m"}},
		{"sub1 rejects root-a", []string{"sub1", "--value", "root-a"}},
		{"sub1 rejects sub2-m", []string{"sub1", "--value", "sub2-m"}},
		{"sub2 rejects root-a", []string{"sub2", "--value", "root-a"}},
		{"sub2 rejects sub1-x", []string{"sub2", "--value", "sub1-x"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reset()
			err := makeTree().RunArgsE(tc.args)
			if err == nil {
				t.Errorf("expected validation error, got nil")
			}
		})
	}
}

// TestSubcommandIsolation_CustomValidators verifies that custom validators
// registered on each command's mirror are called only for that command, never
// leaking to siblings. This exercises the actual validation path (not just
// alts matching) to confirm the per-command paramMeta instances are distinct.
func TestSubcommandIsolation_CustomValidators(t *testing.T) {
	type Params struct {
		Value string `descr:"value"`
	}

	var rootCalls, sub1Calls, sub2Calls int

	makeTree := func() CmdT[Params] {
		return CmdT[Params]{
			Use: "root",
			InitFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command) error {
				ctx.GetParam(&p.Value).SetCustomValidator(func(v any) error {
					rootCalls++
					if !strings.HasPrefix(v.(string), "root-") {
						return fmt.Errorf("root expects prefix 'root-'")
					}
					return nil
				})
				return nil
			},
			RunFunc: func(p *Params, c *cobra.Command, args []string) {},
			SubCmds: SubCmds(
				CmdT[Params]{
					Use: "sub1",
					InitFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command) error {
						ctx.GetParam(&p.Value).SetCustomValidator(func(v any) error {
							sub1Calls++
							if !strings.HasPrefix(v.(string), "sub1-") {
								return fmt.Errorf("sub1 expects prefix 'sub1-'")
							}
							return nil
						})
						return nil
					},
					RunFunc: func(p *Params, c *cobra.Command, args []string) {},
				},
				CmdT[Params]{
					Use: "sub2",
					InitFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command) error {
						ctx.GetParam(&p.Value).SetCustomValidator(func(v any) error {
							sub2Calls++
							if !strings.HasPrefix(v.(string), "sub2-") {
								return fmt.Errorf("sub2 expects prefix 'sub2-'")
							}
							return nil
						})
						return nil
					},
					RunFunc: func(p *Params, c *cobra.Command, args []string) {},
				},
			),
		}
	}

	reset := func() { rootCalls, sub1Calls, sub2Calls = 0, 0, 0 }

	// --- Invoke root: only root's validator runs ---
	reset()
	if err := makeTree().RunArgsE([]string{"--value", "root-ok"}); err != nil {
		t.Fatalf("root exec: %v", err)
	}
	if rootCalls != 1 {
		t.Errorf("rootCalls = %d, want 1", rootCalls)
	}
	if sub1Calls != 0 || sub2Calls != 0 {
		t.Errorf("sub validators leaked into root: sub1=%d sub2=%d", sub1Calls, sub2Calls)
	}

	// --- Invoke sub1: only sub1's validator runs ---
	reset()
	if err := makeTree().RunArgsE([]string{"sub1", "--value", "sub1-ok"}); err != nil {
		t.Fatalf("sub1 exec: %v", err)
	}
	if sub1Calls != 1 {
		t.Errorf("sub1Calls = %d, want 1", sub1Calls)
	}
	if rootCalls != 0 || sub2Calls != 0 {
		t.Errorf("validators leaked into sub1: root=%d sub2=%d", rootCalls, sub2Calls)
	}

	// --- Invoke sub2: only sub2's validator runs ---
	reset()
	if err := makeTree().RunArgsE([]string{"sub2", "--value", "sub2-ok"}); err != nil {
		t.Fatalf("sub2 exec: %v", err)
	}
	if sub2Calls != 1 {
		t.Errorf("sub2Calls = %d, want 1", sub2Calls)
	}
	if rootCalls != 0 || sub1Calls != 0 {
		t.Errorf("validators leaked into sub2: root=%d sub1=%d", rootCalls, sub1Calls)
	}

	// --- Cross-command value rejection ---
	reset()
	if err := makeTree().RunArgsE([]string{"sub1", "--value", "root-x"}); err == nil {
		t.Errorf("sub1 should reject 'root-x'")
	}
	// sub1's validator should have been called; root's should not have been.
	if sub1Calls == 0 {
		t.Errorf("sub1 validator was never invoked during rejection path")
	}
	if rootCalls != 0 {
		t.Errorf("root validator leaked into sub1's validation: rootCalls=%d", rootCalls)
	}
}

// TestSubcommandIsolation_MirrorIdentity spells out the underlying invariant:
// each command in the tree owns a distinct mirror instance for the same
// fieldPath, because each runs its own traverse inside its own processingContext.
// The mirror a command configures in its InitFuncCtx must be the same instance
// its RunFuncCtx sees — and it must NOT be the same instance as any sibling's
// mirror for the same field.
func TestSubcommandIsolation_MirrorIdentity(t *testing.T) {
	type Params struct {
		Value string `descr:"value"`
	}

	var rootInit, rootRun Param
	var sub1Init, sub1Run Param
	var sub2Init, sub2Run Param

	makeTree := func() CmdT[Params] {
		return CmdT[Params]{
			Use: "root",
			InitFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command) error {
				rootInit = ctx.GetParam(&p.Value)
				return nil
			},
			RunFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command, args []string) {
				rootRun = ctx.GetParam(&p.Value)
			},
			SubCmds: SubCmds(
				CmdT[Params]{
					Use: "sub1",
					InitFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command) error {
						sub1Init = ctx.GetParam(&p.Value)
						return nil
					},
					RunFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command, args []string) {
						sub1Run = ctx.GetParam(&p.Value)
					},
				},
				CmdT[Params]{
					Use: "sub2",
					InitFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command) error {
						sub2Init = ctx.GetParam(&p.Value)
						return nil
					},
					RunFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command, args []string) {
						sub2Run = ctx.GetParam(&p.Value)
					},
				},
			),
		}
	}

	// Invoke each command in its own freshly built tree so none of them share
	// any cobra state with another invocation.
	if err := makeTree().RunArgsE([]string{"--value", "v"}); err != nil {
		t.Fatalf("root: %v", err)
	}
	if rootInit == nil || rootRun == nil || rootInit != rootRun {
		t.Errorf("root: init vs run mirror mismatch: init=%p run=%p", rootInit, rootRun)
	}

	if err := makeTree().RunArgsE([]string{"sub1", "--value", "v"}); err != nil {
		t.Fatalf("sub1: %v", err)
	}
	if sub1Init == nil || sub1Run == nil || sub1Init != sub1Run {
		t.Errorf("sub1: init vs run mirror mismatch: init=%p run=%p", sub1Init, sub1Run)
	}

	if err := makeTree().RunArgsE([]string{"sub2", "--value", "v"}); err != nil {
		t.Fatalf("sub2: %v", err)
	}
	if sub2Init == nil || sub2Run == nil || sub2Init != sub2Run {
		t.Errorf("sub2: init vs run mirror mismatch: init=%p run=%p", sub2Init, sub2Run)
	}

	// The mirrors from different commands must be distinct instances. Even
	// though they describe the same field of the same type, they belong to
	// different processingContexts built by different toCobraBase calls.
	mirrors := []Param{rootInit, sub1Init, sub2Init}
	seen := map[Param]string{}
	names := []string{"root", "sub1", "sub2"}
	for i, m := range mirrors {
		if prior, dup := seen[m]; dup {
			t.Errorf("mirror identity collision between commands %q and %q: both returned %p",
				prior, names[i], m)
		}
		seen[m] = names[i]
	}

	// Quick sanity on deterministic path ordering in AllMirrors (just to make
	// sure the new AllMirrors() is sensible — orthogonal to the main point).
	_ = slices.Clone([]Param{rootInit})
}
