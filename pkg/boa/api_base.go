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
	"sort"
	"strings"
	"sync"

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
	//
	// For richer support (including key-presence probing that lets boa detect
	// zero-valued or default-matching writes to optional struct-pointer groups),
	// prefer the ConfigFormat field, which accepts a full ConfigFormat value.
	// When both are set, ConfigFormat takes precedence.
	ConfigUnmarshal func([]byte, any) error
	// ConfigFormat specifies a per-command config file format — both the
	// unmarshaler and an optional key-tree probe used for set-by-config detection.
	// When set, this takes precedence over ConfigUnmarshal and over any format
	// registered via RegisterConfigFormat for the file extension.
	ConfigFormat ConfigFormat
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
//
// Internally, mirrors are stored keyed by their declared-index path from the root
// parameters struct. This keeps the identity of a mirror stable even when pointer
// substructs are reassigned, and makes subtree operations efficient string-prefix
// queries rather than address walks. The address → path cache exists solely to
// support the ergonomic GetParam(&params.Field) API.
type HookContext struct {
	ctx *processingContext
}

// newHookContext wraps a processingContext for hook exposure.
// Returns a usable (even if empty) HookContext — it never returns nil.
func newHookContext(pctx *processingContext) *HookContext {
	return &HookContext{ctx: pctx}
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
//
// GetParam returns nil and logs a descriptive slog.Error if the field pointer
// does not belong to the parameters struct associated with this HookContext —
// for example, if the caller passes a pointer to a field in an unrelated
// struct, in a different command's params, or in a substruct that was never
// registered (e.g., tagged boa:"ignore"). Callers using the idiomatic
// ctx.GetParam(&p.X).SetY(...) pattern will still see a nil-dereference crash
// if they chain against a nil return, but the preceding slog.Error log line
// will carry the descriptive cause so the real bug is visible in the output.
// Callers that want "probe without crashing" semantics can explicitly check
// for nil.
func (c *HookContext) GetParam(fieldPtr any) Param {
	if param, ok := fieldPtr.(Param); ok {
		return param
	}
	if c == nil || c.ctx == nil || c.ctx.mirrorByPath == nil {
		slog.Error("boa.HookContext.GetParam: called on a nil or uninitialized HookContext (no parameters are registered)")
		return nil
	}
	if fieldPtr == nil {
		slog.Error("boa.HookContext.GetParam: fieldPtr is nil")
		return nil
	}
	rv := reflect.ValueOf(fieldPtr)
	if rv.Kind() != reflect.Ptr {
		slog.Error("boa.HookContext.GetParam: fieldPtr must be a pointer to a struct field",
			"got_type", fmt.Sprintf("%T", fieldPtr))
		return nil
	}
	if rv.IsNil() {
		slog.Error("boa.HookContext.GetParam: fieldPtr is a typed nil",
			"got_type", fmt.Sprintf("%T", fieldPtr))
		return nil
	}

	addr := rv.UnsafePointer()
	// Rebuild the address cache if it was invalidated (e.g., after a subtree removal).
	if c.ctx.addrToPath == nil {
		c.ctx.rebuildAddrToPath()
	}
	if path, ok := c.ctx.addrToPath[addr]; ok {
		return c.ctx.mirrorByPath[path]
	}
	// Fallback: walk the root to discover a matching leaf address. This handles
	// the case where the user reassigned a substruct after the cache was built.
	// On a hit, repair the cache opportunistically so the next lookup for this
	// address is O(1) again.
	if c.ctx.rootStructPtr != nil {
		if path, ok := c.ctx.findPathByAddr(addr); ok {
			if c.ctx.addrToPath != nil {
				c.ctx.addrToPath[addr] = path
			}
			return c.ctx.mirrorByPath[path]
		}
	}
	slog.Error(
		"boa.HookContext.GetParam: the field pointer does not belong to the parameters struct associated with this HookContext. "+
			"Likely causes: "+
			"(1) you passed a pointer to a field in an unrelated struct; "+
			"(2) you passed a field from a different command's params in the same command tree (each command has its own mirror set); "+
			"(3) you passed a field from a substruct that was not registered (e.g., one tagged boa:\"ignore\" or of an unsupported type); "+
			"(4) you passed a field from a different instance of the same params type than the one registered with this command.",
		"got_type", fmt.Sprintf("%T", fieldPtr),
		"address", fmt.Sprintf("%p", addr),
	)
	return nil
}

// AllMirrors returns all parameter mirrors in the context in declaration/insertion order.
func (c *HookContext) AllMirrors() []Param {
	if c == nil || c.ctx == nil || c.ctx.mirrorByPath == nil {
		return nil
	}
	result := make([]Param, 0, len(c.ctx.pathOrder))
	for _, p := range c.ctx.pathOrder {
		if m, ok := c.ctx.mirrorByPath[p]; ok {
			result = append(result, m)
		}
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

// DumpBytes serializes the parameters associated with this HookContext to
// config bytes, emitting only fields where HasValue reports true. That is:
// any field set via CLI, environment, config file, or a struct-tag / hook
// default is written out; fields that were never set are omitted entirely.
//
// This is the right helper for persisting resolved config between runs:
// because defaults count as "set", the dump pins the current default values
// into the file, so the next run sees the same config even if a future
// release changes the app's built-in defaults. Truly-unset fields stay
// omitted, so the output doesn't fill up with Go zero values (0, "", false,
// nil) for options the user never touched.
//
// ext selects the registered format (e.g. ".yaml", ".toml"). A leading dot
// is optional. Pass "" for pretty-printed JSON. If marshalFunc is non-nil
// it takes precedence over ext. Same format-resolution rules and error
// behaviour as DumpConfigBytes.
//
// Key names are taken from the output format's struct tag when present:
//
//   - .json output honours `json:"name"` tags
//   - .yaml / .yml output honours `yaml:"name"` tags
//   - .toml output honours `toml:"name"` tags
//
// Fields with a `-` tag for the output format are skipped entirely. When no
// tag is set, the Go struct field name is used verbatim — matching what
// standard marshalers do when you hand them the original struct. This makes
// source-aware dumps round-trip cleanly through LoadConfig* for structs
// that mix format-specific tags with untagged fields.
//
// The configfile param itself (the field tagged configfile:"true") is
// omitted from the output — a dumped file that references its own path as
// a field is self-referential and surprising on the next load.
func (c *HookContext) DumpBytes(ext string, marshalFunc func(v any) ([]byte, error)) ([]byte, error) {
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	tree, err := c.buildSetValueTree(structTagForExt(ext))
	if err != nil {
		return nil, err
	}
	marshal, err := resolveConfigMarshalByExt(ext, marshalFunc)
	if err != nil {
		return nil, err
	}
	return marshal(tree)
}

// structTagForExt maps a config file extension to the struct tag that
// boa should consult when mapping config-file keys to Go struct fields,
// in both directions — the dump path (source-aware map key generation)
// and the load path (set-by-config detection for optional struct-pointer
// groups and for HasValue). Keeping a single source of truth across both
// paths means renames like `json:"host"` → `"host"` round-trip from a
// config file back out to a dump of the same format without drift.
//
// For unrecognised extensions we default to the extension minus its
// leading dot — so registering a custom `.kvp` format automatically
// uses the `kvp` struct tag without any extra plumbing, and registering
// `.mycustom` uses `mycustom`. Users who want a different convention
// today need a per-command `Cmd.ConfigFormat` override. We keep the
// explicit `.yml` entry because yaml parsers consult the `yaml` tag,
// not a hypothetical `yml` one. Empty ext falls back to `json` so
// programmatic Dump/Load calls without a file path stay JSON-shaped.
func structTagForExt(ext string) string {
	switch ext {
	case "", ".json":
		return "json"
	case ".yml":
		return "yaml"
	}
	return strings.TrimPrefix(ext, ".")
}

// resolveDumpFieldName picks the key name for a struct field in a
// source-aware dump. It honours the format-appropriate struct tag so
// `Host string `json:"hostname"`` dumps as `"hostname"` (and round-trips
// through LoadConfigFile). A "-" tag value means "skip this field";
// callers should drop the field entirely when this returns ("", true).
// An empty tagName means "no tag lookup"; fall back to the field name.
func resolveDumpFieldName(sf reflect.StructField, tagName string) (name string, skip bool) {
	if tagName == "" {
		return sf.Name, false
	}
	tag := sf.Tag.Get(tagName)
	if tag == "" {
		return sf.Name, false
	}
	// Tag value may be `name,opt1,opt2` (e.g. json:"name,omitempty").
	// We only care about the name part.
	if comma := strings.IndexByte(tag, ','); comma >= 0 {
		tag = tag[:comma]
	}
	if tag == "-" {
		return "", true
	}
	if tag == "" {
		return sf.Name, false
	}
	return tag, false
}

// DumpFile is the file-writing counterpart to DumpBytes. The marshaler is
// resolved from filePath's extension; the file is written with mode 0644
// and overwrites any existing file.
func (c *HookContext) DumpFile(filePath string, marshalFunc func(v any) ([]byte, error)) error {
	if filePath == "" {
		return NewUserInputError(fmt.Errorf("HookContext.DumpFile: filePath must not be empty"))
	}
	data, err := c.DumpBytes(filepath.Ext(filePath), marshalFunc)
	if err != nil {
		return fmt.Errorf("failed to marshal config for %s: %w", filePath, err)
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", filePath, err)
	}
	return nil
}

// buildSetValueTree walks the root parameters struct and returns a nested
// map[string]any containing only fields where the corresponding param
// mirror's HasValue() is true. Nested structs appear as nested maps and are
// omitted entirely when they have no set descendants.
//
// tagName picks the struct tag used for map key names (see
// structTagForExt). An empty tagName falls back to the Go field name.
func (c *HookContext) buildSetValueTree(tagName string) (map[string]any, error) {
	if c == nil || c.ctx == nil {
		return nil, fmt.Errorf("boa: HookContext: uninitialized (no parameters registered)")
	}
	if c.ctx.rootStructPtr == nil {
		return nil, fmt.Errorf("boa: HookContext: no root parameters struct")
	}
	rv := reflect.ValueOf(c.ctx.rootStructPtr)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, fmt.Errorf("boa: HookContext: root parameters struct is nil")
		}
		rv = rv.Elem()
	}
	return buildSetValueMapNode(rv, c.ctx, nil, tagName), nil
}

// shouldEmitInDump decides whether a leaf parameter should appear in a
// source-aware config dump. The rule is HasValue with one deliberate
// exception for bools:
//
// ParamEnricherBool auto-installs a `false` default for every bool
// parameter that does not specify one, so a plain `Silent bool` field
// would otherwise always satisfy HasValue even when the user never touched
// it. For dump purposes we treat a bool that is still sitting at the
// default-false value as unset unless CLI / env / config / inject says
// otherwise — otherwise every dump fills up with `"Silent": false`
// noise for flags the user never flipped.
//
// Trade-off: a user who explicitly writes `default:"false"` on a bool in a
// struct tag will see the same omission, since the enricher's default and
// a hand-written `false` default are indistinguishable. Users who want the
// explicit emission can run with --their-flag=false once; the source
// tracking then records wasSetOnCli and the dump emits it.
func shouldEmitInDump(f Param, v reflect.Value) bool {
	if f.wasSetOnCli() || f.wasSetByEnv() || f.wasSetByInject() {
		return true
	}
	if pm, ok := f.(*paramMeta); ok && pm.setByConfig {
		return true
	}
	if !f.hasDefaultValue() {
		return false
	}
	if f.GetKind() == reflect.Bool {
		if b, ok := v.Interface().(bool); ok && !b {
			return false
		}
	}
	return true
}

// buildSetValueMapNode is the recursive walker used by buildSetValueTree.
// It returns nil when the entire subtree has no set fields so the caller
// can omit the parent key. tagName is the struct tag to consult for field
// key names (see structTagForExt); an empty tagName means "use the Go
// field name".
func buildSetValueMapNode(v reflect.Value, ctx *processingContext, pathIdx []int, tagName string) map[string]any {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()
	out := map[string]any{}
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}
		fv := v.Field(i)
		childIdx := append(append([]int{}, pathIdx...), i)
		pathKey := joinPath(childIdx)

		if mirror, ok := ctx.mirrorByPath[pathKey]; ok {
			// Leaf parameter with a registered mirror.
			if mirror.IsConfigFile() {
				// Never write the configfile path back into the dumped file —
				// self-reference on the next load is a surprise.
				continue
			}
			if shouldEmitInDump(mirror, fv) {
				name, skip := resolveDumpFieldName(sf, tagName)
				if skip {
					continue
				}
				out[name] = fv.Interface()
			}
			continue
		}
		// No direct mirror at this path → might be a nested anonymous or
		// pointer struct whose children carry the mirrors. Recurse.
		// Anonymous (embedded) fields flatten their children into the
		// parent map, matching boa's flag-generation semantics (an embedded
		// DBConfig produces --host, not --db-config-host) and encoding/json's
		// default struct-embedding flattening.
		switch fv.Kind() {
		case reflect.Struct:
			if sub := buildSetValueMapNode(fv, ctx, childIdx, tagName); len(sub) > 0 {
				if sf.Anonymous {
					for k, v := range sub {
						out[k] = v
					}
				} else {
					name, skip := resolveDumpFieldName(sf, tagName)
					if skip {
						continue
					}
					out[name] = sub
				}
			}
		case reflect.Ptr:
			if !fv.IsNil() && fv.Elem().Kind() == reflect.Struct {
				if sub := buildSetValueMapNode(fv, ctx, childIdx, tagName); len(sub) > 0 {
					if sf.Anonymous {
						for k, v := range sub {
							out[k] = v
						}
					} else {
						name, skip := resolveDumpFieldName(sf, tagName)
						if skip {
							continue
						}
						out[name] = sub
					}
				}
			}
		}
		// Other kinds with no mirror → boa:"ignore" or unsupported; skip.
	}
	if len(out) == 0 {
		return nil
	}
	return out
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
//
// If unmarshalFunc is non-nil it is used directly. If unmarshalFunc is nil the
// resolution order is the same as for configfile:"true" fields: the registered
// format matching the file's extension first (RegisterConfigFormat /
// RegisterConfigFormatFull), and json.Unmarshal as the final fallback when no
// registration matches.
func LoadConfigFile[T any](filePath string, target *T, unmarshalFunc func([]byte, any) error) error {
	override := ConfigFormat{}
	if unmarshalFunc != nil {
		override.Unmarshal = unmarshalFunc
	}
	_, _, err := loadConfigFileInto(filePath, target, override)
	return err
}

