// Package boa provides a declarative CLI and environment variable parameter utility.
package boa

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"reflect"
)

// NoParams is an empty struct that can be used when a command doesn't need parameters.
type NoParams struct{}

// CmdT is a generic version of Cmd with type-safe parameter handling.
// It provides a fluent builder API for creating commands with strongly-typed parameter structs.
// The Struct type parameter represents the parameter struct type for this command.
type CmdT[Struct any] struct {
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
	// Params is a pointer to the struct containing command parameters
	Params *Struct
	// ParamEnrich is a function that enriches parameter definitions
	ParamEnrich ParamEnricher
	// RunFunc is the function to run when this command is called, with type-safe parameters
	RunFunc func(params *Struct, cmd *cobra.Command, args []string)
	// InitFunc runs during initialization with type-safe parameters
	InitFunc func(params *Struct) error
	// PreValidateFunc runs after flags are parsed but before validation with type-safe parameters
	PreValidateFunc func(params *Struct, cmd *cobra.Command, args []string) error
	// PreExecuteFunc runs after validation but before command execution with type-safe parameters
	PreExecuteFunc func(params *Struct, cmd *cobra.Command, args []string) error
	// UseCobraErrLog determines whether to use Cobra's error logging
	UseCobraErrLog bool
	// SortFlags determines whether to sort command flags alphabetically
	SortFlags bool
	// ValidArgs is a list of valid non-flag arguments
	ValidArgs []string
	// ValidArgsFunc is a function returning valid arguments for bash completion with type-safe parameters
	ValidArgsFunc func(params *Struct, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)
	// RawArgs allows injecting command line arguments instead of using os.Args
	RawArgs []string
}

// NewCmdT creates a new command with type-safe parameters.
// It automatically creates a parameter struct of type Struct and returns a command builder.
// This is the primary entry point for the fluent builder API.
func NewCmdT[Struct any](use string) CmdT[Struct] {
	var params Struct
	return NewCmdT2(use, &params)
}

