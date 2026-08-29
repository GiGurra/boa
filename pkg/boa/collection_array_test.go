package boa

import (
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCollectionArray_StringOccurrenceSemantics(t *testing.T) {
	type Params struct {
		Labels []string `long:"label" collection:"array" optional:"true"`
	}

	var got []string
	err := (CmdT[Params]{
		Use: "test",
		RunFunc: func(p *Params, _ *cobra.Command, _ []string) {
			got = append([]string(nil), p.Labels...)
		},
	}).RunArgsE([]string{"--label", "a,b", "--label", "second", "--label="})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"a,b", "second", ""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected each occurrence to remain opaque: want %#v, got %#v", want, got)
	}
}

func TestCollectionArray_StringDefaultAndEnvKeepCSVSemantics(t *testing.T) {
	type Params struct {
		Labels []string `long:"label" collection:"array" default:"default,values" env:"BOA_ARRAY_LABELS" optional:"true"`
	}

	t.Run("default is CSV when CLI is absent", func(t *testing.T) {
		var got []string
		err := (CmdT[Params]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFunc: func(p *Params, _ *cobra.Command, _ []string) {
				got = append([]string(nil), p.Labels...)
			},
		}).RunArgsE(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if want := []string{"default", "values"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("want %#v, got %#v", want, got)
		}
	})

	t.Run("first CLI occurrence replaces default", func(t *testing.T) {
		var got []string
		err := (CmdT[Params]{
			Use: "test",
			RunFunc: func(p *Params, _ *cobra.Command, _ []string) {
				got = append([]string(nil), p.Labels...)
			},
		}).RunArgsE([]string{"--label", "cli,opaque", "--label", "next"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if want := []string{"cli,opaque", "next"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("want %#v, got %#v", want, got)
		}
	})

	t.Run("environment remains CSV", func(t *testing.T) {
		t.Setenv("BOA_ARRAY_LABELS", "env,values")
		var got []string
		err := (CmdT[Params]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			RunFunc: func(p *Params, _ *cobra.Command, _ []string) {
				got = append([]string(nil), p.Labels...)
			},
		}).RunArgsE(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if want := []string{"env", "values"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("want %#v, got %#v", want, got)
		}
	})
}

func TestCollectionArray_StringSliceAlias(t *testing.T) {
	type Labels []string
	type Params struct {
		Labels Labels `long:"label" collection:"array" optional:"true"`
	}

	var got Labels
	err := (CmdT[Params]{
		Use: "test",
		RunFunc: func(p *Params, _ *cobra.Command, _ []string) {
			got = append(Labels(nil), p.Labels...)
		},
	}).RunArgsE([]string{"--label", "a,b", "--label", "c"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := Labels{"a,b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %#v, got %#v", want, got)
	}
}

func TestCollectionArray_NumericSlicesUseOneScalarPerOccurrence(t *testing.T) {
	type Params struct {
		Numbers []int `collection:"array" default:"9,10" optional:"true"`
	}

	var got []int
	err := (CmdT[Params]{
		Use: "test",
		RunFunc: func(p *Params, _ *cobra.Command, _ []string) {
			got = append([]int(nil), p.Numbers...)
		},
	}).RunArgsE([]string{"--numbers", "1", "--numbers", "2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []int{1, 2}; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %#v, got %#v", want, got)
	}

	err = (CmdT[Params]{Use: "test", RunFunc: func(*Params, *cobra.Command, []string) {}}).
		RunArgsE([]string{"--numbers", "1,2"})
	if err == nil || !strings.Contains(err.Error(), "invalid syntax") {
		t.Fatalf("expected comma-containing numeric occurrence to be parsed as one invalid scalar, got %v", err)
	}
}

func TestCollectionArray_ProgrammaticConfiguration(t *testing.T) {
	type Params struct {
		Labels []string `optional:"true"`
	}

	var got []string
	err := (CmdT[Params]{
		Use: "test",
		InitFuncCtx: func(ctx *HookContext, p *Params, _ *cobra.Command) error {
			GetParamT(ctx, &p.Labels).SetCollection(CollectionArray)
			return nil
		},
		RunFunc: func(p *Params, _ *cobra.Command, _ []string) {
			got = append([]string(nil), p.Labels...)
		},
	}).RunArgsE([]string{"--labels", "a,b"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []string{"a,b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("want %#v, got %#v", want, got)
	}
}

func TestCollectionArray_PersistentFlagIsInherited(t *testing.T) {
	type Params struct {
		Labels []string `long:"label" collection:"array" persistent:"true" optional:"true"`
	}

	params := Params{}
	leaf := &cobra.Command{Use: "leaf", Run: func(*cobra.Command, []string) {}}
	root := (CmdT[Params]{
		Use:     "root",
		Params:  &params,
		SubCmds: []*cobra.Command{leaf},
	}).ToCobra()
	root.SetArgs([]string{"leaf", "--label", "a,b", "--label", "second"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []string{"a,b", "second"}; !reflect.DeepEqual(params.Labels, want) {
		t.Fatalf("want %#v, got %#v", want, params.Labels)
	}
}

func TestCollectionTagRejectsInvalidUses(t *testing.T) {
	t.Run("unknown mode", func(t *testing.T) {
		type Params struct {
			Labels []string `collection:"unknown" optional:"true"`
		}
		err := (CmdT[Params]{Use: "test", RunFunc: func(*Params, *cobra.Command, []string) {}}).RunArgsE(nil)
		if err == nil || !strings.Contains(err.Error(), "invalid collection mode") {
			t.Fatalf("expected invalid collection mode error, got %v", err)
		}
	})

	t.Run("non-slice", func(t *testing.T) {
		type Params struct {
			Label string `collection:"array" optional:"true"`
		}
		err := (CmdT[Params]{Use: "test", RunFunc: func(*Params, *cobra.Command, []string) {}}).RunArgsE(nil)
		if err == nil || !strings.Contains(err.Error(), "only slice fields") {
			t.Fatalf("expected non-slice collection error, got %v", err)
		}
	})
}
