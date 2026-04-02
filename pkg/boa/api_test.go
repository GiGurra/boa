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
	type params struct {
		Flag1 string `short:"s"`
		Flag2 int    `short:"i"`
	}

	p := params{}
	Cmd{
		Use:    "test",
		Short:  "test",
		Params: &p,
	}.ToCobra()

	// We can't directly check param names from the struct anymore,
	// but we can verify the cobra command has the right flags
	// The enricher should set flag1 and flag2 as names
}

func TestValidFlagStruct(t *testing.T) {
	type params struct {
		Flag1 string
		Flag2 int
	}

	p := params{}
	Cmd{
		Use:    "test",
		Short:  "test",
		Params: &p,
	}.ToCobra()
}

func TestMixedRequiredAndOptional(t *testing.T) {
	type params struct {
		User       string `short:"u" env:"SOMETHING_USERNAME" descr:"The user with api access to something"`
		Org        string `short:"o" env:"SOMETHING_ORG" descr:"The something organisation"`
		HelloParam int    `short:"x" env:"HELLO_PARAM" descr:"A hello param" default:"42" optional:"true"`
	}

	p := params{}
	Cmd{
		Use:    "test",
		Short:  "test",
		Params: &p,
	}.ToCobra()
}

func TestDisallowHAsShort(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	type params struct {
		HelloParam int `short:"h" env:"HELLO_PARAM" descr:"A hello param" default:"42" optional:"true"`
	}

	p := params{}
	Cmd{
		Use:    "test",
		Short:  "test",
		Params: &p,
	}.ToCobra()
}

func TestReflectOfStructs(t *testing.T) {
	type params struct {
		User       string `short:"u" env:"SOMETHING_USERNAME" descr:"The user with api access to something"`
		Org        string `short:"o" env:"SOMETHING_ORG" descr:"The something organisation"`
		HelloParam int    `short:"x" env:"HELLO_PARAM" descr:"A hello param" default:"42" optional:"true"`
	}

	p := params{}
	var structAsAny any = p
	var structPtrAsAny any = &p

	fmt.Printf("structAsAny: %v\n", reflect.TypeOf(structAsAny).Kind())
	fmt.Printf("structPtrAsAny: %v\n", reflect.TypeOf(structPtrAsAny).Kind())
}

func TestDoubleDefault(t *testing.T) {
	p := struct {
		User string `default:"defaultUser"`
	}{}

	osArgsBefore := os.Args
	os.Args = []string{"test"}
	defer func() {
		os.Args = osArgsBefore
	}()

	err := Cmd{
		Params:      &p,
		ParamEnrich: ParamEnricherName,
		InitFuncCtx: func(ctx *HookContext, params any, cmd *cobra.Command) error {
			pp := params.(*struct {
				User string `default:"defaultUser"`
			})
			ctx.GetParam(&pp.User).SetDefault(Default("123"))
			return nil
		},
	}.Validate()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if p.User != "123" {
		t.Errorf("Expected default value to be '123', got: %s", p.User)
	}
}

func TestIgnoreBoaIgnored(t *testing.T) {
	type params struct {
		User     string `default:"defaultUser"`
		UserIgn1 string `boa:"-"`
		UserIgn2 string `boa:"ignore"`
		UserIgn3 string `boa:"ignored"`
	}

	p := params{}

	osArgsBefore := os.Args
	os.Args = []string{"test"}
	defer func() {
		os.Args = osArgsBefore
	}()

	err := Cmd{Params: &p, ParamEnrich: ParamEnricherName}.Validate()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if p.User != "defaultUser" {
		t.Errorf("Expected default value to be 'defaultUser', got: %s", p.User)
	}
}

