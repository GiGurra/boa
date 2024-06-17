package boa

import (
	"fmt"
	"github.com/spf13/cobra"
	"reflect"
)

type Required[T SupportedTypes] struct {
	Name            string
	Short           string
	Env             string
	Default         *T
	Descr           string
	CustomValidator func(T) error
	Positional      bool

	Alternatives     []string
	AlternativesFunc func(cmd *cobra.Command, args []string, toComplete string) []string

	setByEnv        bool
	setPositionally bool
	valuePtr        any
	parent          *cobra.Command
}

func (f *Required[T]) IsEnabled() bool {
	return true
}

func (f *Required[T]) GetIsEnabledFn() func() bool {
	return nil
}

func (f *Required[T]) SetAlternatives(strings []string) {
	f.Alternatives = strings
}

// prove that Optional[T] implements Param
var _ Param = &Required[string]{}

func (f *Required[T]) wasSetPositionally() bool {
	return f.setPositionally
}

func (f *Required[T]) markSetPositionally() {
	f.setPositionally = true
}

func (f *Required[T]) isPositional() bool {
	return f.Positional
}

// SetDefault Only to be used from ParamEnrichers. Use the regular parameters to set the default with type safety otherwise.
func (f *Required[T]) SetDefault(val any) {
	f.Default = val.(*T)
}

func (f *Required[T]) SetEnv(val string) {
	f.Env = val
}

func (f *Required[T]) SetShort(val string) {
	f.Short = val
}

func (f *Required[T]) SetName(val string) {
	f.Name = val
}

func (f *Required[T]) wasSetByEnv() bool {
	return f.setByEnv
}

func (f *Required[T]) markSetFromEnv() {
	f.setByEnv = true
}

func (f *Required[T]) Value() T {
	if !HasValue(f) {
		panic(fmt.Errorf("tried to access flag.Value() of '%s', which was not set. This is a bug in util_cobra", f.GetName()))
	}
	if f.valuePtr != nil {
		return *f.valuePtr.(*T)
	} else {
		if f.hasDefaultValue() {
			return *f.Default
		} else {
			panic(fmt.Errorf("tried to access flag.Value() of '%s', which was not set. This is a bug in util_cobra", f.GetName()))
		}
	}
}

func (f *Required[T]) setDescription(state string) {
	f.Descr = state
}

func (f *Required[T]) setPositional(state bool) {
	f.Positional = state
}

func (f *Required[T]) customValidatorOfPtr() func(any) error {
	return func(val any) error {
		if f.CustomValidator == nil {
			return nil
		}
		return f.CustomValidator(*val.(*T))
	}
}

func (f *Required[T]) wasSetOnCli() bool {
	if f.Positional {
		return f.wasSetPositionally()
	} else {
		if f.parent == nil {
			return false
		} else {
			return f.parent.Flags().Changed(f.Name)
		}
	}
}

func (f *Required[T]) GetShort() string {
	return f.Short
}

func (f *Required[T]) GetName() string {
	return f.Name
}

func (f *Required[T]) GetEnv() string {
	return f.Env
}

func (f *Required[T]) defaultValuePtr() any {
	return f.Default
}

func (f *Required[T]) hasDefaultValue() bool {
	return f.Default != nil
}

func (f *Required[T]) descr() string {
	return f.Descr
}

func (f *Required[T]) IsRequired() bool {
	return true
}

func (f *Required[T]) valuePtrF() any {
	return f.valuePtr
}

func (f *Required[T]) parentCmd() *cobra.Command {
	return f.parent
}

func (f *Required[T]) defaultValueStr() string {
	if !f.hasDefaultValue() {
		panic("flag has no default value")
	}
	return fmt.Sprintf("%v", *f.Default)
}

func (f *Required[T]) HasValue() bool {
	return HasValue(f)
}

func (f *Required[T]) GetKind() reflect.Kind {
	return f.GetType().Kind()
}

func (f *Required[T]) GetType() reflect.Type {
	var zero T
	return reflect.TypeOf(zero)
}

func (f *Required[T]) setParentCmd(cmd *cobra.Command) {
	f.parent = cmd
}

func (f *Required[T]) setValuePtr(val any) {
	f.valuePtr = val
}

func (f *Required[T]) GetAlternatives() []string {
	return f.Alternatives
}

func (f *Required[T]) GetAlternativesFunc() func(cmd *cobra.Command, args []string, toComplete string) []string {
	return f.AlternativesFunc
}
