// Package boa provides a declarative CLI and environment variable parameter utility.
package boa

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"log/slog"
	"reflect"
)

// Optional represents a parameter that may or may not have a value.
// It can be set via command line flags, environment variables, default values,
// or programmatic injection. Unlike Required parameters, Optional parameters
// don't cause validation errors when not set.
//
// The type parameter T must be one of the types supported by SupportedTypes.
//
// Deprecated: Use raw Go types with struct tags instead.
// Example: `Name string `descr:"User name" optional:"true"`` instead of `Name Optional[string]`.
// Access values directly (params.Name) instead of params.Name.Value().
// For programmatic configuration, use HookContext.GetParam().
type Optional[T SupportedTypes] struct {
	// Name is the flag name (without the -- prefix)
	Name string
	// Short is the short flag name (single character, without the - prefix)
	Short string
	// Env is the environment variable name that can set this parameter
	Env string
	// Default is the default value pointer for this parameter
	Default *T
	// Descr is the description shown in help text
	Descr string
	// CustomValidator is an optional function to validate the parameter value
	CustomValidator func(T) error
	// Positional indicates if this is a positional argument rather than a flag
	Positional bool

	// Alternatives provides a list of allowed values for this parameter
	Alternatives []string
	// AlternativesFunc provides a dynamic function to generate valid value suggestions for bash completion
	AlternativesFunc func(cmd *cobra.Command, args []string, toComplete string) []string
	// StrictAlts controls whether Alternatives are strictly enforced (validated) or just used as suggestions.
	// When nil or true, values must be in the Alternatives list. When false, any value is accepted.
	StrictAlts *bool

	// Internal state fields
	setByEnv        bool
	setPositionally bool
	injected        bool
	valuePtr        any
	parent          *cobra.Command

	// Dynamic requirement/enablement conditions
	requiredFn func() bool
	enabledFn  func() bool
}

func (f *Optional[T]) GetIsEnabledFn() func() bool {
	return f.enabledFn
}

func (f *Optional[T]) IsEnabled() bool {
	if f.enabledFn != nil {
		return f.enabledFn()
	}
	return true
}

func (f *Optional[T]) SetIsEnabled(b bool) {
	f.enabledFn = func() bool {
		return b
	}
}

func (f *Optional[T]) SetIsEnabledFn(f2 func() bool) {
	f.enabledFn = f2
}

// GetAlternatives returns the list of allowed values for this parameter.
// Used for command line completion and validation.
func (f *Optional[T]) GetAlternatives() []string {
	return f.Alternatives
}

// GetAlternativesFunc returns the function that provides dynamic value
// suggestions for bash completion.
func (f *Optional[T]) GetAlternativesFunc() func(cmd *cobra.Command, args []string, toComplete string) []string {
	return f.AlternativesFunc
}

// SetAlternativesFunc sets the function that provides dynamic value
// suggestions for bash completion.
func (f *Optional[T]) SetAlternativesFunc(fn func(cmd *cobra.Command, args []string, toComplete string) []string) {
	f.AlternativesFunc = fn
}

// SetAlternatives sets the list of allowed values for this parameter.
func (f *Optional[T]) SetAlternatives(strings []string) {
	f.Alternatives = strings
}

// SetStrictAlts sets whether Alternatives are strictly enforced.
func (f *Optional[T]) SetStrictAlts(strict bool) {
	f.StrictAlts = &strict
}

// GetStrictAlts returns whether Alternatives should be strictly enforced.
// Returns true if StrictAlts is nil (default behavior) or explicitly set to true.
func (f *Optional[T]) GetStrictAlts() bool {
	return f.StrictAlts == nil || *f.StrictAlts
}

// prove that Optional[T] implements Param
var _ Param = &Optional[string]{}

// wasSetPositionally returns whether this parameter was set via a positional argument.
// Internal method used by boa.
func (f *Optional[T]) wasSetPositionally() bool {
	return f.setPositionally
}