// LoadConfigBytes unmarshals raw config bytes into the target struct, using
// the same format-resolution rules as LoadConfigFile. This is the in-memory
// counterpart for cases where the bytes do not come from a local file —
// typical callers are //go:embed assets, stdin pipes, HTTP response bodies,
// and test fixtures.
//
// ext selects the registered format (e.g. ".yaml", ".toml"). A leading dot is
// optional — "yaml" and ".yaml" both work. Pass "" to use the default JSON
// parser. If unmarshalFunc is non-nil it takes precedence over ext.
//
// Passing an empty data slice is a no-op (returns nil) so callers can hand in
// the result of an optional read without a preceding len check.
//
// CLI and env var values still take precedence when this is used inside
// PreValidateFunc, exactly as with LoadConfigFile.
func LoadConfigBytes[T any](data []byte, ext string, target *T, unmarshalFunc func([]byte, any) error) error {
	if len(data) == 0 {
		return nil
	}
	override := ConfigFormat{}
	if unmarshalFunc != nil {
		override.Unmarshal = unmarshalFunc
	}
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	_, err := loadConfigBytesInto(data, ext, target, override)
	return err
}

// DumpConfigBytes serializes v to config bytes using the marshaler resolved
// from ext. This is the *naive* dump — every exported field on v is emitted,
// including Go zero values. Useful for "generate an example config with
// every option" or round-trip tests.
//
// For "persist the resolved config between runs, but don't emit fields the
// user never set" semantics, use HookContext.DumpBytes / HookContext.DumpFile
// instead — those honour HasValue so unset fields are omitted entirely and
// defaults are still pinned in place across app upgrades.
//
// ext selects the registered format (e.g. ".yaml", ".toml"). A leading dot
// is optional. Pass "" to use the default pretty-printed JSON marshaler.
// If marshalFunc is non-nil it takes precedence over ext.
//
// Unlike LoadConfigBytes, there is no silent cross-format fallback: if the
// requested extension is registered but has no Marshal, DumpConfigBytes
// returns a clear error pointing at RegisterConfigMarshaler. Writing JSON
// bytes to a file named `.yaml` would be a nasty surprise, so the API
// refuses rather than guessing.
//
// The JSON default is indented with two spaces and ends with a trailing
// newline — the shape you'd expect when the bytes are about to land on disk.
func DumpConfigBytes[T any](v *T, ext string, marshalFunc func(v any) ([]byte, error)) ([]byte, error) {
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	marshal, err := resolveConfigMarshalByExt(ext, marshalFunc)
	if err != nil {
		return nil, err
	}
	return marshal(v)
}

