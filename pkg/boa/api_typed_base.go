package boa

import (
	"fmt"
	"github.com/spf13/cobra"
	"reflect"
)

type NoParams struct{}

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
	InitFunc       func(params *Struct) error
	PreExecuteFunc func(params *Struct, cmd *cobra.Command, args []string) error
	UseCobraErrLog bool
	SortFlags      bool
	ValidArgs      []string
	ValidArgsFunc  func(params *Struct, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)
	RawArgs        []string
}

func NewCmdBuilder[Struct any](use string) Wrap2[Struct] {
	var params Struct
	return NewCmdBuilder2(use, &params)
}

func NewCmdBuilder2[Struct any](use string, params *Struct) Wrap2[Struct] {

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

func (b Wrap2[Struct]) WithRunFunc(run func(params *Struct)) Wrap2[Struct] {
	return b.WithRunFunc3(func(params *Struct, _ *cobra.Command, _ []string) {
		run(params)
	})
}

func (b Wrap2[Struct]) WithRunFunc3(run func(params *Struct, cmd *cobra.Command, args []string)) Wrap2[Struct] {
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

func (b Wrap2[Struct]) WithPreExecuteFunc(preExecuteFunc func(params *Struct, cmd *cobra.Command, args []string)) Wrap2[Struct] {
	return b.WithPreExecuteFuncE(func(params *Struct, cmd *cobra.Command, args []string) error {
		preExecuteFunc(params, cmd, args)
		return nil
	})
}

func (b Wrap2[Struct]) WithPreExecuteFuncE(preExecuteFunc func(params *Struct, cmd *cobra.Command, args []string) error) Wrap2[Struct] {
	b.PreExecuteFunc = preExecuteFunc
	return b
}

func (b Wrap2[Struct]) WithInitFunc(initFunc func(params *Struct)) Wrap2[Struct] {
	return b.WithInitFuncE(func(params *Struct) error {
		initFunc(params)
		return nil
	})
}

func (b Wrap2[Struct]) WithInitFuncE(initFunc func(params *Struct) error) Wrap2[Struct] {
	b.InitFunc = initFunc
	return b
}

// WithRawArgs sets the raw args to be used instead of os.Args. Mostly used for testing purposes.
func (b Wrap2[Struct]) WithRawArgs(rawArgs []string) Wrap2[Struct] {
	b.RawArgs = rawArgs
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

	var initFunc func(params any) error = nil
	if b.InitFunc != nil {
		initFunc = func(params any) error {
			return b.InitFunc(params.(*Struct))
		}
	}

	var preExecuteFunc func(params any, cmd *cobra.Command, args []string) error = nil
	if b.PreExecuteFunc != nil {
		preExecuteFunc = func(params any, cmd *cobra.Command, args []string) error {
			return b.PreExecuteFunc(params.(*Struct), cmd, args)
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
		InitFunc:       initFunc,
		PreExecuteFunc: preExecuteFunc,
		RawArgs:        b.RawArgs,
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
