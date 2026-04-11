package boa

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unsafe"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// osExit is the exit function used by Run(). It defaults to os.Exit but can be
// replaced in tests to verify exit behavior without terminating the test process.
var osExit = os.Exit

// UserInputError wraps errors caused by invalid user input (missing required
// params, invalid flag values, etc.). These errors should result in a clean
// exit with error message rather than a panic with stack trace.
type UserInputError struct {
	Err error
}

func (e *UserInputError) Error() string {
	return e.Err.Error()
}

func (e *UserInputError) Unwrap() error {
	return e.Err
}

// NewUserInputError wraps an error as a UserInputError.
// Use this in hooks like PreValidateFunc when returning user input validation errors
// to ensure they result in a clean exit (no stack trace) when using Run().
func NewUserInputError(err error) error {
	if err == nil {
		return nil
	}
	return &UserInputError{Err: err}
}

// NewUserInputErrorf creates a new UserInputError with a formatted message.
// Use this in hooks like PreValidateFunc when returning user input validation errors
// to ensure they result in a clean exit (no stack trace) when using Run().
func NewUserInputErrorf(format string, args ...any) error {
	return &UserInputError{Err: fmt.Errorf(format, args...)}
}

// internal aliases for backwards compatibility within the package
var newUserInputError = NewUserInputError
var newUserInputErrorf = NewUserInputErrorf

// IsUserInputError checks if an error is (or wraps) a UserInputError
// or is one of pflag's known validation error types
func IsUserInputError(err error) bool {
	if err == nil {
		return false
	}

	// Check for our custom UserInputError
	var uie *UserInputError
	if errors.As(err, &uie) {
		return true
	}

	// Check for pflag validation error types
	var invalidSyntax *pflag.InvalidSyntaxError
	var invalidValue *pflag.InvalidValueError
	var notExist *pflag.NotExistError
	var valueRequired *pflag.ValueRequiredError

	if errors.As(err, &invalidSyntax) ||
		errors.As(err, &invalidValue) ||
		errors.As(err, &notExist) ||
		errors.As(err, &valueRequired) {
		return true
	}

	return false
}

// runFuncError marks errors originating from RunFuncE/RunFuncCtxE so that
// runImpl can distinguish them from cobra errors (unknown command, bad flags).
// Cobra errors should be treated as user input; RunFuncE errors are programming errors.
type runFuncError struct{ Err error }

func (e *runFuncError) Error() string { return e.Err.Error() }
func (e *runFuncError) Unwrap() error { return e.Err }

// Execute runs a cobra command with boa's error handling convention:
// usage is printed first, then the error message, both to stderr.
// Use this instead of cmd.Execute() when working with commands from ToCobra().
func Execute(cmd *cobra.Command) error {
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	executedCmd, err := cmd.ExecuteC()
	if err != nil {
		fmt.Fprintln(os.Stderr, executedCmd.UsageString())
		fmt.Fprintln(os.Stderr, "Error:", err.Error())
	}
	return err
}

// wrapArgsValidator wraps a cobra.PositionalArgs validator to return UserInputError
func wrapArgsValidator(validator cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := validator(cmd, args); err != nil {
			return newUserInputError(err)
		}
		return nil
	}
}

type Param interface {
	HasValue() bool
	GetShort() string
	GetName() string
	GetEnv() string
	GetKind() reflect.Kind
	GetType() reflect.Type
	SetDefault(any)
	SetEnv(string)
	SetShort(string)
	SetName(string)
	SetAlternatives([]string)
	defaultValuePtr() any
	getDescr() string
	IsRequired() bool
	valuePtrF() any
	parentCmd() *cobra.Command
	wasSetOnCli() bool
	wasSetByEnv() bool
	wasSetByInject() bool
	customValidatorOfPtr() func(any) error
	SetCustomValidator(func(any) error)
	hasDefaultValue() bool
	defaultValueStr() string
	setParentCmd(cmd *cobra.Command)
	setValuePtr(any)
	injectValuePtr(any)
	markSetFromEnv()
	isPositional() bool
	wasSetPositionally() bool
	markSetPositionally()
	setPositional(bool)
	setDescription(descr string)
	IsEnabled() bool
	GetAlternatives() []string
	GetAlternativesFunc() func(cmd *cobra.Command, args []string, toComplete string) []string
	SetAlternativesFunc(func(cmd *cobra.Command, args []string, toComplete string) []string)
	GetIsEnabledFn() func() bool
	SetIsEnabledFn(func() bool)
	SetRequiredFn(func() bool)
	GetRequiredFn() func() bool
	SetStrictAlts(bool)
	GetStrictAlts() bool

	// --- Exported parity for struct-tag-only features ---
	// These mirror struct tags so anything configurable by tag is also
	// configurable programmatically (e.g. for params built from third-party
	// structs you can't add tags to).

	// IsNoFlag reports whether CLI flag registration is suppressed for this
	// parameter (still reads env vars and config files). Mirrors `boa:"noflag"`.
	IsNoFlag() bool
	// SetNoFlag toggles CLI flag suppression. Must be called before cobra
	// flag binding (e.g. inside InitFunc / InitFuncCtx) to take effect.
	SetNoFlag(bool)

	// IsNoEnv reports whether env var reading is suppressed for this
	// parameter (CLI flags and config files still apply). Mirrors `boa:"noenv"`.
	IsNoEnv() bool
	// SetNoEnv toggles env var suppression.
	SetNoEnv(bool)

	// IsIgnored reports whether the parameter is fully ignored by boa
	// (no CLI flag, no env reading, no validation). Config files can still
	// populate the underlying field via the unmarshaler.
	IsIgnored() bool
	// SetIgnored marks the parameter as ignored. Must be called before
	// cobra flag binding and env parsing.
	SetIgnored(bool)

	// GetDescription / SetDescription expose the help/descr text.
	GetDescription() string
	SetDescription(string)

	// IsPositional / SetPositional mirror `positional:"true"`.
	IsPositional() bool
	SetPositional(bool)

	// GetMin / SetMin / GetMax / SetMax / GetPattern / SetPattern mirror the
	// validation tags. Pass nil to clear a min/max bound.
	GetMin() *float64
	SetMin(*float64)
	GetMax() *float64
	SetMax(*float64)
	GetPattern() string
	SetPattern(string)

	// SetRequired is a convenience that fixes the parameter as required or
	// optional regardless of the original tag. Equivalent to
	// SetRequiredFn(func() bool { return val }).
	SetRequired(bool)
}

// configFileEntry tracks a configfile:"true" field and the struct it should load into.
type configFileEntry struct {
	mirror     Param     // the string param holding the file path
	target     any       // pointer to the struct to unmarshal into
	targetPath fieldPath // path from root to the target struct (empty for root)
}

// fieldPath is the dot-separated sequence of struct-field declaration indices
// that locates a parameter from the root params struct. Example: "0.2.1" = root
// field 0, its field 2, its field 1. Empty string means the root itself.
//
// Paths are the authoritative key for mirror storage — they are stable under
// pointer-substruct reassignment (reflect.FieldByIndex walks through live
// pointers at lookup time), debuggable, and support O(1) subtree operations
// via string-prefix matching.
type fieldPath string

// joinPath serializes a []int index path to the canonical fieldPath string form.
func joinPath(p []int) fieldPath {
	if len(p) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, v := range p {
		if i > 0 {
			sb.WriteByte('.')
		}
		sb.WriteString(strconv.Itoa(v))
	}
	return fieldPath(sb.String())
}

// hasSubtreePrefix reports whether this path is prefix itself or a descendant of it.
func (p fieldPath) hasSubtreePrefix(prefix fieldPath) bool {
	if prefix == "" {
		return true
	}
	s := string(p)
	ps := string(prefix)
	return s == ps || strings.HasPrefix(s, ps+".")
}

// preallocatedPtrInfo tracks a struct pointer field that was nil and got preallocated
// so that traverse could discover and register its child fields as CLI flags.
// After all value sources are applied (CLI, env, config), the pointer is nil'd back
// if none of its fields were explicitly set.
type preallocatedPtrInfo struct {
	ptrField reflect.Value // the pointer field in the parent struct (e.g., *DBConfig)
	path     fieldPath     // path from root to this pointer field
}

type processingContext struct {
	context.Context
	// rootStructPtr is a pointer to the root parameters struct (e.g., *Params).
	// All field paths are expressed relative to this root.
	rootStructPtr any
	// mirrorByPath is the authoritative mirror store, keyed by field-index path.
	// Paths are stable under pointer-substruct reassignment and support efficient
	// subtree queries via string-prefix matching.
	mirrorByPath map[fieldPath]Param
	// pathOrder preserves traverse insertion order for operations that need
	// deterministic iteration (e.g., syncMirrors reads raw → mirror → raw in
	// the order fields were discovered).
	pathOrder []fieldPath
	// addrToPath is a non-authoritative reverse index built during traverse.
	// Its sole purpose is to support HookContext.GetParam(&params.Field),
	// where users pass a Go field pointer and expect a mirror back. When the
	// root parameters struct is reassigned mid-flight, this cache can be
	// invalidated and rebuilt — mirrorByPath remains correct regardless.
	addrToPath map[unsafe.Pointer]fieldPath
	// walkFallbackCount / cacheRebuildCount are instrumentation counters for
	// tests that need to assert whether a GetParam call hit the fast path or
	// fell through to the slower fallback walk / cache rebuild. Incremented
	// unconditionally — cheap, zero-cost for production code.
	walkFallbackCount int
	cacheRebuildCount int
	// ConfigFiles tracks all configfile:"true" fields and their target structs.
	// Ordered: substruct entries first, root entry last (so root overrides inner).
	ConfigFiles []configFileEntry
	// PreallocatedPtrs tracks struct pointer fields that were nil and got preallocated.
	// Ordered depth-first (innermost first) so cleanup processes leaves before parents.
	PreallocatedPtrs []preallocatedPtrInfo
	// ConfigPresentPtrs tracks preallocated struct pointers that were explicitly
	// mentioned in a config file (even if no child fields were set, e.g., "DB": {}).
	// These survive cleanup regardless of whether individual fields were set.
	ConfigPresentPtrs map[uintptr]bool
}

