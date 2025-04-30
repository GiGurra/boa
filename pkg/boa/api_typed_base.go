package boa

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"reflect"
)

type NoParams struct{}

type CmdT[Struct any] struct {
	Use             string
	Short           string
	Long            string
	Version         string
	Args            cobra.PositionalArgs
	SubCommands     []*cobra.Command
	Params          *Struct
	ParamEnrich     ParamEnricher
	RunFunc         func(params *Struct, cmd *cobra.Command, args []string)
	InitFunc        func(params *Struct) error
	PreValidateFunc func(params *Struct, cmd *cobra.Command, args []string) error
	PreExecuteFunc  func(params *Struct, cmd *cobra.Command, args []string) error
	UseCobraErrLog  bool
	SortFlags       bool
	ValidArgs       []string
	ValidArgsFunc   func(params *Struct, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)
	RawArgs         []string
}

func NewCmdT[Struct any](use string) CmdT[Struct] {
	var params Struct
	return NewCmdT2(use, &params)
}

func NewCmdT2[Struct any](use string, params *Struct) CmdT[Struct] {

	if reflect.TypeOf(params).Kind() != reflect.Ptr {
		panic(fmt.Errorf("expected pointer to struct"))
	}

	if reflect.TypeOf(params).Elem().Kind() != reflect.Struct {
		panic(fmt.Errorf("expected pointer to struct"))
	}

	return CmdT[Struct]{
		Use:         use,
		Params:      params,
		ParamEnrich: ParamEnricherDefault,
	}
}

func (b CmdT[Struct]) WithUse(use string) CmdT[Struct] {
	b.Use = use
	return b
}

func (b CmdT[Struct]) WithShort(short string) CmdT[Struct] {
	b.Short = short
	return b
}

func (b CmdT[Struct]) WithLong(long string) CmdT[Struct] {
	b.Long = long
	return b
}

func (b CmdT[Struct]) WithVersion(version string) CmdT[Struct] {
	b.Version = version
	return b
}

func (b CmdT[Struct]) WithArgs(args cobra.PositionalArgs) CmdT[Struct] {
	b.Args = args
	return b
}

func (b CmdT[Struct]) WithParamEnrich(enricher ParamEnricher) CmdT[Struct] {
	b.ParamEnrich = enricher
	return b
}

func (b CmdT[Struct]) WithRunFunc(run func(params *Struct)) CmdT[Struct] {
	return b.WithRunFunc3(func(params *Struct, _ *cobra.Command, _ []string) {
		run(params)
	})
}

func (b CmdT[Struct]) WithRunFunc3(run func(params *Struct, cmd *cobra.Command, args []string)) CmdT[Struct] {
	b.RunFunc = run
	return b
}

func (b CmdT[Struct]) WithUseCobraErrLog(useCobraErrLog bool) CmdT[Struct] {
	b.UseCobraErrLog = useCobraErrLog
	return b
}

func (b CmdT[Struct]) WithSortFlags(sortFlags bool) CmdT[Struct] {
	b.SortFlags = sortFlags
	return b
}

func (b CmdT[Struct]) WithValidArgs(validArgs []string) CmdT[Struct] {
	b.ValidArgs = validArgs
	return b
}

func (b CmdT[Struct]) WithValidArgsFunc(validArgsFunc func(params *Struct, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)) CmdT[Struct] {
	b.ValidArgsFunc = validArgsFunc
	return b
}

func (b CmdT[Struct]) WithSubCmds(cmd ...*cobra.Command) CmdT[Struct] {
	b.SubCommands = append(b.SubCommands, cmd...)
	return b
}

func (b CmdT[Struct]) WithPreValidateFunc(preValidateFunc func(params *Struct, cmd *cobra.Command, args []string)) CmdT[Struct] {
	return b.WithPreValidateFuncE(func(params *Struct, cmd *cobra.Command, args []string) error {
		preValidateFunc(params, cmd, args)
		return nil
	})
}

func (b CmdT[Struct]) WithPreValidateFuncE(preValidateFunc func(params *Struct, cmd *cobra.Command, args []string) error) CmdT[Struct] {
	b.PreValidateFunc = preValidateFunc
	return b
}

