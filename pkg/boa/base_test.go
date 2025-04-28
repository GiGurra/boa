package boa

import (
	"fmt"
	"os"
	"reflect"
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

	Wrap{
		Use:    "test",
		Short:  "test",
		Params: &params,
	}.ToCmd()

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

	Wrap{
		Use:    "test",
		Short:  "test",
		Params: &params,
	}.ToCmd()
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

	Wrap{
		Use:    "test",
		Short:  "test",
		Params: &params,
	}.ToCmd()
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

	Wrap{
		Use:    "test",
		Short:  "test",
		Params: &params,
	}.ToCmd()
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

	err := Validate(&params, Wrap{ParamEnrich: ParamEnricherName})
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if params.User.Value() != "123" {
		t.Errorf("Expected default value to be '123', got: %s", params.User.Value())
	}
}
