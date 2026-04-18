package boa

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// typeHandler defines how a specific Go type is handled as a CLI parameter.
// Each handler covers the full lifecycle: flag binding, string parsing, and post-parse conversion.
type typeHandler struct {
	// bindFlag registers a cobra flag and returns the pointer cobra writes to.
	// name, short, descr are the flag metadata. defaultVal is from paramMeta.defaultValuePtr() (may be nil).
	bindFlag func(cmd *cobra.Command, name, short, descr string, defaultVal any) any

	// parse converts a string value (from env var, default tag, positional arg) into a typed pointer.
	// Returns *T for the appropriate type.
	parse func(name, strVal string) (any, error)

	// convert is called during validation to convert cobra's stored type to the final type.
	// Only needed for types stored as strings in cobra (time.Time, *url.URL).
	// nil means no conversion needed.
	convert func(name string, val any) (any, error)

	// baseType is the canonical Go type, used by normalizeType().
	// e.g., for time.Duration this is reflect.TypeOf(time.Duration(0))
	baseType reflect.Type
}

// typeHandlerRegistry maps reflect.Type → handler for special types (time.Time, net.IP, etc.)
// and reflect.Kind → handler for basic types (string, int, bool, etc.)
var (
	exactTypeHandlers = map[reflect.Type]*typeHandler{}
	kindHandlers      = map[reflect.Kind]*typeHandler{}

	// sliceTypeHandlers maps the element type to a handler for slices of that type.
	// Used for []time.Duration, []time.Time, []net.IP, []*url.URL.
	sliceExactTypeHandlers = map[reflect.Type]*typeHandler{}
	sliceKindHandlers      = map[reflect.Kind]*typeHandler{}

	// mapTypeHandlers maps the exact map type (e.g., map[string]string) to a handler.
	mapTypeHandlers = map[reflect.Type]*typeHandler{}
)

func init() {
	registerBuiltinTypes()
}

// TypeDef defines how a custom type is parsed from and formatted to strings.
// Use with RegisterType to add support for user-defined types as CLI parameters.
type TypeDef[T any] struct {
	// Parse converts a CLI string into the typed value.
	Parse func(string) (T, error)
	// Format converts the typed value back to a string (for default display).
	// If nil, fmt.Sprintf("%v", val) is used.
	Format func(T) string
}

// RegisterType registers a custom type for use as a CLI parameter.
// The type will be stored as a string flag in cobra and converted using
// the provided Parse/Format functions.
//
// Example:
//
//	boa.RegisterType[SemVer](boa.TypeDef[SemVer]{
//	    Parse:  func(s string) (SemVer, error) { return parseSemVer(s) },
//	    Format: func(v SemVer) string { return v.String() },
//	})
func RegisterType[T any](def TypeDef[T]) {
	t := reflect.TypeOf((*T)(nil)).Elem()
	formatFn := def.Format
	if formatFn == nil {
		formatFn = func(v T) string { return fmt.Sprintf("%v", v) }
	}

	exactTypeHandlers[t] = &typeHandler{
		baseType: t,
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			def := ""
			if defaultVal != nil {
				v := reflect.ValueOf(defaultVal)
				if v.Kind() == reflect.Ptr && !v.IsNil() {
					def = formatFn(v.Elem().Interface().(T))
				}
			}
			return cmd.Flags().StringP(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			v, err := def.Parse(strVal)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %w", name, err)
			}
			return &v, nil
		},
		convert: func(name string, val any) (any, error) {
			if strPtr, ok := val.(*string); ok {
				if *strPtr == "" {
					var zero T
					return &zero, nil
				}
				v, err := def.Parse(*strPtr)
				if err != nil {
					return nil, fmt.Errorf("invalid value for param '%s': %w", name, err)
				}
				return &v, nil
			}
			return val, nil // already converted
		},
	}
}

