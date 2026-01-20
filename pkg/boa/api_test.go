package boa

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestSetsDefaultName(t *testing.T) {
	var params = struct {
		Flag1 Required[string]
		Flag2 Required[int]
	}{
		Flag1: Required[string]{Short: "s"},
		Flag2: Required[int]{Short: "i"},
	}

	Cmd{
		Use:    "test",
		Short:  "test",
		Params: &params,
	}.ToCobra()

	if params.Flag1.Name != "flag1" {
		t.Errorf("Name of flag1 is not set to default")
	}

	if params.Flag2.Name != "flag2" {
		t.Errorf("Name of flag2 is not set to default")
	}
}

func TestValidFlagStruct(t *testing.T) {

	var params = struct {
		Flag1 Required[string]
		Flag2 Required[int]
	}{
		Flag1: Required[string]{Name: "s"},
		Flag2: Required[int]{Name: "i"},
	}

	Cmd{
		Use:    "test",
		Short:  "test",
		Params: &params,
	}.ToCobra()
}

func TestMixedRequiredAndOptional(t *testing.T) {

	var params = struct {
		User       Required[string]
		Org        Required[string]
		HelloParam Optional[int]
	}{
		User:       Required[string]{Name: "user", Short: "u", Env: "SOMETHING_USERNAME", Descr: "The user with api access to something"},
		Org:        Required[string]{Name: "org", Short: "o", Env: "SOMETHING_ORG", Descr: "The something organisation"},
		HelloParam: Optional[int]{Name: "hello-param", Short: "x", Env: "HELLO_PARAM", Descr: "A hello param", Default: Default(42)},
	}

	Cmd{
		Use:    "test",
		Short:  "test",
		Params: &params,
	}.ToCobra()
}

func TestDisallowHAsShort(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	var params = struct {
		HelloParam Optional[int]
	}{
		HelloParam: Optional[int]{Name: "hello-param", Short: "h", Env: "HELLO_PARAM", Descr: "A hello param", Default: Default(42)},
	}

	Cmd{
		Use:    "test",
		Short:  "test",
		Params: &params,
	}.ToCobra()
}

func TestReflectOfStructs(t *testing.T) {
	var params = struct {
		User       Required[string]
		Org        Required[string]
		HelloParam Optional[int]
	}{
		User:       Required[string]{Name: "user", Short: "u", Env: "SOMETHING_USERNAME", Descr: "The user with api access to something"},
		Org:        Required[string]{Name: "org", Short: "o", Env: "SOMETHING_ORG", Descr: "The something organisation"},
		HelloParam: Optional[int]{Name: "hello-param", Short: "x", Env: "HELLO_PARAM", Descr: "A hello param", Default: Default(42)},
	}

	var structAsAny any = params
	var structPtrAsAny any = &params

	fmt.Printf("structAsAny: %v\n", reflect.TypeOf(structAsAny).Kind())
	fmt.Printf("structPtrAsAny: %v\n", reflect.TypeOf(structPtrAsAny).Kind())
}

func TestDoubleDefault(t *testing.T) {
	var params = struct {
		User Required[string] `default:"defaultUser"`
	}{
		User: Required[string]{Default: Default("123")},
	}

	osArgsBefore := os.Args
	os.Args = []string{"test"}
	defer func() {
		os.Args = osArgsBefore
	}()

	err := Cmd{Params: &params, ParamEnrich: ParamEnricherName}.Validate()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if params.User.Value() != "123" {
		t.Errorf("Expected default value to be '123', got: %s", params.User.Value())
	}
}