// DumpConfigFile serializes v and writes it to filePath. The marshaler is
// resolved from the file extension using the same rules as DumpConfigBytes.
// The file is written with mode 0644; if a file already exists at filePath
// it is overwritten.
//
// Pass a non-nil marshalFunc to override the extension-registered marshaler
// (mirrors LoadConfigFile's unmarshalFunc parameter).
//
// If filePath is empty, returns a clear user-input error — unlike
// LoadConfigFile, there is no "empty is a no-op" shortcut, because silently
// dropping a dump request is more surprising than a missing load.
func DumpConfigFile[T any](filePath string, v *T, marshalFunc func(v any) ([]byte, error)) error {
	if filePath == "" {
		return NewUserInputError(fmt.Errorf("DumpConfigFile: filePath must not be empty"))
	}
	data, err := DumpConfigBytes(v, filepath.Ext(filePath), marshalFunc)
	if err != nil {
		return fmt.Errorf("failed to marshal config for %s: %w", filePath, err)
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", filePath, err)
	}
	return nil
}

// ConfigFormat describes how to parse and introspect a config file format.
//
// boa ships with a built-in handler for JSON only. To support other formats
// (YAML, TOML, HCL, …) without forcing those dependencies on every boa user,
// bring your own parser and register it via RegisterConfigFormatFull — or set
// Cmd.ConfigFormat on a single command.
type ConfigFormat struct {
	// Unmarshal parses raw bytes into the target struct. Required for
	// LoadConfigFile / LoadConfigBytes. A ConfigFormat with a nil Unmarshal
	// is "dump-only" — it can still be used by DumpConfigFile /
	// DumpConfigBytes if Marshal is set, but reading will fall through to
	// the JSON fallback.
	Unmarshal func(data []byte, target any) error

	// Marshal serializes a value back out to raw bytes. Optional — only
	// required for DumpConfigFile / DumpConfigBytes. Callers that ask to
	// dump a format that has no registered Marshal get a clear error
	// instead of a silent cross-format fallback, because (for example)
	// writing JSON bytes to a file named `.yaml` would be a nasty surprise.
	Marshal func(v any) ([]byte, error)

	// KeyTree returns a nested map[string]any representing the top-level and
	// nested key structure of the raw bytes. boa uses this to detect which
	// struct fields — and which optional struct-pointer parameter groups —
	// were explicitly mentioned in the config file, even when the written
	// value equals Go's zero value or the parameter's default.
	//
	// Only key presence matters. Nested objects should appear as map[string]any
	// so boa can recurse; scalars and arrays may be any non-nil placeholder.
	//
	// Optional. If nil, boa falls back to snapshot comparison, which detects
	// changed values but not zero-value or same-as-default writes to optional
	// struct-pointer parameter groups.
	KeyTree func(data []byte) (map[string]any, error)
}

