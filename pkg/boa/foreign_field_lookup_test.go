package boa

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// The tests in this file verify that HookContext.GetParam fails loudly *via
// logging* (but still returns nil) when called with a field pointer that does
// not belong to the parameters struct associated with the context.
//
// Silent nil was the prior behavior. That led to delayed "invalid memory
// address or nil pointer dereference" crashes at the caller's chained method
// call, with no indication that the root cause was an upstream mis-lookup.
//
// The current behavior is: return nil, but emit a descriptive slog.Error so
// the cause is visible in logs. Callers that want "probe without crashing"
// semantics can still check for nil explicitly; callers that chain methods
// will still crash, but the preceding log line carries the real diagnostic.

// captureSlog redirects the default slog logger into an in-memory buffer for
// the duration of fn, then restores it. Returns the captured output.
func captureSlog(fn func()) string {
	var buf bytes.Buffer
	orig := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})))
	defer slog.SetDefault(orig)
	fn()
	return buf.String()
}

// assertBoaError checks that captured output contains the expected substrings.
func assertBoaError(t *testing.T, captured string, wantSubstrings ...string) {
	t.Helper()
	if !strings.Contains(captured, "boa.HookContext.GetParam") {
		t.Errorf("slog output does not carry boa prefix: %q", captured)
	}
	for _, w := range wantSubstrings {
		if !strings.Contains(captured, w) {
			t.Errorf("slog output missing expected substring %q: %q", w, captured)
		}
	}
}

// TestGetParam_ForeignFieldLogsError: pass a pointer to a field in an
// unrelated struct. GetParam must return nil AND emit a descriptive slog.Error
// naming the likely causes.
func TestGetParam_ForeignFieldLogsError(t *testing.T) {
	type Params struct {
		Name string `descr:"name" default:"default-name"`
	}
	type Other struct {
		Foreign string
	}
	other := &Other{Foreign: "unrelated"}

	var gotMirror Param

	captured := captureSlog(func() {
		cmd := (CmdT[Params]{
			Use: "test",
			InitFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command) error {
				gotMirror = ctx.GetParam(&other.Foreign)
				return nil
			},
			RunFunc: func(p *Params, c *cobra.Command, args []string) {},
		}).ToCobra()

		cmd.SetArgs([]string{})
		if err := Execute(cmd); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if gotMirror != nil {
		t.Errorf("expected nil mirror for foreign field, got %p", gotMirror)
	}
	assertBoaError(t, captured, "does not belong to the parameters struct")
}

// TestGetParam_SameTypeDifferentInstanceLogsError: pass a pointer to a field
// in a separately-allocated instance of the SAME Params type. Must return nil
// and log the same descriptive error.
func TestGetParam_SameTypeDifferentInstanceLogsError(t *testing.T) {
	type Params struct {
		Name string `descr:"name" default:"default-name"`
	}
	stranger := &Params{Name: "stranger"}

	var gotMirror Param

	captured := captureSlog(func() {
		cmd := (CmdT[Params]{
			Use: "test",
			InitFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command) error {
				gotMirror = ctx.GetParam(&stranger.Name)
				return nil
			},
			RunFunc: func(p *Params, c *cobra.Command, args []string) {},
		}).ToCobra()

		cmd.SetArgs([]string{})
		if err := Execute(cmd); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if gotMirror != nil {
		t.Errorf("expected nil mirror for foreign-instance-same-type lookup, got %p", gotMirror)
	}
	assertBoaError(t, captured, "does not belong to the parameters struct")
}

// TestGetParam_NilFieldPointerLogsError: passing untyped nil must return nil
// and produce a descriptive log line, not a generic reflect crash.
func TestGetParam_NilFieldPointerLogsError(t *testing.T) {
	type Params struct {
		Name string `descr:"name" default:"default-name"`
	}

	var gotMirror Param
	gotMirror = nil // keep compiler happy when set inside closure

	captured := captureSlog(func() {
		cmd := (CmdT[Params]{
			Use: "test",
			InitFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command) error {
				gotMirror = ctx.GetParam(nil)
				return nil
			},
			RunFunc: func(p *Params, c *cobra.Command, args []string) {},
		}).ToCobra()

		cmd.SetArgs([]string{})
		if err := Execute(cmd); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if gotMirror != nil {
		t.Errorf("expected nil mirror for nil fieldPtr, got %p", gotMirror)
	}
	assertBoaError(t, captured, "fieldPtr is nil")
}

// TestGetParam_NonPointerLogsError: passing a non-pointer value must return nil
// with a descriptive message rather than crashing inside reflect.
func TestGetParam_NonPointerLogsError(t *testing.T) {
	type Params struct {
		Name string `descr:"name" default:"default-name"`
	}

	var gotMirror Param

	captured := captureSlog(func() {
		cmd := (CmdT[Params]{
			Use: "test",
			InitFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command) error {
				gotMirror = ctx.GetParam("not-a-pointer")
				return nil
			},
			RunFunc: func(p *Params, c *cobra.Command, args []string) {},
		}).ToCobra()

		cmd.SetArgs([]string{})
		if err := Execute(cmd); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if gotMirror != nil {
		t.Errorf("expected nil mirror for non-pointer fieldPtr, got %p", gotMirror)
	}
	assertBoaError(t, captured, "must be a pointer")
}

// TestGetParam_RegisteredFieldStillWorks is a paired sanity check: the happy
// path still returns a non-nil mirror and the log stream should be empty.
func TestGetParam_RegisteredFieldStillWorks(t *testing.T) {
	type Params struct {
		Name string `descr:"name" default:"default-name"`
	}

	var gotMirror Param

	captured := captureSlog(func() {
		cmd := (CmdT[Params]{
			Use: "test",
			InitFuncCtx: func(ctx *HookContext, p *Params, c *cobra.Command) error {
				gotMirror = ctx.GetParam(&p.Name)
				return nil
			},
			RunFunc: func(p *Params, c *cobra.Command, args []string) {},
		}).ToCobra()

		cmd.SetArgs([]string{})
		if err := Execute(cmd); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if gotMirror == nil {
		t.Errorf("registered field lookup returned nil")
	}
	if strings.Contains(captured, "boa.HookContext.GetParam") {
		t.Errorf("unexpected boa GetParam error in log for happy path: %q", captured)
	}
}
