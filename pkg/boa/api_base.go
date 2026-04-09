// Package boa provides a declarative CLI and environment variable parameter utility.
// It enhances and simplifies github.com/spf13/cobra by enabling straightforward
// and declarative CLI interfaces through struct-based parameter definitions.
package boa

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"unsafe"

	"github.com/spf13/cobra"
)

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
	// Params is a pointer to a struct containing command parameters
	Params any
	// ParamEnrich is a function that enriches parameter definitions
	ParamEnrich ParamEnricher
	// RunFunc is the function to run when this command is called
	RunFunc func(cmd *cobra.Command, args []string)
	// RunFuncCtx is the function to run when this command is called, with access to HookContext
	RunFuncCtx func(ctx *HookContext, cmd *cobra.Command, args []string)
	// RunFuncE is like RunFunc but returns an error instead of requiring manual error handling
	RunFuncE func(cmd *cobra.Command, args []string) error
	// RunFuncCtxE is like RunFuncCtx but returns an error instead of requiring manual error handling
	RunFuncCtxE func(ctx *HookContext, cmd *cobra.Command, args []string) error
	// UseCobraErrLog determines whether to use Cobra's error logging
	UseCobraErrLog bool
	// SortFlags determines whether to sort command flags alphabetically
	SortFlags bool
	// ValidArgs is a list of valid non-flag arguments
	ValidArgs []string
	// ValidArgsFunc is a function returning valid arguments for bash completion
	ValidArgsFunc func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)
	// Lifecycle hook functions
	// InitFunc runs during initialization before any cobra flags are parsed or created
	InitFunc func(params any, cmd *cobra.Command) error
	// PostCreateFunc runs after cobra flags are created but before parsing
	PostCreateFunc func(params any, cmd *cobra.Command) error
	// PreValidateFunc runs after flags are parsed but before validation
	PreValidateFunc func(params any, cmd *cobra.Command, args []string) error
	// PreExecuteFunc runs after validation but before command execution
	PreExecuteFunc func(params any, cmd *cobra.Command, args []string) error
	// Context-aware lifecycle hooks (provide access to parameter mirrors)
	// InitFuncCtx runs during initialization with access to HookContext
	InitFuncCtx func(ctx *HookContext, params any, cmd *cobra.Command) error
	// PostCreateFuncCtx runs after cobra flags are created with access to HookContext
	PostCreateFuncCtx func(ctx *HookContext, params any, cmd *cobra.Command) error
	// PreValidateFuncCtx runs after flags are parsed but before validation with access to HookContext
	PreValidateFuncCtx func(ctx *HookContext, params any, cmd *cobra.Command, args []string) error
	// PreExecuteFuncCtx runs after validation but before command execution with access to HookContext
	PreExecuteFuncCtx func(ctx *HookContext, params any, cmd *cobra.Command, args []string) error
	// ConfigUnmarshal specifies the unmarshal function for config files loaded via the configfile tag.
	// If nil, defaults to json.Unmarshal.
	ConfigUnmarshal func([]byte, any) error
	// RawArgs allows injecting command line arguments instead of using os.Args
	RawArgs []string
}