func (b CmdT[Struct]) WithPreExecuteFunc(preExecuteFunc func(params *Struct, cmd *cobra.Command, args []string)) CmdT[Struct] {
	return b.WithPreExecuteFuncE(func(params *Struct, cmd *cobra.Command, args []string) error {
		preExecuteFunc(params, cmd, args)
		return nil
	})
}

func (b CmdT[Struct]) WithPreExecuteFuncE(preExecuteFunc func(params *Struct, cmd *cobra.Command, args []string) error) CmdT[Struct] {
	b.PreExecuteFunc = preExecuteFunc
	return b
}

func (b CmdT[Struct]) WithInitFunc(initFunc func(params *Struct)) CmdT[Struct] {
	return b.WithInitFuncE(func(params *Struct) error {
		initFunc(params)
		return nil
	})
}

func (b CmdT[Struct]) WithInitFuncE(initFunc func(params *Struct) error) CmdT[Struct] {
	b.InitFunc = initFunc
	return b
}

// WithRawArgs sets the raw args to be used instead of os.Args. Mostly used for testing purposes.
func (b CmdT[Struct]) WithRawArgs(rawArgs []string) CmdT[Struct] {
	b.RawArgs = rawArgs
	return b
}

func (b CmdT[Struct]) ToCmd() Cmd {

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

	var preValidateFunc func(params any, cmd *cobra.Command, args []string) error = nil
	if b.PreValidateFunc != nil {
		preValidateFunc = func(params any, cmd *cobra.Command, args []string) error {
			return b.PreValidateFunc(params.(*Struct), cmd, args)
		}
	}

	return Cmd{
		Use:             b.Use,
		Short:           b.Short,
		Long:            b.Long,
		Version:         b.Version,
		Args:            b.Args,
		SubCommands:     b.SubCommands,
		Params:          b.Params,
		ParamEnrich:     b.ParamEnrich,
		RunFunc:         runFcn,
		UseCobraErrLog:  b.UseCobraErrLog,
		SortFlags:       b.SortFlags,
		ValidArgs:       b.ValidArgs,
		ValidArgsFunc:   validArgsFunc,
		InitFunc:        initFunc,
		PreValidateFunc: preValidateFunc,
		PreExecuteFunc:  preExecuteFunc,
		RawArgs:         b.RawArgs,
	}
}

func (b CmdT[Struct]) ToCobra() *cobra.Command {
	return b.ToCmd().ToCobra()
}

func (b CmdT[Struct]) Run() {
	RunH(b.ToCobra(), ResultHandler{})
}

func (b CmdT[Struct]) RunArgs(rawArgs []string) {
	b.WithRawArgs(rawArgs).Run()
}

func (b CmdT[Struct]) RunH(handler ResultHandler) {
	RunH(b.ToCobra(), handler)
}

func (b CmdT[Struct]) RunHArgs(handler ResultHandler, rawArgs []string) {
	b.WithRawArgs(rawArgs).RunH(handler)
}

func (b CmdT[Struct]) Validate() error {
	return Validate(b.Params, b.ToCmd())
}

func UnMarshalFromFileParam[T any](
	fileParam Param,
	v *T,
	unmarshalFunc func(data []byte, v any) error,
) error {
	if !fileParam.HasValue() {
		return nil
	} else {
		valuePtrAny := fileParam.valuePtrF()
		valuePtrStr, ok := valuePtrAny.(*string)
		if !ok {
			return fmt.Errorf("expected string value, got %T", valuePtrAny)
		}
		if valuePtrStr == nil {
			return fmt.Errorf("expected string value, got nil")
		}
		if *valuePtrStr == "" {
			return fmt.Errorf("expected string value, got empty string")
		}
		fileContents, err := os.ReadFile(*valuePtrStr)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", *valuePtrStr, err)
		}

		if unmarshalFunc == nil {
			unmarshalFunc = json.Unmarshal
		}

		err = unmarshalFunc(fileContents, v)
		if err != nil {
			return fmt.Errorf("failed to unmarshal file %s: %w", *valuePtrStr, err)
		}

		return nil
	}
}
