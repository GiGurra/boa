package boa

import (
	"fmt"
	"github.com/spf13/cobra"
	"reflect"
	"time"
)

type SupportedTypes interface {
	string | int | int32 | int64 | bool | float64 | float32 | time.Time |
		[]int | []int32 | []int64 | []float32 | []float64 | []string
}

type Cmd struct {
	Use            string
	Short          string
	Long           string
	Version        string
	Args           cobra.PositionalArgs
	SubCommands    []*cobra.Command
	Params         any
	ParamEnrich    ParamEnricher
	RunFunc        func(cmd *cobra.Command, args []string)
	UseCobraErrLog bool
	SortFlags      bool
	ValidArgs      []string
	ValidArgsFunc  func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)
	// To customize params
	InitFunc        func(params any) error
	PreValidateFunc func(params any, cmd *cobra.Command, args []string) error
	PreExecuteFunc  func(params any, cmd *cobra.Command, args []string) error
	// To inject raw args instead of using os.Args
	RawArgs []string
}

func HasValue(f Param) bool {
	return f.wasSetByEnv() || f.wasSetOnCli() || f.hasDefaultValue() || f.wasSetByInject()
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

func (b Cmd) WithSubCmds(cmd ...*cobra.Command) Cmd {
	b.SubCommands = append(b.SubCommands, cmd...)
	return b
}

func Compose(structPtrs ...any) *StructComposition {
	return &StructComposition{
		StructPtrs: structPtrs,
	}
}

type StructComposition struct {
	StructPtrs []any
}

func (b Cmd) ToCobra() *cobra.Command {
	return b.toCobraImpl()
}

type ResultHandler struct {
	Panic   func(any)
	Failure func(error)
	Success func()
}

func RunH(cmd *cobra.Command, handler ResultHandler) {
	runImpl(cmd, handler)
}

//goland:noinspection GoUnusedExportedFunction
func Run(cmd *cobra.Command) {
	RunH(cmd, ResultHandler{})
}

func (b Cmd) Run() {
	b.RunH(ResultHandler{})
}

func (b Cmd) RunH(handler ResultHandler) {
	RunH(b.ToCobra(), handler)
}

func Default[T SupportedTypes](val T) *T {
	return &val
}

func Validate[T any](structPtr *T, w Cmd) error {
	w.Params = structPtr
	w.RunFunc = func(cmd *cobra.Command, args []string) {}
	w.UseCobraErrLog = false
	var err error
	handler := ResultHandler{
		Panic: func(a any) {
			err = fmt.Errorf("panic: %v", a)
		},
		Failure: func(e error) {
			err = e
		},
	}
	cobraCmd := w.ToCobra()
	cobraCmd.SilenceErrors = true
	cobraCmd.SilenceUsage = true
	RunH(cobraCmd, handler)
	return err
}

type CfgStructInit interface {
	Init() error
}

type CfgStructPreExecute interface {
	PreExecute() error
}

type CfgStructPreValidate interface {
	PreValidate() error
}
