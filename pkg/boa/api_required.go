package boa

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/spf13/cobra"
)

// required represents a parameter that must have a value.
// If a required parameter is not set via command line, environment variable,
// default value, or programmatic injection, it will cause a validation error.
type required[T SupportedTypes] struct {
	name            string
	short           string
	env             string
	defaultVal      *T
	descr           string
	customValidator func(T) error
	positional      bool

	alternatives     []string
	alternativesFunc func(cmd *cobra.Command, args []string, toComplete string) []string
	strictAlts       *bool

	setByEnv        bool
	setPositionally bool
	injected        bool
	valuePtr        any
	parent          *cobra.Command
}

var _ Param = &required[string]{}

func (f *required[T]) IsEnabled() bool {
	return true
}

func (f *required[T]) GetIsEnabledFn() func() bool {
	return nil
}

func (f *required[T]) SetAlternatives(strings []string) {
	f.alternatives = strings
}

func (f *required[T]) SetStrictAlts(strict bool) {
	f.strictAlts = &strict
}

func (f *required[T]) GetStrictAlts() bool {
	return f.strictAlts == nil || *f.strictAlts
}

func (f *required[T]) wasSetPositionally() bool {
	return f.setPositionally
}

func (f *required[T]) markSetPositionally() {
	f.setPositionally = true
}

func (f *required[T]) isPositional() bool {
	return f.positional
}

func (f *required[T]) SetDefault(val any) {
	if typedVal, ok := val.(*T); ok {
		f.defaultVal = typedVal
	} else {
		var zero T
		targetType := reflect.TypeOf(zero)
		valPtr := reflect.ValueOf(val)
		if valPtr.Kind() == reflect.Ptr && !valPtr.IsNil() {
			valElem := valPtr.Elem()
			if valElem.Type().ConvertibleTo(targetType) {
				converted := valElem.Convert(targetType)
				result := converted.Interface().(T)
				f.defaultVal = &result
				return
			}
		}
		f.defaultVal = val.(*T)
	}
}

func (f *required[T]) SetEnv(val string) {
	f.env = val
}

func (f *required[T]) SetShort(val string) {
	f.short = val
}

func (f *required[T]) SetName(val string) {
	f.name = val
}

func (f *required[T]) wasSetByEnv() bool {
	return f.setByEnv
}

func (f *required[T]) markSetFromEnv() {
	f.setByEnv = true
}

func (f *required[T]) value() T {
	if HasValue(f) {
		if f.valuePtr != nil {
			if val, ok := f.valuePtr.(*T); ok {
				return *val
			}
			valPtr := reflect.ValueOf(f.valuePtr)
			if valPtr.Kind() == reflect.Ptr && !valPtr.IsNil() {
				valElem := valPtr.Elem()
				var zero T
				targetType := reflect.TypeOf(zero)
				if valElem.Type().ConvertibleTo(targetType) {
					converted := valElem.Convert(targetType)
					return converted.Interface().(T)
				}
			}
			return *f.valuePtr.(*T)
		} else {
			return *f.defaultVal
		}
	} else {
		slog.Warn(fmt.Sprintf("tried to access required[..].value() of '%s', which was not set.", f.GetName()))
		var zero T
		return zero
	}
}

func (f *required[T]) setDescription(state string) {
	f.descr = state
}

func (f *required[T]) setPositional(state bool) {
	f.positional = state
}

func (f *required[T]) customValidatorOfPtr() func(any) error {
	return func(val any) error {
		if f.customValidator == nil {
			return nil
		}
		return f.customValidator(*val.(*T))
	}
}

func (f *required[T]) SetCustomValidator(validator func(any) error) {
	if validator == nil {
		f.customValidator = nil
		return
	}
	f.customValidator = func(val T) error {
		return validator(val)
	}
}

func (f *required[T]) wasSetOnCli() bool {
	if f.positional {
		return f.wasSetPositionally()
	} else {
		if f.parent == nil {
			return false
		} else {
			return f.parent.Flags().Changed(f.name)
		}
	}
}

func (f *required[T]) GetShort() string {
	return f.short
}

func (f *required[T]) GetName() string {
	return f.name
}

func (f *required[T]) GetEnv() string {
	return f.env
}

func (f *required[T]) defaultValuePtr() any {
	return f.defaultVal
}

func (f *required[T]) hasDefaultValue() bool {
	return f.defaultVal != nil
}

func (f *required[T]) getDescr() string {
	return f.descr
}

func (f *required[T]) IsRequired() bool {
	return true
}

func (f *required[T]) valuePtrF() any {
	if f.valuePtr != nil {
		return f.valuePtr
	} else {
		return f.defaultVal
	}
}

func (f *required[T]) wasSetByInject() bool {
	return f.injected && f.valuePtr != nil
}

func (f *required[T]) parentCmd() *cobra.Command {
	return f.parent
}

func (f *required[T]) defaultValueStr() string {
	if !f.hasDefaultValue() {
		slog.Error(fmt.Sprintf("defaultValueStr called on required parameter '%s' without default value", f.name))
		return ""
	}
	return fmt.Sprintf("%v", *f.defaultVal)
}

func (f *required[T]) HasValue() bool {
	return HasValue(f)
}

func (f *required[T]) GetKind() reflect.Kind {
	return f.GetType().Kind()
}

func (f *required[T]) GetType() reflect.Type {
	var zero T
	return reflect.TypeOf(zero)
}

func (f *required[T]) setParentCmd(cmd *cobra.Command) {
	f.parent = cmd
}

func (f *required[T]) setValuePtr(val any) {
	f.valuePtr = val
}

func (f *required[T]) injectValuePtr(val any) {
	f.valuePtr = val
	f.injected = val != nil
}

func (f *required[T]) GetAlternatives() []string {
	return f.alternatives
}

func (f *required[T]) GetAlternativesFunc() func(cmd *cobra.Command, args []string, toComplete string) []string {
	return f.alternativesFunc
}

func (f *required[T]) SetAlternativesFunc(fn func(cmd *cobra.Command, args []string, toComplete string) []string) {
	f.alternativesFunc = fn
}

func (p required[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.value())
}

func (p *required[T]) UnmarshalJSON(data []byte) error {
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

func (f *required[T]) SetIsEnabledFn(_ func() bool) {}

func (f *required[T]) SetRequiredFn(_ func() bool) {}

func (f *required[T]) GetRequiredFn() func() bool { return nil }