// preallocateStructPtrs walks the struct tree and allocates any nil struct pointer fields,
// tracking them in ctx.PreallocatedPtrs so they can be nil'd back after parsing if unused.
// This must be called before the first traverse so that traverse discovers the child fields.
// The list is built depth-first (innermost first) for correct cleanup ordering.
//
// path is the declared-index path from the root to the struct being walked, using
// reflect.StructField.Index numbering. Ignored fields retain their slot.
func preallocateStructPtrs(ctx *processingContext, structPtr any, path []int) {
	val := reflect.ValueOf(structPtr).Elem()
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if isBoaIgnored(field) {
			continue
		}
		childPath := append(append([]int(nil), path...), i)
		// Recurse into non-pointer structs (they're always present)
		if field.Type.Kind() == reflect.Struct && !isSupportedType(field.Type) {
			preallocateStructPtrs(ctx, val.Field(i).Addr().Interface(), childPath)
			continue
		}
		// Preallocate nil struct pointer fields
		if field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct && !isSupportedType(field.Type) {
			fieldVal := val.Field(i)
			if fieldVal.IsNil() {
				fieldVal.Set(reflect.New(field.Type.Elem()))
				// Recurse into the newly allocated struct (inner pointers first)
				preallocateStructPtrs(ctx, fieldVal.Interface(), childPath)
				// Append after recursion so innermost entries come first
				ctx.PreallocatedPtrs = append(ctx.PreallocatedPtrs, preallocatedPtrInfo{
					ptrField: fieldVal,
					path:     joinPath(childPath),
				})
			} else {
				// Already non-nil (user pre-initialized) — still recurse for nested nil ptrs
				preallocateStructPtrs(ctx, fieldVal.Interface(), childPath)
			}
			continue
		}
	}
}

// cleanupPreallocatedPtrs nils back any preallocated struct pointers whose fields
// were never explicitly set (via CLI, env, or config injection). Processes innermost
// first so that nested struct pointers are cleaned before their parents.
func cleanupPreallocatedPtrs(ctx *processingContext) {
	for _, info := range ctx.PreallocatedPtrs {
		if info.ptrField.IsNil() {
			// Already cleaned up (e.g., parent was nil'd in a previous iteration)
			continue
		}

		structPtr := info.ptrField.Interface()
		anySet := false

		// Check if this struct was explicitly mentioned in a config file
		if ctx.ConfigPresentPtrs[reflect.ValueOf(structPtr).Pointer()] {
			anySet = true
		}

		// Walk the struct's fields and check if any mirror was explicitly set.
		// Use the preallocation's recorded path so we resolve via mirrorByPath.
		if !anySet {
			collectAndCheck(ctx, structPtr, splitPath(info.path), &anySet)
		}

		if !anySet {
			// Check if any nested struct pointer survived cleanup (is still non-nil).
			// This handles the case where a deeply nested field was set, which keeps
			// the nested ptr alive — the parent should also survive.
			structVal := reflect.ValueOf(structPtr).Elem()
			for i := 0; i < structVal.NumField(); i++ {
				f := structVal.Field(i)
				if f.Kind() == reflect.Ptr && !f.IsNil() && f.Elem().Kind() == reflect.Struct {
					anySet = true
					break
				}
			}
		}

		if !anySet {
			// Remove mirrors for all fields within this subtree via path-prefix purge
			removeMirrorsForSubtree(ctx, info.path)
			// Nil the pointer
			info.ptrField.Set(reflect.Zero(info.ptrField.Type()))
		}
	}
}

// markConfigKeysPresent probes the raw config file data for key presence to detect
// which preallocated struct pointer fields were explicitly mentioned in the config,
// even if the values are zero or match the defaults.
//
// It delegates to the ConfigFormat's KeyTree to build a nested map[string]any
// representing the raw bytes' key structure, then matches entries against
// struct field names using case-insensitive logic (the same rule encoding/json
// applies). For nested structs, it recurses into sub-maps.
//
// Returns true if key-presence detection succeeded. Returns false when the
// format has no KeyTree, when the probe errors out, or when there are no
// preallocated pointer groups to care about — in those cases the caller
// should fall back to snapshot comparison.
func markConfigKeysPresent(ctx *processingContext, target any, targetPath fieldPath, rawData []byte, format ConfigFormat) bool {
	if len(ctx.PreallocatedPtrs) == 0 || len(rawData) == 0 {
		return false
	}
	if format.KeyTree == nil {
		return false
	}
	topLevel, err := format.KeyTree(rawData)
	if err != nil || topLevel == nil {
		return false
	}
	markConfigKeysPresentInStruct(ctx, target, topLevel, splitPath(targetPath))
	return true
}

// splitPath parses a fieldPath string back into the []int index path form.
// Accepts an empty path.
//
// splitPath is internal and its only legal input is a string previously emitted
// by joinPath — which always produces digit segments separated by dots. A parse
// error therefore represents a library-internal invariant violation, not user
// input, and has no sensible recovery: silently substituting 0 would corrupt
// downstream FieldByIndex lookups and surface as a confusing "mirror not found"
// failure far from the real bug. Panic is the right response.
func splitPath(p fieldPath) []int {
	if p == "" {
		return nil
	}
	parts := strings.Split(string(p), ".")
	out := make([]int, len(parts))
	for i, s := range parts {
		n, err := strconv.Atoi(s)
		if err != nil {
			panic(fmt.Errorf("boa: malformed fieldPath %q: segment %q is not a valid integer (this is a library bug, not user input)", p, s))
		}
		out[i] = n
	}
	return out
}

// jsonFieldKey returns the key that encoding/json would use for a struct field.
// If a json tag is present, use its name. Otherwise, use the Go field name.
func jsonFieldKey(field reflect.StructField) string {
	if tag := field.Tag.Get("json"); tag != "" {
		parts := strings.SplitN(tag, ",", 2)
		if parts[0] != "" && parts[0] != "-" {
			return parts[0]
		}
		if parts[0] == "-" {
			return "" // explicitly skipped
		}
	}
	return field.Name
}

// configKeyLookup does a case-insensitive key lookup matching encoding/json behavior:
// exact match first, then case-insensitive fallback. Works on any KeyTree output,
// regardless of the underlying config format (JSON, YAML, TOML, …).
func configKeyLookup(keys map[string]any, target string) (any, bool) {
	if target == "" {
		return nil, false
	}
	// Exact match first (fast path)
	if v, ok := keys[target]; ok {
		return v, true
	}
	// Case-insensitive fallback (matches encoding/json behavior)
	for k, v := range keys {
		if strings.EqualFold(k, target) {
			return v, true
		}
	}
	return nil, false
}

// asKeyMap coerces a KeyTree sub-value to a map[string]any when it is one,
// tolerating the map[any]any shape that some YAML parsers (notably yaml.v2)
// produce for nested mappings. Returns nil if the value is not a map-like.
func asKeyMap(v any) map[string]any {
	switch m := v.(type) {
	case map[string]any:
		return m
	case map[any]any:
		out := make(map[string]any, len(m))
		for k, val := range m {
			if ks, ok := k.(string); ok {
				out[ks] = val
			}
		}
		return out
	}
	return nil
}

func markConfigKeysPresentInStruct(ctx *processingContext, structPtr any, keys map[string]any, path []int) {
	val := reflect.ValueOf(structPtr).Elem()
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if isBoaIgnored(field) {
			continue
		}
		childPath := append(append([]int(nil), path...), i)
		key := jsonFieldKey(field)
		rawVal, keyPresent := configKeyLookup(keys, key)
		if !keyPresent {
			continue
		}
		fieldVal := val.Field(i)
		// If this is a struct pointer field and it's preallocated (non-nil)
		if field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct && !isSupportedType(field.Type) && !fieldVal.IsNil() {
			// The config mentioned this struct — mark it as present so the
			// pointer survives cleanup, even if the object is empty `{}`.
			// Individual fields only get setByConfig if they appear as keys.
			markStructPtrPresentByConfig(ctx, fieldVal.Interface())
			if subMap := asKeyMap(rawVal); len(subMap) > 0 {
				markConfigKeysPresentInStruct(ctx, fieldVal.Interface(), subMap, childPath)
			}
			continue
		}
		// If this is a non-pointer struct, recurse
		if field.Type.Kind() == reflect.Struct && !isSupportedType(field.Type) {
			if subMap := asKeyMap(rawVal); subMap != nil {
				markConfigKeysPresentInStruct(ctx, fieldVal.Addr().Interface(), subMap, childPath)
			}
			continue
		}
		// Leaf field present in config — mark its mirror
		if isSupportedType(field.Type) {
			if mirror, ok := ctx.mirrorByPath[joinPath(childPath)]; ok {
				if pm, isPM := mirror.(*paramMeta); isPM {
					pm.setByConfig = true
				}
			}
		}
	}
}

// snapshotPreallocatedStructs takes a deep copy of each preallocated struct's value.
// Used as fallback for non-JSON config formats where key-presence detection can't work.
func snapshotPreallocatedStructs(ctx *processingContext) []reflect.Value {
	snapshots := make([]reflect.Value, len(ctx.PreallocatedPtrs))
	for i, info := range ctx.PreallocatedPtrs {
		if info.ptrField.IsNil() {
			continue
		}
		orig := info.ptrField.Elem()
		cp := reflect.New(orig.Type()).Elem()
		cp.Set(orig)
		snapshots[i] = cp
	}
	return snapshots
}

// markConfigChangedStructs compares preallocated struct values against pre-config
// snapshots. If any struct changed, marks all its mirrors as setByConfig.
// This is the fallback for formats whose KeyTree cannot describe the literal
// key structure (no KeyTree set, or the KeyTree returned an error).
//
// The fallback is scoped per-load via fallbackRoots: a preallocated pointer
// is only considered if its path lies within one of those subtrees. This
// prevents a failing sub-load from blanket-marking fields that a *separate*
// load already covered precisely via its own KeyTree. An empty fieldPath in
// the list means "the entire root" (the legacy whole-tree behaviour, used
// when the root config itself has no KeyTree).
//
// An empty fallbackRoots slice is a no-op.
func markConfigChangedStructs(ctx *processingContext, snapshots []reflect.Value, fallbackRoots []fieldPath) {
	if len(fallbackRoots) == 0 {
		return
	}
	for i, info := range ctx.PreallocatedPtrs {
		if info.ptrField.IsNil() || !snapshots[i].IsValid() {
			continue
		}
		if !pathWithinAny(info.path, fallbackRoots) {
			continue
		}
		current := info.ptrField.Elem()
		if !reflect.DeepEqual(current.Interface(), snapshots[i].Interface()) {
			markAllMirrorsInSubtree(ctx, info.ptrField.Interface(), splitPath(info.path))
		}
	}
}

// pathWithinAny reports whether child lies within (or equals) any of roots.
// Uses fieldPath.hasSubtreePrefix so segment boundaries are respected (i.e.,
// path "12" is NOT considered within root "1"; only "1", "1.X", "1.X.Y"... are).
func pathWithinAny(child fieldPath, roots []fieldPath) bool {
	return slices.ContainsFunc(roots, child.hasSubtreePrefix)
}

// markStructPtrPresentByConfig records that a preallocated struct pointer was
// explicitly mentioned in a config file. This ensures the pointer survives
// cleanup even if no individual child fields were set (e.g., "DB": {}).
// Individual field HasValue is not affected — only cleanup is.
func markStructPtrPresentByConfig(ctx *processingContext, structPtr any) {
	addr := reflect.ValueOf(structPtr).Pointer()
	if ctx.ConfigPresentPtrs == nil {
		ctx.ConfigPresentPtrs = make(map[uintptr]bool)
	}
	ctx.ConfigPresentPtrs[addr] = true
}

