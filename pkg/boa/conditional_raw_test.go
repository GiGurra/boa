package boa

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestRawParamSetRequiredFn tests that SetRequiredFn works for raw parameters via HookContext
// Note: To use SetRequiredFn on raw params, they must be marked as optional (optional:"true")
// since SetRequiredFn is only meaningful for Optional mirrors (Required mirrors are always required)
func TestRawParamSetRequiredFn(t *testing.T) {
	t.Run("conditionally required - missing when required", func(t *testing.T) {
		type Params struct {
			Mode     string
			FilePath string `optional:"true"` // required when Mode == "file"
		}

		cmd := NewCmdT[Params]("test").
			WithInitFuncCtx(func(ctx *HookContext, p *Params, _ *cobra.Command) error {
				// FilePath is required when Mode is "file"
				filePathParam := ctx.GetParam(&p.FilePath)
				filePathParam.SetRequiredFn(func() bool {
					return p.Mode == "file"
				})
				return nil
			}).
			WithRunFunc(func(_ *Params) {})

		// Set Mode to "file" but don't provide FilePath - should fail
		err := cmd.WithRawArgs([]string{"--mode", "file"}).Validate()
		if err == nil {
			t.Error("expected validation error for missing required FilePath when Mode=file")
		}
	})

	t.Run("conditionally required - provided when required", func(t *testing.T) {
		type Params struct {
			Mode     string
			FilePath string `optional:"true"` // required when Mode == "file"
		}

		cmd := NewCmdT[Params]("test").
			WithInitFuncCtx(func(ctx *HookContext, p *Params, _ *cobra.Command) error {
				filePathParam := ctx.GetParam(&p.FilePath)
				filePathParam.SetRequiredFn(func() bool {
					return p.Mode == "file"
				})
				return nil
			}).
			WithRunFunc(func(_ *Params) {})

		// Set Mode to "file" and provide FilePath - should pass
		err := cmd.WithRawArgs([]string{"--mode", "file", "--file-path", "/path/to/file"}).Validate()
		if err != nil {
			t.Errorf("unexpected validation error: %v", err)
		}
	})

	t.Run("conditionally required - not required when condition false", func(t *testing.T) {
		type Params struct {
			Mode     string
			FilePath string `optional:"true"` // required when Mode == "file"
		}

		cmd := NewCmdT[Params]("test").
			WithInitFuncCtx(func(ctx *HookContext, p *Params, _ *cobra.Command) error {
				filePathParam := ctx.GetParam(&p.FilePath)
				filePathParam.SetRequiredFn(func() bool {
					return p.Mode == "file"
				})
				return nil
			}).
			WithRunFunc(func(_ *Params) {})

		// Set Mode to "stream" - FilePath should not be required
		err := cmd.WithRawArgs([]string{"--mode", "stream"}).Validate()
		if err != nil {
			t.Errorf("unexpected validation error when Mode != file: %v", err)
		}
	})
}

// TestRawParamSetIsEnabledFn tests that SetIsEnabledFn works for raw parameters via HookContext
func TestRawParamSetIsEnabledFn(t *testing.T) {
	t.Run("conditionally enabled - disabled flag is hidden", func(t *testing.T) {
		type Params struct {
			Debug   bool
			Verbose bool // only enabled when Debug is true
		}

		cmd := NewCmdT[Params]("test").
			WithInitFuncCtx(func(ctx *HookContext, p *Params, _ *cobra.Command) error {
				verboseParam := ctx.GetParam(&p.Verbose)
				verboseParam.SetIsEnabledFn(func() bool {
					return p.Debug
				})
				return nil
			}).
			WithRunFunc(func(_ *Params) {})

		// When Debug is false, Verbose is disabled - should pass even without verbose
		err := cmd.WithRawArgs([]string{}).Validate()
		if err != nil {
			t.Errorf("unexpected validation error: %v", err)
		}
	})

	t.Run("conditionally enabled - enabled flag works", func(t *testing.T) {
		type Params struct {
			Debug   bool
			Verbose bool `optional:"true"` // only enabled when Debug is true
		}

		var capturedVerbose bool

		cmd := NewCmdT[Params]("test").
			WithInitFuncCtx(func(ctx *HookContext, p *Params, _ *cobra.Command) error {
				verboseParam := ctx.GetParam(&p.Verbose)
				verboseParam.SetIsEnabledFn(func() bool {
					return p.Debug
				})
				return nil
			}).
			WithRunFunc(func(p *Params) {
				capturedVerbose = p.Verbose
			})

		// When Debug is true, Verbose is enabled and can be set
		cmd.WithRawArgs([]string{"--debug", "--verbose"}).Run()
		if !capturedVerbose {
			t.Error("expected Verbose to be true when Debug is true and --verbose is passed")
		}
	})
}

