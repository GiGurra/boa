package boa

import (
	"fmt"
	"reflect"

	"github.com/spf13/cobra"
)

// NoParams is an empty struct that can be used when a command doesn't need parameters.
type NoParams struct{}

// CmdT is a generic command type with type-safe parameter handling.
// Create commands using struct literal syntax:
//
//	boa.CmdT[Params]{
//	    Use:   "my-app",
//	    Short: "description",
//	    RunFunc: func(params *Params, cmd *cobra.Command, args []string) {
//	        // use params directly
//	    },
//	}.Run()
type CmdT[Struct any] struct {
	// Use is the one-line usage message shown in help
	Use string
	// Short is a short description shown in the 'help' output
	Short string
	// Long is the long description shown in the 'help <this-command>' output
	Long string
	// Version is the version for this command
	Version string
	// Aliases are alternative names for this command
	Aliases []string
	// GroupID is the group id to which this command belongs (for help categorization)
	GroupID string
	// Groups defines command groups for organizing subcommands in help output (optional, auto-generated if not specified)
	Groups []*cobra.Group
	// Args defines how cobra should validate positional arguments
	Args cobra.PositionalArgs
	// SubCmds contains sub-commands for this command
	SubCmds []*cobra.Command
	// Params is a pointer to the struct containing command parameters
	Params *Struct
	// ParamEnrich is a function that enriches parameter definitions
	ParamEnrich ParamEnricher
	// RunFunc is the function to run when this command is called, with type-safe parameters
	RunFunc func(params *Struct, cmd *cobra.Command, args []string)
	// RunFuncCtx is the function to run when this command is called, with access to HookContext
	RunFuncCtx func(ctx *HookContext, params *Struct, cmd *cobra.Command, args []string)
	// RunFuncE is like RunFunc but returns an error
	RunFuncE func(params *Struct, cmd *cobra.Command, args []string) error
	// RunFuncCtxE is like RunFuncCtx but returns an error
	RunFuncCtxE func(ctx *HookContext, params *Struct, cmd *cobra.Command, args []string) error
	// InitFunc runs during initialization with type-safe parameters
	InitFunc func(params *Struct, cmd *cobra.Command) error
	// PostCreateFunc runs after cobra flags are created but before parsing
	PostCreateFunc func(params *Struct, cmd *cobra.Command) error
	// PreValidateFunc runs after flags are parsed but before validation
	PreValidateFunc func(params *Struct, cmd *cobra.Command, args []string) error
	// PreExecuteFunc runs after validation but before command execution
	PreExecuteFunc func(params *Struct, cmd *cobra.Command, args []string) error
	// InitFuncCtx runs during initialization with access to HookContext
	InitFuncCtx func(ctx *HookContext, params *Struct, cmd *cobra.Command) error
	// PostCreateFuncCtx runs after cobra flags are created with access to HookContext
	PostCreateFuncCtx func(ctx *HookContext, params *Struct, cmd *cobra.Command) error
	// PreValidateFuncCtx runs after flags are parsed but before validation with HookContext
	PreValidateFuncCtx func(ctx *HookContext, params *Struct, cmd *cobra.Command, args []string) error
	// PreExecuteFuncCtx runs after validation but before execution with HookContext
	PreExecuteFuncCtx func(ctx *HookContext, params *Struct, cmd *cobra.Command, args []string) error
	// UseCobraErrLog determines whether to use Cobra's error logging
	UseCobraErrLog bool
	// SortFlags determines whether to sort command flags alphabetically
	SortFlags bool
	// ValidArgs is a list of valid non-flag arguments
	ValidArgs []string
	// ValidArgsFunc is a function returning valid arguments for bash completion
	ValidArgsFunc func(params *Struct, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)
	// RawArgs allows injecting command line arguments instead of using os.Args
	RawArgs []string
}

