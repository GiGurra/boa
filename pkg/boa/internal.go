package boa

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unsafe"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// osExit is the exit function used by Run(). It defaults to os.Exit but can be
// replaced in tests to verify exit behavior without terminating the test process.
var osExit = os.Exit

// UserInputError wraps errors caused by invalid user input (missing required
// params, invalid flag values, etc.). These errors should result in a clean
// exit with error message rather than a panic with stack trace.
type UserInputError struct {
	Err error
}

func (e *UserInputError) Error() string {
	return e.Err.Error()
}

func (e *UserInputError) Unwrap() error {
	return e.Err
}

// NewUserInputError wraps an error as a UserInputError.
// Use this in hooks like PreValidateFunc when returning user input validation errors
// to ensure they result in a clean exit (no stack trace) when using Run().
func NewUserInputError(err error) error {
	if err == nil {
		return nil
	}
	return &UserInputError{Err: err}
}

// NewUserInputErrorf creates a new UserInputError with a formatted message.
// Use this in hooks like PreValidateFunc when returning user input validation errors
// to ensure they result in a clean exit (no stack trace) when using Run().
func NewUserInputErrorf(format string, args ...any) error {
	return &UserInputError{Err: fmt.Errorf(format, args...)}
}

// internal aliases for backwards compatibility within the package
var newUserInputError = NewUserInputError
var newUserInputErrorf = NewUserInputErrorf

// IsUserInputError checks if an error is (or wraps) a UserInputError
// or is one of pflag's known validation error types
func IsUserInputError(err error) bool {
	if err == nil {
		return false
	}

	// Check for our custom UserInputError
	var uie *UserInputError
	if errors.As(err, &uie) {
		return true
	}

	// Check for pflag validation error types
	var invalidSyntax *pflag.InvalidSyntaxError
	var invalidValue *pflag.InvalidValueError
	var notExist *pflag.NotExistError
	var valueRequired *pflag.ValueRequiredError

	if errors.As(err, &invalidSyntax) ||
		errors.As(err, &invalidValue) ||
		errors.As(err, &notExist) ||
		errors.As(err, &valueRequired) {
		return true
	}

	return false
}

// wrapArgsValidator wraps a cobra.PositionalArgs validator to return UserInputError
func wrapArgsValidator(validator cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := validator(cmd, args); err != nil {
			return newUserInputError(err)
		}
		return nil
	}
}

type Param interface {
	HasValue() bool
	GetShort() string
	GetName() string
	GetEnv() string
	GetKind() reflect.Kind
	GetType() reflect.Type
	SetDefault(any)
	SetEnv(string)
	SetShort(string)
	SetName(string)
	SetAlternatives([]string)
	defaultValuePtr() any
	getDescr() string
	IsRequired() bool
	valuePtrF() any
	parentCmd() *cobra.Command
	wasSetOnCli() bool
	wasSetByEnv() bool
	wasSetByInject() bool
	customValidatorOfPtr() func(any) error
	SetCustomValidator(func(any) error)
	hasDefaultValue() bool
	defaultValueStr() string
	setParentCmd(cmd *cobra.Command)
	setValuePtr(any)
	injectValuePtr(any)
	markSetFromEnv()
	isPositional() bool
	wasSetPositionally() bool
	markSetPositionally()
	setPositional(bool)
	setDescription(descr string)
	IsEnabled() bool
	GetAlternatives() []string
	GetAlternativesFunc() func(cmd *cobra.Command, args []string, toComplete string) []string
	SetAlternativesFunc(func(cmd *cobra.Command, args []string, toComplete string) []string)
	GetIsEnabledFn() func() bool
	SetIsEnabledFn(func() bool)
	SetRequiredFn(func() bool)
	GetRequiredFn() func() bool
	SetStrictAlts(bool)
	GetStrictAlts() bool
}

// configFileEntry tracks a configfile:"true" field and the struct it should load into.
type configFileEntry struct {
	mirror Param // the string param holding the file path
	target any   // pointer to the struct to unmarshal into
}

type processingContext struct {
	context.Context
	RawAddrToMirror map[unsafe.Pointer]Param
	// We need to keep track of raw params, so we can
	// override the raw values with cli values in case
	// the user may have mapped config files to the params
	// as well - since the config file deserialization will
	// not be aware of the raw values, and just overwrite them.
	RawAddresses []unsafe.Pointer
	// ConfigFiles tracks all configfile:"true" fields and their target structs.
	// Ordered: substruct entries first, root entry last (so root overrides inner).
	ConfigFiles []configFileEntry
}

func parseEnv(ctx *processingContext, structPtr any) error {

	err := traverse(ctx, structPtr, func(param Param, _ string, _ reflect.StructTag) error {

		if !param.IsEnabled() {
			return nil
		}

		if err := readEnv(param); err != nil {
			return err
		}

		return nil
	}, nil)
	return newUserInputError(err)
}

func validate(ctx *processingContext, structPtr any) error {

	err := traverse(ctx, structPtr, func(param Param, _ string, _ reflect.StructTag) error {

		if !param.IsEnabled() {
			return nil
		}

		envHint := ""
		if param.GetEnv() != "" {
			envHint = fmt.Sprintf(" (env: %s)", param.GetEnv())
		}

		if param.IsRequired() && !HasValue(param) {
			return fmt.Errorf("missing required param '%s'%s", param.GetName(), envHint)
		}

		// Post-parse conversion for types stored as strings in cobra (time.Time, *url.URL, JSON fallback, etc.)
		if HasValue(param) {
			converted := false
			if handler, _ := lookupHandler(param.GetType()); handler != nil && handler.convert != nil {
				res, err := handler.convert(param.GetName(), param.valuePtrF())
				if err != nil {
					return err
				}
				param.setValuePtr(res)
				converted = true
			} else if param.GetKind() == reflect.Slice {
				if sliceHandler := lookupSliceHandler(param.GetType().Elem()); sliceHandler != nil && sliceHandler.convert != nil {
					res, err := sliceHandler.convert(param.GetName(), param.valuePtrF())
					if err != nil {
						return err
					}
					param.setValuePtr(res)
					converted = true
				}
			}

			// JSON fallback conversion: if value is still a *string but the target type
			// is a complex type (map, nested slice, etc.) without a native handler, try JSON unmarshal
			if !converted {
				needsJsonFallback := false
				if param.GetKind() == reflect.Map {
					needsJsonFallback = lookupMapHandler(param.GetType()) == nil
				} else if param.GetKind() == reflect.Slice && lookupSliceHandler(param.GetType().Elem()) == nil {
					needsJsonFallback = true
				}
				if needsJsonFallback {
					if strPtr, ok := param.valuePtrF().(*string); ok && strPtr != nil && *strPtr != "" {
						fallback := jsonFallbackHandler(param.GetType())
						res, err := fallback.convert(param.GetName(), param.valuePtrF())
						if err != nil {
							return err
						}
						param.setValuePtr(res)
					}
				}
			}
			if alts := param.GetAlternatives(); alts != nil && param.GetStrictAlts() {

				ptrVal := param.valuePtrF()
				// check if it is a slice param
				kind := reflect.TypeOf(ptrVal).Elem().Kind()
				if kind == reflect.Slice {
					// run the validation for each slice element
					sliceVal := reflect.ValueOf(ptrVal).Elem()
					for i := 0; i < sliceVal.Len(); i++ {
						elem := sliceVal.Index(i)
						strVal := fmt.Sprintf("%v", elem.Interface())
						if !slices.Contains(alts, strVal) {
							return fmt.Errorf("invalid value for param '%s': '%s' is not in the list of allowed values: %v", param.GetName(), strVal, alts)
						}
					}
				} else {
					if ptrVal != nil {
						strVal := ptrToAnyToString(ptrVal)
						if !slices.Contains(alts, strVal) {
							return fmt.Errorf("invalid value for param '%s': '%s' is not in the list of allowed values: %v", param.GetName(), strVal, alts)
						}
					}
				}
			}

			if err := param.customValidatorOfPtr()(param.valuePtrF()); err != nil {
				return fmt.Errorf("invalid value for param '%s': %s", param.GetName(), err.Error())
			}

			// min/max/pattern tag validation
			if pm, ok := param.(*paramMeta); ok {
				if err := validateMinMaxPattern(pm, param.valuePtrF()); err != nil {
					return fmt.Errorf("invalid value for param '%s': %s", param.GetName(), err.Error())
				}
			}
		}

		return nil
	}, nil)
	return newUserInputError(err)
}