// TestRawParamGetRequiredFn tests that GetRequiredFn returns the function set via SetRequiredFn
// Note: The field must be optional:"true" for SetRequiredFn to work (Required mirrors ignore it)
func TestRawParamGetRequiredFn(t *testing.T) {
	type Params struct {
		Name string `optional:"true"`
	}

	cmd := NewCmdT[Params]("test").
		WithInitFuncCtx(func(ctx *HookContext, p *Params, _ *cobra.Command) error {
			nameParam := ctx.GetParam(&p.Name)

			// Initially, GetRequiredFn should return nil for raw params
			if nameParam.GetRequiredFn() != nil {
				t.Error("expected GetRequiredFn to return nil initially for raw param")
			}

			// Set a required function
			requiredFn := func() bool { return true }
			nameParam.SetRequiredFn(requiredFn)

			// GetRequiredFn should now return a function
			if nameParam.GetRequiredFn() == nil {
				t.Error("expected GetRequiredFn to return non-nil after SetRequiredFn")
			}

			return nil
		}).
		WithRunFunc(func(_ *Params) {})

	_ = cmd.WithRawArgs([]string{"--name", "test"}).Validate()
}

// TestRawParamMixedWithWrapped tests that GetParam works for both raw and wrapped fields
func TestRawParamMixedWithWrapped(t *testing.T) {
	type Params struct {
		RawName     string
		WrappedPort Optional[int] //nolint:staticcheck // testing deprecated type
	}

	cmd := NewCmdT[Params]("test").
		WithInitFuncCtx(func(ctx *HookContext, p *Params, _ *cobra.Command) error {
			// GetParam should work for raw fields
			rawParam := ctx.GetParam(&p.RawName)
			if rawParam == nil {
				t.Error("expected GetParam to return non-nil for raw field")
			}
			rawParam.SetDefault(Default("default-name")) //nolint:staticcheck // testing with deprecated function

			// GetParam should also work for wrapped fields
			wrappedParam := ctx.GetParam(&p.WrappedPort)
			if wrappedParam == nil {
				t.Error("expected GetParam to return non-nil for wrapped field")
			}
			wrappedParam.SetDefault(Default(8080)) //nolint:staticcheck // testing with deprecated function

			return nil
		}).
		WithRunFunc(func(_ *Params) {})

	err := cmd.WithRawArgs([]string{}).Validate()
	if err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

// TestRawParamConditionalWithDefault tests that conditional params with defaults work correctly
// Note: The conditionally required field must be optional:"true" for SetRequiredFn to work
func TestRawParamConditionalWithDefault(t *testing.T) {
	type Params struct {
		Mode   string `default:"auto"`
		Target string `optional:"true"` // required when Mode != "auto"
	}

	cmd := NewCmdT[Params]("test").
		WithInitFuncCtx(func(ctx *HookContext, p *Params, _ *cobra.Command) error {
			targetParam := ctx.GetParam(&p.Target)
			targetParam.SetRequiredFn(func() bool {
				return p.Mode != "auto"
			})
			return nil
		}).
		WithRunFunc(func(_ *Params) {})

	// With default Mode="auto", Target should not be required
	err := cmd.WithRawArgs([]string{}).Validate()
	if err != nil {
		t.Errorf("unexpected validation error with default Mode: %v", err)
	}

	// Override Mode to something else, Target becomes required
	cmd2 := NewCmdT[Params]("test2").
		WithInitFuncCtx(func(ctx *HookContext, p *Params, _ *cobra.Command) error {
			targetParam := ctx.GetParam(&p.Target)
			targetParam.SetRequiredFn(func() bool {
				return p.Mode != "auto"
			})
			return nil
		}).
		WithRunFunc(func(_ *Params) {})

	err = cmd2.WithRawArgs([]string{"--mode", "manual"}).Validate()
	if err == nil {
		t.Error("expected validation error when Mode=manual and Target is missing")
	}
}
