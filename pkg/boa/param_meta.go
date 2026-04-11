package boa

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
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

	// pathKey is the fieldPath this param is stored under in the
	// processingContext's mirrorByPath map. Stashed at creation so callers
	// (e.g., the configfile tag handler) that have a *paramMeta can read its
	// path directly instead of scanning pathOrder.
	pathKey fieldPath

	// Default value — stored as typed reflect.Value
	defaultVal *reflect.Value

	// State
	setByEnv        bool
	setByConfig     bool
	setPositionally bool
	injected        bool
	valuePtr        any            // cobra flag pointer (e.g., *string from StringP)
	parent          *cobra.Command

	// Validation
	customValidator func(any) error
	// minVal / maxVal hold the bound as a typed pointer. The concrete type
	// depends on the field kind:
	//   - signed int field   → *int64
	//   - unsigned int field → *uint64
	//   - float field        → *float64
	//   - string/slice/map   → *int (length bound)
	// nil means "no bound". The typed storage keeps int64 bounds lossless
	// past 2^53, which is the whole point of going through an any here
	// instead of always-float64.
	minVal  any
	maxVal  any
	pattern string // regex pattern for string validation

	// noFlag indicates the field should not be registered as a CLI flag,
	// but is still populated from env vars and config files. Set via the
	// `boa:"noflag"` tag (alias `boa:"nocli"`).
	noFlag bool

	// noEnv indicates the field should not read from environment variables.
	// CLI flags and config files still populate it normally. Set via the
	// `boa:"noenv"` tag.
	noEnv bool

	// ignored marks the mirror as fully ignored by boa: skip CLI flag,
	// skip env reading, skip required/min/max/pattern validation. The
	// only remaining write path is config-file unmarshal, which writes
	// to the raw struct field directly. The tag-level equivalent
	// (`boa:"ignore"` / `boa:"-"`) skips traversal entirely so the mirror
	// never exists; this flag is for programmatic equivalence
	// post-traversal. Note: `boa:"configonly"` is NOT an alias for this —
	// it desugars to noFlag+noEnv with the mirror preserved so validation
	// and required checks still run.
	ignored bool

	// isConfigFile marks this string parameter as the config-file path for
	// its parent struct. Mirrors `configfile:"true"`. When set, the field
	// must be a string and its value (CLI / env / default) is used as a
	// path to a config file that's unmarshaled into the enclosing struct.
	// Set either via the tag or programmatically via SetConfigFile(true).
	isConfigFile bool
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
		v := reflect.ValueOf(val)
		if v.Kind() == reflect.Ptr && !v.IsNil() {
			if f.isPointer {
				// For pointer fields (*int, *string, etc.), pass the pointer value
				// so the validator receives the same type the user declared.
				return f.customValidator(v.Interface())
			}
			// For non-pointer fields, dereference so the validator receives the value.
			return f.customValidator(v.Elem().Interface())
		}
		return f.customValidator(val)
	}
}

func (f *paramMeta) SetCustomValidator(validator func(any) error) {
	f.customValidator = validator
}

// --- noFlag / ignored ---

func (f *paramMeta) IsNoFlag() bool         { return f.noFlag }
func (f *paramMeta) SetNoFlag(val bool)     { f.noFlag = val }
func (f *paramMeta) IsNoEnv() bool          { return f.noEnv }
func (f *paramMeta) SetNoEnv(val bool)      { f.noEnv = val }
func (f *paramMeta) IsIgnored() bool        { return f.ignored }
func (f *paramMeta) SetIgnored(val bool)    { f.ignored = val }
func (f *paramMeta) IsConfigFile() bool     { return f.isConfigFile }
func (f *paramMeta) SetConfigFile(val bool) { f.isConfigFile = val }

// --- min / max / pattern ---

// boundKind classifies a field type for min/max purposes. It collapses the
// reflect.Kind zoo into the four shapes a bound actually has: signed int,
// unsigned int, float, or length (string/slice/map). unsupportedBound means
// min/max are meaningless on this field.
type boundKind int

const (
	unsupportedBound boundKind = iota
	signedIntBound
	unsignedIntBound
	floatBound
	lengthBound
)

// boundKindOf returns the boundKind for a reflect.Type. Uses Kind() so type
// aliases (e.g., `type Port int`) work transparently.
func boundKindOf(t reflect.Type) boundKind {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return signedIntBound
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return unsignedIntBound
	case reflect.Float32, reflect.Float64:
		return floatBound
	case reflect.String, reflect.Slice, reflect.Map:
		return lengthBound
	}
	return unsupportedBound
}

// boundKind returns this param's boundKind.
func (f *paramMeta) boundKind() boundKind { return boundKindOf(f.fieldType) }

// supportsPattern reports whether this param's underlying type is one the
// pattern validator will act on (strings only).
func (f *paramMeta) supportsPattern() bool {
	return f.fieldType.Kind() == reflect.String
}