// validateMinMaxPattern checks min/max/pattern tag constraints.
// For numeric types, min/max compare the value. For strings, min/max compare length.
func validateMinMaxPattern(pm *paramMeta, valPtr any) error {
	if pm.minVal == nil && pm.maxVal == nil && pm.pattern == "" {
		return nil
	}

	v := reflect.ValueOf(valPtr)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64:
		val := float64(v.Int())
		if pm.minVal != nil && val < *pm.minVal {
			return fmt.Errorf("value %v is below min %v", v.Int(), *pm.minVal)
		}
		if pm.maxVal != nil && val > *pm.maxVal {
			return fmt.Errorf("value %v exceeds max %v", v.Int(), *pm.maxVal)
		}
	case reflect.Float32, reflect.Float64:
		val := v.Float()
		if pm.minVal != nil && val < *pm.minVal {
			return fmt.Errorf("value %v is below min %v", val, *pm.minVal)
		}
		if pm.maxVal != nil && val > *pm.maxVal {
			return fmt.Errorf("value %v exceeds max %v", val, *pm.maxVal)
		}
	case reflect.String:
		str := v.String()
		if pm.minVal != nil && float64(len(str)) < *pm.minVal {
			return fmt.Errorf("length %d is below min %v", len(str), *pm.minVal)
		}
		if pm.maxVal != nil && float64(len(str)) > *pm.maxVal {
			return fmt.Errorf("length %d exceeds max %v", len(str), *pm.maxVal)
		}
	}

	if pm.pattern != "" && v.Kind() == reflect.String {
		matched, err := regexp.MatchString(pm.pattern, v.String())
		if err != nil {
			return fmt.Errorf("invalid pattern %q: %w", pm.pattern, err)
		}
		if !matched {
			return fmt.Errorf("value %q does not match pattern %q", v.String(), pm.pattern)
		}
	}

	return nil
}

func ptrToAnyToString(ptr any) string {
	if ptr == nil {
		panic("ptrToAnyToString called with nil")
	}

	val := reflect.ValueOf(ptr)
	if val.Kind() != reflect.Ptr {
		panic("ptrToAnyToString called with non-pointer")
	}

	elem := val.Elem()
	return fmt.Sprintf("%v", elem.Interface())
}

func doParsePositional(f Param, strVal string) error {
	if strVal == "" && f.IsRequired() {
		if f.hasDefaultValue() || f.wasSetByEnv() {
			return nil
		} else {
			return newUserInputErrorf("empty positional arg: %s", f.GetName())
		}
	}

	if err := readFrom(f, strVal); err != nil {
		return newUserInputError(err)
	}

	f.markSetPositionally()

	return nil
}

func toTypedSlice[T any](slice any) []T {
	if slice == nil {
		return nil
	}
	// Try direct type assertion first
	if typed, ok := slice.([]T); ok {
		return typed
	}
	// Handle type aliases by converting via reflection
	sliceVal := reflect.ValueOf(slice)
	if sliceVal.Kind() == reflect.Slice {
		result := make([]T, sliceVal.Len())
		var zero T
		targetElemType := reflect.TypeOf(zero)
		for i := 0; i < sliceVal.Len(); i++ {
			elem := sliceVal.Index(i)
			if elem.Type().ConvertibleTo(targetElemType) {
				result[i] = elem.Convert(targetElemType).Interface().(T)
			} else {
				result[i] = elem.Interface().(T)
			}
		}
		return result
	}
	// Fallback to original behavior (will panic with clear message)
	return slice.([]T)
}

