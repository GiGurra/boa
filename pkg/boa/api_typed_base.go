package boa

import (
	"fmt"
	"github.com/spf13/cobra"
	"reflect"
)

type NoParamsT struct{}

var NoParams = &NoParamsT{}

type Wrap2[Struct any] struct {
	Use            string
	Short          string
	Long           string
	Version        string
	Args           cobra.PositionalArgs
	SubCommands    []*cobra.Command
	Params         *Struct
	ParamEnrich    ParamEnricher
	RunFunc        func(params *Struct, cmd *cobra.Command, args []string)
	UseCobraErrLog bool
	SortFlags      bool
	ValidArgs      []string
	ValidArgsFunc  func(params *Struct, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)
}

func NewCmdBuilder[Struct any](use string) Wrap2[Struct] {
	var params *Struct

	// Instantiate the config object for simplicity
	configType := reflect.TypeOf((*Struct)(nil))
	switch configType.Kind() {
	case reflect.Ptr | reflect.Interface:
		elemType := configType.Elem()
		newInstance := reflect.New(elemType).Interface()
		if typedInstance, ok := newInstance.(*Struct); ok {
			params = typedInstance
		}
	default:
		// For value types, leave the zero value as is
	}

	if reflect.TypeOf(params).Kind() != reflect.Ptr {
		panic(fmt.Errorf("expected pointer to struct"))
	}

	if reflect.TypeOf(params).Elem().Kind() != reflect.Struct {
		panic(fmt.Errorf("expected pointer to struct"))
	}

	return Wrap2[Struct]{
		Use:         use,
		Params:      params,
		ParamEnrich: ParamEnricherDefault,
	}
}

func (b Wrap2[Struct]) WithUse(use string) Wrap2[Struct] {
	b.Use = use
	return b
}

func (b Wrap2[Struct]) WithShort(short string) Wrap2[Struct] {
	b.Short = short
	return b
}

func (b Wrap2[Struct]) WithLong(long string) Wrap2[Struct] {
	b.Long = long
	return b
}

func (b Wrap2[Struct]) WithVersion(version string) Wrap2[Struct] {
	b.Version = version
	return b
}

func (b Wrap2[Struct]) WithArgs(args cobra.PositionalArgs) Wrap2[Struct] {
	b.Args = args
	return b
}

func (b Wrap2[Struct]) WithParamEnrich(enricher ParamEnricher) Wrap2[Struct] {
	b.ParamEnrich = enricher
	return b
}

func (b Wrap2[Struct]) WithRunFunc(run func(params *Struct, cmd *cobra.Command, args []string)) Wrap2[Struct] {
	b.RunFunc = run
	return b
}

func (b Wrap2[Struct]) WithUseCobraErrLog(useCobraErrLog bool) Wrap2[Struct] {
	b.UseCobraErrLog = useCobraErrLog
	return b
}

func (b Wrap2[Struct]) WithSortFlags(sortFlags bool) Wrap2[Struct] {
	b.SortFlags = sortFlags
	return b
}

func (b Wrap2[Struct]) WithValidArgs(validArgs []string) Wrap2[Struct] {
	b.ValidArgs = validArgs
	return b
}

func (b Wrap2[Struct]) WithValidArgsFunc(validArgsFunc func(params *Struct, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)) Wrap2[Struct] {
	b.ValidArgsFunc = validArgsFunc
	return b
}

func (b Wrap2[Struct]) WithSubCommands(cmd ...*cobra.Command) Wrap2[Struct] {
	b.SubCommands = append(b.SubCommands, cmd...)
	return b
}

func (b Wrap2[Struct]) ToWrap() Wrap {

	var runFcn func(cmd *cobra.Command, args []string) = nil
	if b.RunFunc != nil {
		runFcn = func(cmd *cobra.Command, args []string) {
			b.RunFunc(b.Params, cmd, args)
		}
	}

	var validArgsFunc func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) = nil
	if b.ValidArgsFunc != nil {
		validArgsFunc = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return b.ValidArgsFunc(b.Params, cmd, args, toComplete)
		}
	}

	return Wrap{
		Use:            b.Use,
		Short:          b.Short,
		Long:           b.Long,
		Version:        b.Version,
		Args:           b.Args,
		SubCommands:    b.SubCommands,
		Params:         b.Params,
		ParamEnrich:    b.ParamEnrich,
		Run:            runFcn,
		UseCobraErrLog: b.UseCobraErrLog,
		SortFlags:      b.SortFlags,
		ValidArgs:      b.ValidArgs,
		ValidArgsFunc:  validArgsFunc,
	}
}

func (b Wrap2[Struct]) ToCmd() *cobra.Command {
	return b.ToWrap().toCmdImpl()
}

func (b Wrap2[Struct]) ToApp() {
	ToAppH(b.ToCmd(), ResultHandler{})
}

func (b Wrap2[Struct]) Run() {
	b.ToApp()
}

func (b Wrap2[Struct]) ToAppH(handler ResultHandler) {
	ToAppH(b.ToCmd(), handler)
}

func (b Wrap2[Struct]) RunH(handler ResultHandler) {
	b.ToAppH(handler)
}

func (b Wrap2[Struct]) Validate() error {
	return Validate(b.Params, b.ToWrap())
}