func TestIgnoreBoaIgnored(t *testing.T) {
	var params = struct {
		User     Required[string] `default:"defaultUser"`
		UserIgn1 Required[string] `boa:"-"`
		UserIgn2 Required[string] `boa:"ignore"`
		UserIgn3 Required[string] `boa:"ignored"`
	}{
		User: Required[string]{Default: Default("123")},
	}

	osArgsBefore := os.Args
	os.Args = []string{"test"}
	defer func() {
		os.Args = osArgsBefore
	}()

	err := Cmd{Params: &params, ParamEnrich: ParamEnricherName}.Validate()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if params.User.Value() != "123" {
		t.Errorf("Expected default value to be '123', got: %s", params.User.Value())
	}
}

func TestUseHInsteadOFHelp(t *testing.T) {

	var params = struct {
		User Required[string] `short:"h"`
	}{
		User: Required[string]{Default: Default("123")},
	}

	ran := false
	Cmd{
		Use:    "test",
		Short:  "test",
		Params: &params,
		InitFunc: func(params any, cmd *cobra.Command) error {
			cmd.Flags().BoolP("help", "", false, "help for test")
			return nil
		},
		RunFunc: func(cmd *cobra.Command, args []string) {
			ran = true
		},
	}.RunArgs([]string{"-h", "user-x"})

	if !ran {
		t.Errorf("Expected command to run")
	}

	if params.User.Value() != "user-x" {
		t.Errorf("Expected user to be 'user-x', got: %s", params.User.Value())
	}
}

func TestUseHInsteadOFHelpIncorrectUse(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	var params = struct {
		User Required[string] `short:"h"`
	}{
		User: Required[string]{Default: Default("123")},
	}

	Cmd{
		Use:    "test",
		Short:  "test",
		Params: &params,
		RunFunc: func(cmd *cobra.Command, args []string) {
		},
	}.Run()
}

type InitTestStruct struct {
	User Required[string] `default:"defaultUser"`
}

func (i *InitTestStruct) Init() error {
	i.User.Default = Default("123")
	return nil
}

func TestInit(t *testing.T) {
	var params = InitTestStruct{}

	osArgsBefore := os.Args
	os.Args = []string{"test"}
	defer func() {
		os.Args = osArgsBefore
	}()

	err := Cmd{Params: &params, ParamEnrich: ParamEnricherName}.Validate()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if params.User.Value() != "123" {
		t.Errorf("Expected default value to be '123', got: %s", params.User.Value())
	}
}

type PreExecuteTestStruct struct {
	User Required[string] `default:"defaultUser"`
}

var errExpected = fmt.Errorf("<expected error>")

func (i *PreExecuteTestStruct) PreExecute() error {
	if i.User.Value() != "defaultUser" {
		return fmt.Errorf("user value is not defaultUser")
	}
	return errExpected
}

func TestPreExecute(t *testing.T) {
	var params = PreExecuteTestStruct{}

	osArgsBefore := os.Args
	os.Args = []string{"test"}
	defer func() {
		os.Args = osArgsBefore
	}()

	err := Cmd{Params: &params, ParamEnrich: ParamEnricherName}.Validate()
	if err != nil {
		if !strings.Contains(err.Error(), errExpected.Error()) {
			t.Errorf("Expected error to contain: %s, got: %v", errExpected.Error(), err)
		}
	} else {
		t.Errorf("Expected error, got: nil")
	}
}

type CustomValidatorTestStruct struct {
	Flag2 Required[int]
}

func (s *CustomValidatorTestStruct) Init() error {
	s.Flag2.CustomValidator = func(s int) error {
		if s < 0 {
			return fmt.Errorf("value must be greater than 0")
		}
		return nil
	}
	return nil
}