func connect(f Param, cmd *cobra.Command, posArgs []Param, ctx *processingContext) error {

	if f.GetName() == "" {
		return fmt.Errorf("invalid conf for param '%s': long param name cannot be empty", f.GetName())
	}

	if f.GetShort() == "h" {
		// Check if we already have a help flag
		if hf := cmd.Flags().Lookup("help"); hf != nil {
			if hf.Shorthand == "h" {
				return fmt.Errorf("invalid conf for param '%s': short param cannot be 'h'. It collides with -h for help", f.GetName())
			}
		} else {
			return fmt.Errorf("invalid conf for param '%s': short param cannot be 'h'. It collides with the default help flag. Set a custom help flag if you wish to override -h", f.GetName())
		}
	}

	if f.GetName() == "help" {
		return fmt.Errorf("invalid conf for param '%s': name cannot be 'help'. It collides with the standard help param", f.GetName())
	}

	extraInfos := make([]string, 0)

	descr := f.getDescr()
	if f.GetEnv() != "" {
		extraInfos = append(extraInfos, fmt.Sprintf("env: %s", f.GetEnv()))
	}

	if f.IsRequired() && !f.hasDefaultValue() {
		extraInfos = append(extraInfos, "required")
	}

	if f.GetIsEnabledFn() != nil {
		extraInfos = append(extraInfos, "conditional")
	}

	if len(extraInfos) > 0 {
		descr = fmt.Sprintf("%s (%s)", descr, strings.Join(extraInfos, ", "))
	}

	if f.hasDefaultValue() {
		if f.GetKind() == reflect.Bool {
			// cobra doesn't show if the default is false. So we must do it ourselves
			if f.defaultValueStr() == "false" {
				descr = fmt.Sprintf("%s (default false)", descr)
			}
		} else if f.defaultValueStr() == "" {
			// cobra doesn't show explicitly empty defaults. So we must do it ourselves
			descr = fmt.Sprintf("%s (default \"\")", descr)
		}
	}

	if f.parentCmd() != nil {
		return fmt.Errorf("param '%s' already connected to a command. Looks like you are trying to use the same parameter struct for two commands. This is not possible. Pleas instantiate one separate struct instance per command", f.GetName())
	}
	f.setParentCmd(cmd)

	if f.isPositional() {
		startSign := func() string {
			if f.IsRequired() {
				return "<"
			} else {
				return "["
			}
		}()
		endSign := func() string {
			if f.IsRequired() {
				return ">"
			} else {
				return "]"
			}
		}()
		cmd.Use += " " + startSign + f.GetName() + endSign

		if cmd.Args == nil {
			cmd.Args = func(cmd *cobra.Command, args []string) error {
				return nil
			}
		}
		// Add the positional arg to the Args function
		oldFn := cmd.Args
		cmd.Args = func(cmd *cobra.Command, args []string) error {
			if err := oldFn(cmd, args); err != nil {
				return err
			}
			posArgIndex := -1
			for i, posArg := range posArgs {
				if posArg.GetName() == f.GetName() {
					posArgIndex = i
				}
			}
			if posArgIndex == -1 {
				if f.IsRequired() {
					return fmt.Errorf("positional arg '%s' not found. This is a bug in boa", f.GetName())
				} else {
					return nil
				}
			}
			if posArgIndex >= len(args) {
				if f.IsRequired() {
					if f.hasDefaultValue() {
						f.setValuePtr(f.defaultValuePtr())
						return nil
					} else {
						return newUserInputErrorf("missing positional arg '%s'", f.GetName())
					}
				} else {
					return nil
				}
			}
			if f.GetKind() == reflect.Slice {
				// Concat all remaining args into a single string, to be parsed as a slice
				remainingArgs := args[posArgIndex:]
				joinedArgs := "[" + strings.Join(remainingArgs, ",") + "]"
				return doParsePositional(f, joinedArgs)
			} else {
				return doParsePositional(f, args[posArgIndex])
			}
		}
		return nil // no need to attach cobra flags
	}

	// Must happen last, because the flags must have been created
	defer func() {
		if f.GetAlternatives() != nil {
			err := cmd.RegisterFlagCompletionFunc(f.GetName(), func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				return f.GetAlternatives(), cobra.ShellCompDirectiveDefault
			})
			if err != nil {
				panic(fmt.Errorf("failed to register static flag completion func for flag '%s': %v", f.GetName(), err))
			}
		}
		if f.GetAlternativesFunc() != nil {
			err := cmd.RegisterFlagCompletionFunc(f.GetName(), func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				// Sync cobra's parsed flag values into raw struct fields before calling
				// the user's completion function. Without this, raw fields (plain string,
				// int, etc.) are zero during completion because PreRunE (which normally
				// calls syncMirrors) is never executed for shell completion.
				syncMirrors(ctx)
				return f.GetAlternativesFunc()(cmd, args, toComplete), cobra.ShellCompDirectiveDefault
			})
			if err != nil {
				panic(fmt.Errorf("failed to register dynamic flag completion func for flag '%s': %v", f.GetName(), err))
			}
		}
	}()

	// Look up type handler for scalar types (including net.IP which is []byte but treated as scalar)
	if handler, _ := lookupHandler(f.GetType()); handler != nil {
		var defVal any
		if f.hasDefaultValue() {
			defVal = f.defaultValuePtr()
		}
		f.setValuePtr(handler.bindFlag(cmd, f.GetName(), f.GetShort(), descr, defVal))
		return nil
	}

	// Map types — try native handler first, then JSON fallback
	if f.GetKind() == reflect.Map {
		mapHandler := lookupMapHandler(f.GetType())
		if mapHandler == nil {
			mapHandler = jsonFallbackHandler(f.GetType())
		}
		var defVal any
		if f.hasDefaultValue() {
			defVal = f.defaultValuePtr()
		}
		f.setValuePtr(mapHandler.bindFlag(cmd, f.GetName(), f.GetShort(), descr, defVal))
		return nil
	}

	// Slice types — try native handler first, then JSON fallback
	if f.GetKind() == reflect.Slice {
		elemType := f.GetType().Elem()
		sliceHandler := lookupSliceHandler(elemType)

		if sliceHandler != nil {
			var defVal any
			if f.hasDefaultValue() {
				defValRef := reflect.ValueOf(f.defaultValuePtr()).Elem()
				// If default was parsed from string tag, it might need parsing into the proper slice type
				if defValRef.Kind() != reflect.Slice {
					parsed, err := sliceHandler.parse(f.GetName(), f.defaultValueStr())
					if err != nil {
						return fmt.Errorf("default value for slice param '%s' is invalid: %s", f.GetName(), err.Error())
					}
					f.SetDefault(parsed)
				}
				defVal = f.defaultValuePtr()
			}
			f.setValuePtr(sliceHandler.bindFlag(cmd, f.GetName(), f.GetShort(), descr, defVal))
			return nil
		}

		// JSON fallback for complex slice types (nested slices, etc.)
		fallback := jsonFallbackHandler(f.GetType())
		var defVal any
		if f.hasDefaultValue() {
			defVal = f.defaultValuePtr()
		}
		f.setValuePtr(fallback.bindFlag(cmd, f.GetName(), f.GetShort(), descr, defVal))
		return nil
	}

	if f.GetKind() == reflect.Array {
		return fmt.Errorf("unsupported param type (Array): %s: ", f.GetKind().String())
	}

	return fmt.Errorf("unsupported param type: %s", f.GetKind().String())
}

func readEnv(f Param) error {
	if f.GetEnv() == "" {
		return nil
	}

	if f.wasSetOnCli() {
		return nil
	}

	envVal := os.Getenv(f.GetEnv())
	if envVal == "" {
		return nil
	}

	err := readFrom(f, envVal)
	if err != nil {
		return err
	}

	f.markSetFromEnv()
	return nil
}

func readFrom(f Param, strVal string) error {

	ptr, err := parsePtr(f.GetName(), f.GetType(), f.GetKind(), strVal)
	if err != nil {
		return err
	}

	f.setValuePtr(ptr)

	return nil
}

// parseTimeString parses a time string trying multiple common formats
func parseTimeString(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %s", s)
}