// markAllMirrorsInSubtree marks every mirror within the given subtree as set by config.
// The subtree is identified by its path from the root (empty path == entire root).
// Only direct-descendant mirrors are marked; nested pointer-to-struct boundaries are
// handled separately by key-presence recursion in markConfigKeysPresentInStruct.
//
// We iterate mirrorByPath looking for entries whose path starts with prefix and does
// NOT cross a pointer-struct boundary further down (i.e., they're the immediate
// flat-descendant leaves reachable by struct-walking without ptr indirection).
// For simplicity — and to match the prior reflect-walking semantics — we walk the
// actual struct at `structPtr` and compute child paths to look up.
func markAllMirrorsInSubtree(ctx *processingContext, structPtr any, path []int) {
	val := reflect.ValueOf(structPtr).Elem()
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if isBoaIgnored(field) {
			continue
		}
		childPath := append(append([]int(nil), path...), i)
		fieldVal := val.Field(i)
		if isSupportedType(field.Type) {
			if mirror, ok := ctx.mirrorByPath[joinPath(childPath)]; ok {
				if pm, isPM := mirror.(*paramMeta); isPM {
					pm.setByConfig = true
				}
			}
			continue
		}
		if field.Type.Kind() == reflect.Struct {
			markAllMirrorsInSubtree(ctx, fieldVal.Addr().Interface(), childPath)
			continue
		}
		// Do NOT recurse into pointer-to-struct fields here.
		// Key-presence detection handles them.
	}
}

// collectAndCheck walks a struct's fields and checks if any mirror was explicitly set
// by the user (CLI, env var, or config file). Default values alone don't count.
// path is the path-from-root of structPtr (empty for root).
func collectAndCheck(ctx *processingContext, structPtr any, path []int, anySet *bool) {
	val := reflect.ValueOf(structPtr).Elem()
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		if *anySet {
			return
		}
		field := typ.Field(i)
		if isBoaIgnored(field) {
			continue
		}
		childPath := append(append([]int(nil), path...), i)
		fieldVal := val.Field(i)
		if isSupportedType(field.Type) {
			if mirror, ok := ctx.mirrorByPath[joinPath(childPath)]; ok {
				pm, isPM := mirror.(*paramMeta)
				if mirror.wasSetOnCli() || mirror.wasSetByEnv() {
					*anySet = true
					return
				}
				if isPM && pm.setByConfig {
					*anySet = true
					return
				}
			}
			continue
		}
		// Recurse into non-pointer structs
		if field.Type.Kind() == reflect.Struct {
			collectAndCheck(ctx, fieldVal.Addr().Interface(), childPath, anySet)
			continue
		}
		// Recurse into non-nil pointer structs
		if field.Type.Kind() == reflect.Ptr && !fieldVal.IsNil() && field.Type.Elem().Kind() == reflect.Struct {
			collectAndCheck(ctx, fieldVal.Interface(), childPath, anySet)
		}
	}
}

// removeMirrorsForSubtree removes all mirrors whose path is equal to or descends from
// the given prefix path. Uses string-prefix matching on the authoritative path store;
// no struct walking required.
func removeMirrorsForSubtree(ctx *processingContext, prefix fieldPath) {
	// Purge mirrorByPath
	for p := range ctx.mirrorByPath {
		if p.hasSubtreePrefix(prefix) {
			delete(ctx.mirrorByPath, p)
		}
	}
	// Rebuild pathOrder without the removed entries
	kept := ctx.pathOrder[:0]
	for _, p := range ctx.pathOrder {
		if _, still := ctx.mirrorByPath[p]; still {
			kept = append(kept, p)
		}
	}
	// Zero out tail references so GC can reclaim the removed fieldPath strings
	for i := len(kept); i < len(ctx.pathOrder); i++ {
		ctx.pathOrder[i] = ""
	}
	ctx.pathOrder = kept
	// Invalidate reverse address index — addresses inside the removed subtree are now stale
	ctx.addrToPath = nil
}

func parseEnv(ctx *processingContext, structPtr any) error {

	err := traverse(ctx, structPtr, func(param Param, _ string, _ reflect.StructTag) error {

		if !param.IsEnabled() {
			return nil
		}

		// Skip env reading for params that opt out of it:
		//   - IsIgnored: fully disabled; only config-file writes.
		//   - IsNoEnv:   env-only suppression; CLI and config still work.
		// Note that noflag is NOT checked here — a noflag param still reads env.
		if param.IsIgnored() || param.IsNoEnv() {
			return nil
		}

		if err := readEnv(param); err != nil {
			return err
		}

		return nil
	}, nil)
	return newUserInputError(err)
}

func validate(ctx *processingContext, structPtr any) error {

	err := traverse(ctx, structPtr, func(param Param, _ string, _ reflect.StructTag) error {

		if !param.IsEnabled() {
			return nil
		}

		// Fully ignored params are skipped end-to-end: no required check,
		// no conversion, no alt/min/max/pattern/custom validation. Config
		// files write directly to the raw struct field via unmarshal, so
		// their values still land — just without any boa-layer processing.
		if param.IsIgnored() {
			return nil
		}

		envHint := ""
		if param.GetEnv() != "" {
			envHint = fmt.Sprintf(" (env: %s)", param.GetEnv())
		}

		if param.IsRequired() && !HasValue(param) {
			return fmt.Errorf("missing required param '%s'%s", param.GetName(), envHint)
		}

		// Post-parse conversion for types stored as strings in cobra (time.Time, *url.URL, JSON fallback, etc.)
		if HasValue(param) {
			converted := false
			if handler, _ := lookupHandler(param.GetType()); handler != nil && handler.convert != nil {
				res, err := handler.convert(param.GetName(), param.valuePtrF())
				if err != nil {
					return err
				}
				param.setValuePtr(res)
				converted = true
			} else if param.GetKind() == reflect.Map {
				if mapHandler := lookupMapHandler(param.GetType()); mapHandler != nil && mapHandler.convert != nil {
					res, err := mapHandler.convert(param.GetName(), param.valuePtrF())
					if err != nil {
						return err
					}
					param.setValuePtr(res)
					converted = true
				}
			} else if param.GetKind() == reflect.Slice {
				if sliceHandler := lookupSliceHandler(param.GetType().Elem()); sliceHandler != nil && sliceHandler.convert != nil {
					res, err := sliceHandler.convert(param.GetName(), param.valuePtrF())
					if err != nil {
						return err
					}
					param.setValuePtr(res)
					converted = true
				}
			}

			// JSON fallback conversion: if value is still a *string but the target type
			// is a complex type (map, nested slice, etc.) without a native handler, try JSON unmarshal
			if !converted {
				needsJsonFallback := false
				if param.GetKind() == reflect.Map {
					needsJsonFallback = lookupMapHandler(param.GetType()) == nil
				} else if param.GetKind() == reflect.Slice && lookupSliceHandler(param.GetType().Elem()) == nil {
					needsJsonFallback = true
				}
				if needsJsonFallback {
					if strPtr, ok := param.valuePtrF().(*string); ok && strPtr != nil && *strPtr != "" {
						fallback := jsonFallbackHandler(param.GetType())
						res, err := fallback.convert(param.GetName(), param.valuePtrF())
						if err != nil {
							return err
						}
						param.setValuePtr(res)
					}
				}
			}
			if alts := param.GetAlternatives(); alts != nil && param.GetStrictAlts() {

				ptrVal := param.valuePtrF()
				// check if it is a slice param
				kind := reflect.TypeOf(ptrVal).Elem().Kind()
				if kind == reflect.Slice {
					// run the validation for each slice element
					sliceVal := reflect.ValueOf(ptrVal).Elem()
					for i := 0; i < sliceVal.Len(); i++ {
						elem := sliceVal.Index(i)
						strVal := fmt.Sprintf("%v", elem.Interface())
						if !slices.Contains(alts, strVal) {
							return fmt.Errorf("invalid value for param '%s': '%s' is not in the list of allowed values: %v", param.GetName(), strVal, alts)
						}
					}
				} else {
					if ptrVal != nil {
						strVal := ptrToAnyToString(ptrVal)
						if !slices.Contains(alts, strVal) {
							return fmt.Errorf("invalid value for param '%s': '%s' is not in the list of allowed values: %v", param.GetName(), strVal, alts)
						}
					}
				}
			}

			if err := param.customValidatorOfPtr()(param.valuePtrF()); err != nil {
				return fmt.Errorf("invalid value for param '%s': %s", param.GetName(), err.Error())
			}

			// min/max/pattern tag validation
			if pm, ok := param.(*paramMeta); ok {
				if err := validateMinMaxPattern(pm, param.valuePtrF()); err != nil {
					return fmt.Errorf("invalid value for param '%s': %s", param.GetName(), err.Error())
				}
			}
		}

		return nil
	}, nil)
	return newUserInputError(err)
}

// validateMinMaxPattern checks min/max/pattern tag constraints.
// For numeric types, min/max compare the value.
// For strings and slices, min/max compare length.
func validateMinMaxPattern(pm *paramMeta, valPtr any) error {
	if pm.minVal == nil && pm.maxVal == nil && pm.pattern == "" {
		return nil
	}

	v := reflect.ValueOf(valPtr)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64:
		val := float64(v.Int())
		if pm.minVal != nil && val < *pm.minVal {
			return fmt.Errorf("value %v is below min %v", v.Int(), *pm.minVal)
		}
		if pm.maxVal != nil && val > *pm.maxVal {
			return fmt.Errorf("value %v exceeds max %v", v.Int(), *pm.maxVal)
		}
	case reflect.Float32, reflect.Float64:
		val := v.Float()
		if pm.minVal != nil && val < *pm.minVal {
			return fmt.Errorf("value %v is below min %v", val, *pm.minVal)
		}
		if pm.maxVal != nil && val > *pm.maxVal {
			return fmt.Errorf("value %v exceeds max %v", val, *pm.maxVal)
		}
	case reflect.String:
		str := v.String()
		if pm.minVal != nil && float64(len(str)) < *pm.minVal {
			return fmt.Errorf("length %d is below min %v", len(str), *pm.minVal)
		}
		if pm.maxVal != nil && float64(len(str)) > *pm.maxVal {
			return fmt.Errorf("length %d exceeds max %v", len(str), *pm.maxVal)
		}
	case reflect.Slice:
		l := v.Len()
		if pm.minVal != nil && float64(l) < *pm.minVal {
			return fmt.Errorf("length %d is below min %v", l, *pm.minVal)
		}
		if pm.maxVal != nil && float64(l) > *pm.maxVal {
			return fmt.Errorf("length %d exceeds max %v", l, *pm.maxVal)
		}
	}

	if pm.pattern != "" && v.Kind() == reflect.String {
		matched, err := regexp.MatchString(pm.pattern, v.String())
		if err != nil {
			return fmt.Errorf("invalid pattern %q: %w", pm.pattern, err)
		}
		if !matched {
			return fmt.Errorf("value %q does not match pattern %q", v.String(), pm.pattern)
		}
	}

	return nil
}