// HasValue checks if a parameter has a value from any source.
// Returns true if the parameter was set by environment variable, command line,
// config file, default value, or programmatic injection.
func HasValue(f Param) bool {
	if f.wasSetByEnv() || f.wasSetOnCli() || f.hasDefaultValue() || f.wasSetByInject() {
		return true
	}
	if pm, ok := f.(*paramMeta); ok && pm.setByConfig {
		return true
	}
	return false
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
			wantShort := string(param.GetName()[0])
			if wantShort == "h" {
				return nil
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
	// It includes name generation, short flag assignment, and boolean default value assignment.
	// Environment variable binding is NOT included by default - add ParamEnricherEnv explicitly if needed.
	ParamEnricherDefault = ParamEnricherCombine(
		ParamEnricherName,
		ParamEnricherShort,
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

// ToCobra converts a Cmd to a cobra.Command by setting up flags, parameter binding,
// and other command properties.
func (b Cmd) ToCobra() *cobra.Command {
	return b.toCobraImpl()
}

// resultHandler defines handlers for different execution outcomes of a command.
// Used internally by Run() and Validate().
type resultHandler struct {
	Panic   func(any)
	Failure func(error)
	Success func()
}

// runH executes a cobra.Command with the specified resultHandler.
func runH(cmd *cobra.Command, handler resultHandler) {
	runImpl(cmd, handler)
}

// Run executes the command with default error handling.
func (b Cmd) Run() {
	runH(b.ToCobra(), resultHandler{})
}

// RunArgs executes the command with the provided arguments and default error handling.
func (b Cmd) RunArgs(rawArgs []string) {
	b.RawArgs = rawArgs
	runH(b.ToCobra(), resultHandler{})
}

// Validate validates parameter values without executing the command's RunFunc.
// This is used mostly in tests.
func (b Cmd) Validate() error {
	b.RunFunc = func(cmd *cobra.Command, args []string) {}
	b.UseCobraErrLog = false
	var err error
	handler := resultHandler{
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
	runH(cobraCmd, handler)
	return err
}

// ToCobraE converts a Cmd to a cobra.Command that uses RunE for error handling.
// Returns an error if command setup fails (e.g., invalid configuration, hook errors).
func (b Cmd) ToCobraE() (*cobra.Command, error) {
	return b.toCobraImplE()
}

// RunE executes the command and returns any error that occurred.
// All errors (from hooks like InitFunc, PreValidate, PreExecute, and RunFuncE) are
// returned as errors rather than causing panics.
func (b Cmd) RunE() error {
	cmd, err := b.ToCobraE()
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	return cmd.Execute()
}

// RunArgsE executes the command with the provided arguments and returns any error.
func (b Cmd) RunArgsE(rawArgs []string) error {
	b.RawArgs = rawArgs
	return b.RunE()
}

// Default creates a pointer to a value of a supported type.
// This is used to define default values for parameters programmatically
// via HookContext.GetParam().SetDefault().
func Default[T any](val T) *T {
	return &val
}

// CfgStructInit is an interface that parameter structs can implement
// to perform initialization logic during command setup.
type CfgStructInit interface {
	Init() error
}

// CfgStructPreExecute is an interface that parameter structs can implement
// to perform logic after validation but before command execution.
type CfgStructPreExecute interface {
	PreExecute() error
}

// CfgStructPreValidate is an interface that parameter structs can implement
// to perform logic after flags are parsed but before validation.
type CfgStructPreValidate interface {
	PreValidate() error
}

// CfgStructInitCtx is an interface that parameter structs can implement
// to perform initialization logic with access to the HookContext.
type CfgStructInitCtx interface {
	InitCtx(ctx *HookContext) error
}

// CfgStructPreValidateCtx is an interface that parameter structs can implement
// to perform logic after flags are parsed but before validation with access to HookContext.
type CfgStructPreValidateCtx interface {
	PreValidateCtx(ctx *HookContext) error
}

// CfgStructPreExecuteCtx is an interface that parameter structs can implement
// to perform logic after validation but before command execution with access to HookContext.
type CfgStructPreExecuteCtx interface {
	PreExecuteCtx(ctx *HookContext) error
}

// CfgStructPostCreate is an interface that parameter structs can implement
// to perform logic after cobra flags are created but before parsing.
type CfgStructPostCreate interface {
	PostCreate() error
}

// CfgStructPostCreateCtx is an interface that parameter structs can implement
// to perform logic after cobra flags are created but before parsing with access to HookContext.
type CfgStructPostCreateCtx interface {
	PostCreateCtx(ctx *HookContext) error
}

// CmdIfc common interface between Cmd and CmdT for reusing code.
type CmdIfc interface {
	ToCobra() *cobra.Command
}

// HookContext provides access to parameter mirrors and advanced configuration APIs
// within startup hooks. This allows hooks to access and modify parameters
// programmatically (SetDefault, SetAlternatives, SetRequiredFn, etc.).
type HookContext struct {
	rawAddrToMirror map[unsafe.Pointer]Param
}

// GetParam returns the Param for any field pointer.
// This provides a unified API for accessing parameter configuration.
//
// Usage:
//
//	type Params struct {
//	    Name string
//	    Age  int
//	}
//	boa.CmdT[Params]{
//	    Use: "cmd",
//	    InitFuncCtx: func(ctx *boa.HookContext, params *Params, cmd *cobra.Command) error {
//	        nameParam := ctx.GetParam(&params.Name)
//	        nameParam.SetDefault(boa.Default("default-name"))
//	        return nil
//	    },
//	}
func (c *HookContext) GetParam(fieldPtr any) Param {
	if param, ok := fieldPtr.(Param); ok {
		return param
	}
	if c.rawAddrToMirror == nil {
		return nil
	}
	addr := reflect.ValueOf(fieldPtr).UnsafePointer()
	return c.rawAddrToMirror[addr]
}

// AllMirrors returns all parameter mirrors in the context.
func (c *HookContext) AllMirrors() []Param {
	if c.rawAddrToMirror == nil {
		return nil
	}
	result := make([]Param, 0, len(c.rawAddrToMirror))
	for _, param := range c.rawAddrToMirror {
		result = append(result, param)
	}
	return result
}

// HasValue returns true if the parameter has a value from any source.
//
// Usage:
//
//	boa.CmdT[Params]{
//	    Use: "cmd",
//	    RunFuncCtx: func(ctx *boa.HookContext, params *Params, cmd *cobra.Command, args []string) {
//	        if ctx.HasValue(&params.Port) {
//	            fmt.Println("Port has a value:", params.Port)
//	        }
//	    },
//	}
func (c *HookContext) HasValue(fieldPtr any) bool {
	param := c.GetParam(fieldPtr)
	if param == nil {
		slog.Error("HookContext.HasValue: could not find param for field pointer", "fieldPtr", fieldPtr)
		return false
	}
	return HasValue(param)
}

// CmdList converts a list of CmdIfc to a slice of cobra.Command.
func CmdList(cmds ...CmdIfc) []*cobra.Command {
	var cobraCmds []*cobra.Command
	for _, cmd := range cmds {
		cobraCmds = append(cobraCmds, cmd.ToCobra())
	}
	return cobraCmds
}

// SubCmds converts a list of CmdIfc to a slice of cobra.Command.
func SubCmds(cmds ...CmdIfc) []*cobra.Command {
	return CmdList(cmds...)
}

// LoadConfigFile reads a config file and unmarshals it into the target struct.
// If filePath is empty, it's a no-op (returns nil).
// CLI and env var values still take precedence when used in PreValidateFunc.
// If unmarshalFunc is nil, defaults to json.Unmarshal.
func LoadConfigFile[T any](filePath string, target *T, unmarshalFunc func([]byte, any) error) error {
	_, err := loadConfigFileInto(filePath, target, unmarshalFunc)
	return err
}

// configFormats maps file extensions to unmarshal functions.
// JSON is registered by default. Users can register additional formats
// (e.g., YAML, TOML) via RegisterConfigFormat.
var configFormats = map[string]func([]byte, any) error{
	".json": json.Unmarshal,
}

// RegisterConfigFormat registers an unmarshal function for a config file extension.
// The extension should include the dot (e.g., ".yaml", ".toml").
// Example:
//
//	boa.RegisterConfigFormat(".yaml", yaml.Unmarshal)
//	boa.RegisterConfigFormat(".toml", toml.Unmarshal)
func RegisterConfigFormat(ext string, unmarshalFunc func([]byte, any) error) {
	configFormats[ext] = unmarshalFunc
}

// ConfigFormatExtensions returns the file extensions that have registered config format handlers.
// Always includes ".json" (registered by default). Additional formats are added via RegisterConfigFormat.
func ConfigFormatExtensions() []string {
	exts := make([]string, 0, len(configFormats))
	for ext := range configFormats {
		exts = append(exts, ext)
	}
	return exts
}

// loadConfigFileInto is the non-generic implementation used internally.
// Resolution order for unmarshal function:
//  1. Explicit unmarshalFunc parameter (from Cmd.ConfigUnmarshal)
//  2. Registered format based on file extension
//  3. json.Unmarshal (default fallback)
func loadConfigFileInto(filePath string, target any, unmarshalFunc func([]byte, any) error) ([]byte, error) {
	if filePath == "" {
		return nil, nil
	}
	fileContents, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}
	if unmarshalFunc == nil {
		// Look up by file extension
		ext := filepath.Ext(filePath)
		if fn, ok := configFormats[ext]; ok {
			unmarshalFunc = fn
		} else {
			unmarshalFunc = json.Unmarshal // ultimate fallback
		}
	}
	if err := unmarshalFunc(fileContents, target); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file %s: %w", filePath, err)
	}
	return fileContents, nil
}

// UnMarshalFromFileParam reads a file path from a parameter and unmarshals its contents into a target struct.
//
// Deprecated: Use LoadConfigFile instead for a simpler API.
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
		return LoadConfigFile(*valuePtrStr, v, unmarshalFunc)
	}
}
