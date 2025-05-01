package boa

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"log/slog"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unsafe"
)

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
	GetIsEnabledFn() func() bool
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

func validate(ctx *processingContext, structPtr any) error {

	return traverse(ctx, structPtr, func(param Param, _ string, _ reflect.StructTag) error {

		if !param.IsEnabled() {
			return nil
		}

		envHint := ""
		if param.GetEnv() != "" {
			envHint = fmt.Sprintf(" (env: %s)", param.GetEnv())
		}

		if err := readEnv(param); err != nil {
			return err
		}
		if param.IsRequired() && !HasValue(param) {
			return fmt.Errorf("missing required param '%s'%s", param.GetName(), envHint)
		}

		// special types validation, e.g. only time.Time so far
		if HasValue(param) {
			if param.GetKind() == reflect.Struct {
				if param.GetType().String() == "time.Time" {
					strVal := *param.valuePtrF().(*string)
					res, err := parsePtr(param.GetName(), param.GetType(), param.GetKind(), strVal)
					if err != nil {
						return fmt.Errorf("invalid value for param '%s': %s", param.GetName(), err.Error())
					}
					param.setValuePtr(res)
				}
			}

			if err := param.customValidatorOfPtr()(param.valuePtrF()); err != nil {
				return fmt.Errorf("invalid value for param '%s': %s", param.GetName(), err.Error())
			}
		}

		return nil
	}, nil)
}

func doParsePositional(f Param, strVal string) error {
	if strVal == "" && f.IsRequired() {
		if f.hasDefaultValue() || f.wasSetByEnv() {
			return nil
		} else {
			return fmt.Errorf("empty positional arg: %s", f.GetName())
		}
	}

	if err := readFrom(f, strVal); err != nil {
		return err
	}

	f.markSetPositionally()

	return nil
}

func toTypedSlice[T SupportedTypes](slice any) []T {
	if slice == nil {
		return nil
	} else {
		return slice.([]T)
	}
}

func connect(f Param, cmd *cobra.Command, posArgs []Param) error {

	if f.GetName() == "" {
		return fmt.Errorf("invalid conf for param '%s': long param name cannot be empty", f.GetName())
	}

	if f.GetShort() == "h" {
		return fmt.Errorf("invalid conf for param '%s': short param cannot be 'h'. It collides with -h for help", f.GetName())
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
						return fmt.Errorf("missing positional arg '%s'", f.GetName())
					}
				} else {
					return nil
				}
			}
			return doParsePositional(f, args[posArgIndex])
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
			def = *reflect.ValueOf(f.defaultValuePtr()).Interface().(*string)
		}
		f.setValuePtr(cmd.Flags().StringP(f.GetName(), f.GetShort(), def, descr))
		return nil
	case reflect.Int:
		def := 0
		if f.hasDefaultValue() {
			def = *reflect.ValueOf(f.defaultValuePtr()).Interface().(*int)
		}
		f.setValuePtr(cmd.Flags().IntP(f.GetName(), f.GetShort(), def, descr))
		return nil
	case reflect.Int32:
		def := int32(0)
		if f.hasDefaultValue() {
			def = *reflect.ValueOf(f.defaultValuePtr()).Interface().(*int32)
		}
		f.setValuePtr(cmd.Flags().Int32P(f.GetName(), f.GetShort(), def, descr))
		return nil
	case reflect.Int64:
		def := int64(0)
		if f.hasDefaultValue() {
			def = *reflect.ValueOf(f.defaultValuePtr()).Interface().(*int64)
		}
		f.setValuePtr(cmd.Flags().Int64P(f.GetName(), f.GetShort(), def, descr))
		return nil
	case reflect.Float64:
		def := 0.0
		if f.hasDefaultValue() {
			def = *reflect.ValueOf(f.defaultValuePtr()).Interface().(*float64)
		}
		f.setValuePtr(cmd.Flags().Float64P(f.GetName(), f.GetShort(), def, descr))
		return nil
	case reflect.Float32:
		def := float32(0.0)
		if f.hasDefaultValue() {
			def = *reflect.ValueOf(f.defaultValuePtr()).Interface().(*float32)
		}
		f.setValuePtr(cmd.Flags().Float32P(f.GetName(), f.GetShort(), def, descr))
		return nil
	case reflect.Bool:
		def := false
		if f.hasDefaultValue() {
			def = *reflect.ValueOf(f.defaultValuePtr()).Interface().(*bool)
		}
		f.setValuePtr(cmd.Flags().BoolP(f.GetName(), f.GetShort(), def, descr))
		return nil
	case reflect.Struct:
		if f.GetType().String() == "time.Time" {
			if f.hasDefaultValue() {
				def := *reflect.ValueOf(f.defaultValuePtr()).Interface().(*time.Time)
				f.setValuePtr(cmd.Flags().StringP(f.GetName(), f.GetShort(), def.Format(time.RFC3339), descr))
			} else {
				f.setValuePtr(cmd.Flags().StringP(f.GetName(), f.GetShort(), "", descr))
			}
			return nil
		} else {
			return fmt.Errorf("general structs not yet supported: " + f.GetKind().String())
		}
	case reflect.Slice:

		elemType := f.GetType().Elem()

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
		return fmt.Errorf("unsupported param type (Array): %s: " + f.GetKind().String())
	case reflect.Pointer:
		return fmt.Errorf("unsupported param type (Pointer): %s: " + f.GetKind().String())
	default:
		return fmt.Errorf("unsupported param type: %s" + f.GetKind().String())
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
	default:
		return nil, fmt.Errorf("unsupported slice element type '%v'. Check parameter '%s'", elemType, name)
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
		if tpe.String() == "time.Time" {
			parsedTime, err := time.Parse(time.RFC3339, strVal)
			if err != nil {
				return nil, fmt.Errorf("invalid value for param %s: %s", name, err.Error())
			}
			return &parsedTime, nil
		} else {
			return nil, fmt.Errorf("general structs not yet supported: " + tpe.String())
		}
	case reflect.Slice:
		return parseSlice(name, strVal, tpe.Elem())
	case reflect.Array:
		return nil, fmt.Errorf("arrays not supported param type. Use a slice instead: " + kind.String())
	case reflect.Pointer:
		return nil, fmt.Errorf("pointers not yet supported param type: " + kind.String())
	default:
		return nil, fmt.Errorf("unsupported param type: %s" + kind.String())
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