// GetOrElse returns the parameter value if it exists, otherwise returns the provided fallback value.
// This is a convenience method for handling optional parameters.
func (f *Optional[T]) GetOrElse(fallback T) T {
	if f.HasValue() {
		return *f.Value()
	} else {
		return fallback
	}
}

// GetOrElseF returns the parameter value if it exists, otherwise calls and returns the result of the fallback function.
// This is useful when the fallback value is expensive to compute.
func (f *Optional[T]) GetOrElseF(fallback func() T) T {
	if f.HasValue() {
		return *f.Value()
	} else {
		return fallback()
	}
}

func (f *Optional[T]) markSetPositionally() {
	f.setPositionally = true
}

func (f *Optional[T]) isPositional() bool {
	return f.Positional
}

func (f *Optional[T]) SetDefault(val any) {
	if typedVal, ok := val.(*T); ok {
		f.Default = typedVal
	} else {
		// Handle type aliases by converting via reflection
		var zero T
		targetType := reflect.TypeOf(zero)
		valPtr := reflect.ValueOf(val)
		if valPtr.Kind() == reflect.Ptr && !valPtr.IsNil() {
			valElem := valPtr.Elem()
			if valElem.Type().ConvertibleTo(targetType) {
				converted := valElem.Convert(targetType)
				result := converted.Interface().(T)
				f.Default = &result
				return
			}
		}
		// Fallback to original behavior (will panic with clear message)
		f.Default = val.(*T)
	}
}

func (f *Optional[T]) SetEnv(val string) {
	f.Env = val
}

func (f *Optional[T]) SetShort(val string) {
	f.Short = val
}

func (f *Optional[T]) SetName(val string) {
	f.Name = val
}

func (f *Optional[T]) wasSetByEnv() bool {
	return f.setByEnv
}

func (f *Optional[T]) markSetFromEnv() {
	f.setByEnv = true
}

// Value returns the parameter value as a pointer, or nil if not set.
// This is the primary method to access the parameter value.
func (f *Optional[T]) Value() *T {
	if HasValue(f) {
		if f.valuePtr != nil {
			// Try direct type assertion first
			if val, ok := f.valuePtr.(*T); ok {
				return val
			}
			// If that fails, use reflection to convert from underlying type to custom type
			// This handles cases like: type CustomStringType string
			valPtr := reflect.ValueOf(f.valuePtr)
			if valPtr.Kind() == reflect.Ptr && !valPtr.IsNil() {
				valElem := valPtr.Elem()
				var zero T
				targetType := reflect.TypeOf(zero)
				// Convert the underlying value to the target custom type
				if valElem.Type().ConvertibleTo(targetType) {
					converted := valElem.Convert(targetType)
					result := converted.Interface().(T)
					return &result
				}
			}
			// Fallback to panic with the original error
			return f.valuePtr.(*T)
		} else {
			return f.Default
		}
	} else {
		slog.Warn(fmt.Sprintf("tried to access Optional[..].Value() of '%s', which was not set.", f.GetName()))
		return nil
	}
}

// HasValue returns whether this parameter has a value from any source.
// It checks if the parameter was set via command line, environment variable,
// default value, or programmatic injection.
func (f *Optional[T]) HasValue() bool {
	return HasValue(f)
}

// setPositional sets whether this parameter is a positional argument.
// Internal method used by boa.
func (f *Optional[T]) setPositional(state bool) {
	f.Positional = state
}

// setDescription sets the description text for this parameter.
// Internal method used by boa.
func (f *Optional[T]) setDescription(state string) {
	f.Descr = state
}

func (f *Optional[T]) customValidatorOfPtr() func(any) error {
	return func(val any) error {
		if f.CustomValidator == nil {
			return nil
		}
		return f.CustomValidator(*val.(*T))
	}
}

// SetCustomValidator sets a custom validation function for this parameter.
// The function receives the value as `any` and should type-assert it to T.
// Example: param.SetCustomValidator(func(v any) error { port := v.(int); ... })
func (f *Optional[T]) SetCustomValidator(validator func(any) error) {
	if validator == nil {
		f.CustomValidator = nil
		return
	}
	f.CustomValidator = func(val T) error {
		return validator(val)
	}
}