func parsePtr(
	name string,
	tpe reflect.Type,
	kind reflect.Kind,
	strVal string,
) (any, error) {
	// Scalar types — look up by exact type first, then by kind
	if handler, _ := lookupHandler(tpe); handler != nil {
		return handler.parse(name, strVal)
	}

	// Map types — native handler or JSON fallback
	if kind == reflect.Map {
		if mapHandler := lookupMapHandler(tpe); mapHandler != nil {
			return mapHandler.parse(name, strVal)
		}
		return jsonFallbackHandler(tpe).parse(name, strVal)
	}

	// Slice types — native handler or JSON fallback
	if kind == reflect.Slice {
		elemType := tpe.Elem()
		if sliceHandler := lookupSliceHandler(elemType); sliceHandler != nil {
			return sliceHandler.parse(name, strVal)
		}
		return jsonFallbackHandler(tpe).parse(name, strVal)
	}

	if kind == reflect.Array {
		return nil, fmt.Errorf("arrays not supported param type. Use a slice instead: %s", kind.String())
	}

	return nil, fmt.Errorf("unsupported param type: %s", kind.String())
}

func camelToKebabCase(in string) string {
	var result strings.Builder
	runes := []rune(in)

	for i, char := range runes {
		if unicode.IsUpper(char) {
			if i > 0 {
				prev := runes[i-1]
				// Insert dash before uppercase if previous was lowercase,
				// or if previous was uppercase but next is lowercase (end of acronym).
				// e.g., "DBHost" → "db-host", "myParam" → "my-param"
				if unicode.IsLower(prev) {
					result.WriteRune('-')
				} else if unicode.IsUpper(prev) && i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
					result.WriteRune('-')
				}
			}
			result.WriteRune(unicode.ToLower(char))
		} else {
			result.WriteRune(char)
		}
	}

	return result.String()
}

func kebabCaseToUpperSnakeCase(in string) string {
	var result strings.Builder

	for _, char := range in {
		if char == '-' {
			result.WriteRune('_')
		} else {
			result.WriteRune(char)
		}
	}

	return strings.ToUpper(result.String())
}

func getBoaTags(field reflect.StructField) []string {
	parts := strings.Split(field.Tag.Get("boa"), ",")
	results := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			results = append(results, part)
		}
	}
	return results
}

func isBoaIgnored(field reflect.StructField) bool {
	boaTags := getBoaTags(field)
	return slices.Contains(boaTags, "ignore") ||
		slices.Contains(boaTags, "ignored") ||
		slices.Contains(boaTags, "-")
}

func traverse(
	ctx *processingContext,
	structPtr any,
	fParam func(param Param, paramFieldName string, tags reflect.StructTag) error,
	fStruct func(structPtr any) error,
	prefixParts ...string,
) error {
	prefix := strings.Join(prefixParts, "")

	if reflect.TypeOf(structPtr).Kind() != reflect.Ptr {
		return fmt.Errorf("expected pointer to struct")
	}

	if reflect.TypeOf(structPtr).Elem().Kind() != reflect.Struct {
		return fmt.Errorf("expected pointer to struct")
	}

	if fStruct != nil {
		err := fStruct(structPtr)
		if err != nil {
			return err
		}
	}

	// use reflection to iterate over all fields of the struct
	fields := reflect.TypeOf(structPtr).Elem()
	rootValue := reflect.ValueOf(structPtr).Elem()
	for i := 0; i < fields.NumField(); i++ {
		field := fields.Field(i)

		if isBoaIgnored(field) {
			continue
		}

		fieldAddr := rootValue.Field(i).Addr()
		// check if field is a param
		param, isParam := fieldAddr.Interface().(Param)
		prefixedName := prefix + field.Name
		if isParam {
			if fParam != nil {
				err := fParam(param, prefixedName, field.Tag)
				if err != nil {
					return err
				}
			}
		} else {

			// check if it is a struct (but not time.Time which is a supported param type)
			if field.Type.Kind() == reflect.Struct && field.Type != timeType {
				// Named (non-anonymous) struct fields get auto-prefixed
				childPrefix := prefix
				if !field.Anonymous {
					childPrefix = prefix + field.Name
				}
				if err := traverse(ctx, fieldAddr.Interface(), fParam, fStruct, childPrefix); err != nil {
					return err
				}
				continue
			}

			// check if it is a pointer to a struct (but not *url.URL or pointer-to-supported-type)
			if field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct && !isSupportedType(field.Type) {
				if !fieldAddr.IsNil() && !fieldAddr.Elem().IsNil() {
					childPrefix := prefix
					if !field.Anonymous {
						childPrefix = prefix + field.Name
					}
					if err := traverse(ctx, fieldAddr.Elem().Interface(), fParam, fStruct, childPrefix); err != nil {
						return err
					}
				}
				continue
			}

			// For raw fields, we store parameter mirrors in the processing context
			if isSupportedType(field.Type) {

				// check if we already have a mirror for this field
				addr := fieldAddr.UnsafePointer()
				var ok bool
				if param, ok = ctx.RawAddrToMirror[addr]; !ok {
					param = newParam(&field, field.Type)
					// Set prefix for named struct nesting
					if prefix != "" {
						if pm, ok := param.(*paramMeta); ok {
							pm.flagPrefix = camelToKebabCase(prefix) + "-"
							pm.envPrefix = kebabCaseToUpperSnakeCase(pm.flagPrefix[:len(pm.flagPrefix)-1]) + "_"
						}
					}
					ctx.RawAddresses = append(ctx.RawAddresses, addr)
					ctx.RawAddrToMirror[addr] = param
				}

				if fParam != nil {
					err := fParam(param, prefixedName, field.Tag)
					if err != nil {
						return err
					}
				}

				continue
			}

			slog.Warn(fmt.Sprintf("field %s is not a type that is interpretable as a boa.Param. It will be ignored", field.Name))
			continue // not a param
		}
	}

	return nil
}