func ptrToAnyToString(ptr any) string {
	if ptr == nil {
		panic("ptrToAnyToString called with nil")
	}

	val := reflect.ValueOf(ptr)
	if val.Kind() != reflect.Ptr {
		panic("ptrToAnyToString called with non-pointer")
	}

	elem := val.Elem()
	return fmt.Sprintf("%v", elem.Interface())
}

func doParsePositional(f Param, strVal string) error {
	if strVal == "" && f.IsRequired() {
		if f.hasDefaultValue() || f.wasSetByEnv() {
			return nil
		} else {
			return newUserInputErrorf("empty positional arg: %s", f.GetName())
		}
	}

	if err := readFrom(f, strVal); err != nil {
		return newUserInputError(err)
	}

	f.markSetPositionally()

	return nil
}

func toTypedSlice[T any](slice any) []T {
	if slice == nil {
		return nil
	}
	// Try direct type assertion first
	if typed, ok := slice.([]T); ok {
		return typed
	}
	// Handle type aliases by converting via reflection
	sliceVal := reflect.ValueOf(slice)
	if sliceVal.Kind() == reflect.Slice {
		result := make([]T, sliceVal.Len())
		var zero T
		targetElemType := reflect.TypeOf(zero)
		for i := 0; i < sliceVal.Len(); i++ {
			elem := sliceVal.Index(i)
			if elem.Type().ConvertibleTo(targetElemType) {
				result[i] = elem.Convert(targetElemType).Interface().(T)
			} else {
				result[i] = elem.Interface().(T)
			}
		}
		return result
	}
	// Fallback to original behavior (will panic with clear message)
	return slice.([]T)
}

func connect(f Param, cmd *cobra.Command, posArgs []Param, ctx *processingContext) error {

	if f.GetName() == "" {
		return fmt.Errorf("invalid conf for param '%s': long param name cannot be empty", f.GetName())
	}

	// Params marked noflag or ignored are not registered with cobra at all.
	// They still participate in env-var reading (unless ignored), config-file
	// loading, and validation — all of which happen outside of cobra.
	if f.IsNoFlag() || f.IsIgnored() {
		if f.IsNoFlag() && f.isPositional() {
			return fmt.Errorf("param '%s': noflag cannot be combined with positional args", f.GetName())
		}
		return nil
	}

	if f.GetShort() == "h" {
		// Check if we already have a help flag
		if hf := cmd.Flags().Lookup("help"); hf != nil {
			if hf.Shorthand == "h" {
				return fmt.Errorf("invalid conf for param '%s': short param cannot be 'h'. It collides with -h for help", f.GetName())
			}
		} else {
			return fmt.Errorf("invalid conf for param '%s': short param cannot be 'h'. It collides with the default help flag. Set a custom help flag if you wish to override -h", f.GetName())
		}
	}

	if f.GetName() == "help" {
		return fmt.Errorf("invalid conf for param '%s': name cannot be 'help'. It collides with the standard help param", f.GetName())
	}

	extraInfos := make([]string, 0)

	descr := f.getDescr()
	if f.GetEnv() != "" {
		extraInfos = append(extraInfos, fmt.Sprintf("env: %s", f.GetEnv()))
	}

	if f.IsRequired() && !f.hasDefaultValue() {
		extraInfos = append(extraInfos, "required")
	}

	if f.GetIsEnabledFn() != nil {
		extraInfos = append(extraInfos, "conditional")
	}

	if len(extraInfos) > 0 {
		descr = fmt.Sprintf("%s (%s)", descr, strings.Join(extraInfos, ", "))
	}

	if f.hasDefaultValue() {
		if f.GetKind() == reflect.Bool {
			// cobra doesn't show if the default is false. So we must do it ourselves
			if f.defaultValueStr() == "false" {
				descr = fmt.Sprintf("%s (default false)", descr)
			}
		} else if f.defaultValueStr() == "" {
			// cobra doesn't show explicitly empty defaults. So we must do it ourselves
			descr = fmt.Sprintf("%s (default \"\")", descr)
		}
	}

	if f.parentCmd() != nil {
		return fmt.Errorf("param '%s' already connected to a command. Looks like you are trying to use the same parameter struct for two commands. This is not possible. Pleas instantiate one separate struct instance per command", f.GetName())
	}
	f.setParentCmd(cmd)

	if f.isPositional() {
		startSign := func() string {
			if f.IsRequired() {
				return "<"
			} else {
				return "["
			}
		}()
		endSign := func() string {
			if f.IsRequired() {
				return ">"
			} else {
				return "]"
			}
		}()
		suffix := ""
		if f.GetType().Kind() == reflect.Slice {
			suffix = "..."
		}
		cmd.Use += " " + startSign + f.GetName() + suffix + endSign

		if cmd.Args == nil {
			cmd.Args = func(cmd *cobra.Command, args []string) error {
				return nil
			}
		}
		// Add the positional arg to the Args function
		oldFn := cmd.Args
		cmd.Args = func(cmd *cobra.Command, args []string) error {
			if err := oldFn(cmd, args); err != nil {
				return err
			}
			posArgIndex := -1
			for i, posArg := range posArgs {
				if posArg.GetName() == f.GetName() {
					posArgIndex = i
				}
			}
			if posArgIndex == -1 {
				if f.IsRequired() {
					return fmt.Errorf("positional arg '%s' not found. This is a bug in boa", f.GetName())
				} else {
					return nil
				}
			}
			if posArgIndex >= len(args) {
				if f.IsRequired() {
					if f.hasDefaultValue() {
						f.setValuePtr(f.defaultValuePtr())
						return nil
					} else {
						return newUserInputErrorf("missing positional arg '%s'", f.GetName())
					}
				} else {
					return nil
				}
			}
			if f.GetKind() == reflect.Slice {
				// Concat all remaining args into a single string, to be parsed as a slice
				remainingArgs := args[posArgIndex:]
				joinedArgs := "[" + strings.Join(remainingArgs, ",") + "]"
				return doParsePositional(f, joinedArgs)
			} else {
				return doParsePositional(f, args[posArgIndex])
			}
		}
		return nil // no need to attach cobra flags
	}

	// Must happen last, because the flags must have been created
	defer func() {
		if f.GetAlternatives() != nil {
			err := cmd.RegisterFlagCompletionFunc(f.GetName(), func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				return f.GetAlternatives(), cobra.ShellCompDirectiveDefault
			})
			if err != nil {
				panic(fmt.Errorf("failed to register static flag completion func for flag '%s': %v", f.GetName(), err))
			}
		}
		if f.GetAlternativesFunc() != nil {
			err := cmd.RegisterFlagCompletionFunc(f.GetName(), func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				// Sync cobra's parsed flag values into raw struct fields before calling
				// the user's completion function. Without this, raw fields (plain string,
				// int, etc.) are zero during completion because PreRunE (which normally
				// calls syncMirrors) is never executed for shell completion.
				syncMirrors(ctx)
				return f.GetAlternativesFunc()(cmd, args, toComplete), cobra.ShellCompDirectiveDefault
			})
			if err != nil {
				panic(fmt.Errorf("failed to register dynamic flag completion func for flag '%s': %v", f.GetName(), err))
			}
		}
	}()

	// Look up type handler for scalar types (including net.IP which is []byte but treated as scalar)
	if handler, _ := lookupHandler(f.GetType()); handler != nil {
		var defVal any
		if f.hasDefaultValue() {
			defVal = f.defaultValuePtr()
		}
		f.setValuePtr(handler.bindFlag(cmd, f.GetName(), f.GetShort(), descr, defVal))
		return nil
	}

	// Map types — try native handler first, then JSON fallback
	if f.GetKind() == reflect.Map {
		mapHandler := lookupMapHandler(f.GetType())
		if mapHandler == nil {
			mapHandler = jsonFallbackHandler(f.GetType())
		}
		var defVal any
		if f.hasDefaultValue() {
			defVal = f.defaultValuePtr()
		}
		f.setValuePtr(mapHandler.bindFlag(cmd, f.GetName(), f.GetShort(), descr, defVal))
		return nil
	}

	// Slice types — try native handler first, then JSON fallback
	if f.GetKind() == reflect.Slice {
		elemType := f.GetType().Elem()
		sliceHandler := lookupSliceHandler(elemType)

		if sliceHandler != nil {
			var defVal any
			if f.hasDefaultValue() {
				defValRef := reflect.ValueOf(f.defaultValuePtr()).Elem()
				// If default was parsed from string tag, it might need parsing into the proper slice type
				if defValRef.Kind() != reflect.Slice {
					parsed, err := sliceHandler.parse(f.GetName(), f.defaultValueStr())
					if err != nil {
						return fmt.Errorf("default value for slice param '%s' is invalid: %s", f.GetName(), err.Error())
					}
					f.SetDefault(parsed)
				}
				defVal = f.defaultValuePtr()
			}
			f.setValuePtr(sliceHandler.bindFlag(cmd, f.GetName(), f.GetShort(), descr, defVal))
			return nil
		}

		// JSON fallback for complex slice types (nested slices, etc.)
		fallback := jsonFallbackHandler(f.GetType())
		var defVal any
		if f.hasDefaultValue() {
			defVal = f.defaultValuePtr()
		}
		f.setValuePtr(fallback.bindFlag(cmd, f.GetName(), f.GetShort(), descr, defVal))
		return nil
	}

	if f.GetKind() == reflect.Array {
		return fmt.Errorf("unsupported param type (Array): %s: ", f.GetKind().String())
	}

	return fmt.Errorf("unsupported param type: %s", f.GetKind().String())
}

func readEnv(f Param) error {
	if f.GetEnv() == "" {
		return nil
	}

	if f.wasSetOnCli() {
		return nil
	}

	envVal := os.Getenv(f.GetEnv())
	if envVal == "" {
		return nil
	}

	err := readFrom(f, envVal)
	if err != nil {
		return err
	}

	f.markSetFromEnv()
	return nil
}

func readFrom(f Param, strVal string) error {

	ptr, err := parsePtr(f.GetName(), f.GetType(), f.GetKind(), strVal)
	if err != nil {
		return err
	}

	f.setValuePtr(ptr)

	return nil
}

// parseTimeString parses a time string trying multiple common formats
func parseTimeString(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %s", s)
}