func traverse(
	ctx *processingContext,
	structPtr any,
	fParam func(param Param, paramFieldName string, tags reflect.StructTag) error,
	fStruct func(structPtr any) error,
) error {

	if reflect.TypeOf(structPtr).Kind() != reflect.Ptr {
		return fmt.Errorf("foreachParam1: expected pointer to struct")
	}

	if reflect.TypeOf(structPtr).Elem().Kind() != reflect.Struct {
		return fmt.Errorf("foreachParam2: expected pointer to struct")
	}

	if fStruct != nil {
		err := fStruct(structPtr)
		if err != nil {
			return fmt.Errorf("foreachParam3: error in fStruct: %s", err.Error())
		}
	}

	// use reflection to iterate over all fields of the struct
	fields := reflect.TypeOf(structPtr).Elem()
	rootValue := reflect.ValueOf(structPtr).Elem()
	for i := 0; i < fields.NumField(); i++ {
		field := fields.Field(i)
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

			// check if it is a struct
			if field.Type.Kind() == reflect.Struct {
				if err := traverse(ctx, fieldAddr.Interface(), fParam, fStruct); err != nil {
					return err
				}
				continue
			}

			// check if it is a pointer to a struct
			if field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct {
				if !fieldAddr.IsNil() && !fieldAddr.Elem().IsNil() {
					if err := traverse(ctx, fieldAddr.Elem().Interface(), fParam, fStruct); err != nil {
						return err
					}
				}
				continue
			}

			if field.Type.Kind() == reflect.Pointer {
				slog.Warn(fmt.Sprintf("raw pointer types to parameters are not (yet?) supported. Field %s will be ignored", field.Name))
				continue
			}

			// For raw fields, we store parameter mirrors in the processing context
			if isSupportedType(field.Type) {

				// check if we already have a mirror for this field
				var addr uintptr = fieldAddr.Pointer()
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

func (b Cmd) toCobraImpl() *cobra.Command {
	cmd := &cobra.Command{
		Use:               b.Use,
		Short:             b.Short,
		Long:              b.Long,
		Run:               b.RunFunc,
		Args:              b.Args,
		SilenceErrors:     !b.UseCobraErrLog,
		ValidArgs:         b.ValidArgs,
		ValidArgsFunction: b.ValidArgsFunc,
	}

	if b.RawArgs != nil {
		cmd.SetArgs(b.RawArgs)
	}

	ctx := &processingContext{
		Context:         context.Background(), // prepare to override later?
		RawAddrToMirror: map[uintptr]Param{},
		RawAddresses:    []uintptr{},
	}

	// if b.params or any inner struct implements CfgStructPreExecute, call it
	if b.Params != nil {
		err := traverse(ctx, b.Params, nil, func(innerParams any) error {
			if preParse, ok := b.Params.(CfgStructInit); ok {
				err := preParse.Init()
				if err != nil {
					return fmt.Errorf("error in PreParse: %s", err.Error())
				}
			}
			return nil
		})
		if err != nil {
			panic(err)
		}
	}

	// if we have a custom init function, call it
	if b.InitFunc != nil {
		err := b.InitFunc(b.Params, cmd)
		if err != nil {
			panic(fmt.Errorf("error in InitFunc: %s", err.Error()))
		}
	}

	cmd.Flags().SortFlags = b.SortFlags
	cmd.Version = b.Version

	for _, subcommand := range b.SubCommands {
		cmd.AddCommand(subcommand)
	}

	if b.Params != nil {

		// look in tags for info about positional args
		err := traverse(ctx, b.Params, func(param Param, _ string, tags reflect.StructTag) error {
			if tags.Get("positional") == "true" || tags.Get("pos") == "true" {
				param.setPositional(true)
			}
			if descr, ok := tags.Lookup("descr"); ok {
				param.setDescription(descr)
			}
			if descr, ok := tags.Lookup("description"); ok {
				param.setDescription(descr)
			}
			if env, ok := tags.Lookup("env"); ok {
				param.SetEnv(env)
			}
			if shrt, ok := tags.Lookup("short"); ok {
				param.SetShort(shrt)
			}
			if name, ok := tags.Lookup("name"); ok {
				param.SetName(name)
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
			panic(fmt.Errorf("error parsing tags: %w", err))
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
			panic(fmt.Errorf("error enriching params: %s", err.Error()))
		}

		positional := make([]Param, 0)
		for _, param := range processed {
			if param.isPositional() {
				positional = append(positional, param)
			}
		}

		// Check that no required positional arg exists after on optional positional arg
		numReqPositional := 0
		for i, param := range positional {
			if param.IsRequired() {
				numReqPositional++
			}
			if param.IsRequired() && i >= 1 {
				prev := positional[i-1]
				if !prev.IsRequired() {
					panic(fmt.Errorf("required positional arg %s must come before optional positional arg %s", param.GetName(), prev.GetName()))
				}
			}
		}

		if cmd.Args == nil {
			cmd.Args = cobra.RangeArgs(numReqPositional, len(positional))
		}

		err = traverse(ctx, b.Params, func(param Param, _ string, tags reflect.StructTag) error {
			err := connect(param, cmd, positional)
			if err != nil {
				return err
			}

			return nil
		}, nil)

		if err != nil {
			panic(fmt.Errorf("error connecting params: %s", err.Error()))
		}
	}

	// now wrap the run function of the command to validate the flags
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if b.Params != nil {

			syncMirrors(ctx)

			// if b.params or any inner struct implements CfgStructPreValidate, call it
			err := traverse(ctx, b.Params, nil, func(innerParams any) error {
				if s, ok := innerParams.(CfgStructPreValidate); ok {
					err := s.PreValidate()
					if err != nil {
						return fmt.Errorf("error in PreValidate: %s", err.Error())
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
					return fmt.Errorf("error in PreValidate: %s", err.Error())
				}
			}

			syncMirrors(ctx)

			if err = validate(ctx, b.Params); err != nil {
				return err
			}

			// if b.params or any inner struct implements CfgStructPreExecute, call it
			err = traverse(ctx, b.Params, nil, func(innerParams any) error {
				if preExecute, ok := innerParams.(CfgStructPreExecute); ok {
					err := preExecute.PreExecute()
					if err != nil {
						return fmt.Errorf("error in PreExecute: %s", err.Error())
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
					return fmt.Errorf("error in PreExecuteFunc: %s", err.Error())
				}
			}

		}
		return nil
	}

	return cmd
}

func syncMirrors(ctx *processingContext) {
	// 1. First, copy non-zero values from the raw fields -> mirrors as injected values.
	// 2. Then copy back cli & env set values to the raw fields

	for _, rawAddr := range ctx.RawAddresses {
		mirror := ctx.RawAddrToMirror[rawAddr]

		// Convert unsafe.Pointer to a reflect.Value of the appropriate pointer type
		// without needing to create an unused ptrType variable
		//goland:noinspection ALL
		ptrToRawValue := reflect.NewAt(mirror.GetType(), unsafe.Pointer(rawAddr))

		if !mirror.wasSetOnCli() && !mirror.wasSetByEnv() && !ptrToRawValue.Elem().IsZero() {
			mirror.injectValuePtr(ptrToRawValue.Interface())
		}

		if mirror.wasSetOnCli() || mirror.wasSetByEnv() || (mirror.HasValue() && ptrToRawValue.Elem().IsZero()) {
			mirrorValue := reflect.ValueOf(mirror.valuePtrF()).Elem()

			// Get the element that the pointer points to
			rawValue := ptrToRawValue.Elem()

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
			fmt.Printf("error executing command: %v\n", err)
			os.Exit(1)
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
		reflect.Int64,
		reflect.Bool,
		reflect.Float32,
		reflect.Float64:
		return true
	case reflect.Struct:
		if t.String() == "time.Time" {
			return true
		} else {
			return false
		}
	case reflect.Slice:
		if t.Elem().Kind() == reflect.String ||
			t.Elem().Kind() == reflect.Int ||
			t.Elem().Kind() == reflect.Int32 ||
			t.Elem().Kind() == reflect.Int64 ||
			t.Elem().Kind() == reflect.Float32 ||
			t.Elem().Kind() == reflect.Float64 {
			return true
		} else {
			return false
		}
	default:
		return false
	}
}

func newParam(field *reflect.StructField, t reflect.Type) Param {

	required := true
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
		if t.String() == "time.Time" {
			if required {
				return &Required[time.Time]{}
			} else {
				return &Optional[time.Time]{}
			}
		} else {
			panic(fmt.Errorf("unsupported type %s", t.String()))
		}
	case reflect.Slice:
		switch t.Elem().Kind() {
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
		default:
			panic(fmt.Errorf("unsupported slice type %s", t.String()))
		}
	default:
		panic(fmt.Errorf("unsupported type %s", t.String()))
	}
}