// NewCmdT2 creates a new command with the provided parameter struct.
// This allows using an existing or pre-configured parameter struct.
func NewCmdT2[Struct any](use string, params *Struct) CmdT[Struct] {
	// Validate that params is a pointer to a struct
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

// WithUse sets the command's use string and returns the updated command.
func (b CmdT[Struct]) WithUse(use string) CmdT[Struct] {
	b.Use = use
	return b
}

// WithShort sets the command's short description and returns the updated command.
func (b CmdT[Struct]) WithShort(short string) CmdT[Struct] {
	b.Short = short
	return b
}

// WithLong sets the command's long description and returns the updated command.
func (b CmdT[Struct]) WithLong(long string) CmdT[Struct] {
	b.Long = long
	return b
}

// WithVersion sets the command's version and returns the updated command.
func (b CmdT[Struct]) WithVersion(version string) CmdT[Struct] {
	b.Version = version
	return b
}

// WithArgs sets the command's positional arguments cobra validation configuration
func (b CmdT[Struct]) WithArgs(args cobra.PositionalArgs) CmdT[Struct] {
	b.Args = args
	return b
}

// WithParamEnrich sets the parameter enrichment function.
func (b CmdT[Struct]) WithParamEnrich(enricher ParamEnricher) CmdT[Struct] {
	b.ParamEnrich = enricher
	return b
}

// WithRunFunc sets the command's run function with a simplified signature
// that only includes the parameter struct. This is useful when you don't need
// access to the cobra.Command or args.
func (b CmdT[Struct]) WithRunFunc(run func(params *Struct)) CmdT[Struct] {
	return b.WithRunFunc3(func(params *Struct, _ *cobra.Command, _ []string) {
		run(params)
	})
}

// WithRunFunc3 sets the command's run function with the full signature
// that includes the parameter struct, cobra.Command, and args.
func (b CmdT[Struct]) WithRunFunc3(run func(params *Struct, cmd *cobra.Command, args []string)) CmdT[Struct] {
	b.RunFunc = run
	return b
}

// WithUseCobraErrLog sets the UseCobraErrLog flag, determining if Cobra's error logging should be used, and returns the updated command.
func (b CmdT[Struct]) WithUseCobraErrLog(useCobraErrLog bool) CmdT[Struct] {
	b.UseCobraErrLog = useCobraErrLog
	return b
}

// WithSortFlags sets the SortFlags option for the command, enabling or disabling flag sorting, and returns the updated command.
func (b CmdT[Struct]) WithSortFlags(sortFlags bool) CmdT[Struct] {
	b.SortFlags = sortFlags
	return b
}

// WithValidArgs sets the valid positional arguments for the command and returns the updated command configuration.
func (b CmdT[Struct]) WithValidArgs(validArgs []string) CmdT[Struct] {
	b.ValidArgs = validArgs
	return b
}

// WithValidArgsFunc sets a dynamic function for generating valid arguments and shell completions for this command.
func (b CmdT[Struct]) WithValidArgsFunc(validArgsFunc func(params *Struct, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)) CmdT[Struct] {
	b.ValidArgsFunc = validArgsFunc
	return b
}

// WithCobraSubCmds sets the sub-commands for this command.
func (b CmdT[Struct]) WithCobraSubCmds(cmd ...*cobra.Command) CmdT[Struct] {
	b.SubCommands = append(b.SubCommands, cmd...)
	return b
}

// WithSubCmds sets the sub-commands for this command.
func (b CmdT[Struct]) WithSubCmds(cmd ...CmdIfc) CmdT[Struct] {
	for _, c := range cmd {
		b.SubCommands = append(b.SubCommands, c.ToCobra())
	}
	return b
}

// WithPreValidateFunc sets a function to run after flags are parsed but before validation.
// This version does not return an error and always succeeds.
func (b CmdT[Struct]) WithPreValidateFunc(preValidateFunc func(params *Struct, cmd *cobra.Command, args []string)) CmdT[Struct] {
	return b.WithPreValidateFuncE(func(params *Struct, cmd *cobra.Command, args []string) error {
		preValidateFunc(params, cmd, args)
		return nil
	})
}

// WithPreValidateFuncE sets a function to run after flags are parsed but before validation.
// This version can return an error to abort command execution.
// This is useful for loading configurations from files or other pre-validation setup.
func (b CmdT[Struct]) WithPreValidateFuncE(preValidateFunc func(params *Struct, cmd *cobra.Command, args []string) error) CmdT[Struct] {
	b.PreValidateFunc = preValidateFunc
	return b
}

// WithPreExecuteFunc sets a function to run after validation but before command execution.
// This version does not return an error and always succeeds.
func (b CmdT[Struct]) WithPreExecuteFunc(preExecuteFunc func(params *Struct, cmd *cobra.Command, args []string)) CmdT[Struct] {
	return b.WithPreExecuteFuncE(func(params *Struct, cmd *cobra.Command, args []string) error {
		preExecuteFunc(params, cmd, args)
		return nil
	})
}

// WithPreExecuteFuncE sets a function to run after validation but before command execution.
// This version can return an error to abort command execution.
// This is useful for setting up resources needed for command execution.
func (b CmdT[Struct]) WithPreExecuteFuncE(preExecuteFunc func(params *Struct, cmd *cobra.Command, args []string) error) CmdT[Struct] {
	b.PreExecuteFunc = preExecuteFunc
	return b
}

// WithInitFunc sets a function to run during initialization, before any flags are parsed.
// This version does not return an error and always succeeds.
func (b CmdT[Struct]) WithInitFunc(initFunc func(params *Struct)) CmdT[Struct] {
	return b.WithInitFuncE(func(params *Struct) error {
		initFunc(params)
		return nil
	})
}

// WithInitFuncE sets a function to run during initialization, before any flags are parsed.
// This version can return an error to abort command execution.
// This is useful for setting up default values and parameter relationships.
func (b CmdT[Struct]) WithInitFuncE(initFunc func(params *Struct) error) CmdT[Struct] {
	b.InitFunc = initFunc
	return b
}

// WithRawArgs sets the raw args to be used instead of os.Args. Mostly used for testing purposes.
func (b CmdT[Struct]) WithRawArgs(rawArgs []string) CmdT[Struct] {
	b.RawArgs = rawArgs
	return b
}

// ToCmd converts a type-safe CmdT to a non-generic Cmd.
// This converts the type-safe functions to their non-generic equivalents.
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

// ToCobra converts this command to a cobra.Command.
// This is used when you want to integrate with existing Cobra command structures.
func (b CmdT[Struct]) ToCobra() *cobra.Command {
	return b.ToCmd().ToCobra()
}

// Run executes the command with default error handling.
// This is the most common way to run a command.
func (b CmdT[Struct]) Run() {
	RunH(b.ToCobra(), ResultHandler{})
}

// RunArgs executes the command with the provided arguments and default error handling.
// This is useful for testing and programmatic execution.
func (b CmdT[Struct]) RunArgs(rawArgs []string) {
	b.WithRawArgs(rawArgs).Run()
}

// RunH executes the command with the specified ResultHandler for
// custom error and panic handling.
func (b CmdT[Struct]) RunH(handler ResultHandler) {
	RunH(b.ToCobra(), handler)
}

// RunHArgs executes the command with the provided arguments and custom error handling.
// This is useful for testing and programmatic execution with custom error handling.
func (b CmdT[Struct]) RunHArgs(handler ResultHandler, rawArgs []string) {
	b.WithRawArgs(rawArgs).RunH(handler)
}

// Validate validates parameter values without executing the command's RunFunc.
// This is useful for testing.
func (b CmdT[Struct]) Validate() error {
	return b.ToCmd().Validate()
}

// UnMarshalFromFileParam reads a file path from a parameter and unmarshals its contents into a target struct.
// This is useful for loading configuration from files specified as command-line arguments.
//
// Parameters:
//   - fileParam: The parameter containing the file path (must be a string parameter)
//   - v: Pointer to the target struct to unmarshal into
//   - unmarshalFunc: Optional custom unmarshal function (defaults to json.Unmarshal if nil)
//
// Returns an error if the file can't be read or unmarshalled properly.
// Returns nil if fileParam has no value (skipping the unmarshalling).
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
