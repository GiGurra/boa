// Package boa provides a declarative CLI and environment variable parameter utility.
package boa

import (
	"log/slog"
	"reflect"

	"github.com/spf13/cobra"
)

// ParamT is a typed view over a parameter's configuration.
// It wraps the internal Param mirror and provides type-safe methods
// for configuring the parameter.
//
// Usage:
//
//	cmd := boa.NewCmdT[Params]("cmd").
//	    WithInitFuncCtx(func(ctx *boa.HookContext, params *Params, cmd *cobra.Command) error {
//	        // Get typed parameter view
//	        nameParam := boa.GetParamT(ctx, &params.Name)
//	        nameParam.SetDefaultT("default-value")
//	        nameParam.SetCustomValidatorT(func(val string) error {
//	            if len(val) < 3 {
//	                return fmt.Errorf("name must be at least 3 characters")
//	            }
//	            return nil
//	        })
//	        return nil
//	    })
type ParamT[T SupportedTypes] interface {
	// Param returns the underlying untyped Param interface.
	Param() Param

	// --- Typed methods ---

	// SetDefaultT sets the default value for this parameter with type safety.
	SetDefaultT(val T)

	// SetCustomValidatorT sets a typed custom validation function for this parameter.
	// The function receives the actual typed value instead of `any`.
	SetCustomValidatorT(fn func(T) error)

	// --- Pass-through methods (convenience wrappers) ---

	// SetAlternatives sets the list of allowed values for this parameter.
	SetAlternatives(alts []string)

	// SetStrictAlts sets whether alternatives are strictly enforced (validated) or just suggestions.
	SetStrictAlts(strict bool)

	// SetAlternativesFunc sets a function that provides dynamic value suggestions for bash completion.
	SetAlternativesFunc(fn func(cmd *cobra.Command, args []string, toComplete string) []string)

	// SetEnv sets the environment variable name for this parameter.
	SetEnv(env string)

	// SetShort sets the short flag name (single character) for this parameter.
	SetShort(short string)

	// SetName sets the flag name for this parameter.
	SetName(name string)

	// SetIsEnabledFn sets a function that determines if this parameter is enabled.
	SetIsEnabledFn(fn func() bool)

	// SetRequiredFn sets a function that determines if this parameter is required.
	// This allows making optional parameters conditionally required.
	SetRequiredFn(fn func() bool)
}

// GetParamT returns a typed ParamT[T] view for the given field pointer.
// It wraps the parameter's internal mirror to provide type-safe configuration.
//
// Usage:
//
//	type Params struct {
//	    Name string `descr:"User name"`
//	    Port int    `descr:"Port number"`
//	}
//	cmd := boa.NewCmdT[Params]("cmd").
//	    WithInitFuncCtx(func(ctx *boa.HookContext, params *Params, cmd *cobra.Command) error {
//	        nameParam := boa.GetParamT(ctx, &params.Name)
//	        nameParam.SetDefaultT("default-name")
//	        nameParam.SetCustomValidatorT(func(val string) error {
//	            if len(val) < 3 {
//	                return fmt.Errorf("name must be at least 3 characters")
//	            }
//	            return nil
//	        })
//
//	        portParam := boa.GetParamT(ctx, &params.Port)
//	        portParam.SetCustomValidatorT(func(port int) error {
//	            if port < 1 || port > 65535 {
//	                return fmt.Errorf("port must be between 1 and 65535")
//	            }
//	            return nil
//	        })
//	        return nil
//	    })
func GetParamT[T SupportedTypes](ctx *HookContext, fieldPtr *T) ParamT[T] {
	param := ctx.GetParam(fieldPtr)
	if param == nil {
		slog.Error("GetParamT: could not find param for field pointer", "fieldPtr", fieldPtr)
		return nil
	}

	return &ParamTView[T]{
		param: param,
	}
}

// ParamTView is a typed view over a parameter's configuration.
// It wraps an untyped Param and provides type-safe methods for configuration.
// Use GetParamT to obtain an instance.
type ParamTView[T SupportedTypes] struct {
	param Param
}

// Param returns the underlying untyped Param interface.
func (w *ParamTView[T]) Param() Param {
	return w.param
}

// SetDefaultT sets the default value with type safety.
func (w *ParamTView[T]) SetDefaultT(val T) {
	w.param.SetDefault(&val)
}

// SetCustomValidatorT sets a typed validation function.
func (w *ParamTView[T]) SetCustomValidatorT(fn func(T) error) {
	if fn == nil {
		w.param.SetCustomValidator(nil)
		return
	}
	w.param.SetCustomValidator(func(val any) error {
		// Handle both pointer and non-pointer values
		switch v := val.(type) {
		case T:
			return fn(v)
		case *T:
			if v != nil {
				return fn(*v)
			}
			var zero T
			return fn(zero)
		default:
			// Try reflection-based conversion for type aliases
			valReflect := reflect.ValueOf(val)
			if valReflect.Kind() == reflect.Ptr && !valReflect.IsNil() {
				valReflect = valReflect.Elem()
			}
			var zero T
			targetType := reflect.TypeOf(zero)
			if valReflect.Type().ConvertibleTo(targetType) {
				converted := valReflect.Convert(targetType).Interface().(T)
				return fn(converted)
			}
			// Fallback - this shouldn't happen in normal usage
			slog.Warn("SetCustomValidatorT: unexpected value type", "expected", targetType, "got", reflect.TypeOf(val))
			return fn(val.(T))
		}
	})
}

// SetAlternatives sets the list of allowed values for this parameter.
func (w *ParamTView[T]) SetAlternatives(alts []string) {
	w.param.SetAlternatives(alts)
}

// SetStrictAlts sets whether alternatives are strictly enforced.
func (w *ParamTView[T]) SetStrictAlts(strict bool) {
	w.param.SetStrictAlts(strict)
}

// SetAlternativesFunc sets a function that provides dynamic value suggestions for bash completion.
func (w *ParamTView[T]) SetAlternativesFunc(fn func(cmd *cobra.Command, args []string, toComplete string) []string) {
	w.param.SetAlternativesFunc(fn)
}

// SetEnv sets the environment variable name for this parameter.
func (w *ParamTView[T]) SetEnv(env string) {
	w.param.SetEnv(env)
}

// SetShort sets the short flag name (single character) for this parameter.
func (w *ParamTView[T]) SetShort(short string) {
	w.param.SetShort(short)
}

// SetName sets the flag name for this parameter.
func (w *ParamTView[T]) SetName(name string) {
	w.param.SetName(name)
}

// SetIsEnabledFn sets a function that determines if this parameter is enabled.
func (w *ParamTView[T]) SetIsEnabledFn(fn func() bool) {
	w.param.SetIsEnabledFn(fn)
}

// SetRequiredFn sets a function that determines if this parameter is required.
func (w *ParamTView[T]) SetRequiredFn(fn func() bool) {
	w.param.SetRequiredFn(fn)
}
