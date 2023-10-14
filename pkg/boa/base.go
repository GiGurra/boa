package boa

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

type SupportedTypes interface {
	string | int | int32 | int64 | bool | float64 | float32
}

type Param interface {
	GetShort() string
	GetName() string
	GetEnv() string
	GetKind() reflect.Kind
	SetDefault(any)
	SetEnv(string)
	SetShort(string)
	SetName(string)
	defaultValuePtr() any
	descr() string
	IsRequired() bool
	valuePtrF() any
	parentCmd() *cobra.Command
	wasSetByFlag() bool
	wasSetByEnv() bool
	customValidatorOfPtr() func(any) error
	markValidated()
	hasDefaultValue() bool
	defaultValueStr() string
	setParentCmd(cmd *cobra.Command)
	setValuePtr(any)
	markSetFromEnv()
}

func Default[T SupportedTypes](val T) *T {
	return &val
}

func validate(structPtr any) {

	foreachParam(structPtr, func(param Param, _ string) {

		envHint := ""
		if param.GetEnv() != "" {
			envHint = fmt.Sprintf(" (env: %s)", param.GetEnv())
		}

		readEnv(param)
		if param.IsRequired() && !hasValue(param) {
			fmt.Printf("Error: required param '%s'%s was not set\n", param.GetName(), envHint)
			panic("required param not set")
		}

		if err := param.customValidatorOfPtr()(param.valuePtrF()); err != nil {
			fmt.Printf("Error: param '%s'%s is invalid: %s\n", param.GetName(), envHint, err.Error())
			panic("invalid param")
		}

		param.markValidated()
	})
}

func connect(f Param, cmd *cobra.Command) {

	if f.GetName() == "" {
		panic(fmt.Errorf("invalid conf for param '%s': long param name cannot be empty", f.GetName()))
	}

	if f.GetShort() == "h" {
		panic(fmt.Errorf("invalid conf for param '%s': short param cannot be 'h'. It collides with -h for help", f.GetName()))
	}

	if f.GetName() == "help" {
		panic(fmt.Errorf("invalid conf for param '%s': name cannot be 'help'. It collides with the standard help param", f.GetName()))
	}

	extraInfos := make([]string, 0)

	descr := f.descr()
	if f.GetEnv() != "" {
		extraInfos = append(extraInfos, fmt.Sprintf("env: %s", f.GetEnv()))
	}

	if f.IsRequired() {
		descr = fmt.Sprintf("%s [required]", descr)
	}

	if len(extraInfos) > 0 {
		descr = fmt.Sprintf("%s (%s)", descr, strings.Join(extraInfos, ", "))
	}

	f.setParentCmd(cmd)
	switch f.GetKind() {
	case reflect.String:
		def := ""
		if f.hasDefaultValue() {
			def = *reflect.ValueOf(f.defaultValuePtr()).Interface().(*string)
		}
		f.setValuePtr(cmd.Flags().StringP(f.GetName(), f.GetShort(), def, descr))
	case reflect.Int:
		def := 0
		if f.hasDefaultValue() {
			def = *reflect.ValueOf(f.defaultValuePtr()).Interface().(*int)
		}
		f.setValuePtr(cmd.Flags().IntP(f.GetName(), f.GetShort(), def, descr))
	case reflect.Int32:
		def := int32(0)
		if f.hasDefaultValue() {
			def = *reflect.ValueOf(f.defaultValuePtr()).Interface().(*int32)
		}
		f.setValuePtr(cmd.Flags().Int32P(f.GetName(), f.GetShort(), def, descr))
	case reflect.Int64:
		def := int64(0)
		if f.hasDefaultValue() {
			def = *reflect.ValueOf(f.defaultValuePtr()).Interface().(*int64)
		}
		f.setValuePtr(cmd.Flags().Int64P(f.GetName(), f.GetShort(), def, descr))
	case reflect.Float64:
		def := 0.0
		if f.hasDefaultValue() {
			def = *reflect.ValueOf(f.defaultValuePtr()).Interface().(*float64)
		}
		f.setValuePtr(cmd.Flags().Float64P(f.GetName(), f.GetShort(), def, descr))
	case reflect.Float32:
		def := float32(0.0)
		if f.hasDefaultValue() {
			def = *reflect.ValueOf(f.defaultValuePtr()).Interface().(*float32)
		}
		f.setValuePtr(cmd.Flags().Float32P(f.GetName(), f.GetShort(), def, descr))
	case reflect.Bool:
		def := false
		if f.hasDefaultValue() {
			def = *reflect.ValueOf(f.defaultValuePtr()).Interface().(*bool)
		}
		f.setValuePtr(cmd.Flags().BoolP(f.GetName(), f.GetShort(), def, descr))
	case reflect.Slice:
		fallthrough
	case reflect.Array:
		panic("arrays or slices not yet supported param type: " + f.GetKind().String())
	case reflect.Pointer:
		panic("pointers not yet supported param type: " + f.GetKind().String())
	default:
		panic("unsupported param type: %s" + f.GetKind().String())
	}
}

