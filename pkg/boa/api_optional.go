// Package boa provides a declarative CLI and environment variable parameter utility.
package boa

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"reflect"
)

// Optional represents a parameter that may or may not have a value.
// It can be set via command line flags, environment variables, default values,
// or programmatic injection. Unlike Required parameters, Optional parameters
// don't cause validation errors when not set.
//
// The type parameter T must be one of the types supported by SupportedTypes.
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

// SetAlternatives sets the list of allowed values for this parameter.
func (f *Optional[T]) SetAlternatives(strings []string) {
	f.Alternatives = strings
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
	f.Default = val.(*T)
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
			return f.valuePtr.(*T)
		} else {
			if f.hasDefaultValue() {
				return f.Default
			} else {
				panic(fmt.Errorf("tried to access flag.Value() of '%s', which was not set. This is a bug in util_cobra", f.GetName()))
			}
		}
	} else {
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
		panic("flag has no default value")
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

func (p Optional[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Value())
}

func (p *Optional[T]) UnmarshalJSON(data []byte) error {
	var v *T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	p.valuePtr = v
	p.injected = v != nil
	return nil
}

// Opt creates an Optional parameter with a default value.
// This is a convenience factory function for creating Optional parameters.
func Opt[T SupportedTypes](defaultValue T) Optional[T] {
	return Optional[T]{
		Default: &defaultValue,
	}
}

// OptP creates an Optional parameter with a default value from a pointer.
// This allows passing nil as a default value or reusing an existing pointer.
func OptP[T SupportedTypes](defaultValue *T) Optional[T] {
	return Optional[T]{
		Default:  defaultValue,
		injected: true,
	}
}