// coerceBound takes a caller-provided value and normalizes it to the storage
// type for the given boundKind. Accepts any numeric value (int/uint/float
// widths) as long as it fits the target kind. Returns an error with a human
// message if the input is the wrong shape (e.g. float bound on an int field,
// or negative length).
func coerceBound(raw any, bk boundKind) (any, error) {
	if raw == nil {
		return nil, fmt.Errorf("nil bound")
	}
	rv := reflect.ValueOf(raw)
	// Unwrap a single level of pointer, e.g. caller passed *int64.
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, fmt.Errorf("nil bound")
		}
		rv = rv.Elem()
	}
	switch bk {
	case signedIntBound:
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			v := rv.Int()
			return &v, nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			u := rv.Uint()
			if u > math.MaxInt64 {
				return nil, fmt.Errorf("bound %d overflows int64", u)
			}
			v := int64(u)
			return &v, nil
		case reflect.Float32, reflect.Float64:
			return nil, fmt.Errorf("float bound on signed-int field is lossy; pass an int value instead")
		}
	case unsignedIntBound:
		switch rv.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			v := rv.Uint()
			return &v, nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			i := rv.Int()
			if i < 0 {
				return nil, fmt.Errorf("negative bound %d on unsigned-int field", i)
			}
			v := uint64(i)
			return &v, nil
		case reflect.Float32, reflect.Float64:
			return nil, fmt.Errorf("float bound on unsigned-int field is lossy; pass an int value instead")
		}
	case floatBound:
		switch rv.Kind() {
		case reflect.Float32, reflect.Float64:
			v := rv.Float()
			return &v, nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			v := float64(rv.Int())
			return &v, nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			v := float64(rv.Uint())
			return &v, nil
		}
	case lengthBound:
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			i := rv.Int()
			if i < 0 {
				return nil, fmt.Errorf("negative length bound %d", i)
			}
			v := int(i)
			return &v, nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			u := rv.Uint()
			if u > math.MaxInt {
				return nil, fmt.Errorf("length bound %d overflows int", u)
			}
			v := int(u)
			return &v, nil
		case reflect.Float32, reflect.Float64:
			return nil, fmt.Errorf("float length bound is not allowed; pass an int")
		}
	}
	return nil, fmt.Errorf("unsupported bound value of type %T", raw)
}

// GetMin returns a copy of the current lower bound, or nil if none is set.
// The concrete type is one of *int64 / *uint64 / *float64 / *int depending on
// the field kind.
func (f *paramMeta) GetMin() any { return copyBound(f.minVal) }

// GetMax returns a copy of the current upper bound. See GetMin for the
// concrete type dispatch.
func (f *paramMeta) GetMax() any { return copyBound(f.maxVal) }

// copyBound returns a defensive copy of a stored typed-pointer bound.
func copyBound(b any) any {
	switch v := b.(type) {
	case nil:
		return nil
	case *int64:
		if v == nil {
			return nil
		}
		out := *v
		return &out
	case *uint64:
		if v == nil {
			return nil
		}
		out := *v
		return &out
	case *float64:
		if v == nil {
			return nil
		}
		out := *v
		return &out
	case *int:
		if v == nil {
			return nil
		}
		out := *v
		return &out
	}
	return nil
}

// SetMin sets a lower bound. Accepts any numeric value; it's coerced to the
// storage type that matches the field's boundKind. Panics if the field type
// cannot carry a bound or if the provided value has the wrong shape. Use
// ClearMin to remove a previously set bound.
func (f *paramMeta) SetMin(val any) {
	bk := f.boundKind()
	if bk == unsupportedBound {
		panic(fmt.Errorf("boa: SetMin on %q: type %s is not a numeric, string, slice, or map — min is only meaningful on those", f.name, f.fieldType.Kind()))
	}
	coerced, err := coerceBound(val, bk)
	if err != nil {
		panic(fmt.Errorf("boa: SetMin on %q: %w", f.name, err))
	}
	f.minVal = coerced
}

// ClearMin removes any lower bound previously set on this parameter. Safe to
// call on any type.
func (f *paramMeta) ClearMin() { f.minVal = nil }

// SetMax sets an upper bound. See SetMin for the type rules.
func (f *paramMeta) SetMax(val any) {
	bk := f.boundKind()
	if bk == unsupportedBound {
		panic(fmt.Errorf("boa: SetMax on %q: type %s is not a numeric, string, slice, or map — max is only meaningful on those", f.name, f.fieldType.Kind()))
	}
	coerced, err := coerceBound(val, bk)
	if err != nil {
		panic(fmt.Errorf("boa: SetMax on %q: %w", f.name, err))
	}
	f.maxVal = coerced
}

// ClearMax removes any upper bound previously set on this parameter. Safe to
// call on any type.
func (f *paramMeta) ClearMax() { f.maxVal = nil }

func (f *paramMeta) GetPattern() string { return f.pattern }

// SetPattern sets a regex pattern. Pass the empty string to clear. Panics if
// called on a non-string field — pattern matching is only defined for strings.
func (f *paramMeta) SetPattern(pat string) {
	if pat != "" && !f.supportsPattern() {
		panic(fmt.Errorf("boa: SetPattern on %q: type %s is not a string — pattern is only meaningful on strings", f.name, f.fieldType.Kind()))
	}
	f.pattern = pat
}

// --- exported description / positional / required convenience ---

func (f *paramMeta) GetDescription() string      { return f.descr }
func (f *paramMeta) SetDescription(descr string) { f.descr = descr }
func (f *paramMeta) IsPositional() bool          { return f.positional }
func (f *paramMeta) SetPositional(state bool)    { f.positional = state }

// SetRequired pins the parameter as required/optional regardless of any
// earlier tag or SetRequiredFn. It is equivalent to
// SetRequiredFn(func() bool { return val }), which means it **replaces**
// (not composes with) any previously installed SetRequiredFn. Call this
// last if you use both.
func (f *paramMeta) SetRequired(val bool) {
	v := val
	f.requiredFn = func() bool { return v }
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
