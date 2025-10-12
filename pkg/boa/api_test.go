package boa

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
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

	// TODO: check that we canot provide e4, when cobra supports it
	// see https://github.com/spf13/pflag/issues/236
}