func registerBuiltinTypes() {
	// --- Basic types (by kind) ---

	kindHandlers[reflect.String] = &typeHandler{
		baseType: reflect.TypeOf(""),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			def := ""
			if defaultVal != nil {
				def = reflect.ValueOf(defaultVal).Elem().Convert(reflect.TypeOf(def)).Interface().(string)
			}
			return cmd.Flags().StringP(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			return &strVal, nil
		},
	}

	kindHandlers[reflect.Int] = &typeHandler{
		baseType: reflect.TypeOf(0),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			def := 0
			if defaultVal != nil {
				def = int(reflect.ValueOf(defaultVal).Elem().Int())
			}
			return cmd.Flags().IntP(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			v, err := strconv.Atoi(strVal)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			return &v, nil
		},
	}

	kindHandlers[reflect.Int32] = &typeHandler{
		baseType: reflect.TypeOf(int32(0)),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			def := int32(0)
			if defaultVal != nil {
				def = int32(reflect.ValueOf(defaultVal).Elem().Int())
			}
			return cmd.Flags().Int32P(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			v, err := strconv.ParseInt(strVal, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			result := int32(v)
			return &result, nil
		},
	}

	kindHandlers[reflect.Int64] = &typeHandler{
		baseType: reflect.TypeOf(int64(0)),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			def := int64(0)
			if defaultVal != nil {
				def = reflect.ValueOf(defaultVal).Elem().Int()
			}
			return cmd.Flags().Int64P(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			v, err := strconv.ParseInt(strVal, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			return &v, nil
		},
	}

	kindHandlers[reflect.Int8] = &typeHandler{
		baseType: reflect.TypeOf(int8(0)),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			def := int8(0)
			if defaultVal != nil {
				def = int8(reflect.ValueOf(defaultVal).Elem().Int())
			}
			return cmd.Flags().Int8P(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			v, err := strconv.ParseInt(strVal, 10, 8)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			result := int8(v)
			return &result, nil
		},
	}

	kindHandlers[reflect.Int16] = &typeHandler{
		baseType: reflect.TypeOf(int16(0)),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			def := int16(0)
			if defaultVal != nil {
				def = int16(reflect.ValueOf(defaultVal).Elem().Int())
			}
			return cmd.Flags().Int16P(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			v, err := strconv.ParseInt(strVal, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			result := int16(v)
			return &result, nil
		},
	}

	kindHandlers[reflect.Uint] = &typeHandler{
		baseType: reflect.TypeOf(uint(0)),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			def := uint(0)
			if defaultVal != nil {
				def = uint(reflect.ValueOf(defaultVal).Elem().Uint())
			}
			return cmd.Flags().UintP(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			v, err := strconv.ParseUint(strVal, 10, 0)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			result := uint(v)
			return &result, nil
		},
	}

	kindHandlers[reflect.Uint8] = &typeHandler{
		baseType: reflect.TypeOf(uint8(0)),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			def := uint8(0)
			if defaultVal != nil {
				def = uint8(reflect.ValueOf(defaultVal).Elem().Uint())
			}
			return cmd.Flags().Uint8P(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			v, err := strconv.ParseUint(strVal, 10, 8)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			result := uint8(v)
			return &result, nil
		},
	}

	kindHandlers[reflect.Uint16] = &typeHandler{
		baseType: reflect.TypeOf(uint16(0)),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			def := uint16(0)
			if defaultVal != nil {
				def = uint16(reflect.ValueOf(defaultVal).Elem().Uint())
			}
			return cmd.Flags().Uint16P(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			v, err := strconv.ParseUint(strVal, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			result := uint16(v)
			return &result, nil
		},
	}

	kindHandlers[reflect.Uint32] = &typeHandler{
		baseType: reflect.TypeOf(uint32(0)),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			def := uint32(0)
			if defaultVal != nil {
				def = uint32(reflect.ValueOf(defaultVal).Elem().Uint())
			}
			return cmd.Flags().Uint32P(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			v, err := strconv.ParseUint(strVal, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			result := uint32(v)
			return &result, nil
		},
	}

	kindHandlers[reflect.Uint64] = &typeHandler{
		baseType: reflect.TypeOf(uint64(0)),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			def := uint64(0)
			if defaultVal != nil {
				def = reflect.ValueOf(defaultVal).Elem().Uint()
			}
			return cmd.Flags().Uint64P(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			v, err := strconv.ParseUint(strVal, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			return &v, nil
		},
	}

	kindHandlers[reflect.Float32] = &typeHandler{
		baseType: reflect.TypeOf(float32(0)),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			def := float32(0)
			if defaultVal != nil {
				def = float32(reflect.ValueOf(defaultVal).Elem().Float())
			}
			return cmd.Flags().Float32P(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			v, err := strconv.ParseFloat(strVal, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			result := float32(v)
			return &result, nil
		},
	}

	kindHandlers[reflect.Float64] = &typeHandler{
		baseType: reflect.TypeOf(float64(0)),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			def := float64(0)
			if defaultVal != nil {
				def = reflect.ValueOf(defaultVal).Elem().Float()
			}
			return cmd.Flags().Float64P(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			v, err := strconv.ParseFloat(strVal, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			return &v, nil
		},
	}

	kindHandlers[reflect.Bool] = &typeHandler{
		baseType: reflect.TypeOf(false),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			def := false
			if defaultVal != nil {
				def = reflect.ValueOf(defaultVal).Elem().Bool()
			}
			return cmd.Flags().BoolP(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			v, err := strconv.ParseBool(strVal)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			return &v, nil
		},
	}

	// --- Special types (by exact type) ---

	exactTypeHandlers[durationType] = &typeHandler{
		baseType: durationType,
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			def := time.Duration(0)
			if defaultVal != nil {
				def = time.Duration(reflect.ValueOf(defaultVal).Elem().Int())
			}
			return cmd.Flags().DurationP(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			v, err := time.ParseDuration(strVal)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			return &v, nil
		},
	}

	exactTypeHandlers[timeType] = &typeHandler{
		baseType: timeType,
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			def := ""
			if defaultVal != nil {
				t := reflect.ValueOf(defaultVal).Elem().Interface().(time.Time)
				def = t.Format(time.RFC3339)
			}
			return cmd.Flags().StringP(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			v, err := parseTimeString(strVal)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			return &v, nil
		},
		convert: func(name string, val any) (any, error) {
			if strPtr, ok := val.(*string); ok {
				v, err := parseTimeString(*strPtr)
				if err != nil {
					return nil, fmt.Errorf("invalid value for param '%s': %s", name, err.Error())
				}
				return &v, nil
			}
			return val, nil // already a *time.Time (e.g., from struct literal)
		},
	}

	exactTypeHandlers[ipType] = &typeHandler{
		baseType: ipType,
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			var def net.IP
			if defaultVal != nil {
				def = reflect.ValueOf(defaultVal).Elem().Interface().(net.IP)
			}
			return cmd.Flags().IPP(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			v := net.ParseIP(strVal)
			if v == nil {
				return nil, fmt.Errorf("invalid IP address for param %s: %s", name, strVal)
			}
			return &v, nil
		},
	}

	exactTypeHandlers[urlPtrType] = &typeHandler{
		baseType: urlPtrType,
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			def := ""
			if defaultVal != nil {
				defVal := reflect.ValueOf(defaultVal).Elem()
				if !defVal.IsNil() {
					def = defVal.Interface().(*url.URL).String()
				}
			}
			return cmd.Flags().StringP(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			if strVal == "" {
				return (*url.URL)(nil), nil
			}
			v, err := url.Parse(strVal)
			if err != nil {
				return nil, fmt.Errorf("invalid URL for param %s: %s", name, err.Error())
			}
			return &v, nil
		},
		convert: func(name string, val any) (any, error) {
			if strPtr, ok := val.(*string); ok {
				v, err := url.Parse(*strPtr)
				if err != nil {
					return nil, fmt.Errorf("invalid value for param '%s': %s", name, err.Error())
				}
				return &v, nil
			}
			return val, nil // already converted
		},
	}

	// --- Slice types (by element type) ---

	// []net.IP
	sliceExactTypeHandlers[ipType] = &typeHandler{
		baseType: reflect.SliceOf(ipType),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			var def []net.IP
			if defaultVal != nil {
				def = reflect.ValueOf(defaultVal).Elem().Interface().([]net.IP)
			}
			return cmd.Flags().IPSliceP(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			return parseSliceWith(strVal, func(s string) (net.IP, error) {
				v := net.ParseIP(s)
				if v == nil {
					return nil, fmt.Errorf("invalid IP for param %s: %s", name, s)
				}
				return v, nil
			})
		},
	}

	// []time.Duration
	sliceExactTypeHandlers[durationType] = &typeHandler{
		baseType: reflect.SliceOf(durationType),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			var def []time.Duration
			if defaultVal != nil {
				defVal := reflect.ValueOf(defaultVal).Elem()
				if defVal.Kind() == reflect.Slice {
					def = defVal.Interface().([]time.Duration)
				}
			}
			return cmd.Flags().DurationSliceP(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			return parseSliceWith(strVal, func(s string) (time.Duration, error) {
				return time.ParseDuration(s)
			})
		},
	}

	// []time.Time — stored as []string, converted later
	sliceExactTypeHandlers[timeType] = &typeHandler{
		baseType: reflect.SliceOf(timeType),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			var def []string
			if defaultVal != nil {
				defVal := reflect.ValueOf(defaultVal).Elem()
				if defVal.Kind() == reflect.Slice && defVal.Type().Elem() == timeType {
					times := defVal.Interface().([]time.Time)
					def = make([]string, len(times))
					for i, t := range times {
						def[i] = t.Format(time.RFC3339)
					}
				}
			}
			return cmd.Flags().StringSliceP(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			return parseSliceWith(strVal, func(s string) (time.Time, error) {
				return parseTimeString(s)
			})
		},
		convert: func(name string, val any) (any, error) {
			if strSlice, ok := val.(*[]string); ok && strSlice != nil {
				times := make([]time.Time, len(*strSlice))
				for i, s := range *strSlice {
					t, err := parseTimeString(s)
					if err != nil {
						return nil, fmt.Errorf("invalid value for param '%s' at index %d: %s", name, i, err.Error())
					}
					times[i] = t
				}
				return &times, nil
			}
			return val, nil
		},
	}

	// []*url.URL — stored as []string, converted later
	sliceExactTypeHandlers[urlPtrType] = &typeHandler{
		baseType: reflect.SliceOf(urlPtrType),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			var def []string
			if defaultVal != nil {
				defVal := reflect.ValueOf(defaultVal).Elem()
				if defVal.Kind() == reflect.Slice && defVal.Type().Elem() == urlPtrType {
					urls := defVal.Interface().([]*url.URL)
					def = make([]string, len(urls))
					for i, u := range urls {
						if u != nil {
							def[i] = u.String()
						}
					}
				}
			}
			return cmd.Flags().StringSliceP(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			return parseSliceWith(strVal, func(s string) (*url.URL, error) {
				return url.Parse(s)
			})
		},
		convert: func(name string, val any) (any, error) {
			if strSlice, ok := val.(*[]string); ok && strSlice != nil {
				urls := make([]*url.URL, len(*strSlice))
				for i, s := range *strSlice {
					u, err := url.Parse(s)
					if err != nil {
						return nil, fmt.Errorf("invalid value for param '%s' at index %d: %s", name, i, err.Error())
					}
					urls[i] = u
				}
				return &urls, nil
			}
			return val, nil
		},
	}

	// --- Basic slice types (by element kind) ---

	sliceKindHandlers[reflect.String] = &typeHandler{
		baseType: reflect.SliceOf(reflect.TypeOf("")),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			return cmd.Flags().StringSliceP(name, short, toTypedSlice[string](derefSliceDefault(defaultVal)), descr)
		},
		parse: func(name, strVal string) (any, error) {
			return parseSliceWith(strVal, func(s string) (string, error) { return s, nil })
		},
	}

	sliceKindHandlers[reflect.Int] = &typeHandler{
		baseType: reflect.SliceOf(reflect.TypeOf(0)),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			return cmd.Flags().IntSliceP(name, short, toTypedSlice[int](derefSliceDefault(defaultVal)), descr)
		},
		parse: func(name, strVal string) (any, error) {
			return parseSliceWith(strVal, func(s string) (int, error) { return strconv.Atoi(s) })
		},
	}

	sliceKindHandlers[reflect.Int32] = &typeHandler{
		baseType: reflect.SliceOf(reflect.TypeOf(int32(0))),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			return cmd.Flags().Int32SliceP(name, short, toTypedSlice[int32](derefSliceDefault(defaultVal)), descr)
		},
		parse: func(name, strVal string) (any, error) {
			return parseSliceWith(strVal, func(s string) (int32, error) {
				v, err := strconv.ParseInt(s, 10, 32)
				return int32(v), err
			})
		},
	}

	sliceKindHandlers[reflect.Int64] = &typeHandler{
		baseType: reflect.SliceOf(reflect.TypeOf(int64(0))),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			return cmd.Flags().Int64SliceP(name, short, toTypedSlice[int64](derefSliceDefault(defaultVal)), descr)
		},
		parse: func(name, strVal string) (any, error) {
			return parseSliceWith(strVal, func(s string) (int64, error) {
				return strconv.ParseInt(s, 10, 64)
			})
		},
	}

	sliceKindHandlers[reflect.Uint] = &typeHandler{
		baseType: reflect.SliceOf(reflect.TypeOf(uint(0))),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			return cmd.Flags().UintSliceP(name, short, toTypedSlice[uint](derefSliceDefault(defaultVal)), descr)
		},
		parse: func(name, strVal string) (any, error) {
			return parseSliceWith(strVal, func(s string) (uint, error) {
				v, err := strconv.ParseUint(s, 10, 0)
				return uint(v), err
			})
		},
	}

	// pflag has no native Uint8/Uint16/Uint32/Uint64/Int8/Int16 slice flags.
	// For these, register a typed pflag.Value via Var() — outward shape matches
	// Int32SliceP / Int64SliceP (proper element type in --help, no string round-trip).
	sliceKindHandlers[reflect.Int8] = makeIntSliceFallbackHandler(reflect.TypeOf(int8(0)), "int8Slice",
		func(s string) (int8, error) { v, err := strconv.ParseInt(s, 10, 8); return int8(v), err },
		func(v int8) string { return strconv.FormatInt(int64(v), 10) })

	sliceKindHandlers[reflect.Int16] = makeIntSliceFallbackHandler(reflect.TypeOf(int16(0)), "int16Slice",
		func(s string) (int16, error) { v, err := strconv.ParseInt(s, 10, 16); return int16(v), err },
		func(v int16) string { return strconv.FormatInt(int64(v), 10) })

	sliceKindHandlers[reflect.Uint8] = makeIntSliceFallbackHandler(reflect.TypeOf(uint8(0)), "uint8Slice",
		func(s string) (uint8, error) { v, err := strconv.ParseUint(s, 10, 8); return uint8(v), err },
		func(v uint8) string { return strconv.FormatUint(uint64(v), 10) })

	sliceKindHandlers[reflect.Uint16] = makeIntSliceFallbackHandler(reflect.TypeOf(uint16(0)), "uint16Slice",
		func(s string) (uint16, error) { v, err := strconv.ParseUint(s, 10, 16); return uint16(v), err },
		func(v uint16) string { return strconv.FormatUint(uint64(v), 10) })

	sliceKindHandlers[reflect.Uint32] = makeIntSliceFallbackHandler(reflect.TypeOf(uint32(0)), "uint32Slice",
		func(s string) (uint32, error) { v, err := strconv.ParseUint(s, 10, 32); return uint32(v), err },
		func(v uint32) string { return strconv.FormatUint(uint64(v), 10) })

	sliceKindHandlers[reflect.Uint64] = makeIntSliceFallbackHandler(reflect.TypeOf(uint64(0)), "uint64Slice",
		func(s string) (uint64, error) { return strconv.ParseUint(s, 10, 64) },
		func(v uint64) string { return strconv.FormatUint(v, 10) })

	sliceKindHandlers[reflect.Float32] = &typeHandler{
		baseType: reflect.SliceOf(reflect.TypeOf(float32(0))),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			return cmd.Flags().Float32SliceP(name, short, toTypedSlice[float32](derefSliceDefault(defaultVal)), descr)
		},
		parse: func(name, strVal string) (any, error) {
			return parseSliceWith(strVal, func(s string) (float32, error) {
				v, err := strconv.ParseFloat(s, 32)
				return float32(v), err
			})
		},
	}

	sliceKindHandlers[reflect.Float64] = &typeHandler{
		baseType: reflect.SliceOf(reflect.TypeOf(float64(0))),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			return cmd.Flags().Float64SliceP(name, short, toTypedSlice[float64](derefSliceDefault(defaultVal)), descr)
		},
		parse: func(name, strVal string) (any, error) {
			return parseSliceWith(strVal, func(s string) (float64, error) {
				return strconv.ParseFloat(s, 64)
			})
		},
	}

	sliceKindHandlers[reflect.Bool] = &typeHandler{
		baseType: reflect.SliceOf(reflect.TypeOf(false)),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			return cmd.Flags().BoolSliceP(name, short, toTypedSlice[bool](derefSliceDefault(defaultVal)), descr)
		},
		parse: func(name, strVal string) (any, error) {
			return parseSliceWith(strVal, func(s string) (bool, error) { return strconv.ParseBool(s) })
		},
	}
}

