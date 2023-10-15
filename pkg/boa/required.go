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
	validated       bool
	setByEnv        bool
	valuePtr        any
	parent          *cobra.Command
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
	if !f.validated {
		panic(fmt.Errorf("flag %s was not validated. Cannot use flag before validation. Did you call Validate(..) on the parent struct", f.GetName()))
	}
	if !hasValue(f) {
		panic(fmt.Errorf("tried to access flag.Value() of '%s', which was not set. This is a bug in util_cobra", f.GetName()))
	}
	return *f.valuePtr.(*T)
}

func (f *Required[T]) markValidated() {
	f.validated = true
}

func (f *Required[T]) customValidatorOfPtr() func(any) error {
	return func(val any) error {
		if f.CustomValidator == nil {
			return nil
		}
		return f.CustomValidator(*val.(*T))
	}
}

func (f *Required[T]) wasSetByFlag() bool {
	if f.parent == nil {
		panic("flag has no parent command. Did you try to .validate() before .ToCmd()?")
	}
	return f.parent.Flags().Changed(f.Name)
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

func (f *Required[T]) GetKind() reflect.Kind {
	var zero T
	return reflect.TypeOf(zero).Kind()
}

func (f *Required[T]) setParentCmd(cmd *cobra.Command) {
	f.parent = cmd
}

func (f *Required[T]) setValuePtr(val any) {
	f.valuePtr = val
}