func TestCustomValidator(t *testing.T) {

	err := Cmd{Params: &CustomValidatorTestStruct{}, ParamEnrich: ParamEnricherName, RawArgs: []string{"--flag2", "-1"}}.Validate()
	if err == nil {
		t.Errorf("Expected error, got: nil")
	} else {
		if !strings.Contains(err.Error(), "value must be greater than 0") {
			t.Errorf("Expected error to contain: %s, got: %v", "value must be greater than 0", err)
		}
	}

	err = Cmd{Params: &CustomValidatorTestStruct{}, ParamEnrich: ParamEnricherName, RawArgs: []string{"--flag2", "0"}}.Validate()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	err = Cmd{
		Params:      &CustomValidatorTestStruct{Flag2: Required[int]{Default: Default(42)}},
		ParamEnrich: ParamEnricherName, RawArgs: []string{},
	}.Validate()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	err = Cmd{
		Params:      &CustomValidatorTestStruct{Flag2: Required[int]{Default: Default(-42)}},
		ParamEnrich: ParamEnricherName, RawArgs: []string{},
	}.Validate()
	if err == nil {
		t.Errorf("Expected error, got: nil")
	} else {
		if !strings.Contains(err.Error(), "value must be greater than 0") {
			t.Errorf("Expected error to contain: %s, got: %v", "value must be greater than 0", err)
		}
	}
}

func TestAlternatives(t *testing.T) {
	type Conf struct {
		MyEnum Required[string] `short:"e" default:"e1" alts:"e1,e2,e3"`
	}

	if err := NewCmdT[Conf]("test").WithRawArgs([]string{}).Validate(); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if err := NewCmdT[Conf]("test").WithRawArgs([]string{"-e", "e2"}).Validate(); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if err := NewCmdT[Conf]("test").WithRawArgs([]string{"-e", "e3"}).Validate(); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if err := NewCmdT[Conf]("test").WithRawArgs([]string{"-e", "e4"}).Validate(); err == nil {
		t.Errorf("Expected error, got: nil")
	}
}

func TestProgrammaticAlternativesMustBeEnforced(t *testing.T) {
	type Conf struct {
		MyEnum Required[string] `short:"e" default:"e1"`
	}
	preValidateFunc := func(params *Conf, cmd *cobra.Command, args []string) {
		params.MyEnum.SetAlternatives([]string{"e1", "e2", "e3"})
	}

	if err := NewCmdT[Conf]("test").WithRawArgs([]string{}).WithPreValidateFunc(preValidateFunc).Validate(); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if err := NewCmdT[Conf]("test").WithRawArgs([]string{"-e", "e2"}).WithPreValidateFunc(preValidateFunc).Validate(); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if err := NewCmdT[Conf]("test").WithRawArgs([]string{"-e", "e3"}).WithPreValidateFunc(preValidateFunc).Validate(); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if err := NewCmdT[Conf]("test").WithRawArgs([]string{"-e", "e4"}).WithPreValidateFunc(preValidateFunc).Validate(); err == nil {
		t.Errorf("Expected error, got: nil")
	}
}

func TestNonStrictAlternativesAllowAnyValue(t *testing.T) {
	type Conf struct {
		MyEnum Required[string] `short:"e" default:"e1" alts:"e1,e2,e3" strict:"false"`
	}

	// Valid alternative should work
	if err := NewCmdT[Conf]("test").WithRawArgs([]string{"-e", "e2"}).Validate(); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Non-listed value should also be accepted when strict is false
	if err := NewCmdT[Conf]("test").WithRawArgs([]string{"-e", "custom-value"}).Validate(); err != nil {
		t.Errorf("Expected no error for non-strict alts, got: %v", err)
	}
}

func TestStrictAlternativesEnforceValidation(t *testing.T) {
	type Conf struct {
		MyEnum Required[string] `short:"e" default:"e1" alts:"e1,e2,e3" strict:"true"`
	}

	// Valid alternative should work
	if err := NewCmdT[Conf]("test").WithRawArgs([]string{"-e", "e2"}).Validate(); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Non-listed value should be rejected when strict is true
	if err := NewCmdT[Conf]("test").WithRawArgs([]string{"-e", "e4"}).Validate(); err == nil {
		t.Errorf("Expected error for strict alts, got: nil")
	}
}

