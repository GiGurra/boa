// Package boa provides a declarative CLI and environment variable parameter utility.
// It enhances and simplifies github.com/spf13/cobra by enabling straightforward
// and declarative CLI interfaces through struct-based parameter definitions.
package boa

import (
	"fmt"
	"github.com/spf13/cobra"
	"reflect"
	"time"
)

// SupportedTypes defines the Go types that can be used as parameter values.
// These types are supported for both Required and Optional parameter wrappers.
type SupportedTypes interface {
	string | int | int32 | int64 | bool | float64 | float32 | time.Time |
		[]int | []int32 | []int64 | []float32 | []float64 | []string
}

// Cmd represents a CLI command with all its configuration options.
// It serves as a wrapper around cobra.Command with additional functionality
// for parameter handling, validation, and lifecycle hooks.
type Cmd struct {
	// Use is the one-line usage message shown in help
	Use string
	// Short is a short description shown in the 'help' output
	Short string
	// Long is the long description shown in the 'help <this-command>' output
	Long string
	// Version is the version for this command
	Version string
	// Args defines how cobra should validate positional arguments
	Args cobra.PositionalArgs
	// SubCommands contains sub-commands for this command
	SubCommands []*cobra.Command
	// Params is a pointer to a struct containing command parameters
	Params any
	// ParamEnrich is a function that enriches parameter definitions
	ParamEnrich ParamEnricher
	// RunFunc is the function to run when this command is called
	RunFunc func(cmd *cobra.Command, args []string)
	// UseCobraErrLog determines whether to use Cobra's error logging
	UseCobraErrLog bool
	// SortFlags determines whether to sort command flags alphabetically
	SortFlags bool
	// ValidArgs is a list of valid non-flag arguments
	ValidArgs []string
	// ValidArgsFunc is a function returning valid arguments for bash completion
	ValidArgsFunc func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)
	// Lifecycle hook functions
	// InitFunc runs during initialization before any flags are parsed
	InitFunc func(params any, cmd *cobra.Command) error
	// PreValidateFunc runs after flags are parsed but before validation
	PreValidateFunc func(params any, cmd *cobra.Command, args []string) error
	// PreExecuteFunc runs after validation but before command execution
	PreExecuteFunc func(params any, cmd *cobra.Command, args []string) error
	// RawArgs allows injecting command line arguments instead of using os.Args
	RawArgs []string
}

// HasValue checks if a parameter has a value from any source.
// Returns true if the parameter was set by environment variable, command line,
// default value, or programmatic injection.
func HasValue(f Param) bool {
	return f.wasSetByEnv() || f.wasSetOnCli() || f.hasDefaultValue() || f.wasSetByInject()
}

// ParamEnricher is a function type that can add or modify parameter metadata.
// It's used to implement auto-generation of parameter properties like names,
// environment variables, short flags, etc.
type ParamEnricher func(alreadyProcessed []Param, param Param, paramFieldName string) error

// ParamEnricherCombine combines multiple parameter enrichers into a single function.
// The enrichers are applied in the order they are provided and an error from any
// enricher will stop the process and return the error.
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
	// ParamEnricherBool sets a default value of false for boolean parameters
	// that don't already have a default value.
	ParamEnricherBool ParamEnricher = func(alreadyProcessed []Param, param Param, paramFieldName string) error {
		if param.GetKind() == reflect.Bool && !param.hasDefaultValue() {
			param.SetDefault(Default(false))
		}
		return nil
	}

	// ParamEnricherName sets the flag name for a parameter based on its field name
	// if a name isn't already set. Converts from camelCase to kebab-case.
	ParamEnricherName ParamEnricher = func(alreadyProcessed []Param, param Param, paramFieldName string) error {
		if param.GetName() == "" {
			param.SetName(camelToKebabCase(paramFieldName))
		}
		return nil
	}

	// ParamEnricherShort sets a short name (single character) for a parameter
	// using the first character of the parameter name if available.
	// Skips setting if the character would be 'h' (reserved for help) or
	// if another parameter already uses that character.
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

	// ParamEnricherEnv sets an environment variable name for a parameter
	// based on its flag name. Converts from kebab-case to UPPER_SNAKE_CASE.
	// Only applies to non-positional parameters.
	ParamEnricherEnv ParamEnricher = func(alreadyProcessed []Param, param Param, paramFieldName string) error {
		if param.GetEnv() == "" && param.GetName() != "" && !param.isPositional() {
			param.SetEnv(kebabCaseToUpperSnakeCase(param.GetName()))
		}
		return nil
	}

	// ParamEnricherDefault is the default combination of enrichers applied to parameters.
	// It includes name generation, short flag assignment, environment variable naming,
	// and boolean default value assignment.
	ParamEnricherDefault = ParamEnricherCombine(
		ParamEnricherName,
		ParamEnricherShort,
		ParamEnricherEnv,
		ParamEnricherBool,
	)

	// ParamEnricherNone is an empty enricher that doesn't modify parameters.
	// Use this when you want to opt out of automatic parameter enrichment.
	ParamEnricherNone = ParamEnricherCombine()
)