// lookupHandler finds the appropriate handler for a type.
// Returns the handler and whether the type is a slice.
func lookupHandler(t reflect.Type) (*typeHandler, bool) {
	// Exact type match first (special types like time.Duration, time.Time, net.IP, *url.URL)
	if h, ok := exactTypeHandlers[t]; ok {
		return h, false
	}
	// Kind-based match for basic types
	if h, ok := kindHandlers[t.Kind()]; ok {
		return h, false
	}
	return nil, false
}

// lookupSliceHandler finds the handler for a slice type based on its element type.
func lookupSliceHandler(elemType reflect.Type) *typeHandler {
	// Exact element type match first
	if h, ok := sliceExactTypeHandlers[elemType]; ok {
		return h
	}
	// Kind-based match
	if h, ok := sliceKindHandlers[elemType.Kind()]; ok {
		return h
	}
	return nil
}

// jsonFallbackHandler creates a handler for any type that uses StringP for cobra binding
// and json.Unmarshal for parsing. This handles nested slices, complex maps, and any
// other type that Go's JSON decoder can handle.
func jsonFallbackHandler(t reflect.Type) *typeHandler {
	return &typeHandler{
		baseType: t,
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			def := ""
			if defaultVal != nil {
				// Marshal the default to JSON for display
				v := reflect.ValueOf(defaultVal)
				if v.Kind() == reflect.Ptr && !v.IsNil() {
					b, err := json.Marshal(v.Elem().Interface())
					if err == nil {
						def = string(b)
					}
				}
			}
			return cmd.Flags().StringP(name, short, def, descr)
		},
		parse: func(name, strVal string) (any, error) {
			ptr := reflect.New(t)
			if err := json.Unmarshal([]byte(strVal), ptr.Interface()); err != nil {
				return nil, fmt.Errorf("invalid JSON for param %s: %w", name, err)
			}
			return ptr.Interface(), nil
		},
		convert: func(name string, val any) (any, error) {
			// val is *string from StringP — unmarshal it
			if strPtr, ok := val.(*string); ok {
				if *strPtr == "" {
					return val, nil // no conversion needed for empty default
				}
				ptr := reflect.New(t)
				if err := json.Unmarshal([]byte(*strPtr), ptr.Interface()); err != nil {
					return nil, fmt.Errorf("invalid JSON for param '%s': %w", name, err)
				}
				return ptr.Interface(), nil
			}
			return val, nil // already converted
		},
	}
}