func TestStrictAlternativesDefaultBehavior(t *testing.T) {
	// Without strict tag, alts should be enforced (default behavior)
	type Conf struct {
		MyEnum Required[string] `short:"e" default:"e1" alts:"e1,e2,e3"`
	}

	if err := NewCmdT[Conf]("test").WithRawArgs([]string{"-e", "e4"}).Validate(); err == nil {
		t.Errorf("Expected error when strict is not specified (default should enforce), got: nil")
	}
}

func TestProgrammaticStrictAlts(t *testing.T) {
	type Conf struct {
		MyEnum Required[string] `short:"e" default:"e1"`
	}
	preValidateFunc := func(params *Conf, cmd *cobra.Command, args []string) {
		params.MyEnum.SetAlternatives([]string{"e1", "e2", "e3"})
		params.MyEnum.SetStrictAlts(false)
	}

	// With strict set to false programmatically, any value should be accepted
	if err := NewCmdT[Conf]("test").WithRawArgs([]string{"-e", "custom-value"}).WithPreValidateFunc(preValidateFunc).Validate(); err != nil {
		t.Errorf("Expected no error for programmatic non-strict alts, got: %v", err)
	}
}

// TestUserInputErrorType verifies that user input validation errors are wrapped as UserInputError
func TestUserInputErrorType(t *testing.T) {
	type Params struct {
		Name string `short:"n" required:"true"`
	}

	// Test missing required param returns UserInputError
	// Use ParamEnricherName to avoid env var interference (e.g., NAME env var)
	err := NewCmdT[Params]("test").WithParamEnrich(ParamEnricherName).WithRawArgs([]string{}).Validate()
	if err == nil {
		t.Fatal("Expected error for missing required param")
	}
	if !IsUserInputError(err) {
		t.Errorf("Expected UserInputError for missing required param, got: %T", err)
	}
	if !strings.Contains(err.Error(), "missing required param") {
		t.Errorf("Expected error message to contain 'missing required param', got: %s", err.Error())
	}
}

func TestUserInputErrorInvalidAlternatives(t *testing.T) {
	type Params struct {
		Mode string `short:"m" default:"a" alts:"a,b,c"`
	}

	// Test invalid alternative returns UserInputError
	err := NewCmdT[Params]("test").WithRawArgs([]string{"-m", "invalid"}).Validate()
	if err == nil {
		t.Fatal("Expected error for invalid alternative")
	}
	if !IsUserInputError(err) {
		t.Errorf("Expected UserInputError for invalid alternative, got: %T", err)
	}
	if !strings.Contains(err.Error(), "not in the list of allowed values") {
		t.Errorf("Expected error message to contain 'not in the list of allowed values', got: %s", err.Error())
	}
}

func TestUserInputErrorCustomValidator(t *testing.T) {
	type Params struct {
		Port int `short:"p" required:"true"`
	}

	// Test custom validator error returns UserInputError
	err := NewCmdT[Params]("test").
		WithRawArgs([]string{"-p", "-1"}).
		WithInitFuncCtx(func(ctx *HookContext, p *Params, cmd *cobra.Command) error {
			ctx.GetParam(&p.Port).SetCustomValidator(func(v any) error {
				if v.(int) < 0 {
					return fmt.Errorf("port must be non-negative")
				}
				return nil
			})
			return nil
		}).
		Validate()
	if err == nil {
		t.Fatal("Expected error for invalid port")
	}
	if !IsUserInputError(err) {
		t.Errorf("Expected UserInputError for custom validator failure, got: %T", err)
	}
	if !strings.Contains(err.Error(), "port must be non-negative") {
		t.Errorf("Expected error message to contain 'port must be non-negative', got: %s", err.Error())
	}
}