// toCobraBase sets up the cobra command with all common configuration (flags, validation, lifecycle hooks)
// but does NOT set the Run/RunE function. Returns both the command and the processing context
// so callers can set up the appropriate run function with access to the context.
// Returns an error if any setup/initialization fails.
func (b Cmd) toCobraBase() (*cobra.Command, *processingContext, error) {
	cmd := &cobra.Command{
		Use:           b.Use,
		Short:         b.Short,
		Long:          b.Long,
		Args:          b.Args,
		SilenceErrors: !b.UseCobraErrLog,
		ValidArgs:     b.ValidArgs,
	}

	if b.RawArgs != nil {
		cmd.SetArgs(b.RawArgs)
	}

	ctx := &processingContext{
		Context:         context.Background(), // prepare to override later?
		RawAddrToMirror: map[unsafe.Pointer]Param{},
		RawAddresses:    []unsafe.Pointer{},
	}

	// build mirrors
	if b.Params != nil {
		_ = traverse(ctx, b.Params, nil, func(innerParams any) error {
			return nil
		})
	}

	syncMirrors(ctx)

	// if b.params or any inner struct implements CfgStructInit, call it
	if b.Params != nil {
		err := traverse(ctx, b.Params, nil, func(innerParams any) error {
			if toInit, ok := b.Params.(CfgStructInit); ok {
				err := toInit.Init()
				if err != nil {
					return fmt.Errorf("error in CfgStructInit.Init(): %w", err)
				}
			}
			// context-aware interface
			if toInitCtx, ok := b.Params.(CfgStructInitCtx); ok {
				hookCtx := &HookContext{rawAddrToMirror: ctx.RawAddrToMirror}
				err := toInitCtx.InitCtx(hookCtx)
				if err != nil {
					return fmt.Errorf("error in CfgStructInitCtx.InitCtx(): %w", err)
				}
			}
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	}

	// if we have a custom init function, call it
	if b.InitFunc != nil {
		err := b.InitFunc(b.Params, cmd)
		if err != nil {
			return nil, nil, fmt.Errorf("error in InitFunc: %w", err)
		}
	}

	// if we have a context-aware init function, call it
	if b.InitFuncCtx != nil {
		hookCtx := &HookContext{rawAddrToMirror: ctx.RawAddrToMirror}
		err := b.InitFuncCtx(hookCtx, b.Params, cmd)
		if err != nil {
			return nil, nil, fmt.Errorf("error in InitFuncCtx: %w", err)
		}
	}

	syncMirrors(ctx)

	cmd.Flags().SortFlags = b.SortFlags
	cmd.Version = b.Version
	cmd.Aliases = b.Aliases
	cmd.GroupID = b.GroupID

	for _, subcommand := range b.SubCmds {
		cmd.AddCommand(subcommand)
	}

	// Add explicit groups first, tracking which IDs are defined
	definedGroups := make(map[string]bool)
	for _, group := range b.Groups {
		cmd.AddGroup(group)
		definedGroups[group.ID] = true
	}

	// Auto-generate groups for any subcommand GroupIDs not already defined
	for _, subcommand := range b.SubCmds {
		if subcommand.GroupID != "" && !definedGroups[subcommand.GroupID] {
			cmd.AddGroup(&cobra.Group{ID: subcommand.GroupID, Title: subcommand.GroupID + ":"})
			definedGroups[subcommand.GroupID] = true
		}
	}

	var positional []Param

	if b.Params != nil {

		// look in tags for info about positional args
		currentStructPtr := b.Params
		err := traverse(ctx, b.Params, func(param Param, _ string, tags reflect.StructTag) error {
			if tags.Get("positional") == "true" || tags.Get("pos") == "true" {
				param.setPositional(true)
			}
			if param.getDescr() == "" {
				if descr, ok := tags.Lookup("help"); ok {
					param.setDescription(descr)
				} else if descr, ok := tags.Lookup("desc"); ok {
					param.setDescription(descr)
				} else if descr, ok := tags.Lookup("descr"); ok {
					param.setDescription(descr)
				} else if descr, ok := tags.Lookup("description"); ok {
					param.setDescription(descr)
				}
			}
			if param.GetEnv() == "" {
				if env, ok := tags.Lookup("env"); ok {
					// Apply struct prefix to explicit env tags
					if pm, ok2 := param.(*paramMeta); ok2 && pm.envPrefix != "" {
						param.SetEnv(pm.envPrefix + env)
					} else {
						param.SetEnv(env)
					}
				}
			}
			if param.GetShort() == "" {
				if shrt, ok := tags.Lookup("short"); ok {
					param.SetShort(shrt)
				}
			}
			if param.GetName() == "" {
				if name, ok := tags.Lookup("name"); ok {
					// Apply struct prefix to explicit name tags
					if pm, ok2 := param.(*paramMeta); ok2 && pm.flagPrefix != "" {
						param.SetName(pm.flagPrefix + name)
					} else {
						param.SetName(name)
					}
				} else if name, ok := tags.Lookup("long"); ok {
					if pm, ok2 := param.(*paramMeta); ok2 && pm.flagPrefix != "" {
						param.SetName(pm.flagPrefix + name)
					} else {
						param.SetName(name)
					}
				}
			}

			setAlts := func(alts string) {
				strVal := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(alts), "["), "]")
				elements := strings.Split(strVal, ",")
				for i, element := range elements {
					elements[i] = strings.TrimSpace(element)
				}
				// Remove empty
				nonEmpty := make([]string, 0)
				for _, element := range elements {
					if element != "" {
						nonEmpty = append(nonEmpty, element)
					}
				}
				param.SetAlternatives(nonEmpty)
			}

			if alts, ok := tags.Lookup("alts"); ok {
				setAlts(alts)
			}
			if alts, ok := tags.Lookup("alternatives"); ok {
				setAlts(alts)
			}

			if strictAlts, ok := tags.Lookup("strict-alts"); ok {
				param.SetStrictAlts(strictAlts == "true")
			}
			if strictAlts, ok := tags.Lookup("strict"); ok {
				param.SetStrictAlts(strictAlts == "true")
			}

			if !param.hasDefaultValue() {
				// Default values are used for injection. So we can't just overwrite them
				if defaultPtr, ok := tags.Lookup("default"); ok {
					ptr, err := parsePtr(param.GetName(), param.GetType(), param.GetKind(), defaultPtr)
					if err != nil {
						return fmt.Errorf("invalid default value for param %s: %s", param.GetName(), err.Error())
					}
					param.SetDefault(ptr)
				}
			}

			// Parse min/max/pattern validation tags
			if pm, ok := param.(*paramMeta); ok {
				if minStr, ok := tags.Lookup("min"); ok {
					v, err := strconv.ParseFloat(minStr, 64)
					if err != nil {
						return fmt.Errorf("invalid min value for param %s: %s", param.GetName(), err.Error())
					}
					pm.minVal = &v
				}
				if maxStr, ok := tags.Lookup("max"); ok {
					v, err := strconv.ParseFloat(maxStr, 64)
					if err != nil {
						return fmt.Errorf("invalid max value for param %s: %s", param.GetName(), err.Error())
					}
					pm.maxVal = &v
				}
				if pat, ok := tags.Lookup("pattern"); ok {
					pm.pattern = pat
				}
			}

			// Detect configfile tag
			if cfgTag, ok := tags.Lookup("configfile"); ok && cfgTag == "true" {
				if param.GetType().Kind() != reflect.String {
					return fmt.Errorf("configfile tag on param %s: must be a string field", param.GetName())
				}
				ctx.ConfigFiles = append(ctx.ConfigFiles, configFileEntry{
					mirror: param,
					target: currentStructPtr,
				})
			}

			return nil
		}, func(structPtr any) error {
			currentStructPtr = structPtr
			return nil
		})

		if err != nil {
			return nil, nil, fmt.Errorf("error parsing tags: %w", err)
		}

		if b.ParamEnrich == nil {
			b.ParamEnrich = ParamEnricherDefault
		}
		processed := make([]Param, 0)
		err = traverse(ctx, b.Params, func(param Param, paramFieldName string, _ reflect.StructTag) error {
			err := b.ParamEnrich(processed, param, paramFieldName)
			if err != nil {
				return err
			}
			processed = append(processed, param)
			return nil
		}, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("error enriching params: %s", err.Error())
		}

		for _, param := range processed {
			if param.isPositional() {
				// if the last positional is a slice, error out
				if len(positional) >= 1 {
					lastPos := positional[len(positional)-1]
					if lastPos.GetKind() == reflect.Slice {
						return nil, nil, fmt.Errorf("positional param %s cannot come after slice positional param %s", param.GetName(), lastPos.GetName())
					}
				}
				positional = append(positional, param)
			}
		}

		// Check that no required positional arg exists after on optional positional arg
		numReqPositional := 0
		allowArbitraryNumPositional := false
		for i, param := range positional {
			if param.GetKind() == reflect.Slice {
				allowArbitraryNumPositional = true
			}
			if param.IsRequired() {
				numReqPositional++
			}
			if param.IsRequired() && i >= 1 {
				prev := positional[i-1]
				if !prev.IsRequired() {
					return nil, nil, fmt.Errorf("required positional arg %s must come before optional positional arg %s", param.GetName(), prev.GetName())
				}
			}
		}

		if cmd.Args == nil {
			if allowArbitraryNumPositional {
				cmd.Args = wrapArgsValidator(cobra.MinimumNArgs(numReqPositional))
			} else {
				cmd.Args = wrapArgsValidator(cobra.RangeArgs(numReqPositional, len(positional)))
			}
		} else {
			cmd.Args = wrapArgsValidator(cmd.Args)
		}

		syncMirrors(ctx)

		err = traverse(ctx, b.Params, func(param Param, _ string, tags reflect.StructTag) error {
			err := connect(param, cmd, positional, ctx)
			if err != nil {
				return err
			}

			return nil
		}, nil)

		// if b.Params implements CfgStructPostCreate, call it
		if postCreate, ok := b.Params.(CfgStructPostCreate); ok {
			if err := postCreate.PostCreate(); err != nil {
				return nil, nil, fmt.Errorf("error in CfgStructPostCreate.PostCreate(): %w", err)
			}
		}
		if postCreateCtx, ok := b.Params.(CfgStructPostCreateCtx); ok {
			hookCtx := &HookContext{rawAddrToMirror: ctx.RawAddrToMirror}
			if err := postCreateCtx.PostCreateCtx(hookCtx); err != nil {
				return nil, nil, fmt.Errorf("error in CfgStructPostCreateCtx.PostCreateCtx(): %w", err)
			}
		}
		if b.PostCreateFunc != nil {
			err := b.PostCreateFunc(b.Params, cmd)
			if err != nil {
				return nil, nil, fmt.Errorf("error in PostCreateFunc: %w", err)
			}
		}
		if b.PostCreateFuncCtx != nil {
			hookCtx := &HookContext{rawAddrToMirror: ctx.RawAddrToMirror}
			err := b.PostCreateFuncCtx(hookCtx, b.Params, cmd)
			if err != nil {
				return nil, nil, fmt.Errorf("error in PostCreateFuncCtx: %w", err)
			}
		}
		if err != nil {
			return nil, nil, fmt.Errorf("error connecting params: %s", err.Error())
		}
	}

	// Build ValidArgsFunction from per-positional-param Alternatives/AlternativesFunc
	// and/or the user-provided ValidArgsFunc. Per-param completions are checked first
	// for the current position; the user's ValidArgsFunc is used as fallback.
	{
		hasPositionalCompletion := false
		for _, p := range positional {
			if p.GetAlternatives() != nil || p.GetAlternativesFunc() != nil {
				hasPositionalCompletion = true
				break
			}
		}
		if hasPositionalCompletion || b.ValidArgsFunc != nil {
			userValidArgsFunc := b.ValidArgsFunc
			cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				syncMirrors(ctx)
				posIdx := len(args)
				if posIdx < len(positional) {
					p := positional[posIdx]
					if p.GetAlternativesFunc() != nil {
						return p.GetAlternativesFunc()(cmd, args, toComplete), cobra.ShellCompDirectiveDefault
					}
					if p.GetAlternatives() != nil {
						return p.GetAlternatives(), cobra.ShellCompDirectiveDefault
					}
				}
				if userValidArgsFunc != nil {
					return userValidArgsFunc(cmd, args, toComplete)
				}
				return nil, cobra.ShellCompDirectiveDefault
			}
		}
	}

	// now wrap the run function of the command to validate the flags
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if b.Params != nil {

			// Must read env values before running any prevalidate code
			if err := parseEnv(ctx, b.Params); err != nil {
				return err
			}

			syncMirrors(ctx)

			// Auto-load config files tagged with configfile:"true".
			// Substruct configs load first, root config loads last (root overrides inner).
			// Priority: CLI > env > root config > substruct config > defaults
			if len(ctx.ConfigFiles) > 0 {
				// Separate root and substruct entries
				var subEntries, rootEntries []configFileEntry
				for _, entry := range ctx.ConfigFiles {
					if entry.target == b.Params {
						rootEntries = append(rootEntries, entry)
					} else {
						subEntries = append(subEntries, entry)
					}
				}
				// Load substruct configs first
				for _, entry := range subEntries {
					if entry.mirror.HasValue() {
						filePath := *(entry.mirror.valuePtrF().(*string))
						if filePath != "" {
							if err := loadConfigFileInto(filePath, entry.target, b.ConfigUnmarshal); err != nil {
								return NewUserInputError(fmt.Errorf("configfile %s: %w", entry.mirror.GetName(), err))
							}
						}
					}
				}
				// Then load root config (overrides substruct values)
				for _, entry := range rootEntries {
					if entry.mirror.HasValue() {
						filePath := *(entry.mirror.valuePtrF().(*string))
						if filePath != "" {
							if err := loadConfigFileInto(filePath, entry.target, b.ConfigUnmarshal); err != nil {
								return NewUserInputError(fmt.Errorf("configfile %s: %w", entry.mirror.GetName(), err))
							}
						}
					}
				}
				syncMirrors(ctx)
			}

			// if b.params or any inner struct implements CfgStructPreValidate, call it
			err := traverse(ctx, b.Params, nil, func(innerParams any) error {
				if s, ok := innerParams.(CfgStructPreValidate); ok {
					err := s.PreValidate()
					if err != nil {
						return fmt.Errorf("error in PreValidate: %w", err)
					}
				}
				// context-aware interface
				if s, ok := innerParams.(CfgStructPreValidateCtx); ok {
					hookCtx := &HookContext{rawAddrToMirror: ctx.RawAddrToMirror}
					err := s.PreValidateCtx(hookCtx)
					if err != nil {
						return fmt.Errorf("error in PreValidateCtx: %w", err)
					}
				}
				return nil
			})
			if err != nil {
				return err
			}

			// if we have a custom pre-execute function, call it
			if b.PreValidateFunc != nil {
				err := b.PreValidateFunc(b.Params, cmd, args)
				if err != nil {
					return fmt.Errorf("error in PreValidateFunc: %w", err)
				}
			}

			// if we have a context-aware pre-validate function, call it
			if b.PreValidateFuncCtx != nil {
				hookCtx := &HookContext{rawAddrToMirror: ctx.RawAddrToMirror}
				err := b.PreValidateFuncCtx(hookCtx, b.Params, cmd, args)
				if err != nil {
					return fmt.Errorf("error in PreValidateFuncCtx: %w", err)
				}
			}

			syncMirrors(ctx)

			if err = validate(ctx, b.Params); err != nil {
				return err
			}

			// Sync mirrors again after validation to copy converted values (e.g., *url.URL from string)
			syncMirrors(ctx)

			// if b.params or any inner struct implements CfgStructPreExecute, call it
			err = traverse(ctx, b.Params, nil, func(innerParams any) error {
				if preExecute, ok := innerParams.(CfgStructPreExecute); ok {
					err := preExecute.PreExecute()
					if err != nil {
						return fmt.Errorf("error in PreExecute: %w", err)
					}
				}
				// context-aware interface
				if preExecuteCtx, ok := innerParams.(CfgStructPreExecuteCtx); ok {
					hookCtx := &HookContext{rawAddrToMirror: ctx.RawAddrToMirror}
					err := preExecuteCtx.PreExecuteCtx(hookCtx)
					if err != nil {
						return fmt.Errorf("error in PreExecuteCtx: %w", err)
					}
				}
				return nil
			})
			if err != nil {
				return err
			}

			// if we have a custom pre-execute function, call it
			if b.PreExecuteFunc != nil {
				err := b.PreExecuteFunc(b.Params, cmd, args)
				if err != nil {
					return fmt.Errorf("error in PreExecuteFunc: %w", err)
				}
			}

			// if we have a context-aware pre-execute function, call it
			if b.PreExecuteFuncCtx != nil {
				hookCtx := &HookContext{rawAddrToMirror: ctx.RawAddrToMirror}
				err := b.PreExecuteFuncCtx(hookCtx, b.Params, cmd, args)
				if err != nil {
					return fmt.Errorf("error in PreExecuteFuncCtx: %w", err)
				}
			}

		}
		return nil
	}

	return cmd, ctx, nil
}