// lookupMapHandler dynamically builds a handler for map[string]V types by composing
// the value type's scalar handler for parsing. For cobra flag binding, it uses pflag's
// native StringToString/StringToInt/StringToInt64 methods where available, falling back
// to a StringP flag with key=value parsing.
func lookupMapHandler(t reflect.Type) *typeHandler {
	if t.Kind() != reflect.Map || t.Key().Kind() != reflect.String {
		return nil // only map[string]V is supported
	}

	// Check cache first
	if h, ok := mapTypeHandlers[t]; ok {
		return h
	}

	valType := normalizeType(t.Elem())

	// Find the scalar handler for the value type
	valHandler, _ := lookupHandler(valType)
	if valHandler == nil {
		return nil
	}

	parseFn := buildMapParse(t, valType, valHandler)

	// Build a composed handler
	h := &typeHandler{
		baseType: t,
		bindFlag: buildMapBindFlag(t, valType),
		parse:    parseFn,
	}

	// Non-native map types (not string/int/int64) use StringP, so they need a
	// convert function to parse the stored string into the actual map type.
	switch valType {
	case reflect.TypeOf(""), reflect.TypeOf(0), reflect.TypeOf(int64(0)):
		// Native pflag support — no convert needed
	default:
		h.convert = func(name string, val any) (any, error) {
			if strPtr, ok := val.(*string); ok {
				if *strPtr == "" {
					return val, nil
				}
				return parseFn(name, *strPtr)
			}
			return val, nil // already converted
		}
	}

	// Cache it
	mapTypeHandlers[t] = h
	return h
}