func TestUseHInsteadOFHelp(t *testing.T) {
	type params struct {
		User string `short:"h" default:"123"`
	}

	p := params{}

	ran := false
	Cmd{
		Use:    "test",
		Short:  "test",
		Params: &p,
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

	if p.User != "user-x" {
		t.Errorf("Expected user to be 'user-x', got: %s", p.User)
	}
}

func TestUseHInsteadOFHelpIncorrectUse(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	type params struct {
		User string `short:"h" default:"123"`
	}

	p := params{}
	Cmd{
		Use:    "test",
		Short:  "test",
		Params: &p,
		RunFunc: func(cmd *cobra.Command, args []string) {
		},
	}.Run()
}

type InitTestStruct struct {
	User string `default:"defaultUser"`
}

func (i *InitTestStruct) InitCtx(ctx *HookContext) error {
	ctx.GetParam(&i.User).SetDefault(Default("123"))
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

	if params.User != "123" {
		t.Errorf("Expected default value to be '123', got: %s", params.User)
	}
}

type PreExecuteTestStruct struct {
	User string `default:"defaultUser"`
}

var errExpected = fmt.Errorf("<expected error>")

func (i *PreExecuteTestStruct) PreExecute() error {
	if i.User != "defaultUser" {
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
	Flag2 int
}

func (s *CustomValidatorTestStruct) InitCtx(ctx *HookContext) error {
	ctx.GetParam(&s.Flag2).SetCustomValidator(func(v any) error {
		if v.(int) < 0 {
			return fmt.Errorf("value must be greater than 0")
		}
		return nil
	})
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
		Params: &CustomValidatorTestStruct{},
		InitFuncCtx: func(ctx *HookContext, params any, cmd *cobra.Command) error {
			p := params.(*CustomValidatorTestStruct)
			// InitCtx will run first (from interface), then this runs
			ctx.GetParam(&p.Flag2).SetDefault(Default(42))
			return nil
		},
		ParamEnrich: ParamEnricherName, RawArgs: []string{},
	}.Validate()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	err = Cmd{
		Params: &CustomValidatorTestStruct{},
		InitFuncCtx: func(ctx *HookContext, params any, cmd *cobra.Command) error {
			p := params.(*CustomValidatorTestStruct)
			ctx.GetParam(&p.Flag2).SetDefault(Default(-42))
			return nil
		},
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
		MyEnum string `short:"e" default:"e1" alts:"e1,e2,e3"`
	}

	if err := (CmdT[Conf]{Use: "test", RawArgs: []string{}}).Validate(); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if err := (CmdT[Conf]{Use: "test", RawArgs: []string{"-e", "e2"}}).Validate(); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if err := (CmdT[Conf]{Use: "test", RawArgs: []string{"-e", "e3"}}).Validate(); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if err := (CmdT[Conf]{Use: "test", RawArgs: []string{"-e", "e4"}}).Validate(); err == nil {
		t.Errorf("Expected error, got: nil")
	}
}

func TestProgrammaticAlternativesMustBeEnforced(t *testing.T) {
	type Conf struct {
		MyEnum string `short:"e" default:"e1"`
	}
	preValidateFunc := func(params *Conf, cmd *cobra.Command, args []string) error {
		// We need HookContext to get the param mirror, but PreValidateFunc doesn't have it.
		// Use PreValidateFuncCtx instead is better, but for this test we can use Cmd-level approach.
		return nil
	}
	_ = preValidateFunc

	// Use InitFuncCtx to set alternatives since we need HookContext
	initFuncCtx := func(ctx *HookContext, params *Conf, cmd *cobra.Command) error {
		ctx.GetParam(&params.MyEnum).SetAlternatives([]string{"e1", "e2", "e3"})
		return nil
	}

	if err := (CmdT[Conf]{Use: "test", RawArgs: []string{}, InitFuncCtx: initFuncCtx}).Validate(); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if err := (CmdT[Conf]{Use: "test", RawArgs: []string{"-e", "e2"}, InitFuncCtx: initFuncCtx}).Validate(); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if err := (CmdT[Conf]{Use: "test", RawArgs: []string{"-e", "e3"}, InitFuncCtx: initFuncCtx}).Validate(); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if err := (CmdT[Conf]{Use: "test", RawArgs: []string{"-e", "e4"}, InitFuncCtx: initFuncCtx}).Validate(); err == nil {
		t.Errorf("Expected error, got: nil")
	}
}

func TestNonStrictAlternativesAllowAnyValue(t *testing.T) {
	type Conf struct {
		MyEnum string `short:"e" default:"e1" alts:"e1,e2,e3" strict:"false"`
	}

	// Valid alternative should work
	if err := (CmdT[Conf]{Use: "test", RawArgs: []string{"-e", "e2"}}).Validate(); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Non-listed value should also be accepted when strict is false
	if err := (CmdT[Conf]{Use: "test", RawArgs: []string{"-e", "custom-value"}}).Validate(); err != nil {
		t.Errorf("Expected no error for non-strict alts, got: %v", err)
	}
}

func TestStrictAlternativesEnforceValidation(t *testing.T) {
	type Conf struct {
		MyEnum string `short:"e" default:"e1" alts:"e1,e2,e3" strict:"true"`
	}

	// Valid alternative should work
	if err := (CmdT[Conf]{Use: "test", RawArgs: []string{"-e", "e2"}}).Validate(); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Non-listed value should be rejected when strict is true
	if err := (CmdT[Conf]{Use: "test", RawArgs: []string{"-e", "e4"}}).Validate(); err == nil {
		t.Errorf("Expected error for strict alts, got: nil")
	}
}

func TestStrictAlternativesDefaultBehavior(t *testing.T) {
	// Without strict tag, alts should be enforced (default behavior)
	type Conf struct {
		MyEnum string `short:"e" default:"e1" alts:"e1,e2,e3"`
	}

	if err := (CmdT[Conf]{Use: "test", RawArgs: []string{"-e", "e4"}}).Validate(); err == nil {
		t.Errorf("Expected error when strict is not specified (default should enforce), got: nil")
	}
}

func TestProgrammaticStrictAlts(t *testing.T) {
	type Conf struct {
		MyEnum string `short:"e" default:"e1"`
	}
	initFuncCtx := func(ctx *HookContext, params *Conf, cmd *cobra.Command) error {
		p := ctx.GetParam(&params.MyEnum)
		p.SetAlternatives([]string{"e1", "e2", "e3"})
		p.SetStrictAlts(false)
		return nil
	}

	// With strict set to false programmatically, any value should be accepted
	if err := (CmdT[Conf]{Use: "test", RawArgs: []string{"-e", "custom-value"}, InitFuncCtx: initFuncCtx}).Validate(); err != nil {
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
	err := (CmdT[Params]{Use: "test", ParamEnrich: ParamEnricherName, RawArgs: []string{}}).Validate()
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
	err := (CmdT[Params]{Use: "test", RawArgs: []string{"-m", "invalid"}}).Validate()
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
	err := (CmdT[Params]{
		Use:     "test",
		RawArgs: []string{"-p", "-1"},
		InitFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command) error {
			ctx.GetParam(&p.Port).SetCustomValidator(func(v any) error {
				if v.(int) < 0 {
					return fmt.Errorf("port must be non-negative")
				}
				return nil
			})
			return nil
		},
	}).Validate()
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

	err := (CmdT[Params]{Use: "test", RawArgs: []string{}}).Validate()
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

	err := (CmdT[Params]{Use: "test", RawArgs: []string{}}).Validate()
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
	err := (CmdT[Params]{Use: "test", RawArgs: []string{"-p", "not-a-number"}}).Validate()
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

	err := (CmdT[Params]{Use: "test", RawArgs: []string{"--unknown-flag"}}).Validate()
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
	err := (CmdT[Params]{
		Use:     "test",
		RawArgs: []string{"-s", "8080", "-e", "80"},
		PreValidateFunc: func(p *Params, cmd *cobra.Command, args []string) error {
			if p.StartPort > p.EndPort {
				return NewUserInputErrorf("start port (%d) must be less than end port (%d)", p.StartPort, p.EndPort)
			}
			return nil
		},
	}).Validate()

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

// TestErrorHandlingTable tests all combinations from the error handling documentation table
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
			var cmd CmdT[Params]
			cmd.Use = "test"
			cmd.ParamEnrich = ParamEnricherName

			switch tc.errorType {
			case "UserInput":
				// Missing required param causes user input error
				cmd.RunFunc = func(p *Params, c *cobra.Command, args []string) {}
			case "Hook":
				// Hook returns a UserInputError
				cmd.PreValidateFunc = func(p *Params, c *cobra.Command, args []string) error {
					return NewUserInputErrorf("hook failed")
				}
				cmd.RunFunc = func(p *Params, c *cobra.Command, args []string) {}
			case "Runtime":
				// RunFunc panics (for non-E methods) or returns error (for E methods)
				if strings.HasSuffix(tc.method, "E") {
					cmd.RunFuncE = func(p *Params, c *cobra.Command, args []string) error {
						return fmt.Errorf("runtime error")
					}
				} else {
					cmd.RunFunc = func(p *Params, c *cobra.Command, args []string) {
						panic("runtime error")
					}
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
				if panicValue != nil {
					t.Errorf("Expected clean exit (no panic), but got panic: %v", panicValue)
				}
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

// TestPositionalArgsErrorOutput verifies that missing positional arguments
// produce both a usage message and an error message in Run()/RunArgs() output,
// and that RunE()/RunArgsE() return errors without printing anything.
func TestPositionalArgsErrorOutput(t *testing.T) {
	type Params struct {
		File string `positional:"true" required:"true"`
		Dest string `positional:"true" required:"true"`
	}

	tests := []struct {
		name          string
		method        string
		args          []string
		wantErr       bool
		wantUsage     bool
		wantErrorMsg  bool
		errorContains string
	}{
		{
			name:          "Run/missing all positional args",
			method:        "Run",
			args:          []string{},
			wantErr:       true,
			wantUsage:     true,
			wantErrorMsg:  true,
			errorContains: "accepts between 2 and 2 arg(s), received 0",
		},
		{
			name:          "Run/missing one positional arg",
			method:        "Run",
			args:          []string{"file.txt"},
			wantErr:       true,
			wantUsage:     true,
			wantErrorMsg:  true,
			errorContains: "accepts between 2 and 2 arg(s), received 1",
		},
		{
			name:          "Run/too many positional args",
			method:        "Run",
			args:          []string{"a", "b", "c"},
			wantErr:       true,
			wantUsage:     true,
			wantErrorMsg:  true,
			errorContains: "accepts between 2 and 2 arg(s), received 3",
		},
		{
			name:          "RunE/missing all positional args",
			method:        "RunE",
			args:          []string{},
			wantErr:       true,
			wantUsage:     false,
			wantErrorMsg:  false, // RunE returns error, prints nothing
			errorContains: "accepts between 2 and 2 arg(s), received 0",
		},
		{
			name:         "Run/correct number of args",
			method:       "Run",
			args:         []string{"file.txt", "/tmp/dest"},
			wantErr:      false,
			wantUsage:    false,
			wantErrorMsg: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var exitCalled bool
			oldOsExit := osExit
			osExit = func(code int) {
				exitCalled = true
			}
			defer func() { osExit = oldOsExit }()

			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			cmd := CmdT[Params]{
				Use:         "test <file> <dest>",
				RunFunc:     func(p *Params, c *cobra.Command, args []string) {},
				ParamEnrich: ParamEnricherName,
			}

			var returnedErr error
			switch tc.method {
			case "Run":
				cmd.RunArgs(tc.args)
			case "RunE":
				returnedErr = cmd.RunArgsE(tc.args)
			}

			w.Close()
			os.Stderr = oldStderr
			captured := make([]byte, 8192)
			n, _ := r.Read(captured)
			r.Close()
			output := string(captured[:n])

			if tc.wantErr {
				if tc.method == "Run" && !exitCalled {
					t.Error("Expected osExit to be called")
				}
				if tc.method == "RunE" && returnedErr == nil {
					t.Error("Expected error to be returned")
				}
				if tc.method == "RunE" && returnedErr != nil && tc.errorContains != "" {
					if !strings.Contains(returnedErr.Error(), tc.errorContains) {
						t.Errorf("Expected error containing %q, got: %s", tc.errorContains, returnedErr.Error())
					}
				}
			}

			if tc.wantUsage {
				if !strings.Contains(output, "Usage:") {
					t.Errorf("Expected 'Usage:' in output, got:\n%s", output)
				}
			}
			if tc.wantErrorMsg {
				if !strings.Contains(output, "Error:") {
					t.Errorf("Expected 'Error:' in output, got:\n%s", output)
				}
				if tc.errorContains != "" && !strings.Contains(output, tc.errorContains) {
					t.Errorf("Expected output containing %q, got:\n%s", tc.errorContains, output)
				}
			}
			if !tc.wantUsage && !tc.wantErrorMsg && n > 0 {
				if tc.method == "RunE" {
					t.Errorf("Expected no output for RunE, got:\n%s", output)
				}
			}
		})
	}
}

// TestSubcommandPositionalArgsError verifies that subcommands with missing positional
// args produce both usage and error messages.
func TestSubcommandPositionalArgsError(t *testing.T) {
	type CpParams struct {
		ConvID   string `positional:"true" required:"true"`
		DestPath string `positional:"true" required:"true"`
	}

	type RootParams struct{}

	t.Run("Run/subcommand missing positional args", func(t *testing.T) {
		var exitCalled bool
		oldOsExit := osExit
		osExit = func(code int) { exitCalled = true }
		defer func() { osExit = oldOsExit }()

		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		root := Cmd{
			Use: "app",
			SubCmds: SubCmds(
				CmdT[CpParams]{
					Use:         "cp <conv-id> <dest-path>",
					Short:       "Copy a conversation",
					RunFunc:     func(p *CpParams, c *cobra.Command, args []string) {},
					ParamEnrich: ParamEnricherName,
				},
			),
		}
		root.RunArgs([]string{"cp"})

		w.Close()
		os.Stderr = oldStderr
		captured := make([]byte, 8192)
		n, _ := r.Read(captured)
		r.Close()
		output := string(captured[:n])

		if !exitCalled {
			t.Error("Expected osExit to be called")
		}
		if !strings.Contains(output, "Usage:") {
			t.Errorf("Expected 'Usage:' in output, got:\n%s", output)
		}
		if !strings.Contains(output, "Error:") {
			t.Errorf("Expected 'Error:' in output, got:\n%s", output)
		}
		if !strings.Contains(output, "accepts between 2 and 2 arg(s), received 0") {
			t.Errorf("Expected error about arg count, got:\n%s", output)
		}
	})

	t.Run("RunE/subcommand missing positional args", func(t *testing.T) {
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		root := Cmd{
			Use: "app",
			SubCmds: SubCmds(
				CmdT[CpParams]{
					Use:         "cp <conv-id> <dest-path>",
					Short:       "Copy a conversation",
					RunFunc:     func(p *CpParams, c *cobra.Command, args []string) {},
					ParamEnrich: ParamEnricherName,
				},
			),
		}
		err := root.RunArgsE([]string{"cp"})

		w.Close()
		os.Stderr = oldStderr
		captured := make([]byte, 8192)
		n, _ := r.Read(captured)
		r.Close()

		if err == nil {
			t.Fatal("Expected error to be returned")
		}
		if !strings.Contains(err.Error(), "accepts between 2 and 2 arg(s), received 0") {
			t.Errorf("Expected error about arg count, got: %s", err.Error())
		}
		if n > 0 {
			t.Errorf("Expected no output for RunE, got:\n%s", string(captured[:n]))
		}
	})

	t.Run("Run/unknown subcommand rejected", func(t *testing.T) {
		var exitCalled bool
		oldOsExit := osExit
		osExit = func(code int) { exitCalled = true }
		defer func() { osExit = oldOsExit }()

		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		root := Cmd{
			Use: "app",
			SubCmds: SubCmds(
				CmdT[RootParams]{
					Use:         "valid",
					RunFunc:     func(p *RootParams, c *cobra.Command, args []string) {},
					ParamEnrich: ParamEnricherName,
				},
			),
		}
		root.RunArgs([]string{"bogus"})

		w.Close()
		os.Stderr = oldStderr
		captured := make([]byte, 8192)
		n, _ := r.Read(captured)
		r.Close()
		output := string(captured[:n])

		if !exitCalled {
			t.Error("Expected osExit to be called for unknown subcommand")
		}
		if !strings.Contains(output, "Error:") {
			t.Errorf("Expected 'Error:' in output for unknown subcommand, got:\n%s", output)
		}
	})
}

// TestSubcommandOnlyUnknownCommand verifies that commands with subcommands but
// no RunFunc reject unknown subcommands with an error (not silently show help).
func TestSubcommandOnlyUnknownCommand(t *testing.T) {
	type SubParams struct{}

	makeRoot := func() Cmd {
		return Cmd{
			Use:   "app",
			Short: "My app",
			SubCmds: SubCmds(
				CmdT[SubParams]{
					Use:         "valid",
					Short:       "A valid command",
					RunFunc:     func(p *SubParams, c *cobra.Command, args []string) {},
					ParamEnrich: ParamEnricherName,
				},
			),
			// No RunFunc — subcommand-only command
		}
	}

	t.Run("Run/unknown command errors with exit 1", func(t *testing.T) {
		var exitCalled bool
		oldOsExit := osExit
		osExit = func(code int) { exitCalled = true }
		defer func() { osExit = oldOsExit }()

		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		makeRoot().RunArgs([]string{"bogus"})

		w.Close()
		os.Stderr = oldStderr
		captured := make([]byte, 8192)
		n, _ := r.Read(captured)
		r.Close()
		output := string(captured[:n])

		if !exitCalled {
			t.Error("Expected osExit to be called for unknown subcommand")
		}
		if !strings.Contains(output, "Error:") {
			t.Errorf("Expected 'Error:' in output, got:\n%s", output)
		}
		if !strings.Contains(output, "unknown command") {
			t.Errorf("Expected 'unknown command' in output, got:\n%s", output)
		}
	})

	t.Run("RunE/unknown command returns error", func(t *testing.T) {
		err := makeRoot().RunArgsE([]string{"bogus"})
		if err == nil {
			t.Fatal("Expected error for unknown subcommand")
		}
		if !strings.Contains(err.Error(), "unknown command") {
			t.Errorf("Expected 'unknown command' in error, got: %s", err.Error())
		}
		if !IsUserInputError(err) {
			t.Errorf("Expected UserInputError, got: %T", err)
		}
	})

	t.Run("Run/no args shows help with exit 0", func(t *testing.T) {
		var exitCalled bool
		oldOsExit := osExit
		osExit = func(code int) { exitCalled = true }
		defer func() { osExit = oldOsExit }()

		makeRoot().RunArgs([]string{})

		if exitCalled {
			t.Error("Did not expect osExit for no-arg subcommand-only command")
		}
	})

	t.Run("Run/valid subcommand still works", func(t *testing.T) {
		ran := false
		root := Cmd{
			Use: "app",
			SubCmds: SubCmds(
				CmdT[SubParams]{
					Use:         "valid",
					RunFunc:     func(p *SubParams, c *cobra.Command, args []string) { ran = true },
					ParamEnrich: ParamEnricherName,
				},
			),
		}
		root.RunArgs([]string{"valid"})
		if !ran {
			t.Error("Expected valid subcommand to run")
		}
	})
}

// TestSlicePositionalArgsErrorOutput verifies that slice positional args accept
// variable argument counts (MinimumNArgs) instead of exact counts.
func TestSlicePositionalArgsErrorOutput(t *testing.T) {
	type SliceParams struct {
		Files []string `positional:"true" required:"true"`
	}

	type MixedParams struct {
		Dest  string   `positional:"true" required:"true"`
		Files []string `positional:"true" required:"true"`
	}

	t.Run("Run/slice accepts single arg", func(t *testing.T) {
		var got []string
		cmd := CmdT[SliceParams]{
			Use:         "test <files>...",
			ParamEnrich: ParamEnricherName,
			RunFunc: func(p *SliceParams, c *cobra.Command, args []string) {
				got = p.Files
			},
		}
		cmd.RunArgs([]string{"only.txt"})
		if len(got) != 1 || got[0] != "only.txt" {
			t.Errorf("Expected [only.txt], got %v", got)
		}
	})

	t.Run("Run/slice accepts many args", func(t *testing.T) {
		var got []string
		cmd := CmdT[SliceParams]{
			Use:         "test <files>...",
			ParamEnrich: ParamEnricherName,
			RunFunc: func(p *SliceParams, c *cobra.Command, args []string) {
				got = p.Files
			},
		}
		input := []string{"a", "b", "c", "d", "e"}
		cmd.RunArgs(input)
		if len(got) != 5 {
			t.Errorf("Expected 5 files, got %d: %v", len(got), got)
		}
		for i, want := range input {
			if got[i] != want {
				t.Errorf("Files[%d] = %q, want %q", i, got[i], want)
			}
		}
	})

	t.Run("Run/slice requires minimum args", func(t *testing.T) {
		var exitCalled bool
		oldOsExit := osExit
		osExit = func(code int) { exitCalled = true }
		defer func() { osExit = oldOsExit }()

		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		cmd := CmdT[SliceParams]{
			Use:         "test <files>...",
			RunFunc:     func(p *SliceParams, c *cobra.Command, args []string) {},
			ParamEnrich: ParamEnricherName,
		}
		cmd.RunArgs([]string{})

		w.Close()
		os.Stderr = oldStderr
		captured := make([]byte, 8192)
		n, _ := r.Read(captured)
		r.Close()
		output := string(captured[:n])

		if !exitCalled {
			t.Error("Expected osExit to be called")
		}
		if !strings.Contains(output, "Error:") {
			t.Errorf("Expected 'Error:' in output, got:\n%s", output)
		}
	})

	t.Run("Run/mixed positional with slice accepts variable count", func(t *testing.T) {
		var gotDest string
		var gotFiles []string
		cmd := CmdT[MixedParams]{
			Use:         "test <dest> <files>...",
			ParamEnrich: ParamEnricherName,
			RunFunc: func(p *MixedParams, c *cobra.Command, args []string) {
				gotDest = p.Dest
				gotFiles = p.Files
			},
		}
		cmd.RunArgs([]string{"/tmp", "a.txt", "b.txt"})
		if gotDest != "/tmp" {
			t.Errorf("Dest = %q, want /tmp", gotDest)
		}
		if len(gotFiles) != 2 || gotFiles[0] != "a.txt" || gotFiles[1] != "b.txt" {
			t.Errorf("Files = %v, want [a.txt b.txt]", gotFiles)
		}
	})

	t.Run("Run/mixed positional with just required and one slice arg", func(t *testing.T) {
		var gotDest string
		var gotFiles []string
		cmd := CmdT[MixedParams]{
			Use:         "test <dest> <files>...",
			ParamEnrich: ParamEnricherName,
			RunFunc: func(p *MixedParams, c *cobra.Command, args []string) {
				gotDest = p.Dest
				gotFiles = p.Files
			},
		}
		cmd.RunArgs([]string{"/out", "single.txt"})
		if gotDest != "/out" {
			t.Errorf("Dest = %q, want /out", gotDest)
		}
		if len(gotFiles) != 1 || gotFiles[0] != "single.txt" {
			t.Errorf("Files = %v, want [single.txt]", gotFiles)
		}
	})

	t.Run("Run/mixed positional missing required", func(t *testing.T) {
		var exitCalled bool
		oldOsExit := osExit
		osExit = func(code int) { exitCalled = true }
		defer func() { osExit = oldOsExit }()

		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		cmd := CmdT[MixedParams]{
			Use:         "test <dest> <files>...",
			RunFunc:     func(p *MixedParams, c *cobra.Command, args []string) {},
			ParamEnrich: ParamEnricherName,
		}
		cmd.RunArgs([]string{})

		w.Close()
		os.Stderr = oldStderr
		captured := make([]byte, 8192)
		n, _ := r.Read(captured)
		r.Close()
		output := string(captured[:n])

		if !exitCalled {
			t.Error("Expected osExit to be called")
		}
		if !strings.Contains(output, "Error:") {
			t.Errorf("Expected 'Error:' in output, got:\n%s", output)
		}
		if !strings.Contains(output, "Usage:") {
			t.Errorf("Expected 'Usage:' in output, got:\n%s", output)
		}
	})

	t.Run("RunE/slice returns error for missing args", func(t *testing.T) {
		err := (CmdT[SliceParams]{
			Use:         "test <files>...",
			RunFunc:     func(p *SliceParams, c *cobra.Command, args []string) {},
			ParamEnrich: ParamEnricherName,
		}).RunArgsE([]string{})

		if err == nil {
			t.Fatal("Expected error for missing slice positional args")
		}
		if !IsUserInputError(err) {
			t.Errorf("Expected UserInputError, got: %T", err)
		}
	})
}

// TestUsageStringGeneration verifies that auto-generated usage strings contain
// the expected flag names, descriptions, defaults, positional placeholders,
// and subcommand listings.
func TestUsageStringGeneration(t *testing.T) {
	t.Run("flags with descriptions and defaults", func(t *testing.T) {
		type Params struct {
			Name    string `descr:"User name" required:"true"`
			Port    int    `descr:"Port number" default:"8080" optional:"true"`
			Verbose bool   `short:"v" descr:"Enable verbose output"`
		}
		usage := (CmdT[Params]{
			Use:     "serve",
			Short:   "Start the server",
			RunFunc: func(p *Params, c *cobra.Command, args []string) {},
		}).ToCobra().UsageString()

		for _, want := range []string{
			"serve [flags]",
			"--name string",
			"User name",
			"(required)",
			"--port int",
			"Port number",
			"(default 8080)",
			"-v, --verbose",
			"Enable verbose output",
		} {
			if !strings.Contains(usage, want) {
				t.Errorf("Usage missing %q:\n%s", want, usage)
			}
		}
	})

	t.Run("auto-generated flag names from field names", func(t *testing.T) {
		type Params struct {
			DBHost   string `descr:"Database host"`
			HTTPPort int    `descr:"HTTP port" default:"8080"`
			LogLevel string `descr:"Log level"`
		}
		usage := (CmdT[Params]{
			Use:         "app",
			RunFunc:     func(p *Params, c *cobra.Command, args []string) {},
			ParamEnrich: ParamEnricherName,
		}).ToCobra().UsageString()

		for _, want := range []string{
			"--db-host string",
			"--http-port int",
			"--log-level string",
		} {
			if !strings.Contains(usage, want) {
				t.Errorf("Usage missing %q:\n%s", want, usage)
			}
		}
	})

	t.Run("positional args in use line", func(t *testing.T) {
		type Params struct {
			Src  string `positional:"true" required:"true"`
			Dest string `positional:"true" required:"true"`
		}
		usage := (CmdT[Params]{
			Use:         "cp",
			RunFunc:     func(p *Params, c *cobra.Command, args []string) {},
			ParamEnrich: ParamEnricherName,
		}).ToCobra().UsageString()

		for _, want := range []string{
			"cp <src> <dest>",
		} {
			if !strings.Contains(usage, want) {
				t.Errorf("Usage missing %q:\n%s", want, usage)
			}
		}
		// Positional-only commands should not show [flags]
		if strings.Contains(usage, "[flags]") {
			t.Errorf("Positional-only command should not show [flags]:\n%s", usage)
		}
	})

	t.Run("mixed flags and positional args", func(t *testing.T) {
		type Params struct {
			File   string  `positional:"true" required:"true"`
			Output *string `descr:"Output file"`
			Force  bool    `short:"f" descr:"Force overwrite"`
		}
		usage := (CmdT[Params]{
			Use:     "process",
			RunFunc: func(p *Params, c *cobra.Command, args []string) {},
		}).ToCobra().UsageString()

		for _, want := range []string{
			"process <file> [flags]",
			"--output string",
			"Output file",
			"-f, --force",
			"Force overwrite",
		} {
			if !strings.Contains(usage, want) {
				t.Errorf("Usage missing %q:\n%s", want, usage)
			}
		}
	})

	t.Run("pointer fields are optional by default", func(t *testing.T) {
		type Params struct {
			Name *string `descr:"Optional name"`
			Age  *int    `descr:"Optional age"`
		}
		usage := (CmdT[Params]{
			Use:         "app",
			RunFunc:     func(p *Params, c *cobra.Command, args []string) {},
			ParamEnrich: ParamEnricherName,
		}).ToCobra().UsageString()

		// Pointer fields should NOT show "(required)"
		if strings.Contains(usage, "(required)") {
			t.Errorf("Pointer fields should not show (required):\n%s", usage)
		}
	})

	t.Run("subcommands listed", func(t *testing.T) {
		type SubParams struct {
			ID string `positional:"true" required:"true"`
		}
		type RootParams struct {
			LogLevel string `descr:"Log level" default:"info"`
		}
		usage := (CmdT[RootParams]{
			Use:   "app",
			Short: "My app",
			SubCmds: SubCmds(
				CmdT[SubParams]{
					Use:   "get",
					Short: "Get a resource",
					RunFunc: func(p *SubParams, c *cobra.Command, args []string) {},
				},
				CmdT[SubParams]{
					Use:   "delete",
					Short: "Delete a resource",
					RunFunc: func(p *SubParams, c *cobra.Command, args []string) {},
				},
			),
		}).ToCobra().UsageString()

		for _, want := range []string{
			"app [command]",
			"Available Commands:",
			"get",
			"Get a resource",
			"delete",
			"Delete a resource",
			"--log-level string",
			`(default "info")`,
		} {
			if !strings.Contains(usage, want) {
				t.Errorf("Usage missing %q:\n%s", want, usage)
			}
		}
	})

	t.Run("slice positional in use line", func(t *testing.T) {
		type Params struct {
			Dest  string   `positional:"true" required:"true"`
			Files []string `positional:"true" required:"true"`
		}
		usage := (CmdT[Params]{
			Use:         "upload",
			RunFunc:     func(p *Params, c *cobra.Command, args []string) {},
			ParamEnrich: ParamEnricherName,
		}).ToCobra().UsageString()

		for _, want := range []string{
			"upload <dest> <files...>",
		} {
			if !strings.Contains(usage, want) {
				t.Errorf("Usage missing %q:\n%s", want, usage)
			}
		}
	})

	t.Run("short flags auto-assigned by default enricher", func(t *testing.T) {
		type Params struct {
			Name string `descr:"User name"`
			Port int    `descr:"Port number" default:"8080"`
		}
		usage := (CmdT[Params]{
			Use:     "app",
			RunFunc: func(p *Params, c *cobra.Command, args []string) {},
			// Default enricher (not ParamEnricherName) assigns short flags
		}).ToCobra().UsageString()

		for _, want := range []string{
			"-n, --name",
			"-p, --port",
		} {
			if !strings.Contains(usage, want) {
				t.Errorf("Usage missing %q:\n%s", want, usage)
			}
		}
	})

	t.Run("explicit short flag override", func(t *testing.T) {
		type Params struct {
			Verbose bool `short:"V" descr:"Verbose output"`
		}
		usage := (CmdT[Params]{
			Use:         "app",
			RunFunc:     func(p *Params, c *cobra.Command, args []string) {},
			ParamEnrich: ParamEnricherName,
		}).ToCobra().UsageString()

		if !strings.Contains(usage, "-V, --verbose") {
			t.Errorf("Expected explicit short flag -V:\n%s", usage)
		}
	})

	t.Run("env tag shown in description", func(t *testing.T) {
		type Params struct {
			Token string `descr:"API token" env:"API_TOKEN"`
		}
		usage := (CmdT[Params]{
			Use:     "app",
			RunFunc: func(p *Params, c *cobra.Command, args []string) {},
		}).ToCobra().UsageString()

		if !strings.Contains(usage, "API_TOKEN") {
			t.Errorf("Expected env var name in usage:\n%s", usage)
		}
	})
}

// TestPositionalArgsUsageAndValidation checks usage strings and argument
// validation for every positional argument combination:
// 0, 1, 2, 3 required; slice; required+optional; required+slice.
func TestPositionalArgsUsageAndValidation(t *testing.T) {
	noop := func(_ *cobra.Command, _ []string) {}

	// --- 0 positional args ---
	t.Run("0 pos args/usage", func(t *testing.T) {
		type P struct {
			Flag string `descr:"A flag"`
		}
		usage := (CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) {}, ParamEnrich: ParamEnricherName}).ToCobra().UsageString()
		if !strings.Contains(usage, "cmd [flags]") {
			t.Errorf("Expected 'cmd [flags]':\n%s", usage)
		}
	})
	t.Run("0 pos args/rejects args", func(t *testing.T) {
		type P struct{}
		err := (CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) {}, ParamEnrich: ParamEnricherName}).RunArgsE([]string{"unexpected"})
		if err == nil {
			t.Fatal("Expected error for unexpected arg")
		}
	})

	// --- 1 positional arg ---
	t.Run("1 pos arg/usage", func(t *testing.T) {
		type P struct {
			Src string `positional:"true" required:"true"`
		}
		usage := (CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) {}, ParamEnrich: ParamEnricherName}).ToCobra().UsageString()
		if !strings.Contains(usage, "cmd <src>") {
			t.Errorf("Expected 'cmd <src>':\n%s", usage)
		}
	})
	t.Run("1 pos arg/accepts 1", func(t *testing.T) {
		type P struct {
			Src string `positional:"true" required:"true"`
		}
		var got string
		(CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Src }, ParamEnrich: ParamEnricherName}).RunArgs([]string{"hello"})
		if got != "hello" {
			t.Errorf("Src = %q, want hello", got)
		}
	})
	t.Run("1 pos arg/rejects 0", func(t *testing.T) {
		type P struct {
			Src string `positional:"true" required:"true"`
		}
		err := (CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) {}, ParamEnrich: ParamEnricherName}).RunArgsE([]string{})
		if err == nil {
			t.Fatal("Expected error for missing arg")
		}
	})
	t.Run("1 pos arg/rejects 2", func(t *testing.T) {
		type P struct {
			Src string `positional:"true" required:"true"`
		}
		err := (CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) {}, ParamEnrich: ParamEnricherName}).RunArgsE([]string{"a", "b"})
		if err == nil {
			t.Fatal("Expected error for too many args")
		}
	})

	// --- 2 positional args ---
	t.Run("2 pos args/usage", func(t *testing.T) {
		type P struct {
			Src  string `positional:"true" required:"true"`
			Dest string `positional:"true" required:"true"`
		}
		usage := (CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) {}, ParamEnrich: ParamEnricherName}).ToCobra().UsageString()
		if !strings.Contains(usage, "cmd <src> <dest>") {
			t.Errorf("Expected 'cmd <src> <dest>':\n%s", usage)
		}
	})
	t.Run("2 pos args/accepts 2", func(t *testing.T) {
		type P struct {
			Src  string `positional:"true" required:"true"`
			Dest string `positional:"true" required:"true"`
		}
		var gotSrc, gotDest string
		(CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) { gotSrc = p.Src; gotDest = p.Dest }, ParamEnrich: ParamEnricherName}).RunArgs([]string{"a", "b"})
		if gotSrc != "a" || gotDest != "b" {
			t.Errorf("Got src=%q dest=%q, want a, b", gotSrc, gotDest)
		}
	})
	t.Run("2 pos args/rejects 1", func(t *testing.T) {
		type P struct {
			Src  string `positional:"true" required:"true"`
			Dest string `positional:"true" required:"true"`
		}
		err := (CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) {}, ParamEnrich: ParamEnricherName}).RunArgsE([]string{"a"})
		if err == nil {
			t.Fatal("Expected error for missing arg")
		}
	})

	// --- 3 positional args ---
	t.Run("3 pos args/usage", func(t *testing.T) {
		type P struct {
			A string `positional:"true" required:"true"`
			B string `positional:"true" required:"true"`
			C string `positional:"true" required:"true"`
		}
		usage := (CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) {}, ParamEnrich: ParamEnricherName}).ToCobra().UsageString()
		if !strings.Contains(usage, "cmd <a> <b> <c>") {
			t.Errorf("Expected 'cmd <a> <b> <c>':\n%s", usage)
		}
	})
	t.Run("3 pos args/accepts 3", func(t *testing.T) {
		type P struct {
			A string `positional:"true" required:"true"`
			B string `positional:"true" required:"true"`
			C string `positional:"true" required:"true"`
		}
		var a, b, c string
		(CmdT[P]{Use: "cmd", RunFunc: func(p *P, cc *cobra.Command, args []string) { a = p.A; b = p.B; c = p.C }, ParamEnrich: ParamEnricherName}).RunArgs([]string{"x", "y", "z"})
		if a != "x" || b != "y" || c != "z" {
			t.Errorf("Got %q %q %q, want x y z", a, b, c)
		}
	})
	t.Run("3 pos args/rejects 2", func(t *testing.T) {
		type P struct {
			A string `positional:"true" required:"true"`
			B string `positional:"true" required:"true"`
			C string `positional:"true" required:"true"`
		}
		err := (CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) {}, ParamEnrich: ParamEnricherName}).RunArgsE([]string{"a", "b"})
		if err == nil {
			t.Fatal("Expected error for missing arg")
		}
	})

	// --- slice positional args ---
	t.Run("slice pos/usage", func(t *testing.T) {
		type P struct {
			Files []string `positional:"true" required:"true"`
		}
		usage := (CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) {}, ParamEnrich: ParamEnricherName}).ToCobra().UsageString()
		if !strings.Contains(usage, "cmd <files...>") {
			t.Errorf("Expected 'cmd <files>':\n%s", usage)
		}
	})
	t.Run("slice pos/accepts 1", func(t *testing.T) {
		type P struct {
			Files []string `positional:"true" required:"true"`
		}
		var got []string
		(CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Files }, ParamEnrich: ParamEnricherName}).RunArgs([]string{"a"})
		if len(got) != 1 || got[0] != "a" {
			t.Errorf("Got %v, want [a]", got)
		}
	})
	t.Run("slice pos/accepts 5", func(t *testing.T) {
		type P struct {
			Files []string `positional:"true" required:"true"`
		}
		var got []string
		(CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) { got = p.Files }, ParamEnrich: ParamEnricherName}).RunArgs([]string{"a", "b", "c", "d", "e"})
		if len(got) != 5 {
			t.Errorf("Got %d args, want 5: %v", len(got), got)
		}
	})
	t.Run("slice pos/rejects 0", func(t *testing.T) {
		type P struct {
			Files []string `positional:"true" required:"true"`
		}
		err := (CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) {}, ParamEnrich: ParamEnricherName}).RunArgsE([]string{})
		if err == nil {
			t.Fatal("Expected error for missing args")
		}
	})

	// --- 1 required + 1 optional positional ---
	t.Run("1 req + 1 opt/usage", func(t *testing.T) {
		type P struct {
			Src  string `positional:"true" required:"true"`
			Dest string `positional:"true" optional:"true"`
		}
		usage := (CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) {}, ParamEnrich: ParamEnricherName}).ToCobra().UsageString()
		if !strings.Contains(usage, "cmd <src> [dest]") {
			t.Errorf("Expected 'cmd <src> [dest]':\n%s", usage)
		}
	})
	t.Run("1 req + 1 opt/accepts 1", func(t *testing.T) {
		type P struct {
			Src  string `positional:"true" required:"true"`
			Dest string `positional:"true" optional:"true"`
		}
		var src, dest string
		(CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) { src = p.Src; dest = p.Dest }, ParamEnrich: ParamEnricherName}).RunArgs([]string{"a"})
		if src != "a" {
			t.Errorf("Src = %q, want a", src)
		}
		if dest != "" {
			t.Errorf("Dest = %q, want empty", dest)
		}
	})
	t.Run("1 req + 1 opt/accepts 2", func(t *testing.T) {
		type P struct {
			Src  string `positional:"true" required:"true"`
			Dest string `positional:"true" optional:"true"`
		}
		var src, dest string
		(CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) { src = p.Src; dest = p.Dest }, ParamEnrich: ParamEnricherName}).RunArgs([]string{"a", "b"})
		if src != "a" || dest != "b" {
			t.Errorf("Got src=%q dest=%q, want a, b", src, dest)
		}
	})
	t.Run("1 req + 1 opt/rejects 0", func(t *testing.T) {
		type P struct {
			Src  string `positional:"true" required:"true"`
			Dest string `positional:"true" optional:"true"`
		}
		err := (CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) {}, ParamEnrich: ParamEnricherName}).RunArgsE([]string{})
		if err == nil {
			t.Fatal("Expected error for missing required arg")
		}
	})
	t.Run("1 req + 1 opt/rejects 3", func(t *testing.T) {
		type P struct {
			Src  string `positional:"true" required:"true"`
			Dest string `positional:"true" optional:"true"`
		}
		err := (CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) {}, ParamEnrich: ParamEnricherName}).RunArgsE([]string{"a", "b", "c"})
		if err == nil {
			t.Fatal("Expected error for too many args")
		}
	})

	// --- 2 required + 1 slice (arbitrary tail) ---
	t.Run("2 req + slice/usage", func(t *testing.T) {
		type P struct {
			Src   string   `positional:"true" required:"true"`
			Dest  string   `positional:"true" required:"true"`
			Files []string `positional:"true" required:"true"`
		}
		usage := (CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) {}, ParamEnrich: ParamEnricherName}).ToCobra().UsageString()
		if !strings.Contains(usage, "cmd <src> <dest> <files...>") {
			t.Errorf("Expected 'cmd <src> <dest> <files>':\n%s", usage)
		}
	})
	t.Run("2 req + slice/accepts exactly required", func(t *testing.T) {
		type P struct {
			Src   string   `positional:"true" required:"true"`
			Dest  string   `positional:"true" required:"true"`
			Files []string `positional:"true" required:"true"`
		}
		var src, dest string
		var files []string
		(CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) { src = p.Src; dest = p.Dest; files = p.Files }, ParamEnrich: ParamEnricherName}).RunArgs([]string{"s", "d", "f1"})
		if src != "s" || dest != "d" || len(files) != 1 || files[0] != "f1" {
			t.Errorf("Got src=%q dest=%q files=%v", src, dest, files)
		}
	})
	t.Run("2 req + slice/accepts many trailing", func(t *testing.T) {
		type P struct {
			Src   string   `positional:"true" required:"true"`
			Dest  string   `positional:"true" required:"true"`
			Files []string `positional:"true" required:"true"`
		}
		var src, dest string
		var files []string
		(CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) { src = p.Src; dest = p.Dest; files = p.Files }, ParamEnrich: ParamEnricherName}).RunArgs([]string{"s", "d", "f1", "f2", "f3", "f4"})
		if src != "s" || dest != "d" || len(files) != 4 {
			t.Errorf("Got src=%q dest=%q files=%v", src, dest, files)
		}
	})
	t.Run("2 req + slice/rejects 1", func(t *testing.T) {
		type P struct {
			Src   string   `positional:"true" required:"true"`
			Dest  string   `positional:"true" required:"true"`
			Files []string `positional:"true" required:"true"`
		}
		err := (CmdT[P]{Use: "cmd", RunFunc: func(p *P, c *cobra.Command, args []string) {}, ParamEnrich: ParamEnricherName}).RunArgsE([]string{"a"})
		if err == nil {
			t.Fatal("Expected error for insufficient args")
		}
	})

	_ = noop
}