func parsePtr(
	name string,
	tpe reflect.Type,
	kind reflect.Kind,
	strVal string,
) (any, error) {
	// Scalar types — look up by exact type first, then by kind
	if handler, _ := lookupHandler(tpe); handler != nil {
		return handler.parse(name, strVal)
	}

	// Map types — native handler or JSON fallback
	if kind == reflect.Map {
		if mapHandler := lookupMapHandler(tpe); mapHandler != nil {
			return mapHandler.parse(name, strVal)
		}
		return jsonFallbackHandler(tpe).parse(name, strVal)
	}

	// Slice types — native handler or JSON fallback
	if kind == reflect.Slice {
		elemType := tpe.Elem()
		if sliceHandler := lookupSliceHandler(elemType); sliceHandler != nil {
			return sliceHandler.parse(name, strVal)
		}
		return jsonFallbackHandler(tpe).parse(name, strVal)
	}

	if kind == reflect.Array {
		return nil, fmt.Errorf("arrays not supported param type. Use a slice instead: %s", kind.String())
	}

	return nil, fmt.Errorf("unsupported param type: %s", kind.String())
}

func camelToKebabCase(in string) string {
	var result strings.Builder
	runes := []rune(in)

	for i, char := range runes {
		if unicode.IsUpper(char) {
			if i > 0 {
				prev := runes[i-1]
				// Insert dash before uppercase if previous was lowercase,
				// or if previous was uppercase but next is lowercase (end of acronym)
				// AND the lowercase tail is more than 1 char (avoids splitting plurals
				// like "URLs" → "urls" while still splitting "DBHost" → "db-host").
				if unicode.IsLower(prev) {
					result.WriteRune('-')
				} else if unicode.IsUpper(prev) && i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
					// Count lowercase chars that follow
					lowercaseTail := 0
					for j := i + 1; j < len(runes) && unicode.IsLower(runes[j]); j++ {
						lowercaseTail++
					}
					if lowercaseTail > 1 {
						result.WriteRune('-')
					}
				}
			}
			result.WriteRune(unicode.ToLower(char))
		} else {
			result.WriteRune(char)
		}
	}

	return result.String()
}

func kebabCaseToUpperSnakeCase(in string) string {
	var result strings.Builder

	for _, char := range in {
		if char == '-' {
			result.WriteRune('_')
		} else {
			result.WriteRune(char)
		}
	}

	return strings.ToUpper(result.String())
}

func getBoaTags(field reflect.StructField) []string {
	parts := strings.Split(field.Tag.Get("boa"), ",")
	results := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			results = append(results, part)
		}
	}
	return results
}

func isBoaIgnored(field reflect.StructField) bool {
	boaTags := getBoaTags(field)
	return slices.Contains(boaTags, "ignore") ||
		slices.Contains(boaTags, "ignored") ||
		slices.Contains(boaTags, "configonly") ||
		slices.Contains(boaTags, "-")
}

// traverse is the public-facing entrypoint; it walks from the root params struct
// starting with an empty path. It accepts variadic prefix strings for backward
// compatibility with existing call sites that sometimes preload a name prefix.
func traverse(
	ctx *processingContext,
	structPtr any,
	fParam func(param Param, paramFieldName string, tags reflect.StructTag) error,
	fStruct func(structPtr any) error,
	prefixParts ...string,
) error {
	return traverseAt(ctx, structPtr, nil, fParam, fStruct, prefixParts...)
}

// traverseAt walks the struct tree while carrying the declared-index path from
// the root. The path is what identifies each leaf mirror in mirrorByPath.
// Ignored fields retain their declared index slot; only the loop variable i is
// used for path construction (never a visit counter), so interleaved ignored
// fields cannot drift the key.
func traverseAt(
	ctx *processingContext,
	structPtr any,
	path []int,
	fParam func(param Param, paramFieldName string, tags reflect.StructTag) error,
	fStruct func(structPtr any) error,
	prefixParts ...string,
) error {
	prefix := strings.Join(prefixParts, "")

	if reflect.TypeOf(structPtr).Kind() != reflect.Ptr {
		return fmt.Errorf("expected pointer to struct")
	}

	if reflect.TypeOf(structPtr).Elem().Kind() != reflect.Struct {
		return fmt.Errorf("expected pointer to struct")
	}

	if fStruct != nil {
		err := fStruct(structPtr)
		if err != nil {
			return err
		}
	}

	// use reflection to iterate over all fields of the struct
	fields := reflect.TypeOf(structPtr).Elem()
	rootValue := reflect.ValueOf(structPtr).Elem()
	for i := 0; i < fields.NumField(); i++ {
		field := fields.Field(i)

		if isBoaIgnored(field) {
			continue
		}

		// childPath is the declared-index path to this field from the root.
		// Use declared index (i), not a visit counter, so ignored fields do not
		// drift subsequent siblings.
		//
		// Single allocation: len(path)+1 explicitly sized. The old
		// append(append([]int(nil), path...), i) form did two allocations per
		// field visit, which matters during init of larger parameter trees.
		childPath := make([]int, len(path)+1)
		copy(childPath, path)
		childPath[len(path)] = i

		fieldAddr := rootValue.Field(i).Addr()
		// check if field is a param
		param, isParam := fieldAddr.Interface().(Param)
		prefixedName := prefix + field.Name
		if isParam {
			if fParam != nil {
				err := fParam(param, prefixedName, field.Tag)
				if err != nil {
					return err
				}
			}
		} else {

			// check if it is a struct (but not registered types like time.Time or custom types)
			if field.Type.Kind() == reflect.Struct && !isSupportedType(field.Type) {
				// Named (non-anonymous) struct fields get auto-prefixed
				childPrefix := prefix
				if !field.Anonymous {
					childPrefix = prefix + field.Name
				}
				if err := traverseAt(ctx, fieldAddr.Interface(), childPath, fParam, fStruct, childPrefix); err != nil {
					return err
				}
				continue
			}

			// check if it is a pointer to a struct (but not *url.URL or pointer-to-supported-type)
			if field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct && !isSupportedType(field.Type) {
				if !fieldAddr.IsNil() && !fieldAddr.Elem().IsNil() {
					childPrefix := prefix
					if !field.Anonymous {
						childPrefix = prefix + field.Name
					}
					if err := traverseAt(ctx, fieldAddr.Elem().Interface(), childPath, fParam, fStruct, childPrefix); err != nil {
						return err
					}
				}
				continue
			}

			// For raw fields, store parameter mirrors keyed by field-index path.
			if isSupportedType(field.Type) {

				// check if we already have a mirror for this field
				pathKey := joinPath(childPath)
				var ok bool
				if param, ok = ctx.mirrorByPath[pathKey]; !ok {
					param = newParam(&field, field.Type)
					// Set prefix for named struct nesting, and stash the path
					// key on the mirror so callers that have a *paramMeta can
					// read it without scanning pathOrder.
					if pm, ok := param.(*paramMeta); ok {
						pm.pathKey = pathKey
						if prefix != "" {
							pm.flagPrefix = camelToKebabCase(prefix) + "-"
							pm.envPrefix = kebabCaseToUpperSnakeCase(pm.flagPrefix[:len(pm.flagPrefix)-1]) + "_"
						}
					}
					ctx.pathOrder = append(ctx.pathOrder, pathKey)
					ctx.mirrorByPath[pathKey] = param
					// Maintain the reverse address index for HookContext.GetParam(&field).
					// This is a non-authoritative cache — mirrorByPath is the source of truth.
					if ctx.addrToPath != nil {
						ctx.addrToPath[fieldAddr.UnsafePointer()] = pathKey
					}
				}

				if fParam != nil {
					err := fParam(param, prefixedName, field.Tag)
					if err != nil {
						return err
					}
				}

				continue
			}

			slog.Warn(fmt.Sprintf("field %s is not a type that is interpretable as a boa.Param. It will be ignored", field.Name))
			continue // not a param
		}
	}

	return nil
}

