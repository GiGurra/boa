// Package boa provides a declarative CLI and environment variable parameter utility.
package boa

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"log/slog"
	"reflect"
)

// Required represents a parameter that must have a value.
// If a Required parameter is not set via command line, environment variable,
// default value, or programmatic injection, it will cause a validation error.
//
// The type parameter T must be one of the types supported by SupportedTypes.
type Required[T SupportedTypes] struct {
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
}

// IsEnabled always returns true for Required parameters.
// Required parameters cannot be disabled.
func (f *Required[T]) IsEnabled() bool {
	return true
}

// GetIsEnabledFn returns nil for Required parameters.
// Required parameters cannot be disabled.
func (f *Required[T]) GetIsEnabledFn() func() bool {
	return nil
}

// SetAlternatives sets the list of allowed values for this parameter.
func (f *Required[T]) SetAlternatives(strings []string) {
	f.Alternatives = strings
}

// This assertion proves that Required[T] implements the Param interface.
var _ Param = &Required[string]{}

func (f *Required[T]) wasSetPositionally() bool {
	return f.setPositionally
}

func (f *Required[T]) markSetPositionally() {
	f.setPositionally = true
}

func (f *Required[T]) isPositional() bool {
	return f.Positional
}

// SetDefault Only to be used from ParamEnrichers. Use the regular parameters to set the default with type safety otherwise.
func (f *Required[T]) SetDefault(val any) {
	f.Default = val.(*T)
}

func (f *Required[T]) SetEnv(val string) {
	f.Env = val
}

func (f *Required[T]) SetShort(val string) {
	f.Short = val
}

func (f *Required[T]) SetName(val string) {
	f.Name = val
}

func (f *Required[T]) wasSetByEnv() bool {
	return f.setByEnv
}

func (f *Required[T]) markSetFromEnv() {
	f.setByEnv = true
}

// Value returns the parameter value.
// Unlike Optional parameters, this returns the actual value, not a pointer.
// This will panic if the parameter doesn't have a value, which shouldn't happen
// with proper validation as Required parameters must have a value.
func (f *Required[T]) Value() T {
	if HasValue(f) {
		if f.valuePtr != nil {
			return *f.valuePtr.(*T)
		} else {
			return *f.Default
		}
	} else {
		slog.Warn(fmt.Sprintf("tried to access Optional[..].Value() of '%s', which was not set.", f.GetName()))
		var zero T
		return zero
	}
}

func (f *Required[T]) setDescription(state string) {
	f.Descr = state
}

func (f *Required[T]) setPositional(state bool) {
	f.Positional = state
}

func (f *Required[T]) customValidatorOfPtr() func(any) error {
	return func(val any) error {
		if f.CustomValidator == nil {
			return nil
		}
		return f.CustomValidator(*val.(*T))
	}
}

func (f *Required[T]) wasSetOnCli() bool {
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

func (f *Required[T]) GetShort() string {
	return f.Short
}

func (f *Required[T]) GetName() string {
	return f.Name
}

func (f *Required[T]) GetEnv() string {
	return f.Env
}

func (f *Required[T]) defaultValuePtr() any {
	return f.Default
}

func (f *Required[T]) hasDefaultValue() bool {
	return f.Default != nil
}

func (f *Required[T]) descr() string {
	return f.Descr
}

// IsRequired always returns true for Required parameters.
// This is the fundamental difference between Required and Optional parameters.
func (f *Required[T]) IsRequired() bool {
	return true
}

// valuePtrF returns the value pointer or default value pointer.
// Internal method used by boa.
func (f *Required[T]) valuePtrF() any {
	if f.valuePtr != nil {
		return f.valuePtr
	} else {
		return f.Default
	}
}

func (f *Required[T]) wasSetByInject() bool {
	return f.injected && f.valuePtr != nil
}

func (f *Required[T]) parentCmd() *cobra.Command {
	return f.parent
}

func (f *Required[T]) defaultValueStr() string {
	if !f.hasDefaultValue() {
		panic("flag has no default value")
	}
	return fmt.Sprintf("%v", *f.Default)
}

// HasValue returns whether this parameter has a value from any source.
// It checks if the parameter was set via command line, environment variable,
// default value, or programmatic injection.
func (f *Required[T]) HasValue() bool {
	return HasValue(f)
}

func (f *Required[T]) GetKind() reflect.Kind {
	return f.GetType().Kind()
}

func (f *Required[T]) GetType() reflect.Type {
	var zero T
	return reflect.TypeOf(zero)
}

func (f *Required[T]) setParentCmd(cmd *cobra.Command) {
	f.parent = cmd
}

func (f *Required[T]) setValuePtr(val any) {
	f.valuePtr = val
}

// GetAlternatives returns the list of allowed values for this parameter.
// Used for command line completion and validation.
func (f *Required[T]) GetAlternatives() []string {
	return f.Alternatives
}

// GetAlternativesFunc returns the function that provides dynamic value
// suggestions for bash completion.
func (f *Required[T]) GetAlternativesFunc() func(cmd *cobra.Command, args []string, toComplete string) []string {
	return f.AlternativesFunc
}

func (p Required[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Value())
}

func (p *Required[T]) UnmarshalJSON(data []byte) error {
	if !p.wasSetOnCli() && !p.wasSetByEnv() {
		var v *T
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		p.valuePtr = v
		p.injected = v != nil
	}
	return nil
}

// Req creates a Required parameter with a default value.
// This is a convenience factory function for creating Required parameters.
// Even though the parameter is required, providing a default value ensures
// it always has a value, preventing validation errors.
func Req[T SupportedTypes](defaultValue T) Required[T] {
	return Required[T]{
		Default:  &defaultValue,
		injected: true,
	}
}
