package boa

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"reflect"
	"time"
)

type SupportedTypes interface {
	string | int | int32 | int64 | bool | float64 | float32 | time.Time |
		[]int | []int32 | []int64 | []float32 | []float64 | []string
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
	// To customize params
	InitFunc       func(params any) error
	PreExecuteFunc func(params any, cmd *cobra.Command, args []string) error
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

func (b Wrap) WithSubCommands(cmd ...*cobra.Command) Wrap {
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

func (b Wrap) ToCmd() *cobra.Command {
	return b.toCmdImpl()
}

type ResultHandler struct {
	Panic   func(any)
	Failure func(error)
	Success func()
}

func ToAppH(cmd *cobra.Command, handler ResultHandler) {

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

//goland:noinspection GoUnusedExportedFunction
func ToApp(cmd *cobra.Command) {
	ToAppH(cmd, ResultHandler{})
}

func (b Wrap) ToApp() {
	b.ToAppH(ResultHandler{})
}

func (b Wrap) ToAppH(handler ResultHandler) {
	ToAppH(b.ToCmd(), handler)
}

func Default[T SupportedTypes](val T) *T {
	return &val
}

func Validate[T any](structPtr *T, w Wrap) error {
	w.Params = structPtr
	w.Run = func(cmd *cobra.Command, args []string) {}
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
	cobraCmd := w.ToCmd()
	cobraCmd.SilenceErrors = true
	cobraCmd.SilenceUsage = true
	ToAppH(cobraCmd, handler)
	return err
}

type CfgStructInit interface {
	Init() error
}

type CfgStructPreExecute interface {
	PreExecute() error
}