// validateRunFuncs checks that at most one run function is set and returns an error if more than one is configured.
func (b Cmd) validateRunFuncs() error {
	runFuncCount := 0
	if b.RunFunc != nil {
		runFuncCount++
	}
	if b.RunFuncCtx != nil {
		runFuncCount++
	}
	if b.RunFuncE != nil {
		runFuncCount++
	}
	if b.RunFuncCtxE != nil {
		runFuncCount++
	}
	if runFuncCount > 1 {
		return fmt.Errorf("cannot set multiple run functions (RunFunc, RunFuncCtx, RunFuncE, RunFuncCtxE) - use only one")
	}
	return nil
}

// toCobraImpl converts a Cmd to a cobra.Command.
// Always uses cmd.RunE internally so errors flow back through Execute() to runImpl().
func (b Cmd) toCobraImpl() *cobra.Command {
	if err := b.validateRunFuncs(); err != nil {
		panic(err)
	}
	cmd, ctx, err := b.toCobraBase()
	if err != nil {
		panic(err)
	}

	// Always use RunE so errors flow back through Execute() to runImpl()
	// This ensures consistent error handling (UserInputError -> exit(1), others -> panic)
	if b.RunFunc != nil {
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			b.RunFunc(cmd, args)
			return nil
		}
	} else if b.RunFuncCtx != nil {
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			hookCtx := &HookContext{rawAddrToMirror: ctx.RawAddrToMirror}
			b.RunFuncCtx(hookCtx, cmd, args)
			return nil
		}
	} else if b.RunFuncE != nil {
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			return b.RunFuncE(cmd, args)
		}
	} else if b.RunFuncCtxE != nil {
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			hookCtx := &HookContext{rawAddrToMirror: ctx.RawAddrToMirror}
			return b.RunFuncCtxE(hookCtx, cmd, args)
		}
	}

	return cmd
}

