package boa

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/spf13/cobra"
)

// paramMeta is a non-generic implementation of the Param interface.
// It replaces the old required[T] and optional[T] generic types with a single
// reflection-based struct that handles all parameter types uniformly.
type paramMeta struct {
	// Metadata
	name       string
	short      string
	env        string
	descr      string
	positional bool

	alternatives     []string
	alternativesFunc func(cmd *cobra.Command, args []string, toComplete string) []string
	strictAlts       *bool

	// Required/Enabled (unifies the old required[T] vs optional[T] split)
	defaultRequired bool         // set at creation from struct tags + globalConfig
	requiredFn      func() bool  // if set, overrides defaultRequired
	enabledFn       func() bool  // if set, checked for IsEnabled()

	// Type info
	fieldType reflect.Type // the VALUE type (string for *string fields, *url.URL for *url.URL fields)
	isPointer bool         // whether the user's struct field is a pointer (except *url.URL)

	// Prefix for nested named struct fields (e.g., "db-" for DB.Host → --db-host)
	flagPrefix string // kebab-case prefix for flag names
	envPrefix  string // UPPER_SNAKE_CASE prefix for env var names

	// Default value — stored as typed reflect.Value
	defaultVal *reflect.Value

	// State
	setByEnv        bool
	setPositionally bool
	injected        bool
	valuePtr        any            // cobra flag pointer (e.g., *string from StringP)
	parent          *cobra.Command

	// Validation
	customValidator func(any) error
	minVal          *float64 // min value (numeric) or min length (string)
	maxVal          *float64 // max value (numeric) or max length (string)
	pattern         string   // regex pattern for string validation
}

var _ Param = &paramMeta{}

// --- IsEnabled / IsRequired ---

func (f *paramMeta) IsEnabled() bool {
	if f.enabledFn != nil {
		return f.enabledFn()
	}
	return true
}

func (f *paramMeta) GetIsEnabledFn() func() bool {
	return f.enabledFn
}

func (f *paramMeta) SetIsEnabledFn(fn func() bool) {
	f.enabledFn = fn
}

func (f *paramMeta) IsRequired() bool {
	if f.requiredFn != nil {
		return f.requiredFn()
	}
	return f.defaultRequired
}

func (f *paramMeta) SetRequiredFn(fn func() bool) {
	f.requiredFn = fn
}

func (f *paramMeta) GetRequiredFn() func() bool {
	return f.requiredFn
}

// --- Alternatives ---

func (f *paramMeta) SetAlternatives(alts []string) {
	f.alternatives = alts
}

func (f *paramMeta) GetAlternatives() []string {
	return f.alternatives
}

func (f *paramMeta) SetAlternativesFunc(fn func(cmd *cobra.Command, args []string, toComplete string) []string) {
	f.alternativesFunc = fn
}

func (f *paramMeta) GetAlternativesFunc() func(cmd *cobra.Command, args []string, toComplete string) []string {
	return f.alternativesFunc
}

func (f *paramMeta) SetStrictAlts(strict bool) {
	f.strictAlts = &strict
}

func (f *paramMeta) GetStrictAlts() bool {
	return f.strictAlts == nil || *f.strictAlts
}

// --- Positional ---

func (f *paramMeta) isPositional() bool {
	return f.positional
}

func (f *paramMeta) setPositional(state bool) {
	f.positional = state
}

func (f *paramMeta) wasSetPositionally() bool {
	return f.setPositionally
}

func (f *paramMeta) markSetPositionally() {
	f.setPositionally = true
}

// --- Name / Short / Env / Description ---

func (f *paramMeta) GetName() string  { return f.name }
func (f *paramMeta) SetName(val string)  { f.name = val }
func (f *paramMeta) GetShort() string { return f.short }
func (f *paramMeta) SetShort(val string) { f.short = val }
func (f *paramMeta) GetEnv() string   { return f.env }
func (f *paramMeta) SetEnv(val string)   { f.env = val }
func (f *paramMeta) getDescr() string { return f.descr }
func (f *paramMeta) setDescription(descr string) { f.descr = descr }

// --- Type info ---

func (f *paramMeta) GetType() reflect.Type {
	return f.fieldType
}

func (f *paramMeta) GetKind() reflect.Kind {
	return f.fieldType.Kind()
}

// --- Default value ---