// ParamEnricherEnvPrefix creates an enricher that adds a prefix to environment variable names.
// This is useful when you want to namespace your environment variables.
//
//goland:noinspection GoUnusedExportedFunction
func ParamEnricherEnvPrefix(prefix string) ParamEnricher {
	return func(alreadyProcessed []Param, param Param, paramFieldName string) error {
		if param.GetEnv() != "" {
			param.SetEnv(prefix + "_" + param.GetEnv())
		}
		return nil
	}
}

// WithCobraSubCmds adds sub-commands to a Cmd and returns the modified Cmd.
// This method allows for fluent chaining of command configuration.
func (b Cmd) WithCobraSubCmds(cmd ...*cobra.Command) Cmd {
	b.SubCommands = append(b.SubCommands, cmd...)
	return b
}

// WithSubCmds adds sub-commands to a Cmd and returns the modified Cmd.
// This method allows for fluent chaining of command configuration.
func (b Cmd) WithSubCmds(cmd ...CmdIfc) Cmd {
	for _, c := range cmd {
		b.SubCommands = append(b.SubCommands, c.ToCobra())
	}
	return b
}

// Compose creates a composition of multiple struct pointers that will be treated
// as a single parameter struct. This allows for combining parameters from
// different structs under one command.
func Compose(structPtrs ...any) *StructComposition {
	return &StructComposition{
		StructPtrs: structPtrs,
	}
}

// StructComposition represents a composition of multiple parameter structs
// that are treated as a single parameter source.
type StructComposition struct {
	// StructPtrs contains pointers to the structs that form the composition
	StructPtrs []any
}

// ToCobra converts a Cmd to a cobra.Command by setting up flags, parameter binding,
// and other command properties. This is used when you want to create a Cobra command
// to use with an existing Cobra command structure.
func (b Cmd) ToCobra() *cobra.Command {
	return b.toCobraImpl()
}

// ResultHandler defines handlers for different execution outcomes of a command.
// This allows custom handling of success, failure, and panic conditions.
type ResultHandler struct {
	// Panic is called when the command execution panics
	Panic func(any)
	// Failure is called when the command execution returns an error
	Failure func(error)
	// Success is called when the command execution completes successfully
	Success func()
}

// RunH executes a cobra.Command with the specified ResultHandler for
// custom error and panic handling.
func RunH(cmd *cobra.Command, handler ResultHandler) {
	runImpl(cmd, handler)
}

// Run executes a cobra.Command with default error handling.
// This is a convenience wrapper around RunH with an empty ResultHandler.
//
//goland:noinspection GoUnusedExportedFunction
func Run(cmd *cobra.Command) {
	RunH(cmd, ResultHandler{})
}

// Run executes the command with default error handling.
// This is a convenience method that creates a Cobra command from the Cmd
// and runs it with the default ResultHandler.
func (b Cmd) Run() {
	b.RunH(ResultHandler{})
}

// RunH executes the command with the specified ResultHandler for
// custom error and panic handling.
func (b Cmd) RunH(handler ResultHandler) {
	RunH(b.ToCobra(), handler)
}

// Validate validates parameter values without executing the command's RunFunc.
// This is used mostly in tests.
func (b Cmd) Validate() error {
	b.RunFunc = func(cmd *cobra.Command, args []string) {}
	b.UseCobraErrLog = false
	var err error
	handler := ResultHandler{
		Panic: func(a any) {
			err = fmt.Errorf("panic: %v", a)
		},
		Failure: func(e error) {
			err = e
		},
	}
	cobraCmd := b.ToCobra()
	cobraCmd.SilenceErrors = true
	cobraCmd.SilenceUsage = true
	RunH(cobraCmd, handler)
	return err
}

// Default creates a pointer to a value of a supported type.
// This is used to define default values for parameters in a type-safe way.
func Default[T SupportedTypes](val T) *T {
	return &val
}

// CfgStructInit is an interface that parameter structs can implement
// to perform initialization logic during command setup.
type CfgStructInit interface {
	// Init is called during initialization before any flags are parsed
	Init() error
}

// CfgStructPreExecute is an interface that parameter structs can implement
// to perform logic after validation but before command execution.
type CfgStructPreExecute interface {
	// PreExecute is called after validation but before command execution
	PreExecute() error
}

// CfgStructPreValidate is an interface that parameter structs can implement
// to perform logic after flags are parsed but before validation.
type CfgStructPreValidate interface {
	// PreValidate is called after flags are parsed but before validation
	PreValidate() error
}

type CmdIfc interface {
	ToCobra() *cobra.Command
}