// buildMapBindFlag returns a bindFlag function for map[string]V.
// Uses pflag's native methods for string/int/int64, falls back to StringP for others.
func buildMapBindFlag(mapType, valType reflect.Type) func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
	// Check for pflag's native map support
	switch valType {
	case reflect.TypeOf(""):
		return func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			var def map[string]string
			if defaultVal != nil {
				def = reflect.ValueOf(defaultVal).Elem().Interface().(map[string]string)
			}
			return cmd.Flags().StringToStringP(name, short, def, descr)
		}
	case reflect.TypeOf(0):
		return func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			var def map[string]int
			if defaultVal != nil {
				def = reflect.ValueOf(defaultVal).Elem().Interface().(map[string]int)
			}
			return cmd.Flags().StringToIntP(name, short, def, descr)
		}
	case reflect.TypeOf(int64(0)):
		return func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			var def map[string]int64
			if defaultVal != nil {
				def = reflect.ValueOf(defaultVal).Elem().Interface().(map[string]int64)
			}
			return cmd.Flags().StringToInt64P(name, short, def, descr)
		}
	default:
		// Fall back to string flag with custom parsing
		return func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			def := ""
			return cmd.Flags().StringP(name, short, def, descr)
		}
	}
}

// buildMapParse returns a parse function for map[string]V that delegates
// value parsing to the scalar handler.
func buildMapParse(mapType, valType reflect.Type, valHandler *typeHandler) func(name, strVal string) (any, error) {
	return func(name, strVal string) (any, error) {
		result := reflect.MakeMap(mapType)
		if strVal == "" {
			ptr := reflect.New(mapType)
			ptr.Elem().Set(result)
			return ptr.Interface(), nil
		}
		pairs := strings.Split(strVal, ",")
		for _, pair := range pairs {
			kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
			if len(kv) != 2 {
				return nil, fmt.Errorf("invalid map entry for param %s: %q (expected key=value)", name, pair)
			}
			key := strings.TrimSpace(kv[0])
			valStr := strings.TrimSpace(kv[1])
			parsedPtr, err := valHandler.parse(name, valStr)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s key %q: %w", name, key, err)
			}
			// parsedPtr is *V, dereference to get V
			result.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(parsedPtr).Elem())
		}
		ptr := reflect.New(mapType)
		ptr.Elem().Set(result)
		return ptr.Interface(), nil
	}
}