func readEnv(f Param) {
	if f.GetEnv() == "" {
		return
	}

	if f.wasSetByFlag() {
		return
	}

	envVal := os.Getenv(f.GetEnv())
	if envVal == "" {
		return
	}

	switch f.GetKind() {
	case reflect.String:
		f.setValuePtr(&envVal)
	case reflect.Int:
		parsedInt, err := strconv.Atoi(envVal)
		if err != nil {
			panic(fmt.Errorf("invalid env value for param %s: %s", f.GetName(), err.Error()))
		}
		f.setValuePtr(&parsedInt)
	case reflect.Int32:
		parsedInt64, err := strconv.ParseInt(envVal, 10, 32)
		if err != nil {
			panic(fmt.Errorf("invalid env value for param %s: %s", f.GetName(), err.Error()))
		}
		parsedInt32 := int32(parsedInt64)
		f.setValuePtr(&parsedInt32)
	case reflect.Int64:
		parsedInt64, err := strconv.ParseInt(envVal, 10, 64)
		if err != nil {
			panic(fmt.Errorf("invalid env value for param %s: %s", f.GetName(), err.Error()))
		}
		f.setValuePtr(&parsedInt64)
	case reflect.Float32:
		parsedFloat64, err := strconv.ParseFloat(envVal, 32)
		if err != nil {
			panic(fmt.Errorf("invalid env value for param %s: %s", f.GetName(), err.Error()))
		}
		parsedFloat32 := float32(parsedFloat64)
		f.setValuePtr(&parsedFloat32)
	case reflect.Float64:
		parsedFloat64, err := strconv.ParseFloat(envVal, 64)
		if err != nil {
			panic(fmt.Errorf("invalid env value for param %s: %s", f.GetName(), err.Error()))
		}
		f.setValuePtr(&parsedFloat64)
	case reflect.Bool:
		parsedBool, err := strconv.ParseBool(envVal)
		if err != nil {
			panic(fmt.Errorf("invalid env value for param %s: %s", f.GetName(), err.Error()))
		}
		f.setValuePtr(&parsedBool)
	case reflect.Slice:
		fallthrough
	case reflect.Array:
		panic("arrays or slices not yet supported param type: " + f.GetKind().String())
	case reflect.Pointer:
		panic("pointers not yet supported param type: " + f.GetKind().String())
	default:
		panic("unsupported param type: %s" + f.GetKind().String())
	}

	f.markSetFromEnv()
}

func hasValue(f Param) bool {
	return f.wasSetByEnv() || f.wasSetByFlag() || f.hasDefaultValue()
}

type ParamEnricher func(alreadyProcessed []Param, param Param, paramFieldName string)

func ParamEnricherCombine(enrichers ...ParamEnricher) ParamEnricher {
	return func(alreadyProcessed []Param, param Param, paramFieldName string) {
		for _, enricher := range enrichers {
			enricher(alreadyProcessed, param, paramFieldName)
		}
	}
}