// ToCmd converts a type-safe CmdT to a non-generic Cmd.
func (b CmdT[Struct]) ToCmd() Cmd {

	if b.Params == nil {
		b.Params = new(Struct)
	}

	// Validate that Params is a struct
	if reflect.TypeOf(b.Params).Kind() != reflect.Ptr {
		panic(fmt.Errorf("expected pointer to struct"))
	}
	if reflect.TypeOf(b.Params).Elem().Kind() != reflect.Struct {
		panic(fmt.Errorf("expected pointer to struct"))
	}

	var runFcn func(cmd *cobra.Command, args []string)
	if b.RunFunc != nil {
		runFcn = func(cmd *cobra.Command, args []string) {
			b.RunFunc(b.Params, cmd, args)
		}
	}

	var validArgsFunc func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)
	if b.ValidArgsFunc != nil {
		validArgsFunc = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return b.ValidArgsFunc(b.Params, cmd, args, toComplete)
		}
	}

	var initFunc func(params any, cmd *cobra.Command) error
	if b.InitFunc != nil {
		initFunc = func(params any, cmd *cobra.Command) error {
			return b.InitFunc(params.(*Struct), cmd)
		}
	}

	var postCreateFunc func(params any, cmd *cobra.Command) error
	if b.PostCreateFunc != nil {
		postCreateFunc = func(params any, cmd *cobra.Command) error {
			return b.PostCreateFunc(params.(*Struct), cmd)
		}
	}

	var preExecuteFunc func(params any, cmd *cobra.Command, args []string) error
	if b.PreExecuteFunc != nil {
		preExecuteFunc = func(params any, cmd *cobra.Command, args []string) error {
			return b.PreExecuteFunc(params.(*Struct), cmd, args)
		}
	}

	var preValidateFunc func(params any, cmd *cobra.Command, args []string) error
	if b.PreValidateFunc != nil {
		preValidateFunc = func(params any, cmd *cobra.Command, args []string) error {
			return b.PreValidateFunc(params.(*Struct), cmd, args)
		}
	}

	var initFuncCtx func(ctx *HookContext, params any, cmd *cobra.Command) error
	if b.InitFuncCtx != nil {
		initFuncCtx = func(ctx *HookContext, params any, cmd *cobra.Command) error {
			return b.InitFuncCtx(ctx, params.(*Struct), cmd)
		}
	}

	var postCreateFuncCtx func(ctx *HookContext, params any, cmd *cobra.Command) error
	if b.PostCreateFuncCtx != nil {
		postCreateFuncCtx = func(ctx *HookContext, params any, cmd *cobra.Command) error {
			return b.PostCreateFuncCtx(ctx, params.(*Struct), cmd)
		}
	}

	var preValidateFuncCtx func(ctx *HookContext, params any, cmd *cobra.Command, args []string) error
	if b.PreValidateFuncCtx != nil {
		preValidateFuncCtx = func(ctx *HookContext, params any, cmd *cobra.Command, args []string) error {
			return b.PreValidateFuncCtx(ctx, params.(*Struct), cmd, args)
		}
	}

	var preExecuteFuncCtx func(ctx *HookContext, params any, cmd *cobra.Command, args []string) error
	if b.PreExecuteFuncCtx != nil {
		preExecuteFuncCtx = func(ctx *HookContext, params any, cmd *cobra.Command, args []string) error {
			return b.PreExecuteFuncCtx(ctx, params.(*Struct), cmd, args)
		}
	}

	var runFuncCtx func(ctx *HookContext, cmd *cobra.Command, args []string)
	if b.RunFuncCtx != nil {
		runFuncCtx = func(ctx *HookContext, cmd *cobra.Command, args []string) {
			b.RunFuncCtx(ctx, b.Params, cmd, args)
		}
	}

	var runFuncE func(cmd *cobra.Command, args []string) error
	if b.RunFuncE != nil {
		runFuncE = func(cmd *cobra.Command, args []string) error {
			return b.RunFuncE(b.Params, cmd, args)
		}
	}

	var runFuncCtxE func(ctx *HookContext, cmd *cobra.Command, args []string) error
	if b.RunFuncCtxE != nil {
		runFuncCtxE = func(ctx *HookContext, cmd *cobra.Command, args []string) error {
			return b.RunFuncCtxE(ctx, b.Params, cmd, args)
		}
	}

	// Due to golang nil upcast behavior
	var params any
	if b.Params != nil {
		params = b.Params
	}

	return Cmd{
		Use:                b.Use,
		Short:              b.Short,
		Long:               b.Long,
		Version:            b.Version,
		Aliases:            b.Aliases,
		GroupID:            b.GroupID,
		Groups:             b.Groups,
		Args:               b.Args,
		SubCmds:            b.SubCmds,
		Params:             params,
		ParamEnrich:        b.ParamEnrich,
		RunFunc:            runFcn,
		RunFuncCtx:         runFuncCtx,
		RunFuncE:           runFuncE,
		RunFuncCtxE:        runFuncCtxE,
		UseCobraErrLog:     b.UseCobraErrLog,
		SortFlags:          b.SortFlags,
		ValidArgs:          b.ValidArgs,
		ValidArgsFunc:      validArgsFunc,
		InitFunc:           initFunc,
		PostCreateFunc:     postCreateFunc,
		PreValidateFunc:    preValidateFunc,
		PreExecuteFunc:     preExecuteFunc,
		InitFuncCtx:        initFuncCtx,
		PostCreateFuncCtx:  postCreateFuncCtx,
		PreValidateFuncCtx: preValidateFuncCtx,
		PreExecuteFuncCtx:  preExecuteFuncCtx,
		RawArgs:            b.RawArgs,
	}
}

// ToCobra converts this command to a cobra.Command.
func (b CmdT[Struct]) ToCobra() *cobra.Command {
	return b.ToCmd().ToCobra()
}

// Run executes the command with default error handling.
func (b CmdT[Struct]) Run() {
	runH(b.ToCobra(), resultHandler{})
}

// RunArgs executes the command with the provided arguments and default error handling.
func (b CmdT[Struct]) RunArgs(rawArgs []string) {
	b.RawArgs = rawArgs
	b.Run()
}

// Validate validates parameter values without executing the command's RunFunc.
func (b CmdT[Struct]) Validate() error {
	return b.ToCmd().Validate()
}

// ToCobraE converts this command to a cobra.Command that uses RunE for error handling.
func (b CmdT[Struct]) ToCobraE() (*cobra.Command, error) {
	return b.ToCmd().ToCobraE()
}

// RunE executes the command and returns any error that occurred.
func (b CmdT[Struct]) RunE() error {
	return b.ToCmd().RunE()
}

// RunArgsE executes the command with the provided arguments and returns any error.
func (b CmdT[Struct]) RunArgsE(rawArgs []string) error {
	b.RawArgs = rawArgs
	return b.RunE()
}