// toCobraBase sets up the cobra command with all common configuration (flags, validation, lifecycle hooks)
// but does NOT set the Run/RunE function. Returns both the command and the processing context
// so callers can set up the appropriate run function with access to the context.
// Returns an error if any setup/initialization fails.
func (b Cmd) toCobraBase() (*cobra.Command, *processingContext, error) {
	cmd := &cobra.Command{
		Use:           b.Use,
		Short:         b.Short,
		Long:          b.Long,
		Args:          b.Args,
		SilenceErrors: true,
		SilenceUsage:  true,
		ValidArgs:     b.ValidArgs,
	}

	if b.RawArgs != nil {
		cmd.SetArgs(b.RawArgs)
	}

	ctx := &processingContext{
		Context:       context.Background(), // prepare to override later?
		rootStructPtr: b.Params,
		mirrorByPath:  map[fieldPath]Param{},
		pathOrder:     []fieldPath{},
		addrToPath:    map[unsafe.Pointer]fieldPath{},
	}

	// Preallocate nil struct pointer fields so traverse can discover their children.
	// This must happen before the first traverse and before init hooks so users can
	// access fields like &params.DB.Port in InitFunc/InitFuncCtx.
	if b.Params != nil {
		preallocateStructPtrs(ctx, b.Params, nil)
	}

	// build mirrors
	if b.Params != nil {
		_ = traverse(ctx, b.Params, nil, func(innerParams any) error {
			return nil
		})
	}

	syncMirrors(ctx)

	// if b.params or any inner struct implements CfgStructInit, call it
	if b.Params != nil {
		err := traverse(ctx, b.Params, nil, func(innerParams any) error {
			if toInit, ok := b.Params.(CfgStructInit); ok {
				err := toInit.Init()
				if err != nil {
					return fmt.Errorf("error in CfgStructInit.Init(): %w", err)
				}
			}
			// context-aware interface
			if toInitCtx, ok := b.Params.(CfgStructInitCtx); ok {
				hookCtx := newHookContext(ctx)
				err := toInitCtx.InitCtx(hookCtx)
				if err != nil {
					return fmt.Errorf("error in CfgStructInitCtx.InitCtx(): %w", err)
				}
			}
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	}

	// if we have a custom init function, call it
	if b.InitFunc != nil {
		err := b.InitFunc(b.Params, cmd)
		if err != nil {
			return nil, nil, fmt.Errorf("error in InitFunc: %w", err)
		}
	}

	// if we have a context-aware init function, call it
	if b.InitFuncCtx != nil {
		hookCtx := newHookContext(ctx)
		err := b.InitFuncCtx(hookCtx, b.Params, cmd)
		if err != nil {
			return nil, nil, fmt.Errorf("error in InitFuncCtx: %w", err)
		}
	}

	syncMirrors(ctx)

	cmd.Flags().SortFlags = b.SortFlags
	cmd.Version = b.Version
	cmd.Aliases = b.Aliases
	cmd.GroupID = b.GroupID

	for _, subcommand := range b.SubCmds {
		cmd.AddCommand(subcommand)
	}

	// Add explicit groups first, tracking which IDs are defined
	definedGroups := make(map[string]bool)
	for _, group := range b.Groups {
		cmd.AddGroup(group)
		definedGroups[group.ID] = true
	}

	// Auto-generate groups for any subcommand GroupIDs not already defined
	for _, subcommand := range b.SubCmds {
		if subcommand.GroupID != "" && !definedGroups[subcommand.GroupID] {
			cmd.AddGroup(&cobra.Group{ID: subcommand.GroupID, Title: subcommand.GroupID + ":"})
			definedGroups[subcommand.GroupID] = true
		}
	}

	var positional []Param

	if b.Params != nil {

		// look in tags for info about positional args
		currentStructPtr := b.Params
		err := traverse(ctx, b.Params, func(param Param, _ string, tags reflect.StructTag) error {
			if tags.Get("positional") == "true" || tags.Get("pos") == "true" {
				param.setPositional(true)
			}
			if param.getDescr() == "" {
				if descr, ok := tags.Lookup("help"); ok {
					param.setDescription(descr)
				} else if descr, ok := tags.Lookup("desc"); ok {
					param.setDescription(descr)
				} else if descr, ok := tags.Lookup("descr"); ok {
					param.setDescription(descr)
				} else if descr, ok := tags.Lookup("description"); ok {
					param.setDescription(descr)
				}
			}
			if param.GetEnv() == "" {
				if env, ok := tags.Lookup("env"); ok {
					// Apply struct prefix to explicit env tags
					if pm, ok2 := param.(*paramMeta); ok2 && pm.envPrefix != "" {
						param.SetEnv(pm.envPrefix + env)
					} else {
						param.SetEnv(env)
					}
				}
			}
			if param.GetShort() == "" {
				if shrt, ok := tags.Lookup("short"); ok {
					param.SetShort(shrt)
				}
			}
			if param.GetName() == "" {
				if name, ok := tags.Lookup("name"); ok {
					// Apply struct prefix to explicit name tags
					if pm, ok2 := param.(*paramMeta); ok2 && pm.flagPrefix != "" {
						param.SetName(pm.flagPrefix + name)
					} else {
						param.SetName(name)
					}
				} else if name, ok := tags.Lookup("long"); ok {
					if pm, ok2 := param.(*paramMeta); ok2 && pm.flagPrefix != "" {
						param.SetName(pm.flagPrefix + name)
					} else {
						param.SetName(name)
					}
				}
			}

			setAlts := func(alts string) {
				strVal := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(alts), "["), "]")
				elements := strings.Split(strVal, ",")
				for i, element := range elements {
					elements[i] = strings.TrimSpace(element)
				}
				// Remove empty
				nonEmpty := make([]string, 0)
				for _, element := range elements {
					if element != "" {
						nonEmpty = append(nonEmpty, element)
					}
				}
				param.SetAlternatives(nonEmpty)
			}

			if alts, ok := tags.Lookup("alts"); ok {
				setAlts(alts)
			}
			if alts, ok := tags.Lookup("alternatives"); ok {
				setAlts(alts)
			}

			if strictAlts, ok := tags.Lookup("strict-alts"); ok {
				param.SetStrictAlts(strictAlts == "true")
			}
			if strictAlts, ok := tags.Lookup("strict"); ok {
				param.SetStrictAlts(strictAlts == "true")
			}

			if !param.hasDefaultValue() {
				// Default values are used for injection. So we can't just overwrite them
				if defaultPtr, ok := tags.Lookup("default"); ok {
					ptr, err := parsePtr(param.GetName(), param.GetType(), param.GetKind(), defaultPtr)
					if err != nil {
						return fmt.Errorf("invalid default value for param %s: %s", param.GetName(), err.Error())
					}
					param.SetDefault(ptr)
				}
			}

			// Parse min/max/pattern validation tags
			if pm, ok := param.(*paramMeta); ok {
				if minStr, ok := tags.Lookup("min"); ok {
					v, err := strconv.ParseFloat(minStr, 64)
					if err != nil {
						return fmt.Errorf("invalid min value for param %s: %s", param.GetName(), err.Error())
					}
					pm.minVal = &v
				}
				if maxStr, ok := tags.Lookup("max"); ok {
					v, err := strconv.ParseFloat(maxStr, 64)
					if err != nil {
						return fmt.Errorf("invalid max value for param %s: %s", param.GetName(), err.Error())
					}
					pm.maxVal = &v
				}
				if pat, ok := tags.Lookup("pattern"); ok {
					pm.pattern = pat
				}
			}

			// Detect `boa` directives that suppress individual input channels
			// without fully ignoring the param:
			//   - noflag / nocli → skip CLI flag, keep env + config + validation
			//   - noenv          → skip env var reading, keep CLI + config + validation
			// These are orthogonal and can be combined (e.g. config-only fields
			// that still want validation).
			for _, t := range strings.Split(tags.Get("boa"), ",") {
				switch strings.TrimSpace(t) {
				case "noflag", "nocli":
					param.SetNoFlag(true)
				case "noenv":
					param.SetNoEnv(true)
				}
			}
			if param.IsNoFlag() && param.isPositional() {
				return fmt.Errorf("param '%s': `boa:\"noflag\"` cannot be combined with positional args (a CLI positional arg must be a CLI arg)", param.GetName())
			}

			// Detect configfile tag
			if cfgTag, ok := tags.Lookup("configfile"); ok && cfgTag == "true" {
				if param.GetType().Kind() != reflect.String {
					return fmt.Errorf("configfile tag on param %s: must be a string field", param.GetName())
				}
				// The target struct path is the parent of the configfile param's own path.
				// Read it directly from paramMeta.pathKey (stashed during traverse) —
				// no scan over pathOrder needed.
				var targetPath fieldPath
				if pm, ok := param.(*paramMeta); ok {
					if idx := strings.LastIndex(string(pm.pathKey), "."); idx >= 0 {
						targetPath = pm.pathKey[:idx]
					}
					// If there's no dot, the configfile field lives at the root
					// and its target path is the empty fieldPath — already set.
				}
				ctx.ConfigFiles = append(ctx.ConfigFiles, configFileEntry{
					mirror:     param,
					target:     currentStructPtr,
					targetPath: targetPath,
				})
			}

			return nil
		}, func(structPtr any) error {
			currentStructPtr = structPtr
			return nil
		})

		if err != nil {
			return nil, nil, fmt.Errorf("error parsing tags: %w", err)
		}

		if b.ParamEnrich == nil {
			b.ParamEnrich = ParamEnricherDefault
		}
		processed := make([]Param, 0)
		err = traverse(ctx, b.Params, func(param Param, paramFieldName string, _ reflect.StructTag) error {
			err := b.ParamEnrich(processed, param, paramFieldName)
			if err != nil {
				return err
			}
			processed = append(processed, param)
			return nil
		}, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("error enriching params: %s", err.Error())
		}

		for _, param := range processed {
			if param.isPositional() {
				// if the last positional is a slice, error out
				if len(positional) >= 1 {
					lastPos := positional[len(positional)-1]
					if lastPos.GetKind() == reflect.Slice {
						return nil, nil, fmt.Errorf("positional param %s cannot come after slice positional param %s", param.GetName(), lastPos.GetName())
					}
				}
				positional = append(positional, param)
			}
		}

		// Check that no required positional arg exists after on optional positional arg
		numReqPositional := 0
		allowArbitraryNumPositional := false
		for i, param := range positional {
			if param.GetKind() == reflect.Slice {
				allowArbitraryNumPositional = true
			}
			if param.IsRequired() && !param.hasDefaultValue() {
				numReqPositional++
			}
			if param.IsRequired() && i >= 1 {
				prev := positional[i-1]
				if !prev.IsRequired() {
					return nil, nil, fmt.Errorf("required positional arg %s must come before optional positional arg %s", param.GetName(), prev.GetName())
				}
			}
		}

		if cmd.Args == nil {
			if allowArbitraryNumPositional {
				cmd.Args = wrapArgsValidator(cobra.MinimumNArgs(numReqPositional))
			} else {
				cmd.Args = wrapArgsValidator(cobra.RangeArgs(numReqPositional, len(positional)))
			}
		} else {
			cmd.Args = wrapArgsValidator(cmd.Args)
		}

		syncMirrors(ctx)

		err = traverse(ctx, b.Params, func(param Param, _ string, tags reflect.StructTag) error {
			err := connect(param, cmd, positional, ctx)
			if err != nil {
				return err
			}

			return nil
		}, nil)

		// if b.Params implements CfgStructPostCreate, call it
		if postCreate, ok := b.Params.(CfgStructPostCreate); ok {
			if err := postCreate.PostCreate(); err != nil {
				return nil, nil, fmt.Errorf("error in CfgStructPostCreate.PostCreate(): %w", err)
			}
		}
		if postCreateCtx, ok := b.Params.(CfgStructPostCreateCtx); ok {
			hookCtx := newHookContext(ctx)
			if err := postCreateCtx.PostCreateCtx(hookCtx); err != nil {
				return nil, nil, fmt.Errorf("error in CfgStructPostCreateCtx.PostCreateCtx(): %w", err)
			}
		}
		if b.PostCreateFunc != nil {
			err := b.PostCreateFunc(b.Params, cmd)
			if err != nil {
				return nil, nil, fmt.Errorf("error in PostCreateFunc: %w", err)
			}
		}
		if b.PostCreateFuncCtx != nil {
			hookCtx := newHookContext(ctx)
			err := b.PostCreateFuncCtx(hookCtx, b.Params, cmd)
			if err != nil {
				return nil, nil, fmt.Errorf("error in PostCreateFuncCtx: %w", err)
			}
		}
		if err != nil {
			return nil, nil, fmt.Errorf("error connecting params: %s", err.Error())
		}
	}

	// Build ValidArgsFunction from per-positional-param Alternatives/AlternativesFunc
	// and/or the user-provided ValidArgsFunc. Per-param completions are checked first
	// for the current position; the user's ValidArgsFunc is used as fallback.
	{
		hasPositionalCompletion := false
		for _, p := range positional {
			if p.GetAlternatives() != nil || p.GetAlternativesFunc() != nil {
				hasPositionalCompletion = true
				break
			}
		}
		if hasPositionalCompletion || b.ValidArgsFunc != nil {
			userValidArgsFunc := b.ValidArgsFunc
			cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
				syncMirrors(ctx)
				posIdx := len(args)
				if posIdx < len(positional) {
					p := positional[posIdx]
					if p.GetAlternativesFunc() != nil {
						return p.GetAlternativesFunc()(cmd, args, toComplete), cobra.ShellCompDirectiveDefault
					}
					if p.GetAlternatives() != nil {
						return p.GetAlternatives(), cobra.ShellCompDirectiveDefault
					}
				}
				if userValidArgsFunc != nil {
					return userValidArgsFunc(cmd, args, toComplete)
				}
				return nil, cobra.ShellCompDirectiveDefault
			}
		}
	}

	// now wrap the run function of the command to validate the flags
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if b.Params != nil {

			// Must read env values before running any prevalidate code
			if err := parseEnv(ctx, b.Params); err != nil {
				return err
			}

			syncMirrors(ctx)

			// Snapshot preallocated structs before config loading. Used as fallback
			// for non-JSON formats where key-presence detection can't work.
			var preConfigSnapshots []reflect.Value
			if len(ctx.PreallocatedPtrs) > 0 {
				preConfigSnapshots = snapshotPreallocatedStructs(ctx)
			}

			// Auto-load config files tagged with configfile:"true".
			// Substruct configs load first, root config loads last (root overrides inner).
			// Priority: CLI > env > root config > substruct config > defaults
			//
			// We collect (target, rawData) pairs so that after loading we can probe
			// the raw bytes for key presence — this lets us detect config-file writes
			// even when the value equals Go's zero value or the field's default.
			type configLoadResult struct {
				target     any
				targetPath fieldPath
				rawData    []byte
				format     ConfigFormat
			}
			var configResults []configLoadResult

			// Resolve the per-command override once. ConfigFormat takes
			// precedence over the legacy ConfigUnmarshal; if neither is set,
			// loadConfigFileInto falls back to the extension-registered format.
			cmdOverride := b.ConfigFormat
			if cmdOverride.Unmarshal == nil && b.ConfigUnmarshal != nil {
				cmdOverride = ConfigFormat{Unmarshal: b.ConfigUnmarshal}
			}

			if len(ctx.ConfigFiles) > 0 {
				// Separate root and substruct entries
				var subEntries, rootEntries []configFileEntry
				for _, entry := range ctx.ConfigFiles {
					if entry.target == b.Params {
						rootEntries = append(rootEntries, entry)
					} else {
						subEntries = append(subEntries, entry)
					}
				}
				// Load substruct configs first
				for _, entry := range subEntries {
					if entry.mirror.HasValue() {
						filePath := *(entry.mirror.valuePtrF().(*string))
						if filePath != "" {
							rawData, effective, err := loadConfigFileInto(filePath, entry.target, cmdOverride)
							if err != nil {
								return NewUserInputError(fmt.Errorf("configfile %s: %w", entry.mirror.GetName(), err))
							}
							configResults = append(configResults, configLoadResult{target: entry.target, targetPath: entry.targetPath, rawData: rawData, format: effective})
						}
					}
				}
				// Then load root config (overrides substruct values)
				for _, entry := range rootEntries {
					if entry.mirror.HasValue() {
						filePath := *(entry.mirror.valuePtrF().(*string))
						if filePath != "" {
							rawData, effective, err := loadConfigFileInto(filePath, entry.target, cmdOverride)
							if err != nil {
								return NewUserInputError(fmt.Errorf("configfile %s: %w", entry.mirror.GetName(), err))
							}
							configResults = append(configResults, configLoadResult{target: entry.target, targetPath: entry.targetPath, rawData: rawData, format: effective})
						}
					}
				}
				syncMirrors(ctx)
			}

			// Probe raw config data for key presence to detect which preallocated
			// struct pointers were mentioned in config files. This detects writes
			// even when the value equals Go's zero value or the field's default.
			// Formats whose ConfigFormat has no KeyTree (or whose KeyTree errors
			// out) fall back to snapshot comparison — but only for the subtree
			// of that particular load, so a failing sub-load can't corrupt the
			// precision of sibling loads whose KeyTree succeeded.
			var fallbackRoots []fieldPath
			for _, cr := range configResults {
				if !markConfigKeysPresent(ctx, cr.target, cr.targetPath, cr.rawData, cr.format) {
					fallbackRoots = append(fallbackRoots, cr.targetPath)
				}
			}
			if len(fallbackRoots) > 0 && preConfigSnapshots != nil {
				markConfigChangedStructs(ctx, preConfigSnapshots, fallbackRoots)
			}

			// Clean up preallocated struct pointers that had no fields set.
			// This must happen after all value sources (CLI, env, config) and before
			// validation, so that required-field checks don't fire for unused struct groups.
			if len(ctx.PreallocatedPtrs) > 0 {
				cleanupPreallocatedPtrs(ctx)
			}

			// if b.params or any inner struct implements CfgStructPreValidate, call it
			err := traverse(ctx, b.Params, nil, func(innerParams any) error {
				if s, ok := innerParams.(CfgStructPreValidate); ok {
					err := s.PreValidate()
					if err != nil {
						return fmt.Errorf("error in PreValidate: %w", err)
					}
				}
				// context-aware interface
				if s, ok := innerParams.(CfgStructPreValidateCtx); ok {
					hookCtx := newHookContext(ctx)
					err := s.PreValidateCtx(hookCtx)
					if err != nil {
						return fmt.Errorf("error in PreValidateCtx: %w", err)
					}
				}
				return nil
			})
			if err != nil {
				return err
			}

			// if we have a custom pre-execute function, call it
			if b.PreValidateFunc != nil {
				err := b.PreValidateFunc(b.Params, cmd, args)
				if err != nil {
					return fmt.Errorf("error in PreValidateFunc: %w", err)
				}
			}

			// if we have a context-aware pre-validate function, call it
			if b.PreValidateFuncCtx != nil {
				hookCtx := newHookContext(ctx)
				err := b.PreValidateFuncCtx(hookCtx, b.Params, cmd, args)
				if err != nil {
					return fmt.Errorf("error in PreValidateFuncCtx: %w", err)
				}
			}

			syncMirrors(ctx)

			if err = validate(ctx, b.Params); err != nil {
				return err
			}

			// Sync mirrors again after validation to copy converted values (e.g., *url.URL from string)
			syncMirrors(ctx)

			// if b.params or any inner struct implements CfgStructPreExecute, call it
			err = traverse(ctx, b.Params, nil, func(innerParams any) error {
				if preExecute, ok := innerParams.(CfgStructPreExecute); ok {
					err := preExecute.PreExecute()
					if err != nil {
						return fmt.Errorf("error in PreExecute: %w", err)
					}
				}
				// context-aware interface
				if preExecuteCtx, ok := innerParams.(CfgStructPreExecuteCtx); ok {
					hookCtx := newHookContext(ctx)
					err := preExecuteCtx.PreExecuteCtx(hookCtx)
					if err != nil {
						return fmt.Errorf("error in PreExecuteCtx: %w", err)
					}
				}
				return nil
			})
			if err != nil {
				return err
			}

			// if we have a custom pre-execute function, call it
			if b.PreExecuteFunc != nil {
				err := b.PreExecuteFunc(b.Params, cmd, args)
				if err != nil {
					return fmt.Errorf("error in PreExecuteFunc: %w", err)
				}
			}

			// if we have a context-aware pre-execute function, call it
			if b.PreExecuteFuncCtx != nil {
				hookCtx := newHookContext(ctx)
				err := b.PreExecuteFuncCtx(hookCtx, b.Params, cmd, args)
				if err != nil {
					return fmt.Errorf("error in PreExecuteFuncCtx: %w", err)
				}
			}

		}
		return nil
	}

	return cmd, ctx, nil
}

