package boa

import (
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
)

func init() {
	registerBuiltinTypes()
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
			strVal := *val.(*string)
			v, err := parseTimeString(strVal)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param '%s': %s", name, err.Error())
			}
			return &v, nil
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