var (
	ParamEnricherBool ParamEnricher = func(alreadyProcessed []Param, param Param, paramFieldName string) {
		if param.GetKind() == reflect.Bool && !param.hasDefaultValue() {
			param.SetDefault(Default(false))
		}
	}
	ParamEnricherName ParamEnricher = func(alreadyProcessed []Param, param Param, paramFieldName string) {
		if param.GetName() == "" {
			param.SetName(camelToKebabCase(paramFieldName))
		}
	}
	ParamEnricherShort ParamEnricher = func(alreadyProcessed []Param, param Param, paramFieldName string) {
		if param.GetShort() == "" && param.GetName() != "" {
			// check that no other param has the same short name
			wantShort := string(param.GetName()[0])
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
	}
	ParamEnricherEnv ParamEnricher = func(alreadyProcessed []Param, param Param, paramFieldName string) {
		if param.GetEnv() == "" && param.GetName() != "" {
			param.SetEnv(kebabCaseToUpperSnakeCase(param.GetName()))
		}
	}

	ParamEnricherDefault ParamEnricher = ParamEnricherCombine(
		ParamEnricherName,
		ParamEnricherShort,
		ParamEnricherEnv,
		ParamEnricherBool,
	)

	ParamEnricherNone ParamEnricher = ParamEnricherCombine()
)

func ParamEnricherEnvPrefix(prefix string) ParamEnricher {
	return func(alreadyProcessed []Param, param Param, paramFieldName string) {
		if param.GetEnv() != "" {
			param.SetEnv(prefix + "_" + param.GetEnv())
		}
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
	Use         string
	Short       string
	Long        string
	SubCommands []*cobra.Command
	Params      any
	ParamEnrich ParamEnricher
	Run         func(cmd *cobra.Command, args []string)
}

func (b Wrap) WithSubCommands(cmd ...*cobra.Command) Wrap {
	b.SubCommands = append(b.SubCommands, cmd...)
	return b
}

func (b Wrap) ToCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   b.Use,
		Short: b.Short,
		Long:  b.Long,
		Run:   b.Run,
	}

	for _, subcommand := range b.SubCommands {
		cmd.AddCommand(subcommand)
	}

	if b.Params != nil {
		if b.ParamEnrich == nil {
			b.ParamEnrich = ParamEnricherDefault
		}
		processed := make([]Param, 0)
		foreachParam(b.Params, func(param Param, paramFieldName string) {
			b.ParamEnrich(processed, param, paramFieldName)
			processed = append(processed, param)
		})
		foreachParam(b.Params, func(param Param, _ string) {
			connect(param, cmd)
		})
	}

	// now wrap the run function of the command to validate the flags
	oldRun := cmd.Run
	if oldRun != nil {
		cmd.Run = func(cmd *cobra.Command, args []string) {
			if b.Params != nil {
				validate(b.Params)
			}
			if oldRun != nil {
				oldRun(cmd, args)
			}
		}
	}

	return cmd
}

type Handler struct {
	Panic   func(any)
	Failure func(error)
	Success func()
}

func foreachParam(structPtr any, f func(param Param, paramFieldName string)) {

	if reflect.TypeOf(structPtr).Kind() != reflect.Ptr {
		panic("expected pointer to struct")
	}

	if reflect.TypeOf(structPtr).Elem().Kind() != reflect.Struct {
		panic("expected pointer to struct")
	}

	// use reflection to iterate over all fields of the struct
	fields := reflect.TypeOf(structPtr).Elem()
	rootValue := reflect.ValueOf(structPtr).Elem()
	for i := 0; i < fields.NumField(); i++ {
		field := fields.Field(i)
		fieldValue := rootValue.Field(i).Addr()
		// check if field is a param
		param, ok := fieldValue.Interface().(Param)
		if !ok {
			fmt.Printf("WARNING: field %s is not a param. It will be ignored", field.Name)
			continue // not a param
		}

		f(param, field.Name)
	}
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
			fmt.Printf("error executing command: %v", err)
			os.Exit(1)
		}
	} else {
		if handler.Success != nil {
			handler.Success()
		}
	}
}
