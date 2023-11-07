package boa

import (
	"fmt"
	"github.com/spf13/cobra"
	"reflect"
)

type Optional[T SupportedTypes] struct {
	Name            string
	Short           string
	Env             string
	Default         *T
	Descr           string
	CustomValidator func(T) error
	Positional      bool
	validated       bool
	setByEnv        bool
	setPositionally bool
	valuePtr        any
	parent          *cobra.Command
}

func (f *Optional[T]) wasSetPositionally() bool {
	return f.setPositionally
}

func (f *Optional[T]) GetOrElse(fallback T) T {
	if f.HasValue() {
		return *f.Value()
	} else {
		return fallback
	}
}

func (f *Optional[T]) GetOrElseF(fallback func() T) T {
	if f.HasValue() {
		return *f.Value()
	} else {
		return fallback()
	}
}

func (f *Optional[T]) markSetPositionally() {
	f.setPositionally = true
}

func (f *Optional[T]) isPositional() bool {
	return f.Positional
}

func (f *Optional[T]) SetDefault(val any) {
	f.Default = val.(*T)
}

func (f *Optional[T]) SetEnv(val string) {
	f.Env = val
}

func (f *Optional[T]) SetShort(val string) {
	f.Short = val
}

func (f *Optional[T]) SetName(val string) {
	f.Name = val
}

func (f *Optional[T]) wasSetByEnv() bool {
	return f.setByEnv
}

func (f *Optional[T]) markSetFromEnv() {
	f.setByEnv = true
}

func (f *Optional[T]) Value() *T {
	if !f.validated {
		panic(fmt.Errorf("flag %s was not validated. Cannot use flag before validation. Did you call Validate(..) on the parent struct", f.GetName()))
	}
	if HasValue(f) {
		return f.valuePtr.(*T)
	} else {
		return nil
	}
}

func (f *Optional[T]) markValidated() {
	f.validated = true
}

func (f *Optional[T]) HasValue() bool {
	return HasValue(f)
}

func (f *Optional[T]) setPositional(state bool) {
	f.Positional = state
}

func (f *Optional[T]) setDescription(state string) {
	f.Descr = state
}

func (f *Optional[T]) customValidatorOfPtr() func(any) error {
	return func(val any) error {
		if f.CustomValidator == nil {
			return nil
		}
		return f.CustomValidator(*val.(*T))
	}
}

func (f *Optional[T]) wasSetOnCli() bool {
	if f.parent == nil {
		panic("flag has no parent command. Did you try to .validate() before .ToCmd()?")
	}

	if f.Positional {
		return f.wasSetPositionally()
	} else {
		return f.parent.Flags().Changed(f.Name)
	}
}

func (f *Optional[T]) GetShort() string {
	return f.Short
}

func (f *Optional[T]) GetName() string {
	return f.Name
}

func (f *Optional[T]) GetEnv() string {
	return f.Env
}

func (f *Optional[T]) defaultValuePtr() any {
	return f.Default
}

func (f *Optional[T]) hasDefaultValue() bool {
	return f.Default != nil
}

func (f *Optional[T]) descr() string {
	return f.Descr
}

func (f *Optional[T]) IsRequired() bool {
	return false
}

func (f *Optional[T]) valuePtrF() any {
	return f.valuePtr
}

func (f *Optional[T]) parentCmd() *cobra.Command {
	return f.parent
}

func (f *Optional[T]) defaultValueStr() string {
	if !f.hasDefaultValue() {
		panic("flag has no default value")
	}
	return fmt.Sprintf("%v", *f.Default)
}

func (f *Optional[T]) GetKind() reflect.Kind {
	return f.GetType().Kind()
}

func (f *Optional[T]) GetType() reflect.Type {
	var zero T
	return reflect.TypeOf(zero)
}

func (f *Optional[T]) setParentCmd(cmd *cobra.Command) {
	f.parent = cmd
}

func (f *Optional[T]) setValuePtr(val any) {
	f.valuePtr = val
}