// validateRunFuncs checks that at most one run function is set and returns an error if more than one is configured.
func (b Cmd) validateRunFuncs() error {
	runFuncCount := 0
	if b.RunFunc != nil {
		runFuncCount++
	}
	if b.RunFuncCtx != nil {
		runFuncCount++
	}
	if b.RunFuncE != nil {
		runFuncCount++
	}
	if b.RunFuncCtxE != nil {
		runFuncCount++
	}
	if runFuncCount > 1 {
		return fmt.Errorf("cannot set multiple run functions (RunFunc, RunFuncCtx, RunFuncE, RunFuncCtxE) - use only one")
	}
	return nil
}

// toCobraImpl converts a Cmd to a cobra.Command.
// Always uses cmd.RunE internally so errors flow back through Execute() to runImpl().
func (b Cmd) toCobraImpl() *cobra.Command {
	if err := b.validateRunFuncs(); err != nil {
		panic(err)
	}
	cmd, ctx, err := b.toCobraBase()
	if err != nil {
		panic(err)
	}

	// Always use RunE so errors flow back through Execute() to runImpl()
	// This ensures consistent error handling (UserInputError -> exit(1), others -> panic)
	if b.RunFunc != nil {
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			b.RunFunc(cmd, args)
			return nil
		}
	} else if b.RunFuncCtx != nil {
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			hookCtx := newHookContext(ctx)
			b.RunFuncCtx(hookCtx, cmd, args)
			return nil
		}
	} else if b.RunFuncE != nil {
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			err := b.RunFuncE(cmd, args)
			if err != nil && !IsUserInputError(err) {
				return &runFuncError{Err: err}
			}
			return err
		}
	} else if b.RunFuncCtxE != nil {
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			hookCtx := newHookContext(ctx)
			err := b.RunFuncCtxE(hookCtx, cmd, args)
			if err != nil && !IsUserInputError(err) {
				return &runFuncError{Err: err}
			}
			return err
		}
	} else if len(b.SubCmds) > 0 {
		// No RunFunc but has subcommands. Make the command runnable so cobra
		// rejects unknown subcommands with an error instead of silently showing help.
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		}
		if b.Args == nil {
			cmd.Args = wrapArgsValidator(func(cmd *cobra.Command, args []string) error {
				if len(args) > 0 {
					return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
				}
				return nil
			})
		}
	}

	return cmd
}

// toCobraImplE converts a Cmd to a cobra.Command using cmd.RunE (returns error).
// Returns (*cobra.Command, error) to propagate setup errors.
func (b Cmd) toCobraImplE() (*cobra.Command, error) {
	if err := b.validateRunFuncs(); err != nil {
		panic(err) // API misuse - should be caught during development
	}
	cmd, ctx, err := b.toCobraBase()
	if err != nil {
		return nil, err
	}

	// Set the RunE function based on which variant is configured
	if b.RunFuncE != nil {
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			return b.RunFuncE(cmd, args)
		}
	} else if b.RunFuncCtxE != nil {
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			hookCtx := newHookContext(ctx)
			return b.RunFuncCtxE(hookCtx, cmd, args)
		}
	} else if b.RunFunc != nil {
		// Wrap non-E variant to return nil (no error)
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			b.RunFunc(cmd, args)
			return nil
		}
	} else if b.RunFuncCtx != nil {
		// Wrap non-E variant to return nil (no error)
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			hookCtx := newHookContext(ctx)
			b.RunFuncCtx(hookCtx, cmd, args)
			return nil
		}
	} else if len(b.SubCmds) > 0 {
		// No RunFunc but has subcommands. Make the command runnable so cobra
		// rejects unknown subcommands with an error instead of silently showing help.
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		}
		if b.Args == nil {
			cmd.Args = wrapArgsValidator(func(cmd *cobra.Command, args []string) error {
				if len(args) > 0 {
					return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
				}
				return nil
			})
		}
	}

	return cmd, nil
}