func TestUserInputErrorInvalidEnvValue(t *testing.T) {
	type Params struct {
		Port int `short:"p" env:"TEST_PORT_INVALID" required:"true"`
	}

	// Set an invalid env value
	os.Setenv("TEST_PORT_INVALID", "not-a-number")
	defer os.Unsetenv("TEST_PORT_INVALID")

	err := NewCmdT[Params]("test").WithRawArgs([]string{}).Validate()
	if err == nil {
		t.Fatal("Expected error for invalid env value")
	}
	if !IsUserInputError(err) {
		t.Errorf("Expected UserInputError for invalid env value, got: %T", err)
	}
}

func TestUserInputErrorMissingPositionalArg(t *testing.T) {
	type Params struct {
		File string `positional:"true" required:"true"`
	}

	err := NewCmdT[Params]("test").WithRawArgs([]string{}).Validate()
	if err == nil {
		t.Fatal("Expected error for missing positional arg")
	}
	if !IsUserInputError(err) {
		t.Errorf("Expected UserInputError for missing positional arg, got: %T", err)
	}
	// Cobra returns its own error message for missing positional args
	if !strings.Contains(err.Error(), "accepts between") && !strings.Contains(err.Error(), "missing positional arg") {
		t.Errorf("Expected error message about missing positional args, got: %s", err.Error())
	}
}

func TestUserInputErrorUnwrap(t *testing.T) {
	// Verify that UserInputError properly implements error unwrapping
	innerErr := fmt.Errorf("inner error")
	uie := &UserInputError{Err: innerErr}

	if uie.Error() != "inner error" {
		t.Errorf("Expected error message 'inner error', got: %s", uie.Error())
	}
	if uie.Unwrap() != innerErr {
		t.Errorf("Expected Unwrap to return inner error")
	}
}

func TestIsUserInputError(t *testing.T) {
	// Test with direct UserInputError
	uie := &UserInputError{Err: fmt.Errorf("test error")}
	if !IsUserInputError(uie) {
		t.Error("Expected IsUserInputError to return true for UserInputError")
	}

	// Test with wrapped UserInputError
	wrapped := fmt.Errorf("wrapped: %w", uie)
	if !IsUserInputError(wrapped) {
		t.Error("Expected IsUserInputError to return true for wrapped UserInputError")
	}

	// Test with regular error
	regularErr := fmt.Errorf("regular error")
	if IsUserInputError(regularErr) {
		t.Error("Expected IsUserInputError to return false for regular error")
	}

	// Test with nil
	if IsUserInputError(nil) {
		t.Error("Expected IsUserInputError to return false for nil")
	}
}

// TestInvalidFlagValueFromCobra tests that invalid flag values
// from Cobra/pflag are detected as UserInputError
func TestInvalidFlagValueFromCobra(t *testing.T) {
	type Params struct {
		Port int `short:"p" required:"true"`
	}

	// Invalid integer value - this error comes from pflag
	err := NewCmdT[Params]("test").WithRawArgs([]string{"-p", "not-a-number"}).Validate()
	if err == nil {
		t.Fatal("Expected error for invalid integer flag value")
	}
	// pflag returns InvalidValueError which IsUserInputError should detect
	if !IsUserInputError(err) {
		t.Errorf("Expected IsUserInputError to return true for pflag InvalidValueError, got: %T - %v", err, err)
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("Expected error message to mention invalid value, got: %s", err.Error())
	}
}

// TestUnknownFlagFromCobra tests that unknown flags from Cobra/pflag are detected as UserInputError
func TestUnknownFlagFromCobra(t *testing.T) {
	type Params struct {
		Name string `short:"n" default:"test"`
	}

	err := NewCmdT[Params]("test").WithRawArgs([]string{"--unknown-flag"}).Validate()
	if err == nil {
		t.Fatal("Expected error for unknown flag")
	}
	// pflag returns NotExistError which IsUserInputError should detect
	if !IsUserInputError(err) {
		t.Errorf("Expected IsUserInputError to return true for pflag NotExistError, got: %T - %v", err, err)
	}
	if !strings.Contains(err.Error(), "unknown flag") {
		t.Errorf("Expected error message to mention unknown flag, got: %s", err.Error())
	}
}

