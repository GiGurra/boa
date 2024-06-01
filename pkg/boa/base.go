package boa

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type SupportedTypes interface {
	string | int | int32 | int64 | bool | float64 | float32 | time.Time |
		[]int | []int32 | []int64 | []float32 | []float64 | []string
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
	customValidatorOfPtr() func(any) error
	hasDefaultValue() bool
	defaultValueStr() string
	setParentCmd(cmd *cobra.Command)
	setValuePtr(any)
	markSetFromEnv()
	isPositional() bool
	wasSetPositionally() bool
	markSetPositionally()
	setPositional(bool)
	setDescription(descr string)
	GetAlternatives() []string
	GetAlternativesFunc() func(cmd *cobra.Command, args []string, toComplete string) []string
}

func Default[T SupportedTypes](val T) *T {
	return &val
}

func validate(structPtr any) error {

	return foreachParam(structPtr, func(param Param, _ string, _ reflect.StructTag) error {

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
		}

		if err := param.customValidatorOfPtr()(param.valuePtrF()); err != nil {
			return fmt.Errorf("invalid value for param '%s': %s", param.GetName(), err.Error())
		}

		return nil
	})
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
		descr = fmt.Sprintf("%s [required]", descr)
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
		return fmt.Errorf("arrays are supported param type: " + f.GetKind().String())
	case reflect.Pointer:
		return fmt.Errorf("pointers not supported param type: " + f.GetKind().String())
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

func HasValue(f Param) bool {
	return f.wasSetByEnv() || f.wasSetOnCli() || f.hasDefaultValue()
}

type ParamEnricher func(alreadyProcessed []Param, param Param, paramFieldName string) error