func (f *Optional[T]) wasSetOnCli() bool {
	if f.Positional {
		return f.wasSetPositionally()
	} else {
		if f.parent == nil {
			return false
		} else {
			return f.parent.Flags().Changed(f.Name)
		}
	}
}

func (f *Optional[T]) wasSetByInject() bool {
	return f.injected && f.valuePtr != nil
}

func (f *Optional[T]) GetShort() string {
	return f.Short
}

func (f *Optional[T]) GetName() string {
	return f.Name
}

func (f *Optional[T]) GetEnv() string {
	return f.Env
}

func (f *Optional[T]) defaultValuePtr() any {
	return f.Default
}

func (f *Optional[T]) hasDefaultValue() bool {
	return f.Default != nil
}

func (f *Optional[T]) descr() string {
	return f.Descr
}

// IsRequired returns whether this optional parameter is currently required.
// While Optional parameters are not required by default, they can be made
// conditionally required based on other parameter values.
func (f *Optional[T]) IsRequired() bool {
	if f.requiredFn != nil {
		return f.requiredFn()
	}
	return false
}

// SetRequiredFn sets a function that dynamically determines whether this
// parameter is required. This allows for conditional requirements based on
// other parameter values.
func (f *Optional[T]) SetRequiredFn(condition func() bool) {
	f.requiredFn = condition
}

// GetRequiredFn returns the function that determines if this parameter is required.
func (f *Optional[T]) GetRequiredFn() func() bool {
	return f.requiredFn
}

func (f *Optional[T]) valuePtrF() any {
	if f.valuePtr != nil {
		return f.valuePtr
	} else {
		return f.Default
	}
}

func (f *Optional[T]) parentCmd() *cobra.Command {
	return f.parent
}

func (f *Optional[T]) defaultValueStr() string {
	if !f.hasDefaultValue() {
		slog.Error(fmt.Sprintf("defaultValueStr called on Optional parameter '%s' without default value", f.Name))
		return ""
	}
	return fmt.Sprintf("%v", *f.Default)
}

func (f *Optional[T]) GetKind() reflect.Kind {
	return f.GetType().Kind()
}

func (f *Optional[T]) GetType() reflect.Type {
	var zero T
	return reflect.TypeOf(zero)
}

func (f *Optional[T]) setParentCmd(cmd *cobra.Command) {
	f.parent = cmd
}

func (f *Optional[T]) setValuePtr(val any) {
	f.valuePtr = val
}

func (f *Optional[T]) injectValuePtr(val any) {
	f.valuePtr = val
	f.injected = val != nil
}

func (p Optional[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Value())
}

func (p *Optional[T]) UnmarshalJSON(data []byte) error {
	if !p.wasSetOnCli() && !p.wasSetByEnv() {
		// CLI always takes precedence over config file
		var v *T
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		p.valuePtr = v
		p.injected = v != nil
	}
	return nil
}

// Opt creates an Optional parameter with a default value.
// This is a convenience factory function for creating Optional parameters.
//
// Deprecated: Use raw Go types with struct tags instead.
// Example: `Name string `descr:"User name" optional:"true" default:"value"`` instead of `Name: boa.Opt("value")`.
func Opt[T SupportedTypes](defaultValue T) Optional[T] {
	return Optional[T]{
		Default: &defaultValue,
	}
}

// OptP creates an Optional parameter with a default value from a pointer.
// This allows passing nil as a default value or reusing an existing pointer.
//
// Deprecated: Use raw Go types with struct tags instead.
// Example: `Name string `descr:"User name" optional:"true"`` instead of `Name: boa.OptP(nil)`.
func OptP[T SupportedTypes](defaultValue *T) Optional[T] {
	return Optional[T]{
		Default:  defaultValue,
		injected: true,
	}
}