// toCobraImplE converts a Cmd to a cobra.Command using cmd.RunE (returns error).
// Returns (*cobra.Command, error) to propagate setup errors.
func (b Cmd) toCobraImplE() (*cobra.Command, error) {
	if err := b.validateRunFuncs(); err != nil {
		panic(err) // API misuse - should be caught during development
	}
	cmd, ctx, err := b.toCobraBase()
	if err != nil {
		return nil, err
	}

	// Set the RunE function based on which variant is configured
	if b.RunFuncE != nil {
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			return b.RunFuncE(cmd, args)
		}
	} else if b.RunFuncCtxE != nil {
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			hookCtx := &HookContext{rawAddrToMirror: ctx.RawAddrToMirror}
			return b.RunFuncCtxE(hookCtx, cmd, args)
		}
	} else if b.RunFunc != nil {
		// Wrap non-E variant to return nil (no error)
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			b.RunFunc(cmd, args)
			return nil
		}
	} else if b.RunFuncCtx != nil {
		// Wrap non-E variant to return nil (no error)
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			hookCtx := &HookContext{rawAddrToMirror: ctx.RawAddrToMirror}
			b.RunFuncCtx(hookCtx, cmd, args)
			return nil
		}
	}

	return cmd, nil
}

func syncMirrors(ctx *processingContext) {
	// 1. First, copy non-zero values from the raw fields -> mirrors as injected values.
	// 2. Then copy back cli & env set values to the raw fields

	for _, rawAddr := range ctx.RawAddresses {
		mirror := ctx.RawAddrToMirror[rawAddr]

		// Check if this is a pointer field (e.g., *string, *int)
		pm, isParamMeta := mirror.(*paramMeta)
		if isParamMeta && pm.isPointer {
			syncPointerField(rawAddr, mirror, pm)
			continue
		}

		// Create a reflect.Value pointing to the raw field via its stored address
		ptrToRawValue := reflect.NewAt(mirror.GetType(), rawAddr)

		if !mirror.wasSetOnCli() && !mirror.wasSetByEnv() && !ptrToRawValue.Elem().IsZero() {
			mirror.injectValuePtr(ptrToRawValue.Interface())
		}

		if mirror.wasSetOnCli() || mirror.wasSetByEnv() || (mirror.HasValue() && ptrToRawValue.Elem().IsZero()) {
			mirrorValue := reflect.ValueOf(mirror.valuePtrF()).Elem()

			// Get the element that the pointer points to
			rawValue := ptrToRawValue.Elem()

			// Skip if types don't match (e.g., string vs *url.URL before conversion)
			// This allows syncing to work before and after validation/conversion
			if !mirrorValue.Type().AssignableTo(rawValue.Type()) {
				continue
			}

			// Make sure the destination is settable
			if rawValue.CanSet() {
				rawValue.Set(mirrorValue)
			} else {
				panic(fmt.Errorf("could not set value for parameter %s", mirror.GetName()))
			}
		}
	}
}

