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
	descr() string
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

type processingContext struct {
	context.Context
	RawAddrToMirror map[uintptr]Param
	// We need to keep track of raw params, so we can
	// override the raw values with cli values in case
	// the user may have mapped config files to the params
	// as well - since the config file deserialization will
	// not be aware of the raw values, and just overwrite them.
	RawAddresses []uintptr
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

		// special types validation for types stored as strings (time.Time, *url.URL)
		if HasValue(param) {
			if param.GetKind() == reflect.Struct {
				if param.GetType() == timeType {
					strVal := *param.valuePtrF().(*string)
					res, err := parsePtr(param.GetName(), param.GetType(), param.GetKind(), strVal)
					if err != nil {
						return fmt.Errorf("invalid value for param '%s': %s", param.GetName(), err.Error())
					}
					param.setValuePtr(res)
				}
			} else if param.GetKind() == reflect.Pointer && param.GetType() == urlPtrType {
				// Check if the value is still a string (needs conversion) or already converted
				if strPtr, ok := param.valuePtrF().(*string); ok {
					strVal := *strPtr
					res, err := parsePtr(param.GetName(), param.GetType(), param.GetKind(), strVal)
					if err != nil {
						return fmt.Errorf("invalid value for param '%s': %s", param.GetName(), err.Error())
					}
					param.setValuePtr(res)
				}
				// If it's already **url.URL, the conversion was already done
			} else if param.GetKind() == reflect.Slice {
				elem := param.GetType().Elem()
				// []time.Time - stored as []string, needs conversion
				if elem == timeType {
					if strSlice, ok := param.valuePtrF().(*[]string); ok && strSlice != nil {
						times := make([]time.Time, len(*strSlice))
						for i, s := range *strSlice {
							t, err := parseTimeString(s)
							if err != nil {
								return fmt.Errorf("invalid value for param '%s' at index %d: %s", param.GetName(), i, err.Error())
							}
							times[i] = t
						}
						param.setValuePtr(&times)
					}
				}
				// []*url.URL - stored as []string, needs conversion
				if elem == urlPtrType {
					if strSlice, ok := param.valuePtrF().(*[]string); ok && strSlice != nil {
						urls := make([]*url.URL, len(*strSlice))
						for i, s := range *strSlice {
							u, err := url.Parse(s)
							if err != nil {
								return fmt.Errorf("invalid value for param '%s' at index %d: %s", param.GetName(), i, err.Error())
							}
							urls[i] = u
						}
						param.setValuePtr(&urls)
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
		}

		return nil
	}, nil)
	return newUserInputError(err)
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

func toTypedSlice[T SupportedTypes](slice any) []T {
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

	descr := f.descr()
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

	switch f.GetKind() {
	case reflect.String:
		def := ""
		if f.hasDefaultValue() {
			defVal := reflect.ValueOf(f.defaultValuePtr()).Elem()
			def = defVal.Convert(reflect.TypeOf(def)).Interface().(string)
		}
		f.setValuePtr(cmd.Flags().StringP(f.GetName(), f.GetShort(), def, descr))
		return nil
	case reflect.Int:
		def := 0
		if f.hasDefaultValue() {
			defVal := reflect.ValueOf(f.defaultValuePtr()).Elem()
			def = defVal.Convert(reflect.TypeOf(def)).Interface().(int)
		}
		f.setValuePtr(cmd.Flags().IntP(f.GetName(), f.GetShort(), def, descr))
		return nil
	case reflect.Int32:
		def := int32(0)
		if f.hasDefaultValue() {
			defVal := reflect.ValueOf(f.defaultValuePtr()).Elem()
			def = defVal.Convert(reflect.TypeOf(def)).Interface().(int32)
		}
		f.setValuePtr(cmd.Flags().Int32P(f.GetName(), f.GetShort(), def, descr))
		return nil
	case reflect.Int64:
		// Check if this is a time.Duration (which has underlying type int64)
		if f.GetType() == durationType {
			def := time.Duration(0)
			if f.hasDefaultValue() {
				defVal := reflect.ValueOf(f.defaultValuePtr()).Elem()
				def = time.Duration(defVal.Int())
			}
			f.setValuePtr(cmd.Flags().DurationP(f.GetName(), f.GetShort(), def, descr))
			return nil
		}
		def := int64(0)
		if f.hasDefaultValue() {
			defVal := reflect.ValueOf(f.defaultValuePtr()).Elem()
			def = defVal.Convert(reflect.TypeOf(def)).Interface().(int64)
		}
		f.setValuePtr(cmd.Flags().Int64P(f.GetName(), f.GetShort(), def, descr))
		return nil
	case reflect.Float64:
		def := 0.0
		if f.hasDefaultValue() {
			defVal := reflect.ValueOf(f.defaultValuePtr()).Elem()
			def = defVal.Convert(reflect.TypeOf(def)).Interface().(float64)
		}
		f.setValuePtr(cmd.Flags().Float64P(f.GetName(), f.GetShort(), def, descr))
		return nil
	case reflect.Float32:
		def := float32(0.0)
		if f.hasDefaultValue() {
			defVal := reflect.ValueOf(f.defaultValuePtr()).Elem()
			def = defVal.Convert(reflect.TypeOf(def)).Interface().(float32)
		}
		f.setValuePtr(cmd.Flags().Float32P(f.GetName(), f.GetShort(), def, descr))
		return nil
	case reflect.Bool:
		def := false
		if f.hasDefaultValue() {
			defVal := reflect.ValueOf(f.defaultValuePtr()).Elem()
			def = defVal.Convert(reflect.TypeOf(def)).Interface().(bool)
		}
		f.setValuePtr(cmd.Flags().BoolP(f.GetName(), f.GetShort(), def, descr))
		return nil
	case reflect.Struct:
		if f.GetType() == timeType {
			if f.hasDefaultValue() {
				def := *reflect.ValueOf(f.defaultValuePtr()).Interface().(*time.Time)
				f.setValuePtr(cmd.Flags().StringP(f.GetName(), f.GetShort(), def.Format(time.RFC3339), descr))
			} else {
				f.setValuePtr(cmd.Flags().StringP(f.GetName(), f.GetShort(), "", descr))
			}
			return nil
		} else {
			return fmt.Errorf("general structs not yet supported: %s", f.GetKind().String())
		}
	case reflect.Slice:
		// Check if this is net.IP (which is []byte)
		if f.GetType() == ipType {
			var def net.IP
			if f.hasDefaultValue() {
				defVal := reflect.ValueOf(f.defaultValuePtr()).Elem()
				def = defVal.Interface().(net.IP)
			}
			f.setValuePtr(cmd.Flags().IPP(f.GetName(), f.GetShort(), def, descr))
			return nil
		}

		elemType := f.GetType().Elem()

		// Check for special slice types first
		// []net.IP - slice of IP addresses
		if elemType == ipType {
			var def []net.IP
			if f.hasDefaultValue() {
				defVal := reflect.ValueOf(f.defaultValuePtr()).Elem()
				def = defVal.Interface().([]net.IP)
			}
			f.setValuePtr(cmd.Flags().IPSliceP(f.GetName(), f.GetShort(), def, descr))
			return nil
		}

		// []time.Duration - slice of durations
		if elemType == durationType {
			var def []time.Duration
			if f.hasDefaultValue() {
				defVal := reflect.ValueOf(f.defaultValuePtr()).Elem()
				if defVal.Kind() == reflect.Slice {
					def = defVal.Interface().([]time.Duration)
				} else {
					// Parse from string
					parsed, err := parseSliceSpecial(f.GetName(), f.defaultValueStr(), elemType)
					if err != nil {
						return fmt.Errorf("default value for slice param '%s' is invalid: %s", f.GetName(), err.Error())
					}
					def = parsed.([]time.Duration)
					f.SetDefault(&def)
				}
			}
			f.setValuePtr(cmd.Flags().DurationSliceP(f.GetName(), f.GetShort(), def, descr))
			return nil
		}

		// []time.Time - stored as string slice, converted later
		if elemType == timeType {
			var def []string
			if f.hasDefaultValue() {
				defVal := reflect.ValueOf(f.defaultValuePtr()).Elem()
				if defVal.Kind() == reflect.Slice && defVal.Type().Elem() == timeType {
					// Convert []time.Time to []string for storage
					times := defVal.Interface().([]time.Time)
					def = make([]string, len(times))
					for i, t := range times {
						def[i] = t.Format(time.RFC3339)
					}
				}
			}
			f.setValuePtr(cmd.Flags().StringSliceP(f.GetName(), f.GetShort(), def, descr))
			return nil
		}

		// []*url.URL - stored as string slice, converted later
		if elemType == urlPtrType {
			var def []string
			if f.hasDefaultValue() {
				defVal := reflect.ValueOf(f.defaultValuePtr()).Elem()
				if defVal.Kind() == reflect.Slice && defVal.Type().Elem() == urlPtrType {
					// Convert []*url.URL to []string for storage
					urls := defVal.Interface().([]*url.URL)
					def = make([]string, len(urls))
					for i, u := range urls {
						if u != nil {
							def[i] = u.String()
						}
					}
				}
			}
			f.setValuePtr(cmd.Flags().StringSliceP(f.GetName(), f.GetShort(), def, descr))
			return nil
		}

		var defaultValueSlice any = nil
		var err error
		if f.hasDefaultValue() {
			defaultValueSlice = reflect.ValueOf(f.defaultValuePtr()).Elem().Interface()
			// if it already has the correct type, dont repeat
			if reflect.TypeOf(f.defaultValuePtr()).Elem().Kind() != reflect.Slice {
				defaultValueSlice, err = parseSlice(f.GetName(), f.defaultValueStr(), elemType)
				if err != nil {
					return fmt.Errorf("default value for slice param '%s' is invalid: %s", f.GetName(), err.Error())
				}
				f.SetDefault(defaultValueSlice)
			}
		}

		switch elemType.Kind() {
		case reflect.String:
			f.setValuePtr(cmd.Flags().StringSliceP(f.GetName(), f.GetShort(), toTypedSlice[string](defaultValueSlice), descr))
		case reflect.Int:
			f.setValuePtr(cmd.Flags().IntSliceP(f.GetName(), f.GetShort(), toTypedSlice[int](defaultValueSlice), descr))
		case reflect.Int32:
			f.setValuePtr(cmd.Flags().Int32SliceP(f.GetName(), f.GetShort(), toTypedSlice[int32](defaultValueSlice), descr))
		case reflect.Int64:
			f.setValuePtr(cmd.Flags().Int64SliceP(f.GetName(), f.GetShort(), toTypedSlice[int64](defaultValueSlice), descr))
		case reflect.Float32:
			f.setValuePtr(cmd.Flags().Float32SliceP(f.GetName(), f.GetShort(), toTypedSlice[float32](defaultValueSlice), descr))
		case reflect.Float64:
			f.setValuePtr(cmd.Flags().Float64SliceP(f.GetName(), f.GetShort(), toTypedSlice[float64](defaultValueSlice), descr))
		case reflect.Bool:
			f.setValuePtr(cmd.Flags().BoolSliceP(f.GetName(), f.GetShort(), toTypedSlice[bool](defaultValueSlice), descr))
		default:
			return fmt.Errorf("unsupported slice element type '%v'. Check parameter '%s'", elemType, f.GetName())
		}
		return nil
	case reflect.Array:
		return fmt.Errorf("unsupported param type (Array): %s: ", f.GetKind().String())
	case reflect.Pointer:
		// Check if this is *url.URL
		if f.GetType() == urlPtrType {
			def := ""
			if f.hasDefaultValue() {
				defVal := reflect.ValueOf(f.defaultValuePtr()).Elem()
				if !defVal.IsNil() {
					def = defVal.Interface().(*url.URL).String()
				}
			}
			f.setValuePtr(cmd.Flags().StringP(f.GetName(), f.GetShort(), def, descr))
			return nil
		}
		return fmt.Errorf("unsupported param type (Pointer): %s: ", f.GetKind().String())
	default:
		return fmt.Errorf("unsupported param type: %s", f.GetKind().String())
	}
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

func parseSlice(
	name string,
	strVal string,
	elemType reflect.Type,
) (any, error) {

	isEmptySlice := strVal == "[]"

	// remove any brackets
	strVal = strings.TrimSuffix(strings.TrimPrefix(strVal, "["), "]")

	parts := strings.Split(strVal, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	switch elemType.Kind() {
	case reflect.String:

		if isEmptySlice {
			return &[]string{}, nil
		}

		return &parts, nil
	case reflect.Int:
		out := make([]int, len(parts))

		if isEmptySlice {
			return &out, nil
		}

		for i, part := range parts {
			parsedInt, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			out[i] = parsedInt
		}
		return &out, nil
	case reflect.Int32:
		out := make([]int32, len(parts))

		if isEmptySlice {
			return &out, nil
		}

		for i, part := range parts {
			parsedInt64, err := strconv.ParseInt(part, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			out[i] = int32(parsedInt64)
		}
		return &out, nil
	case reflect.Int64:
		out := make([]int64, len(parts))

		if isEmptySlice {
			return &out, nil
		}

		for i, part := range parts {
			parsedInt64, err := strconv.ParseInt(part, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			out[i] = parsedInt64
		}
		return &out, nil
	case reflect.Float32:
		out := make([]float32, len(parts))

		if isEmptySlice {
			return &out, nil
		}

		for i, part := range parts {
			parsedFloat64, err := strconv.ParseFloat(part, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			out[i] = float32(parsedFloat64)
		}
		return &out, nil
	case reflect.Float64:
		out := make([]float64, len(parts))

		if isEmptySlice {
			return &out, nil
		}

		for i, part := range parts {
			parsedFloat64, err := strconv.ParseFloat(part, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			out[i] = parsedFloat64
		}
		return &out, nil
	case reflect.Bool:
		out := make([]bool, len(parts))

		if isEmptySlice {
			return &out, nil
		}

		for i, part := range parts {
			parsedBool, err := strconv.ParseBool(part)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			out[i] = parsedBool
		}
		return &out, nil
	default:
		return nil, fmt.Errorf("unsupported slice element type '%v'. Check parameter '%s'", elemType, name)
	}
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

// parseSliceSpecial parses slices of special types (time.Duration, time.Time, net.IP, *url.URL)
func parseSliceSpecial(
	name string,
	strVal string,
	elemType reflect.Type,
) (any, error) {
	isEmptySlice := strVal == "[]"

	// remove any brackets
	strVal = strings.TrimSuffix(strings.TrimPrefix(strVal, "["), "]")

	parts := strings.Split(strVal, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}

	//nolint:staticcheck // can't use tagged switch with reflect.Type comparisons
	switch {
	case elemType == durationType:
		out := make([]time.Duration, len(parts))
		if isEmptySlice {
			return out, nil
		}
		for i, part := range parts {
			d, err := time.ParseDuration(part)
			if err != nil {
				return nil, fmt.Errorf("invalid duration value for param %s: %s", name, err.Error())
			}
			out[i] = d
		}
		return out, nil

	case elemType == timeType:
		out := make([]time.Time, len(parts))
		if isEmptySlice {
			return out, nil
		}
		for i, part := range parts {
			t, err := parseTimeString(part)
			if err != nil {
				return nil, fmt.Errorf("invalid time value for param %s: %s", name, err.Error())
			}
			out[i] = t
		}
		return out, nil

	case elemType == ipType:
		out := make([]net.IP, len(parts))
		if isEmptySlice {
			return out, nil
		}
		for i, part := range parts {
			ip := net.ParseIP(part)
			if ip == nil {
				return nil, fmt.Errorf("invalid IP address for param %s: %s", name, part)
			}
			out[i] = ip
		}
		return out, nil

	case elemType == urlPtrType:
		out := make([]*url.URL, len(parts))
		if isEmptySlice {
			return out, nil
		}
		for i, part := range parts {
			u, err := url.Parse(part)
			if err != nil {
				return nil, fmt.Errorf("invalid URL for param %s: %s", name, err.Error())
			}
			out[i] = u
		}
		return out, nil

	default:
		return nil, fmt.Errorf("unsupported special slice element type '%v'. Check parameter '%s'", elemType, name)
	}
}

func parsePtr(
	name string,
	tpe reflect.Type,
	kind reflect.Kind,
	strVal string,
) (any, error) {

	switch kind {
	case reflect.String:
		return &strVal, nil
	case reflect.Int:
		parsedInt, err := strconv.Atoi(strVal)
		if err != nil {
			return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
		}
		return &parsedInt, nil
	case reflect.Int32:
		parsedInt64, err := strconv.ParseInt(strVal, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
		}
		parsedInt32 := int32(parsedInt64)
		return &parsedInt32, nil
	case reflect.Int64:
		// Check if this is time.Duration
		if tpe == durationType {
			parsedDuration, err := time.ParseDuration(strVal)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			return &parsedDuration, nil
		}
		parsedInt64, err := strconv.ParseInt(strVal, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
		}
		return &parsedInt64, nil
	case reflect.Float32:
		parsedFloat64, err := strconv.ParseFloat(strVal, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
		}
		parsedFloat32 := float32(parsedFloat64)
		return &parsedFloat32, nil
	case reflect.Float64:
		parsedFloat64, err := strconv.ParseFloat(strVal, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
		}
		return &parsedFloat64, nil
	case reflect.Bool:
		parsedBool, err := strconv.ParseBool(strVal)
		if err != nil {
			return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
		}
		return &parsedBool, nil
	case reflect.Struct:
		if tpe == timeType {
			parsedTime, err := time.Parse(time.RFC3339, strVal)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			return &parsedTime, nil
		} else {
			return nil, fmt.Errorf("general structs not yet supported: %s", tpe.String())
		}
	case reflect.Slice:
		// Check if this is net.IP (single IP, which is []byte)
		if tpe == ipType {
			parsedIP := net.ParseIP(strVal)
			if parsedIP == nil {
				return nil, fmt.Errorf("invalid IP address for param %s: %s", name, strVal)
			}
			return &parsedIP, nil
		}
		// Check for special slice element types
		elem := tpe.Elem()
		if elem == durationType || elem == timeType || elem == ipType || elem == urlPtrType {
			parsed, err := parseSliceSpecial(name, strVal, elem)
			if err != nil {
				return nil, err
			}
			// parseSliceSpecial returns value, not pointer - wrap it
			result := reflect.New(tpe)
			result.Elem().Set(reflect.ValueOf(parsed))
			return result.Interface(), nil
		}
		return parseSlice(name, strVal, tpe.Elem())
	case reflect.Array:
		return nil, fmt.Errorf("arrays not supported param type. Use a slice instead: %s", kind.String())
	case reflect.Pointer:
		// Check if this is *url.URL
		if tpe == urlPtrType {
			if strVal == "" {
				return (*url.URL)(nil), nil
			}
			parsedURL, err := url.Parse(strVal)
			if err != nil {
				return nil, fmt.Errorf("invalid URL for param %s: %s", name, err.Error())
			}
			return &parsedURL, nil
		}
		return nil, fmt.Errorf("pointers not yet supported param type: %s", kind.String())
	default:
		return nil, fmt.Errorf("unsupported param type: %s", kind.String())
	}
}

func camelToKebabCase(in string) string {
	var result strings.Builder

	for _, char := range in {
		if unicode.IsUpper(char) {
			if result.Len() > 0 {
				result.WriteRune('-')
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
) error {

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
		if isParam {
			//if !param.IsEnabled() { // cant do here, because it is not known yet
			//	continue // this parameter is not enabled
			//}
			if fParam != nil {
				err := fParam(param, field.Name, field.Tag)
				if err != nil {
					return err
				}
			}
		} else {

			// check if it is a struct (but not time.Time which is a supported param type)
			if field.Type.Kind() == reflect.Struct && field.Type != timeType {
				if err := traverse(ctx, fieldAddr.Interface(), fParam, fStruct); err != nil {
					return err
				}
				continue
			}

			// check if it is a pointer to a struct (but not *url.URL which is a supported param type)
			if field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct && field.Type != urlPtrType {
				if !fieldAddr.IsNil() && !fieldAddr.Elem().IsNil() {
					if err := traverse(ctx, fieldAddr.Elem().Interface(), fParam, fStruct); err != nil {
						return err
					}
				}
				continue
			}

			if field.Type.Kind() == reflect.Pointer && field.Type != urlPtrType {
				slog.Warn(fmt.Sprintf("raw pointer types to parameters are not (yet?) supported. Field %s will be ignored", field.Name))
				continue
			}

			// For raw fields, we store parameter mirrors in the processing context
			if isSupportedType(field.Type) {

				// check if we already have a mirror for this field
				addr := fieldAddr.Pointer()
				var ok bool
				if param, ok = ctx.RawAddrToMirror[addr]; !ok {
					param = newParam(&field, field.Type)
					ctx.RawAddresses = append(ctx.RawAddresses, addr)
					ctx.RawAddrToMirror[addr] = param
				}

				if fParam != nil {
					err := fParam(param, field.Name, field.Tag)
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
		RawAddrToMirror: map[uintptr]Param{},
		RawAddresses:    []uintptr{},
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

	if b.Params != nil {

		// look in tags for info about positional args
		err := traverse(ctx, b.Params, func(param Param, _ string, tags reflect.StructTag) error {
			if tags.Get("positional") == "true" || tags.Get("pos") == "true" {
				param.setPositional(true)
			}
			if param.descr() == "" {
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
					param.SetEnv(env)
				}
			}
			if param.GetShort() == "" {
				if shrt, ok := tags.Lookup("short"); ok {
					param.SetShort(shrt)
				}
			}
			if param.GetName() == "" {
				if name, ok := tags.Lookup("name"); ok {
					param.SetName(name)
				} else if name, ok := tags.Lookup("long"); ok {
					param.SetName(name)
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
			return nil
		}, nil)

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

		positional := make([]Param, 0)
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

	// Set ValidArgsFunction with syncMirrors so that positional-argument completion
	// functions also see up-to-date raw field values (same reason as flag completion).
	if b.ValidArgsFunc != nil {
		cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			syncMirrors(ctx)
			return b.ValidArgsFunc(cmd, args, toComplete)
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

		// Convert unsafe.Pointer to a reflect.Value of the appropriate pointer type
		// without needing to create an unused ptrType variable
		//goland:noinspection ALL
		//nolint:govet // intentional: rawAddr is a valid uintptr from fieldAddr.Pointer()
		ptrToRawValue := reflect.NewAt(mirror.GetType(), unsafe.Pointer(rawAddr))

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

func runImpl(cmd *cobra.Command, handler ResultHandler) {

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

	// 	string |
	//		int |
	//		int32 |
	//		int64 |
	//		bool |
	//		float64 |
	//		float32 |
	//		time.Time |
	//		time.Duration |
	//		net.IP |
	//		*url.URL |
	//		[]string |
	//		[]int |
	//		[]int32 |
	//		[]int64 |
	//		[]float32 |
	//		[]float64
	switch t.Kind() {
	case
		reflect.String,
		reflect.Int,
		reflect.Int32,
		reflect.Bool,
		reflect.Float32,
		reflect.Float64:
		return true
	case reflect.Int64:
		// int64 and time.Duration (which is int64 underneath)
		return true
	case reflect.Struct:
		if t == timeType {
			return true
		} else {
			return false
		}
	case reflect.Slice:
		// net.IP is []byte
		if t == ipType {
			return true
		}
		elem := t.Elem()
		// Basic slice types
		if elem.Kind() == reflect.String ||
			elem.Kind() == reflect.Int ||
			elem.Kind() == reflect.Int32 ||
			elem.Kind() == reflect.Int64 ||
			elem.Kind() == reflect.Float32 ||
			elem.Kind() == reflect.Float64 ||
			elem.Kind() == reflect.Bool {
			return true
		}
		// []time.Time
		if elem == timeType {
			return true
		}
		// []net.IP (slice of net.IP which is itself []byte)
		if elem == ipType {
			return true
		}
		// []*url.URL
		if elem == urlPtrType {
			return true
		}
		return false
	case reflect.Pointer:
		// *url.URL
		if t == urlPtrType {
			return true
		}
		return false
	default:
		return false
	}
}

func newParam(field *reflect.StructField, t reflect.Type) Param {

	required := !cfg.defaultOptional
	if requiredTag, ok := field.Tag.Lookup("required"); ok {
		switch requiredTag {
		case "true":
			required = true
		case "false":
			required = false
		default:
			panic(fmt.Errorf("invalid value for field %s's required tag: %s", field.Name, requiredTag))
		}
	}
	if requiredTag, ok := field.Tag.Lookup("req"); ok {
		switch requiredTag {
		case "true":
			required = true
		case "false":
			required = false
		default:
			panic(fmt.Errorf("invalid value for field %s's required tag: %s", field.Name, requiredTag))
		}
	}
	if optionalTag, ok := field.Tag.Lookup("optional"); ok {
		switch optionalTag {
		case "true":
			required = false
		case "false":
			required = true
		default:
			panic(fmt.Errorf("invalid value for field %s's optional tag: %s", field.Name, optionalTag))
		}
	}
	if optionalTag, ok := field.Tag.Lookup("opt"); ok {
		switch optionalTag {
		case "true":
			required = false
		case "false":
			required = true
		default:
			panic(fmt.Errorf("invalid value for field %s's optional tag: %s", field.Name, optionalTag))
		}
	}

	switch t.Kind() {
	case reflect.String:
		if required {
			return &Required[string]{}
		} else {
			return &Optional[string]{}
		}
	case reflect.Int:
		if required {
			return &Required[int]{}
		} else {
			return &Optional[int]{}
		}
	case reflect.Int32:
		if required {
			return &Required[int32]{}
		} else {
			return &Optional[int32]{}
		}
	case reflect.Int64:
		// Check if this is time.Duration
		if t == durationType {
			if required {
				return &Required[time.Duration]{}
			} else {
				return &Optional[time.Duration]{}
			}
		}
		if required {
			return &Required[int64]{}
		} else {
			return &Optional[int64]{}
		}
	case reflect.Float32:
		if required {
			return &Required[float32]{}
		} else {
			return &Optional[float32]{}
		}
	case reflect.Float64:
		if required {
			return &Required[float64]{}
		} else {
			return &Optional[float64]{}
		}
	case reflect.Bool:
		if required {
			return &Required[bool]{}
		} else {
			return &Optional[bool]{}
		}
	case reflect.Struct:
		if t == timeType {
			if required {
				return &Required[time.Time]{}
			} else {
				return &Optional[time.Time]{}
			}
		} else {
			panic(fmt.Errorf("unsupported type %s", t.String()))
		}
	case reflect.Slice:
		// Check if this is net.IP (which is []byte)
		if t == ipType {
			if required {
				return &Required[net.IP]{}
			} else {
				return &Optional[net.IP]{}
			}
		}
		elem := t.Elem()
		// Check for special slice types first
		// []net.IP - pflag has IPSliceP
		if elem == ipType {
			if required {
				return &Required[[]net.IP]{}
			} else {
				return &Optional[[]net.IP]{}
			}
		}
		// []time.Duration - pflag has DurationSliceP
		if elem == durationType {
			if required {
				return &Required[[]time.Duration]{}
			} else {
				return &Optional[[]time.Duration]{}
			}
		}
		// []time.Time - stored as []string, converted during validation
		if elem == timeType {
			if required {
				return &Required[[]time.Time]{}
			} else {
				return &Optional[[]time.Time]{}
			}
		}
		// []*url.URL - stored as []string, converted during validation
		if elem == urlPtrType {
			if required {
				return &Required[[]*url.URL]{}
			} else {
				return &Optional[[]*url.URL]{}
			}
		}
		switch elem.Kind() {
		case reflect.String:
			if required {
				return &Required[[]string]{}
			} else {
				return &Optional[[]string]{}
			}
		case reflect.Int:
			if required {
				return &Required[[]int]{}
			} else {
				return &Optional[[]int]{}
			}
		case reflect.Int32:
			if required {
				return &Required[[]int32]{}
			} else {
				return &Optional[[]int32]{}
			}
		case reflect.Int64:
			if required {
				return &Required[[]int64]{}
			} else {
				return &Optional[[]int64]{}
			}
		case reflect.Float32:
			if required {
				return &Required[[]float32]{}
			} else {
				return &Optional[[]float32]{}
			}
		case reflect.Float64:
			if required {
				return &Required[[]float64]{}
			} else {
				return &Optional[[]float64]{}
			}
		case reflect.Bool:
			if required {
				return &Required[[]bool]{}
			} else {
				return &Optional[[]bool]{}
			}
		default:
			panic(fmt.Errorf("unsupported slice type %s", t.String()))
		}
	case reflect.Pointer:
		// Check if this is *url.URL
		if t == urlPtrType {
			if required {
				return &Required[*url.URL]{}
			} else {
				return &Optional[*url.URL]{}
			}
		}
		panic(fmt.Errorf("unsupported pointer type %s", t.String()))
	default:
		panic(fmt.Errorf("unsupported type %s", t.String()))
	}
}

var timeType = reflect.TypeOf(time.Time{})
var durationType = reflect.TypeOf(time.Duration(0))
var ipType = reflect.TypeOf(net.IP{})
var urlPtrType = reflect.TypeOf((*url.URL)(nil))