func (f *paramMeta) SetDefault(val any) {
	if val == nil {
		f.defaultVal = nil
		return
	}

	valRef := reflect.ValueOf(val)

	// val should be a pointer (*T) — dereference to get the value
	if valRef.Kind() == reflect.Ptr && !valRef.IsNil() {
		elem := valRef.Elem()

		// Direct type match
		if elem.Type() == f.fieldType {
			v := reflect.New(f.fieldType).Elem()
			v.Set(elem)
			f.defaultVal = &v
			return
		}

		// Handle type aliases (e.g., MyString → string)
		if elem.Type().ConvertibleTo(f.fieldType) {
			converted := elem.Convert(f.fieldType)
			v := reflect.New(f.fieldType).Elem()
			v.Set(converted)
			f.defaultVal = &v
			return
		}
	}

	// Fallback: try to use the value directly (non-pointer)
	if valRef.Type() == f.fieldType {
		v := reflect.New(f.fieldType).Elem()
		v.Set(valRef)
		f.defaultVal = &v
		return
	}

	panic(fmt.Errorf("paramMeta.SetDefault: cannot set default of type %T for field type %s", val, f.fieldType))
}

func (f *paramMeta) hasDefaultValue() bool {
	return f.defaultVal != nil
}

func (f *paramMeta) defaultValuePtr() any {
	if f.defaultVal == nil {
		return nil
	}
	// Return a pointer to a copy of the default value
	ptr := reflect.New(f.fieldType)
	ptr.Elem().Set(*f.defaultVal)
	return ptr.Interface()
}

func (f *paramMeta) defaultValueStr() string {
	if !f.hasDefaultValue() {
		slog.Error(fmt.Sprintf("defaultValueStr called on parameter '%s' without default value", f.name))
		return ""
	}
	return fmt.Sprintf("%v", f.defaultVal.Interface())
}

// --- Value storage ---

func (f *paramMeta) setValuePtr(val any) {
	f.valuePtr = val
}

func (f *paramMeta) injectValuePtr(val any) {
	f.valuePtr = val
	f.injected = val != nil
}

func (f *paramMeta) valuePtrF() any {
	if f.valuePtr != nil {
		return f.valuePtr
	}
	return f.defaultValuePtr()
}

func (f *paramMeta) HasValue() bool {
	return HasValue(f)
}

// --- CLI/Env state ---

func (f *paramMeta) wasSetOnCli() bool {
	if f.positional {
		return f.wasSetPositionally()
	}
	if f.parent == nil {
		return false
	}
	return f.parent.Flags().Changed(f.name)
}

func (f *paramMeta) wasSetByEnv() bool {
	return f.setByEnv
}

func (f *paramMeta) markSetFromEnv() {
	f.setByEnv = true
}

func (f *paramMeta) wasSetByInject() bool {
	return f.injected && f.valuePtr != nil
}

// --- Parent command ---

func (f *paramMeta) parentCmd() *cobra.Command {
	return f.parent
}

func (f *paramMeta) setParentCmd(cmd *cobra.Command) {
	f.parent = cmd
}

// --- Custom validator ---

func (f *paramMeta) customValidatorOfPtr() func(any) error {
	return func(val any) error {
		if f.customValidator == nil {
			return nil
		}
		// val is a pointer (e.g., *int from valuePtrF()). Dereference it
		// so the validator receives the value, not a pointer to it.
		v := reflect.ValueOf(val)
		if v.Kind() == reflect.Ptr && !v.IsNil() {
			return f.customValidator(v.Elem().Interface())
		}
		return f.customValidator(val)
	}
}

func (f *paramMeta) SetCustomValidator(validator func(any) error) {
	f.customValidator = validator
}

// --- JSON marshaling ---

func (f *paramMeta) MarshalJSON() ([]byte, error) {
	if !f.HasValue() {
		return json.Marshal(nil)
	}
	if f.valuePtr != nil {
		val := reflect.ValueOf(f.valuePtr)
		if val.Kind() == reflect.Ptr && !val.IsNil() {
			return json.Marshal(val.Elem().Interface())
		}
		return json.Marshal(nil)
	}
	if f.defaultVal != nil {
		return json.Marshal(f.defaultVal.Interface())
	}
	return json.Marshal(nil)
}

func (f *paramMeta) UnmarshalJSON(data []byte) error {
	if f.wasSetOnCli() || f.wasSetByEnv() {
		return nil
	}
	// Allocate a new value of the field type
	ptr := reflect.New(f.fieldType)
	if err := json.Unmarshal(data, ptr.Interface()); err != nil {
		return err
	}
	// Check if the unmarshaled value is the zero value (for JSON null)
	if ptr.Elem().IsZero() {
		return nil
	}
	f.valuePtr = ptr.Interface()
	f.injected = true
	return nil
}