func ParamEnricherCombine(enrichers ...ParamEnricher) ParamEnricher {
	return func(alreadyProcessed []Param, param Param, paramFieldName string) error {
		for _, enricher := range enrichers {
			err := enricher(alreadyProcessed, param, paramFieldName)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

//goland:noinspection GoUnusedGlobalVariable
var (
	ParamEnricherBool ParamEnricher = func(alreadyProcessed []Param, param Param, paramFieldName string) error {
		if param.GetKind() == reflect.Bool && !param.hasDefaultValue() {
			param.SetDefault(Default(false))
		}
		return nil
	}
	ParamEnricherName ParamEnricher = func(alreadyProcessed []Param, param Param, paramFieldName string) error {
		if param.GetName() == "" {
			param.SetName(camelToKebabCase(paramFieldName))
		}
		return nil
	}
	ParamEnricherShort ParamEnricher = func(alreadyProcessed []Param, param Param, paramFieldName string) error {
		if param.GetShort() == "" && param.GetName() != "" {
			// check that no other param has the same short name
			wantShort := string(param.GetName()[0])
			if wantShort == "h" {
				return nil // don't override help h
			}
			shortAvailable := true
			for _, other := range alreadyProcessed {
				if other.GetShort() == wantShort {
					shortAvailable = false
				}
			}
			if shortAvailable {
				param.SetShort(wantShort)
			}
		}
		return nil
	}
	ParamEnricherEnv ParamEnricher = func(alreadyProcessed []Param, param Param, paramFieldName string) error {
		if param.GetEnv() == "" && param.GetName() != "" && !param.isPositional() {
			param.SetEnv(kebabCaseToUpperSnakeCase(param.GetName()))
		}
		return nil
	}

	ParamEnricherDefault = ParamEnricherCombine(
		ParamEnricherName,
		ParamEnricherShort,
		ParamEnricherEnv,
		ParamEnricherBool,
	)

	ParamEnricherNone = ParamEnricherCombine()
)

//goland:noinspection GoUnusedExportedFunction
func ParamEnricherEnvPrefix(prefix string) ParamEnricher {
	return func(alreadyProcessed []Param, param Param, paramFieldName string) error {
		if param.GetEnv() != "" {
			param.SetEnv(prefix + "_" + param.GetEnv())
		}
		return nil
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

type Wrap struct {
	Use            string
	Short          string
	Long           string
	Version        string
	Args           cobra.PositionalArgs
	SubCommands    []*cobra.Command
	Params         any
	ParamEnrich    ParamEnricher
	Run            func(cmd *cobra.Command, args []string)
	UseCobraErrLog bool
	SortFlags      bool
	ValidArgs      []string
	ValidArgsFunc  func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)
}

func (b Wrap) WithSubCommands(cmd ...*cobra.Command) Wrap {
	b.SubCommands = append(b.SubCommands, cmd...)
	return b
}

func Compose(structs ...any) *Composition {
	return &Composition{
		StructPtrs: structs,
	}
}

type Composition struct {
	StructPtrs []any
}

func (b Wrap) ToCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               b.Use,
		Short:             b.Short,
		Long:              b.Long,
		Run:               b.Run,
		Args:              b.Args,
		SilenceErrors:     !b.UseCobraErrLog,
		ValidArgs:         b.ValidArgs,
		ValidArgsFunction: b.ValidArgsFunc,
	}

	cmd.Flags().SortFlags = b.SortFlags
	cmd.Version = b.Version

	for _, subcommand := range b.SubCommands {
		cmd.AddCommand(subcommand)
	}

	if b.Params != nil {

		// look in tags for info about positional args
		err := foreachParam(b.Params, func(param Param, _ string, tags reflect.StructTag) error {
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

			if defaultPtr, ok := tags.Lookup("default"); ok {
				ptr, err := parsePtr(param.GetName(), param.GetType(), param.GetKind(), defaultPtr)
				if err != nil {
					return fmt.Errorf("invalid default value for param %s: %s", param.GetName(), err.Error())
				}
				param.SetDefault(ptr)
			}
			return nil
		})

		if err != nil {
			panic(fmt.Errorf("error parsing tags: %s", err.Error()))
		}

		if b.ParamEnrich == nil {
			b.ParamEnrich = ParamEnricherDefault
		}
		processed := make([]Param, 0)
		err = foreachParam(b.Params, func(param Param, paramFieldName string, _ reflect.StructTag) error {
			err := b.ParamEnrich(processed, param, paramFieldName)
			if err != nil {
				return err
			}
			processed = append(processed, param)
			return nil
		})
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

		err = foreachParam(b.Params, func(param Param, _ string, tags reflect.StructTag) error {
			err := connect(param, cmd, positional)
			if err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			panic(fmt.Errorf("error connecting params: %s", err.Error()))
		}
	}

	// now wrap the run function of the command to validate the flags
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if b.Params != nil {
			if err := validate(b.Params); err != nil {
				return err
			}
		}
		return nil
	}

	return cmd
}

type Handler struct {
	Panic   func(any)
	Failure func(error)
	Success func()
}

func foreachParam(structPtr any, f func(param Param, paramFieldName string, tags reflect.StructTag) error) error {

	if reflect.TypeOf(structPtr).Kind() != reflect.Ptr {
		return fmt.Errorf("expected pointer to struct")
	}

	if reflect.TypeOf(structPtr).Elem().Kind() != reflect.Struct {
		return fmt.Errorf("expected pointer to struct")
	}

	if c, ok := reflect.ValueOf(structPtr).Interface().(*Composition); ok {
		for _, structPtr := range c.StructPtrs {
			if err := foreachParam(structPtr, f); err != nil {
				return err
			}
		}
		return nil
	}

	// use reflection to iterate over all fields of the struct
	fields := reflect.TypeOf(structPtr).Elem()
	rootValue := reflect.ValueOf(structPtr).Elem()
	for i := 0; i < fields.NumField(); i++ {
		field := fields.Field(i)
		fieldValue := rootValue.Field(i).Addr()
		// check if field is a param
		param, isParam := fieldValue.Interface().(Param)
		if isParam {
			err := f(param, field.Name, field.Tag)
			if err != nil {
				return err
			}
		} else {

			// check if it is a struct
			if field.Type.Kind() == reflect.Struct {
				if err := foreachParam(fieldValue.Interface(), f); err != nil {
					return err
				}
				continue
			}

			fmt.Printf("WARNING: field %s is not a param. It will be ignored\n", field.Name)
			continue // not a param
		}
	}

	return nil
}

func (b Wrap) ToApp() {
	b.ToAppH(Handler{})
}

func (b Wrap) ToAppH(handler Handler) {

	if handler.Panic != nil {
		defer func() {
			if r := recover(); r != nil {
				handler.Panic(r)
			}
		}()
	}

	cmd := b.ToCmd()
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