// typedIntSliceValue is a pflag.Value / pflag.SliceValue implementation for integer
// slice types pflag has no native slice flag for (uint8, uint16, uint32, uint64,
// int8, int16). Same outward behavior as Int32SliceP / Int64SliceP: comma-split,
// repeated-flag, CSV-with-quotes — pflag handles all of that via Set().
type typedIntSliceValue[T any] struct {
	value      *[]T
	changed    bool
	parseElem  func(string) (T, error)
	formatElem func(T) string
	typeName   string
}

func (s *typedIntSliceValue[T]) Set(val string) error {
	parts, err := readAsCSV(val)
	if err != nil {
		return err
	}
	parsed := make([]T, 0, len(parts))
	for _, p := range parts {
		v, err := s.parseElem(p)
		if err != nil {
			return err
		}
		parsed = append(parsed, v)
	}
	if !s.changed {
		*s.value = parsed
	} else {
		*s.value = append(*s.value, parsed...)
	}
	s.changed = true
	return nil
}

func (s *typedIntSliceValue[T]) Type() string { return s.typeName }

func (s *typedIntSliceValue[T]) String() string {
	parts := make([]string, len(*s.value))
	for i, v := range *s.value {
		parts[i] = s.formatElem(v)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func (s *typedIntSliceValue[T]) Append(val string) error {
	v, err := s.parseElem(val)
	if err != nil {
		return err
	}
	*s.value = append(*s.value, v)
	return nil
}

func (s *typedIntSliceValue[T]) Replace(vals []string) error {
	parsed := make([]T, 0, len(vals))
	for _, str := range vals {
		v, err := s.parseElem(str)
		if err != nil {
			return err
		}
		parsed = append(parsed, v)
	}
	*s.value = parsed
	return nil
}

func (s *typedIntSliceValue[T]) GetSlice() []string {
	parts := make([]string, len(*s.value))
	for i, v := range *s.value {
		parts[i] = s.formatElem(v)
	}
	return parts
}

// makeIntSliceFallbackHandler builds a slice handler for integer types pflag has no
// native slice flag for. Uses Var() with a typedIntSliceValue so the resulting flag
// is a proper typed slice (same shape as Int32SliceP) — no string round-trip, no
// convert function needed.
func makeIntSliceFallbackHandler[T any](
	elemType reflect.Type,
	typeName string,
	parseElem func(string) (T, error),
	formatElem func(T) string,
) *typeHandler {
	return &typeHandler{
		baseType: reflect.SliceOf(elemType),
		bindFlag: func(cmd *cobra.Command, name, short, descr string, defaultVal any) any {
			storage := new([]T)
			if defaultVal != nil {
				if typed := toTypedSlice[T](derefSliceDefault(defaultVal)); typed != nil {
					*storage = typed
				}
			}
			v := &typedIntSliceValue[T]{
				value:      storage,
				parseElem:  parseElem,
				formatElem: formatElem,
				typeName:   typeName,
			}
			cmd.Flags().VarP(v, name, short, descr)
			return storage
		},
		parse: func(name, strVal string) (any, error) {
			return parseSliceWith(strVal, parseElem)
		},
	}
}

// readAsCSV parses one value pflag receives from the command line into individual
// elements, using csv rules so quoted commas survive (matches pflag's own
// stringSliceValue behavior).
func readAsCSV(val string) ([]string, error) {
	if val == "" {
		return nil, nil
	}
	stringReader := csv.NewReader(strings.NewReader(val))
	return stringReader.Read()
}

// derefSliceDefault dereferences a defaultVal pointer (e.g., *[]string → []string) for slice handlers.
// defaultVal comes from paramMeta.defaultValuePtr() which always returns a pointer.
func derefSliceDefault(defaultVal any) any {
	if defaultVal == nil {
		return nil
	}
	v := reflect.ValueOf(defaultVal)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		return v.Elem().Interface()
	}
	return defaultVal
}

// parseSliceWith is a generic helper that parses a bracketed string "[a,b,c]" into a typed slice.
func parseSliceWith[T any](strVal string, parseFn func(string) (T, error)) (*[]T, error) {
	strVal = strings.TrimSpace(strVal)
	if strings.HasPrefix(strVal, "[") && strings.HasSuffix(strVal, "]") {
		strVal = strVal[1 : len(strVal)-1]
	}
	if strVal == "" {
		result := make([]T, 0)
		return &result, nil
	}
	parts := strings.Split(strVal, ",")
	result := make([]T, len(parts))
	for i, part := range parts {
		v, err := parseFn(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		result[i] = v
	}
	return &result, nil
}
