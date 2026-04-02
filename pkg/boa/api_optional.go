package boa

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/spf13/cobra"
)

// optional represents a parameter that may or may not have a value.
type optional[T SupportedTypes] struct {
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

	requiredFn func() bool
	enabledFn  func() bool
}

var _ Param = &optional[string]{}

func (f *optional[T]) GetIsEnabledFn() func() bool {
	return f.enabledFn
}

func (f *optional[T]) IsEnabled() bool {
	if f.enabledFn != nil {
		return f.enabledFn()
	}
	return true
}

func (f *optional[T]) SetIsEnabledFn(f2 func() bool) {
	f.enabledFn = f2
}

func (f *optional[T]) GetAlternatives() []string {
	return f.alternatives
}

func (f *optional[T]) GetAlternativesFunc() func(cmd *cobra.Command, args []string, toComplete string) []string {
	return f.alternativesFunc
}

func (f *optional[T]) SetAlternativesFunc(fn func(cmd *cobra.Command, args []string, toComplete string) []string) {
	f.alternativesFunc = fn
}

func (f *optional[T]) SetAlternatives(strings []string) {
	f.alternatives = strings
}

func (f *optional[T]) SetStrictAlts(strict bool) {
	f.strictAlts = &strict
}

func (f *optional[T]) GetStrictAlts() bool {
	return f.strictAlts == nil || *f.strictAlts
}

func (f *optional[T]) wasSetPositionally() bool {
	return f.setPositionally
}

func (f *optional[T]) markSetPositionally() {
	f.setPositionally = true
}

func (f *optional[T]) isPositional() bool {
	return f.positional
}

func (f *optional[T]) SetDefault(val any) {
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

func (f *optional[T]) SetEnv(val string) {
	f.env = val
}

func (f *optional[T]) SetShort(val string) {
	f.short = val
}

func (f *optional[T]) SetName(val string) {
	f.name = val
}

func (f *optional[T]) wasSetByEnv() bool {
	return f.setByEnv
}

func (f *optional[T]) markSetFromEnv() {
	f.setByEnv = true
}

func (f *optional[T]) value() *T {
	if HasValue(f) {
		if f.valuePtr != nil {
			if val, ok := f.valuePtr.(*T); ok {
				return val
			}
			valPtr := reflect.ValueOf(f.valuePtr)
			if valPtr.Kind() == reflect.Ptr && !valPtr.IsNil() {
				valElem := valPtr.Elem()
				var zero T
				targetType := reflect.TypeOf(zero)
				if valElem.Type().ConvertibleTo(targetType) {
					converted := valElem.Convert(targetType)
					result := converted.Interface().(T)
					return &result
				}
			}
			return f.valuePtr.(*T)
		} else {
			return f.defaultVal
		}
	} else {
		slog.Warn(fmt.Sprintf("tried to access optional[..].value() of '%s', which was not set.", f.GetName()))
		return nil
	}
}

func (f *optional[T]) HasValue() bool {
	return HasValue(f)
}

func (f *optional[T]) setPositional(state bool) {
	f.positional = state
}

func (f *optional[T]) setDescription(state string) {
	f.descr = state
}

func (f *optional[T]) customValidatorOfPtr() func(any) error {
	return func(val any) error {
		if f.customValidator == nil {
			return nil
		}
		return f.customValidator(*val.(*T))
	}
}

func (f *optional[T]) SetCustomValidator(validator func(any) error) {
	if validator == nil {
		f.customValidator = nil
		return
	}
	f.customValidator = func(val T) error {
		return validator(val)
	}
}

func (f *optional[T]) wasSetOnCli() bool {
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

func (f *optional[T]) wasSetByInject() bool {
	return f.injected && f.valuePtr != nil
}

func (f *optional[T]) GetShort() string {
	return f.short
}

func (f *optional[T]) GetName() string {
	return f.name
}

func (f *optional[T]) GetEnv() string {
	return f.env
}

func (f *optional[T]) defaultValuePtr() any {
	return f.defaultVal
}

func (f *optional[T]) hasDefaultValue() bool {
	return f.defaultVal != nil
}

func (f *optional[T]) getDescr() string {
	return f.descr
}

func (f *optional[T]) IsRequired() bool {
	if f.requiredFn != nil {
		return f.requiredFn()
	}
	return false
}

func (f *optional[T]) SetRequiredFn(condition func() bool) {
	f.requiredFn = condition
}

func (f *optional[T]) GetRequiredFn() func() bool {
	return f.requiredFn
}

func (f *optional[T]) valuePtrF() any {
	if f.valuePtr != nil {
		return f.valuePtr
	} else {
		return f.defaultVal
	}
}

func (f *optional[T]) parentCmd() *cobra.Command {
	return f.parent
}

func (f *optional[T]) defaultValueStr() string {
	if !f.hasDefaultValue() {
		slog.Error(fmt.Sprintf("defaultValueStr called on optional parameter '%s' without default value", f.name))
		return ""
	}
	return fmt.Sprintf("%v", *f.defaultVal)
}

func (f *optional[T]) GetKind() reflect.Kind {
	return f.GetType().Kind()
}

func (f *optional[T]) GetType() reflect.Type {
	var zero T
	return reflect.TypeOf(zero)
}

func (f *optional[T]) setParentCmd(cmd *cobra.Command) {
	f.parent = cmd
}

func (f *optional[T]) setValuePtr(val any) {
	f.valuePtr = val
}

func (f *optional[T]) injectValuePtr(val any) {
	f.valuePtr = val
	f.injected = val != nil
}

func (p optional[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.value())
}

func (p *optional[T]) UnmarshalJSON(data []byte) error {
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