// resolveFieldValue walks from the root params struct to the field at the given
// declared-index path and returns an addressable reflect.Value for that field.
// Returns (zero, false) if any intermediate pointer-to-struct is nil (which can
// happen after cleanupPreallocatedPtrs nils an unused substruct).
func (ctx *processingContext) resolveFieldValue(path fieldPath) (reflect.Value, bool) {
	if ctx.rootStructPtr == nil {
		return reflect.Value{}, false
	}
	v := reflect.ValueOf(ctx.rootStructPtr).Elem()
	idx := splitPath(path)
	for _, i := range idx {
		for v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return reflect.Value{}, false
			}
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct {
			return reflect.Value{}, false
		}
		if i >= v.NumField() {
			return reflect.Value{}, false
		}
		v = v.Field(i)
	}
	// For the final value, if it is a pointer-to-struct we do NOT auto-deref:
	// mirror types may themselves be pointers (e.g., *url.URL), and pointer-field
	// semantics are handled by syncPointerField.
	return v, true
}

// rebuildAddrToPath walks the current live struct tree (starting from rootStructPtr)
// and rebuilds the reverse address index. Called on demand when the cache has been
// invalidated (e.g., after a subtree removal).
func (ctx *processingContext) rebuildAddrToPath() {
	ctx.cacheRebuildCount++
	ctx.addrToPath = map[unsafe.Pointer]fieldPath{}
	for _, p := range ctx.pathOrder {
		v, ok := ctx.resolveFieldValue(p)
		if !ok || !v.CanAddr() {
			continue
		}
		ctx.addrToPath[v.Addr().UnsafePointer()] = p
	}
}

// findPathByAddr walks the live struct tree and searches for a leaf whose address
// matches. Used as a fallback when the reverse index is stale (e.g., a substruct
// was reassigned by user code).
func (ctx *processingContext) findPathByAddr(addr unsafe.Pointer) (fieldPath, bool) {
	ctx.walkFallbackCount++
	for _, p := range ctx.pathOrder {
		v, ok := ctx.resolveFieldValue(p)
		if !ok || !v.CanAddr() {
			continue
		}
		if v.Addr().UnsafePointer() == addr {
			return p, true
		}
	}
	return "", false
}

// reinterpretAs returns the raw field value viewed as the target type, without
// copying. This supports type aliases (e.g., a MyString field seen as string) by
// taking the field's address and reconstructing a Value at the same memory with
// the target type. If the types already match, the input is returned unchanged.
//
// This is the one legitimate use of unsafe in the sync path — cobra's flag system
// only knows underlying types, so mirror storage and mirror → raw writes must go
// through the underlying view of the memory.
func reinterpretAs(rawFieldVal reflect.Value, target reflect.Type) reflect.Value {
	if rawFieldVal.Type() == target {
		return rawFieldVal
	}
	return reflect.NewAt(target, rawFieldVal.Addr().UnsafePointer()).Elem()
}

func syncMirrors(ctx *processingContext) {
	// 1. First, copy non-zero values from the raw fields -> mirrors as injected values.
	// 2. Then copy back cli & env set values to the raw fields

	for _, p := range ctx.pathOrder {
		mirror, ok := ctx.mirrorByPath[p]
		if !ok {
			continue
		}
		rawFieldVal, resolved := ctx.resolveFieldValue(p)
		if !resolved {
			// The substruct housing this field has been nil'd out by cleanup — skip.
			continue
		}

		// Check if this is a pointer field (e.g., *string, *int)
		pm, isParamMeta := mirror.(*paramMeta)
		if isParamMeta && pm.isPointer {
			// Reinterpret as pointer-to-underlying-type so type aliases round-trip
			ptrView := reinterpretAs(rawFieldVal, reflect.PointerTo(mirror.GetType()))
			syncPointerField(ptrView, mirror)
			continue
		}

		// Reinterpret as the underlying type (matters for type aliases)
		underlying := reinterpretAs(rawFieldVal, mirror.GetType())

		if !mirror.wasSetOnCli() && !mirror.wasSetByEnv() && !underlying.IsZero() {
			mirror.injectValuePtr(underlying.Addr().Interface())
		}

		if mirror.wasSetOnCli() || mirror.wasSetByEnv() || (mirror.HasValue() && underlying.IsZero()) {
			mirrorValue := reflect.ValueOf(mirror.valuePtrF()).Elem()

			// Skip if types don't match (e.g., string vs *url.URL before conversion)
			// This allows syncing to work before and after validation/conversion
			if !mirrorValue.Type().AssignableTo(underlying.Type()) {
				continue
			}

			// Make sure the destination is settable
			if underlying.CanSet() {
				underlying.Set(mirrorValue)
			} else {
				panic(fmt.Errorf("could not set value for parameter %s", mirror.GetName()))
			}
		}
	}
}

// syncPointerField handles bidirectional sync for pointer fields like *string, *int.
// rawFieldVal is the *string field value itself (an addressable reflect.Value of
// pointer kind). The mirror stores the element type (string).
func syncPointerField(rawFieldVal reflect.Value, mirror Param) {
	// Raw → Mirror: if the user's pointer is non-nil, inject the pointed-to value
	if !mirror.wasSetOnCli() && !mirror.wasSetByEnv() && !rawFieldVal.IsNil() {
		// rawFieldVal.Interface() is *string — same type cobra uses
		mirror.injectValuePtr(rawFieldVal.Interface())
	}

	// Mirror → Raw: if mirror has a value, set the pointer field
	if mirror.wasSetOnCli() || mirror.wasSetByEnv() || (mirror.HasValue() && rawFieldVal.IsNil()) {
		valPtr := mirror.valuePtrF()
		if valPtr != nil {
			mirrorVal := reflect.ValueOf(valPtr) // *string from cobra
			// Skip if types don't match (e.g., string vs *url.URL before conversion)
			if mirrorVal.Type().AssignableTo(rawFieldVal.Type()) {
				if rawFieldVal.CanSet() {
					rawFieldVal.Set(mirrorVal)
				}
			}
		}
	}
}

func runImpl(cmd *cobra.Command, handler resultHandler) {

	if handler.Panic != nil {
		defer func() {
			if r := recover(); r != nil {
				handler.Panic(r)
			}
		}()
	}

	err := Execute(cmd)
	if err != nil {
		if handler.Failure != nil {
			handler.Failure(err)
		} else {
			// Errors from RunFuncE/RunFuncCtxE that aren't UserInputError are
			// programming errors — panic so developers notice.
			var rfe *runFuncError
			if errors.As(err, &rfe) {
				panic(rfe.Unwrap())
			}
			// Everything else: Execute() already printed usage + error.
			osExit(1)
			return // osExit may be mocked in tests, so we need to return explicitly
		}
	} else {
		if handler.Success != nil {
			handler.Success()
		}
	}
}

func isSupportedType(t reflect.Type) bool {
	// Exact type match (time.Time, time.Duration, net.IP, *url.URL)
	if _, ok := exactTypeHandlers[t]; ok {
		return true
	}
	// Kind-based match (string, int, bool, float, etc.)
	if _, ok := kindHandlers[t.Kind()]; ok {
		return true
	}
	// Map types — all map[string]V types are supported (native pflag or JSON fallback)
	if t.Kind() == reflect.Map && t.Key().Kind() == reflect.String {
		return true
	}
	// Slice types — all slices are supported (native pflag for basic elements, JSON fallback for complex)
	if t.Kind() == reflect.Slice {
		return true
	}
	// Pointer-to-supported-type (e.g., *string, *int, *bool)
	if t.Kind() == reflect.Pointer {
		return isSupportedType(t.Elem())
	}
	return false
}

// normalizeType converts type aliases to their base types for cobra compatibility.
// For example, `type MyString string` returns reflect.TypeOf("") (string).
// Special types (time.Time, time.Duration, net.IP, *url.URL) are returned as-is.
func normalizeType(t reflect.Type) reflect.Type {
	// Exact-match handlers have their own baseType (special types stay as-is)
	if handler, ok := exactTypeHandlers[t]; ok {
		return handler.baseType
	}

	// For slices, normalize via slice handlers
	if t.Kind() == reflect.Slice {
		elem := t.Elem()
		// Special slice element types stay as-is
		if _, ok := sliceExactTypeHandlers[elem]; ok {
			return t
		}
		// Normalize basic slice element types
		normElem := normalizeType(elem)
		if normElem != elem {
			return reflect.SliceOf(normElem)
		}
		return t
	}

	// Kind-based handlers provide the baseType for basic types + aliases
	if handler, ok := kindHandlers[t.Kind()]; ok {
		return handler.baseType
	}

	return t
}

func newParam(field *reflect.StructField, t reflect.Type) Param {
	// Determine if this is a pointer-to-value field (e.g., *string, *int)
	// Note: *url.URL is NOT treated as a pointer field — it's a specific supported type
	isPtr := t.Kind() == reflect.Pointer && t != urlPtrType
	valueType := t
	if isPtr {
		valueType = t.Elem()
	}

	// Normalize type aliases to their base types for cobra compatibility.
	// e.g., `type MyString string` → store as string, since cobra's StringP returns *string.
	// This matches the old required[T] behavior where newParam always created required[string]{},
	// required[int]{}, etc. regardless of whether the field was a type alias.
	valueType = normalizeType(valueType)

	// Pointer, map, and nested slice fields default to optional (nil = not set)
	isRequired := !cfg.defaultOptional
	if isPtr || valueType.Kind() == reflect.Map {
		isRequired = false
	}
	// Nested slices ([][]T) default to optional — flat slices keep the global default
	if valueType.Kind() == reflect.Slice && valueType.Elem().Kind() == reflect.Slice {
		isRequired = false
	}

	if requiredTag, ok := field.Tag.Lookup("required"); ok {
		switch requiredTag {
		case "true":
			isRequired = true
		case "false":
			isRequired = false
		default:
			panic(fmt.Errorf("invalid value for field %s's required tag: %s", field.Name, requiredTag))
		}
	}
	if requiredTag, ok := field.Tag.Lookup("req"); ok {
		switch requiredTag {
		case "true":
			isRequired = true
		case "false":
			isRequired = false
		default:
			panic(fmt.Errorf("invalid value for field %s's required tag: %s", field.Name, requiredTag))
		}
	}
	if optionalTag, ok := field.Tag.Lookup("optional"); ok {
		switch optionalTag {
		case "true":
			isRequired = false
		case "false":
			isRequired = true
		default:
			panic(fmt.Errorf("invalid value for field %s's optional tag: %s", field.Name, optionalTag))
		}
	}
	if optionalTag, ok := field.Tag.Lookup("opt"); ok {
		switch optionalTag {
		case "true":
			isRequired = false
		case "false":
			isRequired = true
		default:
			panic(fmt.Errorf("invalid value for field %s's optional tag: %s", field.Name, optionalTag))
		}
	}

	return &paramMeta{
		fieldType:       valueType,
		isPointer:       isPtr,
		defaultRequired: isRequired,
	}
}

var timeType = reflect.TypeOf(time.Time{})
var durationType = reflect.TypeOf(time.Duration(0))
var ipType = reflect.TypeOf(net.IP{})
var urlPtrType = reflect.TypeOf((*url.URL)(nil))