// configFormats maps file extensions to their registered ConfigFormat.
// JSON is registered by default with both Unmarshal and KeyTree. Users can
// register additional formats (e.g., YAML, TOML) via RegisterConfigFormat
// or RegisterConfigFormatFull.
//
// Access to configFormats MUST go through configFormatsMu. Registration
// (write) takes the exclusive lock; resolution / extension enumeration take
// the read lock. The read path is hot — every config file load hits it —
// but it's a pure map lookup so RWMutex contention is negligible in practice.
var (
	configFormatsMu sync.RWMutex
	configFormats   = map[string]ConfigFormat{
		".json": {
			Unmarshal: json.Unmarshal,
			Marshal:   jsonMarshalPretty,
			KeyTree:   jsonKeyTree,
		},
	}
)

// jsonKeyTree is the built-in KeyTree implementation for JSON.
func jsonKeyTree(data []byte) (map[string]any, error) {
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// jsonMarshalPretty is the built-in Marshal used for DumpConfig*.
// It produces human-readable, 2-space-indented JSON with a trailing
// newline — the shape you'd expect when writing a config file to disk.
// json.Marshal's compact output round-trips fine but is unfriendly for a
// human who opens the file after a dump.
func jsonMarshalPretty(v any) ([]byte, error) {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// UniversalConfigFormat builds a ConfigFormat from a single "universal"
// unmarshal function — one that can decode bytes into any target, including
// map[string]any. The returned ConfigFormat has:
//
//   - Unmarshal — the function you passed in.
//   - KeyTree — a synthesized probe that calls the same function against a
//     map[string]any target so boa can inspect the literal key structure.
//
// That covers every mainstream Go config parser (encoding/json,
// gopkg.in/yaml.v3, github.com/BurntSushi/toml, github.com/hashicorp/hcl/v2,
// …), all of which decode into interface{} targets uniformly.
//
// RegisterConfigFormat uses this helper internally, so for registry-based
// dispatch you normally just call RegisterConfigFormat directly. Call
// UniversalConfigFormat yourself when you want to set a format inline on a
// single command via Cmd.ConfigFormat:
//
//	boa.CmdT[Params]{
//	    ConfigFormat: boa.UniversalConfigFormat(yaml.Unmarshal),
//	    ...
//	}
//
// Use the explicit boa.ConfigFormat{Unmarshal, KeyTree} struct literal (and
// RegisterConfigFormatFull) only when the parser cannot decode into
// map[string]any — e.g., a handwritten custom format that only populates
// specific struct types. In that case supply a KeyTree function that
// produces the nested key structure yourself.
//
// Passing nil panics — a missing unmarshal function is a programming error
// and is surfaced eagerly so you don't silently fall through to the JSON
// handler at parse time.
func UniversalConfigFormat(unmarshalFunc func([]byte, any) error) ConfigFormat {
	if unmarshalFunc == nil {
		panic(fmt.Errorf("boa: UniversalConfigFormat: unmarshalFunc must be non-nil"))
	}
	return ConfigFormat{
		Unmarshal: unmarshalFunc,
		KeyTree: func(data []byte) (map[string]any, error) {
			var out map[string]any
			if err := unmarshalFunc(data, &out); err != nil {
				return nil, err
			}
			return out, nil
		},
	}
}

// RegisterConfigFormat registers an unmarshal function for a config file
// extension. The extension should include the dot (e.g., ".yaml", ".toml").
//
// A single call gives you both parsing (the file extension now dispatches to
// unmarshalFunc) and full key-presence detection (including zero-valued and
// same-as-default writes to optional struct-pointer parameter groups).
// Internally, RegisterConfigFormat wraps unmarshalFunc in a
// UniversalConfigFormat — the KeyTree is synthesized by calling the same
// parser against a map[string]any target, which every mainstream Go config
// library supports.
//
// Example:
//
//	boa.RegisterConfigFormat(".yaml", yaml.Unmarshal)
//	boa.RegisterConfigFormat(".toml", toml.Unmarshal)
//	boa.RegisterConfigFormat(".hcl", hcl.Decode)
//
// Registration is goroutine-safe (the registry is guarded by a
// sync.RWMutex), but the common pattern is still to call from init() or
// from main-goroutine startup before any commands run. Passing a nil
// unmarshalFunc panics.
//
// Use RegisterConfigFormatFull instead only when your parser cannot decode
// into map[string]any (e.g., a custom format that only populates specific
// struct types). In that case you must supply a hand-written KeyTree that
// produces the nested key structure yourself.
func RegisterConfigFormat(ext string, unmarshalFunc func([]byte, any) error) {
	RegisterConfigFormatFull(ext, UniversalConfigFormat(unmarshalFunc))
}

// RegisterConfigFormatFull registers a complete ConfigFormat for a config file
// extension. Use this when your parser cannot decode into map[string]any and
// you need to supply a hand-written KeyTree for set-by-config detection.
//
// The extension should include the dot (e.g., ".mycustom").
//
// Registration is goroutine-safe — the registry is guarded by a sync.RWMutex,
// so you can register formats from any goroutine. In practice, calling from
// init() or main-goroutine startup (before any commands run) is still the
// clearest model.
//
// Passing a ConfigFormat with a nil Unmarshal panics — a missing parser is a
// programming error and is surfaced eagerly so you don't silently fall
// through to the JSON handler at parse time.
//
// Example (using gopkg.in/yaml.v3):
//
//	boa.RegisterConfigFormatFull(".yaml", boa.ConfigFormat{
//	    Unmarshal: yaml.Unmarshal,
//	    KeyTree: func(data []byte) (map[string]any, error) {
//	        var out map[string]any
//	        if err := yaml.Unmarshal(data, &out); err != nil {
//	            return nil, err
//	        }
//	        return out, nil
//	    },
//	})
func RegisterConfigFormatFull(ext string, format ConfigFormat) {
	normalized := normalizeConfigExt(ext)
	if format.Unmarshal == nil {
		panic(fmt.Errorf("boa: RegisterConfigFormatFull(%q): ConfigFormat.Unmarshal must be non-nil", ext))
	}
	configFormatsMu.Lock()
	defer configFormatsMu.Unlock()
	configFormats[normalized] = format
}

// RegisterConfigMarshaler attaches a Marshal function to a registered format,
// enabling DumpConfigFile / DumpConfigBytes for that extension. The common
// pattern is one call per format, paired with an earlier RegisterConfigFormat:
//
//	boa.RegisterConfigFormat(".yaml", yaml.Unmarshal)
//	boa.RegisterConfigMarshaler(".yaml", yaml.Marshal)
//
// If the extension hasn't been registered yet, a placeholder entry is created
// with only Marshal set. Reading such a format then falls through to the JSON
// fallback, which is almost never what you want — always register Unmarshal
// too unless the format is genuinely dump-only.
//
// Registration is goroutine-safe. Passing nil panics.
func RegisterConfigMarshaler(ext string, marshalFunc func(v any) ([]byte, error)) {
	normalized := normalizeConfigExt(ext)
	if marshalFunc == nil {
		panic(fmt.Errorf("boa: RegisterConfigMarshaler(%q): marshalFunc must be non-nil", ext))
	}
	configFormatsMu.Lock()
	defer configFormatsMu.Unlock()
	cf := configFormats[normalized]
	cf.Marshal = marshalFunc
	configFormats[normalized] = cf
}

// normalizeConfigExt canonicalises an extension registration key into the
// dot-prefixed form that filepath.Ext produces at lookup time. Accepts either
// "yaml" or ".yaml" — both become ".yaml" — so users don't have to remember
// which form boa expects. An empty string panics because it's unambiguously
// a programmer mistake.
func normalizeConfigExt(ext string) string {
	if ext == "" {
		panic(fmt.Errorf("boa: config format extension must not be empty"))
	}
	if !strings.HasPrefix(ext, ".") {
		return "." + ext
	}
	return ext
}

// ConfigFormatExtensions returns the file extensions that have registered
// config format handlers, sorted alphabetically for deterministic iteration.
// Always includes ".json" (registered by default). Additional formats are
// added via RegisterConfigFormat or RegisterConfigFormatFull.
//
// The sort matters for callers like boaviper.FindConfig that probe the same
// search path with every registered extension: without a stable order, which
// file wins is nondeterministic when two extensions' files both exist.
func ConfigFormatExtensions() []string {
	configFormatsMu.RLock()
	exts := make([]string, 0, len(configFormats))
	for ext := range configFormats {
		exts = append(exts, ext)
	}
	configFormatsMu.RUnlock()
	sort.Strings(exts)
	return exts
}

// loadConfigFileInto is the non-generic implementation used internally.
// Resolution order for the effective ConfigFormat:
//  1. override (from Cmd.ConfigFormat / Cmd.ConfigUnmarshal) when its Unmarshal is non-nil
//  2. Registered format for the file extension
//  3. JSON fallback (unmarshal + key-tree)
//
// Returns the raw bytes and the effective ConfigFormat so callers can reuse
// its KeyTree for key-presence detection.
func loadConfigFileInto(filePath string, target any, override ConfigFormat) ([]byte, ConfigFormat, error) {
	if filePath == "" {
		return nil, ConfigFormat{}, nil
	}
	fileContents, err := os.ReadFile(filePath)
	if err != nil {
		return nil, ConfigFormat{}, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}
	effective, err := loadConfigBytesInto(fileContents, filepath.Ext(filePath), target, override)
	if err != nil {
		return nil, effective, fmt.Errorf("failed to unmarshal config file %s: %w", filePath, err)
	}
	return fileContents, effective, nil
}

// loadConfigBytesInto is the shared bytes-level core used by both file and
// in-memory loaders. It resolves the effective ConfigFormat the same way as
// loadConfigFileInto and runs the unmarshaler against the supplied bytes.
func loadConfigBytesInto(data []byte, ext string, target any, override ConfigFormat) (ConfigFormat, error) {
	effective := resolveConfigFormatByExt(ext, override)
	if err := effective.Unmarshal(data, target); err != nil {
		return effective, err
	}
	return effective, nil
}

// resolveConfigFormatByExt picks the ConfigFormat to use for a given extension,
// honouring the precedence override → extension-registered → JSON fallback.
// The returned ConfigFormat always has a non-nil Unmarshal. ext should be
// dot-prefixed (filepath.Ext form); an empty string means "no hint" and falls
// through to the JSON default.
func resolveConfigFormatByExt(ext string, override ConfigFormat) ConfigFormat {
	if override.Unmarshal != nil {
		return override
	}
	if ext != "" {
		configFormatsMu.RLock()
		cf, ok := configFormats[ext]
		configFormatsMu.RUnlock()
		if ok && cf.Unmarshal != nil {
			return cf
		}
	}
	return ConfigFormat{Unmarshal: json.Unmarshal, KeyTree: jsonKeyTree}
}

// resolveConfigMarshalByExt picks the marshaler for the Dump* helpers.
// Precedence: explicit override → registered format's Marshal → JSON pretty
// fallback when no ext was supplied. Unlike the unmarshal path, a
// *registered* format with a nil Marshal is an error, not a fallback —
// emitting JSON bytes under a `.yaml` extension would silently corrupt the
// file's type on disk.
func resolveConfigMarshalByExt(ext string, override func(v any) ([]byte, error)) (func(v any) ([]byte, error), error) {
	if override != nil {
		return override, nil
	}
	if ext == "" {
		return jsonMarshalPretty, nil
	}
	configFormatsMu.RLock()
	cf, ok := configFormats[ext]
	configFormatsMu.RUnlock()
	if !ok {
		// Unknown extension → same permissive default as the load path:
		// fall through to JSON pretty. Callers that want a hard error for
		// unknown extensions can check ConfigFormatExtensions() first.
		return jsonMarshalPretty, nil
	}
	if cf.Marshal == nil {
		return nil, fmt.Errorf("boa: no marshaler registered for config extension %q — call RegisterConfigMarshaler(%q, ...) or pass a marshalFunc", ext, ext)
	}
	return cf.Marshal, nil
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
