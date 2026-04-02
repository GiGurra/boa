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

		// Set Mode to "file" but don't provide FilePath - should fail
		err := CmdT[Params]{
			Use: "test",
			InitFuncCtx: func(ctx *HookContext, p *Params, _ *cobra.Command) error {
				// FilePath is required when Mode is "file"
				filePathParam := ctx.GetParam(&p.FilePath)
				filePathParam.SetRequiredFn(func() bool {
					return p.Mode == "file"
				})
				return nil
			},
			RunFunc: func(_ *Params, _ *cobra.Command, _ []string) {},
			RawArgs: []string{"--mode", "file"},
		}.Validate()
		if err == nil {
			t.Error("expected validation error for missing required FilePath when Mode=file")
		}
	})

	t.Run("conditionally required - provided when required", func(t *testing.T) {
		type Params struct {
			Mode     string
			FilePath string `optional:"true"` // required when Mode == "file"
		}

		// Set Mode to "file" and provide FilePath - should pass
		err := CmdT[Params]{
			Use: "test",
			InitFuncCtx: func(ctx *HookContext, p *Params, _ *cobra.Command) error {
				filePathParam := ctx.GetParam(&p.FilePath)
				filePathParam.SetRequiredFn(func() bool {
					return p.Mode == "file"
				})
				return nil
			},
			RunFunc: func(_ *Params, _ *cobra.Command, _ []string) {},
			RawArgs: []string{"--mode", "file", "--file-path", "/path/to/file"},
		}.Validate()
		if err != nil {
			t.Errorf("unexpected validation error: %v", err)
		}
	})

	t.Run("conditionally required - not required when condition false", func(t *testing.T) {
		type Params struct {
			Mode     string
			FilePath string `optional:"true"` // required when Mode == "file"
		}

		// Set Mode to "stream" - FilePath should not be required
		err := CmdT[Params]{
			Use: "test",
			InitFuncCtx: func(ctx *HookContext, p *Params, _ *cobra.Command) error {
				filePathParam := ctx.GetParam(&p.FilePath)
				filePathParam.SetRequiredFn(func() bool {
					return p.Mode == "file"
				})
				return nil
			},
			RunFunc: func(_ *Params, _ *cobra.Command, _ []string) {},
			RawArgs: []string{"--mode", "stream"},
		}.Validate()
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

		// When Debug is false, Verbose is disabled - should pass even without verbose
		err := CmdT[Params]{
			Use: "test",
			InitFuncCtx: func(ctx *HookContext, p *Params, _ *cobra.Command) error {
				verboseParam := ctx.GetParam(&p.Verbose)
				verboseParam.SetIsEnabledFn(func() bool {
					return p.Debug
				})
				return nil
			},
			RunFunc: func(_ *Params, _ *cobra.Command, _ []string) {},
			RawArgs: []string{},
		}.Validate()
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

		// When Debug is true, Verbose is enabled and can be set
		CmdT[Params]{
			Use: "test",
			InitFuncCtx: func(ctx *HookContext, p *Params, _ *cobra.Command) error {
				verboseParam := ctx.GetParam(&p.Verbose)
				verboseParam.SetIsEnabledFn(func() bool {
					return p.Debug
				})
				return nil
			},
			RunFunc: func(p *Params, _ *cobra.Command, _ []string) {
				capturedVerbose = p.Verbose
			},
			RawArgs: []string{"--debug", "--verbose"},
		}.Run()
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

	CmdT[Params]{
		Use: "test",
		InitFuncCtx: func(ctx *HookContext, p *Params, _ *cobra.Command) error {
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
		},
		RunFunc: func(_ *Params, _ *cobra.Command, _ []string) {},
		RawArgs: []string{"--name", "test"},
	}.Validate() //nolint:errcheck // test only cares about side effects
}

// TestRawParamMixedWithWrapped tests that GetParam works for both raw fields
func TestRawParamMixedWithWrapped(t *testing.T) {
	type Params struct {
		RawName string
		Port    int `optional:"true"`
	}

	CmdT[Params]{
		Use: "test",
		InitFuncCtx: func(ctx *HookContext, p *Params, _ *cobra.Command) error {
			// GetParam should work for raw fields
			rawParam := ctx.GetParam(&p.RawName)
			if rawParam == nil {
				t.Error("expected GetParam to return non-nil for raw field")
			}
			rawParam.SetDefault(Default("default-name"))

			// GetParam should also work for other raw fields
			portParam := ctx.GetParam(&p.Port)
			if portParam == nil {
				t.Error("expected GetParam to return non-nil for port field")
			}
			portParam.SetDefault(Default(8080))

			return nil
		},
		RunFunc: func(_ *Params, _ *cobra.Command, _ []string) {},
		RawArgs: []string{},
	}.Validate() //nolint:errcheck // test only cares about side effects
}

// TestRawParamConditionalWithDefault tests that conditional params with defaults work correctly
// Note: The conditionally required field must be optional:"true" for SetRequiredFn to work
func TestRawParamConditionalWithDefault(t *testing.T) {
	type Params struct {
		Mode   string `default:"auto"`
		Target string `optional:"true"` // required when Mode != "auto"
	}

	// With default Mode="auto", Target should not be required
	err := CmdT[Params]{
		Use: "test",
		InitFuncCtx: func(ctx *HookContext, p *Params, _ *cobra.Command) error {
			targetParam := ctx.GetParam(&p.Target)
			targetParam.SetRequiredFn(func() bool {
				return p.Mode != "auto"
			})
			return nil
		},
		RunFunc: func(_ *Params, _ *cobra.Command, _ []string) {},
		RawArgs: []string{},
	}.Validate()
	if err != nil {
		t.Errorf("unexpected validation error with default Mode: %v", err)
	}

	// Override Mode to something else, Target becomes required
	err = CmdT[Params]{
		Use: "test2",
		InitFuncCtx: func(ctx *HookContext, p *Params, _ *cobra.Command) error {
			targetParam := ctx.GetParam(&p.Target)
			targetParam.SetRequiredFn(func() bool {
				return p.Mode != "auto"
			})
			return nil
		},
		RunFunc: func(_ *Params, _ *cobra.Command, _ []string) {},
		RawArgs: []string{"--mode", "manual"},
	}.Validate()
	if err == nil {
		t.Error("expected validation error when Mode=manual and Target is missing")
	}
}