// TestNewUserInputErrorInHook tests that users can return UserInputError from hooks
func TestNewUserInputErrorInHook(t *testing.T) {
	type Params struct {
		StartPort int `short:"s" required:"true"`
		EndPort   int `short:"e" required:"true"`
	}

	// Use PreValidateFunc to do cross-field validation
	err := NewCmdT[Params]("test").
		WithRawArgs([]string{"-s", "8080", "-e", "80"}).
		WithPreValidateFuncE(func(p *Params, cmd *cobra.Command, args []string) error {
			if p.StartPort > p.EndPort {
				return NewUserInputErrorf("start port (%d) must be less than end port (%d)", p.StartPort, p.EndPort)
			}
			return nil
		}).
		Validate()

	if err == nil {
		t.Fatal("Expected error for invalid port range")
	}
	if !IsUserInputError(err) {
		t.Errorf("Expected UserInputError from PreValidateFunc, got: %T", err)
	}
	if !strings.Contains(err.Error(), "start port") {
		t.Errorf("Expected error message about start port, got: %s", err.Error())
	}
}

// TestErrorHandlingTable tests all combinations from the error handling documentation table:
//
//	| Method         | Setup Errors | User Input Errors | Hook Errors | Runtime Errors |
//	|----------------|--------------|-------------------|-------------|----------------|
//	| Run()          | Panic        | Exit(1)           | Exit(1)     | Panic          |
//	| RunE()         | Panic        | Return            | Return      | Return         |
//	| RunArgs(args)  | Panic        | Exit(1)           | Exit(1)     | Panic          |
//	| RunArgsE(args) | Panic        | Return            | Return      | Return         |
func TestErrorHandlingTable(t *testing.T) {
	type expectedBehavior int
	const (
		expectPanic expectedBehavior = iota
		expectExit1
		expectReturn
	)

	type outputExpectation struct {
		hasErrorPrefix bool // "Error:" prefix in stderr
		hasUsage       bool // "Usage:" in stderr
		noOutput       bool // no stderr output at all
	}

	tests := []struct {
		name       string
		method     string // "Run", "RunE", "RunArgs", "RunArgsE"
		errorType  string // "UserInput", "Hook", "Runtime"
		behavior   expectedBehavior
		output     outputExpectation
		errorMatch string // substring to match in error (for Return behavior)
	}{
		// Run() behavior
		{"Run/UserInput", "Run", "UserInput", expectExit1, outputExpectation{hasErrorPrefix: true, hasUsage: true}, ""},
		{"Run/Hook", "Run", "Hook", expectExit1, outputExpectation{hasErrorPrefix: true, hasUsage: true}, ""},
		{"Run/Runtime", "Run", "Runtime", expectPanic, outputExpectation{}, "runtime error"},

		// RunE() behavior
		{"RunE/UserInput", "RunE", "UserInput", expectReturn, outputExpectation{noOutput: true}, "missing required param"},
		{"RunE/Hook", "RunE", "Hook", expectReturn, outputExpectation{noOutput: true}, "hook failed"},
		{"RunE/Runtime", "RunE", "Runtime", expectReturn, outputExpectation{noOutput: true}, "runtime error"},

		// RunArgs() behavior
		{"RunArgs/UserInput", "RunArgs", "UserInput", expectExit1, outputExpectation{hasErrorPrefix: true, hasUsage: true}, ""},
		{"RunArgs/Hook", "RunArgs", "Hook", expectExit1, outputExpectation{hasErrorPrefix: true, hasUsage: true}, ""},
		{"RunArgs/Runtime", "RunArgs", "Runtime", expectPanic, outputExpectation{}, "runtime error"},

		// RunArgsE() behavior
		{"RunArgsE/UserInput", "RunArgsE", "UserInput", expectReturn, outputExpectation{noOutput: true}, "missing required param"},
		{"RunArgsE/Hook", "RunArgsE", "Hook", expectReturn, outputExpectation{noOutput: true}, "hook failed"},
		{"RunArgsE/Runtime", "RunArgsE", "Runtime", expectReturn, outputExpectation{noOutput: true}, "runtime error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			type Params struct {
				Name string `short:"n" required:"true"`
			}

			// Track behavior
			var exitCalled bool
			var exitCode int
			var panicValue any
			var returnedErr error

			// Mock osExit
			oldOsExit := osExit
			osExit = func(code int) {
				exitCalled = true
				exitCode = code
			}
			defer func() { osExit = oldOsExit }()

			// Capture stderr
			oldStderr := os.Stderr
			r, w, pipeErr := os.Pipe()
			if pipeErr != nil {
				t.Fatal(pipeErr)
			}
			os.Stderr = w

			// Build command based on error type
			// Use ParamEnricherName to avoid env var interference (e.g., NAME env var)
			cmd := NewCmdT[Params]("test").WithParamEnrich(ParamEnricherName)

			switch tc.errorType {
			case "UserInput":
				// Missing required param causes user input error
				cmd = cmd.WithRunFunc(func(p *Params) {})
			case "Hook":
				// Hook returns a UserInputError
				cmd = cmd.WithPreValidateFuncE(func(p *Params, c *cobra.Command, args []string) error {
					return NewUserInputErrorf("hook failed")
				}).WithRunFunc(func(p *Params) {})
			case "Runtime":
				// RunFunc panics (for non-E methods) or returns error (for E methods)
				if strings.HasSuffix(tc.method, "E") {
					cmd = cmd.WithRunFuncE(func(p *Params) error {
						return fmt.Errorf("runtime error")
					})
				} else {
					cmd = cmd.WithRunFunc(func(p *Params) {
						panic("runtime error")
					})
				}
			}

			// Execute with panic recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicValue = r
					}
				}()

				args := []string{}
				if tc.errorType != "UserInput" {
					args = []string{"-n", "test"} // Provide required param
				}

				switch tc.method {
				case "Run":
					cmd.RunArgs(args)
				case "RunE":
					returnedErr = cmd.RunArgsE(args)
				case "RunArgs":
					cmd.RunArgs(args)
				case "RunArgsE":
					returnedErr = cmd.RunArgsE(args)
				}
			}()

			// Restore stderr and read output
			w.Close()
			os.Stderr = oldStderr
			captured := make([]byte, 8192)
			n, _ := r.Read(captured)
			r.Close()
			output := string(captured[:n])

			// Verify behavior
			switch tc.behavior {
			case expectPanic:
				if panicValue == nil {
					t.Error("Expected panic but none occurred")
				}
			case expectExit1:
				if !exitCalled {
					t.Error("Expected osExit to be called")
				}
				if exitCode != 1 {
					t.Errorf("Expected exit code 1, got %d", exitCode)
				}
			case expectReturn:
				if returnedErr == nil {
					t.Error("Expected error to be returned")
				}
				if tc.errorMatch != "" && !strings.Contains(returnedErr.Error(), tc.errorMatch) {
					t.Errorf("Expected error containing %q, got: %s", tc.errorMatch, returnedErr.Error())
				}
			}

			// Verify output expectations
			if tc.output.noOutput {
				if n > 0 {
					t.Errorf("Expected no stderr output, got: %s", output)
				}
			}
			if tc.output.hasErrorPrefix {
				if !strings.Contains(output, "Error:") {
					t.Errorf("Expected 'Error:' in stderr, got: %s", output)
				}
			}
			if tc.output.hasUsage {
				if !strings.Contains(output, "Usage:") {
					t.Errorf("Expected 'Usage:' in stderr, got: %s", output)
				}
			}
		})
	}
}