// syncPointerField handles bidirectional sync for pointer fields like *string, *int.
// rawAddr points to the pointer field itself (e.g., the *string field in the user's struct).
// The mirror stores the value type (string), while the raw field is a pointer (*string).
func syncPointerField(rawAddr unsafe.Pointer, mirror Param, _ *paramMeta) {
	// rawAddr points to the *string field. reflect.NewAt creates a **string pointing at it.
	ptrType := reflect.PointerTo(mirror.GetType()) // e.g., *string
	rawFieldPtr := reflect.NewAt(ptrType, rawAddr)  // **string pointing at the field
	rawFieldVal := rawFieldPtr.Elem()                // the *string field value itself

	// Raw → Mirror: if the user's pointer is non-nil, inject the pointed-to value
	if !mirror.wasSetOnCli() && !mirror.wasSetByEnv() && !rawFieldVal.IsNil() {
		// rawFieldVal.Interface() is *string — same type cobra uses
		mirror.injectValuePtr(rawFieldVal.Interface())
	}

	// Mirror → Raw: if mirror has a value, set the pointer field
	if mirror.wasSetOnCli() || mirror.wasSetByEnv() || (mirror.HasValue() && rawFieldVal.IsNil()) {
		valPtr := mirror.valuePtrF()
		if valPtr != nil {
			mirrorVal := reflect.ValueOf(valPtr) // *string from cobra
			// Skip if types don't match (e.g., string vs *url.URL before conversion)
			if mirrorVal.Type().AssignableTo(rawFieldVal.Type()) {
				if rawFieldVal.CanSet() {
					rawFieldVal.Set(mirrorVal)
				}
			}
		}
	}
}

func runImpl(cmd *cobra.Command, handler resultHandler) {

	if handler.Panic != nil {
		defer func() {
			if r := recover(); r != nil {
				handler.Panic(r)
			}
		}()
	}

	err := cmd.Execute()
	if err != nil {
		if handler.Failure != nil {
			handler.Failure(err)
		} else {
			// For expected user input errors (missing required params, invalid values, etc.),
			// print the error and exit cleanly (no stack trace).
			// Only panic for unexpected errors (programming errors, runtime failures).
			if IsUserInputError(err) {
				fmt.Fprintln(os.Stderr, "Error:", err.Error())
				osExit(1)
				return // osExit may be mocked in tests, so we need to return explicitly
			}
			panic(err)
		}
	} else {
		if handler.Success != nil {
			handler.Success()
		}
	}
}

func isSupportedType(t reflect.Type) bool {
	// Exact type match (time.Time, time.Duration, net.IP, *url.URL)
	if _, ok := exactTypeHandlers[t]; ok {
		return true
	}
	// Kind-based match (string, int, bool, float, etc.)
	if _, ok := kindHandlers[t.Kind()]; ok {
		return true
	}
	// Map types — all map[string]V types are supported (native pflag or JSON fallback)
	if t.Kind() == reflect.Map && t.Key().Kind() == reflect.String {
		return true
	}
	// Slice types — all slices are supported (native pflag for basic elements, JSON fallback for complex)
	if t.Kind() == reflect.Slice {
		return true
	}
	// Pointer-to-supported-type (e.g., *string, *int, *bool)
	if t.Kind() == reflect.Pointer {
		return isSupportedType(t.Elem())
	}
	return false
}

// normalizeType converts type aliases to their base types for cobra compatibility.
// For example, `type MyString string` returns reflect.TypeOf("") (string).
// Special types (time.Time, time.Duration, net.IP, *url.URL) are returned as-is.
func normalizeType(t reflect.Type) reflect.Type {
	// Exact-match handlers have their own baseType (special types stay as-is)
	if handler, ok := exactTypeHandlers[t]; ok {
		return handler.baseType
	}

	// For slices, normalize via slice handlers
	if t.Kind() == reflect.Slice {
		elem := t.Elem()
		// Special slice element types stay as-is
		if _, ok := sliceExactTypeHandlers[elem]; ok {
			return t
		}
		// Normalize basic slice element types
		normElem := normalizeType(elem)
		if normElem != elem {
			return reflect.SliceOf(normElem)
		}
		return t
	}

	// Kind-based handlers provide the baseType for basic types + aliases
	if handler, ok := kindHandlers[t.Kind()]; ok {
		return handler.baseType
	}

	return t
}

func newParam(field *reflect.StructField, t reflect.Type) Param {
	// Determine if this is a pointer-to-value field (e.g., *string, *int)
	// Note: *url.URL is NOT treated as a pointer field — it's a specific supported type
	isPtr := t.Kind() == reflect.Pointer && t != urlPtrType
	valueType := t
	if isPtr {
		valueType = t.Elem()
	}

	// Normalize type aliases to their base types for cobra compatibility.
	// e.g., `type MyString string` → store as string, since cobra's StringP returns *string.
	// This matches the old required[T] behavior where newParam always created required[string]{},
	// required[int]{}, etc. regardless of whether the field was a type alias.
	valueType = normalizeType(valueType)

	// Pointer, map, and nested slice fields default to optional (nil = not set)
	isRequired := !cfg.defaultOptional
	if isPtr || valueType.Kind() == reflect.Map {
		isRequired = false
	}
	// Nested slices ([][]T) default to optional — flat slices keep the global default
	if valueType.Kind() == reflect.Slice && valueType.Elem().Kind() == reflect.Slice {
		isRequired = false
	}

	if requiredTag, ok := field.Tag.Lookup("required"); ok {
		switch requiredTag {
		case "true":
			isRequired = true
		case "false":
			isRequired = false
		default:
			panic(fmt.Errorf("invalid value for field %s's required tag: %s", field.Name, requiredTag))
		}
	}
	if requiredTag, ok := field.Tag.Lookup("req"); ok {
		switch requiredTag {
		case "true":
			isRequired = true
		case "false":
			isRequired = false
		default:
			panic(fmt.Errorf("invalid value for field %s's required tag: %s", field.Name, requiredTag))
		}
	}
	if optionalTag, ok := field.Tag.Lookup("optional"); ok {
		switch optionalTag {
		case "true":
			isRequired = false
		case "false":
			isRequired = true
		default:
			panic(fmt.Errorf("invalid value for field %s's optional tag: %s", field.Name, optionalTag))
		}
	}
	if optionalTag, ok := field.Tag.Lookup("opt"); ok {
		switch optionalTag {
		case "true":
			isRequired = false
		case "false":
			isRequired = true
		default:
			panic(fmt.Errorf("invalid value for field %s's optional tag: %s", field.Name, optionalTag))
		}
	}

	return &paramMeta{
		fieldType:       valueType,
		isPointer:       isPtr,
		defaultRequired: isRequired,
	}
}

var timeType = reflect.TypeOf(time.Time{})
var durationType = reflect.TypeOf(time.Duration(0))
var ipType = reflect.TypeOf(net.IP{})
var urlPtrType = reflect.TypeOf((*url.URL)(nil))
