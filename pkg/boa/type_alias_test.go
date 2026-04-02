package boa

import (
	"testing"

	"github.com/spf13/cobra"
)

// Type aliases for basic types
type MyString string
type MyInt int
type MyInt32 int32
type MyInt64 int64
type MyFloat32 float32
type MyFloat64 float64
type MyBool bool

// Test that type aliases work as required fields (plain types)
func TestTypeAlias_Required(t *testing.T) {
	type Config struct {
		Name  MyString  `descr:"A custom string type"`
		Count MyInt     `descr:"A custom int type"`
		Big   MyInt64   `descr:"A custom int64 type"`
		Ratio MyFloat64 `descr:"A custom float64 type"`
	}

	ran := false

	CmdT[Config]{
		Use: "test",
		RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
			ran = true
			if params.Name != "hello" {
				t.Fatalf("expected Name to be 'hello' but got '%s'", params.Name)
			}
			if params.Count != 42 {
				t.Fatalf("expected Count to be 42 but got %d", params.Count)
			}
			if params.Big != 9999999999 {
				t.Fatalf("expected Big to be 9999999999 but got %d", params.Big)
			}
			if params.Ratio != 3.14 {
				t.Fatalf("expected Ratio to be 3.14 but got %f", params.Ratio)
			}
		},
	}.RunArgs([]string{
		"--name", "hello",
		"--count", "42",
		"--big", "9999999999",
		"--ratio", "3.14",
	})

	if !ran {
		t.Fatal("expected command to run")
	}
}

// Test that type aliases work as optional fields
func TestTypeAlias_Optional(t *testing.T) {
	type Config struct {
		Name  MyString  `descr:"A custom string type" optional:"true"`
		Count MyInt     `descr:"A custom int type" optional:"true"`
		Small MyInt32   `descr:"A custom int32 type" optional:"true"`
		Ratio MyFloat32 `descr:"A custom float32 type" optional:"true"`
	}

	ran := false

	CmdT[Config]{
		Use: "test",
		RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
			ran = true
			if params.Name != "test" {
				t.Fatalf("expected Name to be 'test'")
			}
			if params.Count != 100 {
				t.Fatalf("expected Count to be 100")
			}
			if params.Small != 32 {
				t.Fatalf("expected Small to be 32")
			}
			if params.Ratio != 1.5 {
				t.Fatalf("expected Ratio to be 1.5")
			}
		},
	}.RunArgs([]string{
		"--name", "test",
		"--count", "100",
		"--small", "32",
		"--ratio", "1.5",
	})

	if !ran {
		t.Fatal("expected command to run")
	}
}

// Test that bool type aliases work (with explicit default)
func TestTypeAlias_Bool(t *testing.T) {
	type Config struct {
		Flag MyBool `descr:"A custom bool type" default:"false" optional:"true"`
	}

	ran := false

	CmdT[Config]{
		Use: "test",
		RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
			ran = true
			if params.Flag != true {
				t.Fatalf("expected Flag to be true")
			}
		},
	}.RunArgs([]string{"--flag"})

	if !ran {
		t.Fatal("expected command to run")
	}
}

// Test that bool type aliases work WITHOUT explicit default (tests enricher path)
func TestTypeAlias_BoolWithEnricher(t *testing.T) {
	type Config struct {
		Flag MyBool `descr:"A custom bool type" optional:"true"`
	}

	ran := false

	CmdT[Config]{
		Use: "test",
		RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
			ran = true
			if params.Flag != true {
				t.Fatalf("expected Flag to be true")
			}
		},
	}.RunArgs([]string{"--flag"})

	if !ran {
		t.Fatal("expected command to run")
	}
}

// Test that type aliases work with defaults
func TestTypeAlias_WithDefaults(t *testing.T) {
	type Config struct {
		Name  MyString `default:"default-value" optional:"true"`
		Count MyInt    `default:"123" optional:"true"`
	}

	ran := false

	CmdT[Config]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
			ran = true
			if params.Name != "default-value" {
				t.Fatalf("expected Name to be 'default-value' but got '%v'", params.Name)
			}
			if params.Count != 123 {
				t.Fatalf("expected Count to be 123 but got '%v'", params.Count)
			}
		},
	}.RunArgs([]string{})

	if !ran {
		t.Fatal("expected command to run")
	}
}

// Test that type aliases work as raw parameters (not wrapped)
func TestTypeAlias_Raw(t *testing.T) {
	type Config struct {
		Name  MyString  `descr:"A custom string type" optional:"true"`
		Count MyInt     `descr:"A custom int type" optional:"true"`
		Ratio MyFloat64 `descr:"A custom float type" optional:"true"`
	}

	ran := false

	CmdT[Config]{
		Use: "test",
		RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
			ran = true
			if params.Name != "raw-test" {
				t.Fatalf("expected Name to be 'raw-test' but got '%s'", params.Name)
			}
			if params.Count != 999 {
				t.Fatalf("expected Count to be 999 but got %d", params.Count)
			}
			if params.Ratio != 2.718 {
				t.Fatalf("expected Ratio to be 2.718 but got %f", params.Ratio)
			}
		},
	}.RunArgs([]string{
		"--name", "raw-test",
		"--count", "999",
		"--ratio", "2.718",
	})

	if !ran {
		t.Fatal("expected command to run")
	}
}

// TestTypeAlias_RawAllTypes tests all primitive type aliases as raw parameters
func TestTypeAlias_RawAllTypes(t *testing.T) {
	t.Run("all primitive types with CLI values", func(t *testing.T) {
		type Config struct {
			Str   MyString  `descr:"string alias" optional:"true"`
			Int   MyInt     `descr:"int alias" optional:"true"`
			Int32 MyInt32   `descr:"int32 alias" optional:"true"`
			Int64 MyInt64   `descr:"int64 alias" optional:"true"`
			F32   MyFloat32 `descr:"float32 alias" optional:"true"`
			F64   MyFloat64 `descr:"float64 alias" optional:"true"`
			Bool  MyBool    `descr:"bool alias" optional:"true"`
		}

		ran := false

		CmdT[Config]{
			Use: "test",
			RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
				ran = true
				if params.Str != "test-str" {
					t.Errorf("Str: expected 'test-str', got '%s'", params.Str)
				}
				if params.Int != 42 {
					t.Errorf("Int: expected 42, got %d", params.Int)
				}
				if params.Int32 != 32 {
					t.Errorf("Int32: expected 32, got %d", params.Int32)
				}
				if params.Int64 != 64 {
					t.Errorf("Int64: expected 64, got %d", params.Int64)
				}
				if params.F32 != 3.2 {
					t.Errorf("F32: expected 3.2, got %f", params.F32)
				}
				if params.F64 != 6.4 {
					t.Errorf("F64: expected 6.4, got %f", params.F64)
				}
				if params.Bool != true {
					t.Errorf("Bool: expected true, got %v", params.Bool)
				}
			},
		}.RunArgs([]string{
			"--str", "test-str",
			"--int", "42",
			"--int32", "32",
			"--int64", "64",
			"--f32", "3.2",
			"--f64", "6.4",
			"--bool",
		})

		if !ran {
			t.Fatal("expected command to run")
		}
	})

	t.Run("all primitive types with defaults", func(t *testing.T) {
		type Config struct {
			Str   MyString  `descr:"string alias" default:"default-str"`
			Int   MyInt     `descr:"int alias" default:"100"`
			Int32 MyInt32   `descr:"int32 alias" default:"320"`
			Int64 MyInt64   `descr:"int64 alias" default:"640"`
			F32   MyFloat32 `descr:"float32 alias" default:"32.0"`
			F64   MyFloat64 `descr:"float64 alias" default:"64.0"`
			Bool  MyBool    `descr:"bool alias" default:"true"`
		}

		ran := false

		CmdT[Config]{
			Use: "test",
			RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
				ran = true
				if params.Str != "default-str" {
					t.Errorf("Str: expected 'default-str', got '%s'", params.Str)
				}
				if params.Int != 100 {
					t.Errorf("Int: expected 100, got %d", params.Int)
				}
				if params.Int32 != 320 {
					t.Errorf("Int32: expected 320, got %d", params.Int32)
				}
				if params.Int64 != 640 {
					t.Errorf("Int64: expected 640, got %d", params.Int64)
				}
				if params.F32 != 32.0 {
					t.Errorf("F32: expected 32.0, got %f", params.F32)
				}
				if params.F64 != 64.0 {
					t.Errorf("F64: expected 64.0, got %f", params.F64)
				}
				if params.Bool != true {
					t.Errorf("Bool: expected true, got %v", params.Bool)
				}
			},
		}.RunArgs([]string{})

		if !ran {
			t.Fatal("expected command to run")
		}
	})

	t.Run("bool without explicit default uses enricher", func(t *testing.T) {
		type Config struct {
			Flag MyBool `descr:"bool flag without default" optional:"true"`
		}

		ran := false

		CmdT[Config]{
			Use: "test",
			RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
				ran = true
				// When --flag is passed, it becomes true
				if params.Flag != true {
					t.Errorf("expected Flag to be true, got %v", params.Flag)
				}
			},
		}.RunArgs([]string{"--flag"})

		if !ran {
			t.Fatal("expected command to run")
		}
	})

	t.Run("bool without explicit default - default value check", func(t *testing.T) {
		type Config struct {
			Flag MyBool `descr:"bool flag without default" optional:"true"`
		}

		ran := false

		CmdT[Config]{
			Use: "test",
			RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
				ran = true
				// Enricher should have set default to false
				if params.Flag != false {
					t.Errorf("expected Flag default to be false, got %v", params.Flag)
				}
			},
		}.RunArgs([]string{})

		if !ran {
			t.Fatal("expected command to run")
		}
	})
}

// TestTypeAlias_RawSliceTypes tests slice type aliases as raw parameters
func TestTypeAlias_RawSliceTypes(t *testing.T) {
	t.Run("with CLI values", func(t *testing.T) {
		type Config struct {
			Strings MyStringSlice  `descr:"string slice alias" optional:"true"`
			Ints    MyIntSlice     `descr:"int slice alias" optional:"true"`
			Int32s  MyInt32Slice   `descr:"int32 slice alias" optional:"true"`
			Int64s  MyInt64Slice   `descr:"int64 slice alias" optional:"true"`
			F32s    MyFloat32Slice `descr:"float32 slice alias" optional:"true"`
			F64s    MyFloat64Slice `descr:"float64 slice alias" optional:"true"`
		}

		ran := false

		CmdT[Config]{
			Use: "test",
			RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
				ran = true
				if len(params.Strings) != 2 || params.Strings[0] != "a" || params.Strings[1] != "b" {
					t.Errorf("Strings: expected [a,b], got %v", params.Strings)
				}
				if len(params.Ints) != 3 || params.Ints[0] != 1 || params.Ints[1] != 2 || params.Ints[2] != 3 {
					t.Errorf("Ints: expected [1,2,3], got %v", params.Ints)
				}
				if len(params.Int32s) != 2 || params.Int32s[0] != 10 || params.Int32s[1] != 20 {
					t.Errorf("Int32s: expected [10,20], got %v", params.Int32s)
				}
				if len(params.Int64s) != 2 || params.Int64s[0] != 100 || params.Int64s[1] != 200 {
					t.Errorf("Int64s: expected [100,200], got %v", params.Int64s)
				}
				if len(params.F32s) != 2 || params.F32s[0] != 1.1 || params.F32s[1] != 2.2 {
					t.Errorf("F32s: expected [1.1,2.2], got %v", params.F32s)
				}
				if len(params.F64s) != 2 || params.F64s[0] != 11.1 || params.F64s[1] != 22.2 {
					t.Errorf("F64s: expected [11.1,22.2], got %v", params.F64s)
				}
			},
		}.RunArgs([]string{
			"--strings", "a,b",
			"--ints", "1,2,3",
			"--int32s", "10,20",
			"--int64s", "100,200",
			"--f32s", "1.1,2.2",
			"--f64s", "11.1,22.2",
		})

		if !ran {
			t.Fatal("expected command to run")
		}
	})

	t.Run("with defaults", func(t *testing.T) {
		type Config struct {
			Strings MyStringSlice  `descr:"string slice alias" default:"x,y,z"`
			Ints    MyIntSlice     `descr:"int slice alias" default:"7,8,9"`
			F64s    MyFloat64Slice `descr:"float64 slice alias" default:"1.1,2.2"`
		}

		ran := false

		CmdT[Config]{
			Use: "test",
			RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
				ran = true
				if len(params.Strings) != 3 || params.Strings[0] != "x" || params.Strings[1] != "y" || params.Strings[2] != "z" {
					t.Errorf("Strings: expected [x,y,z], got %v", params.Strings)
				}
				if len(params.Ints) != 3 || params.Ints[0] != 7 || params.Ints[1] != 8 || params.Ints[2] != 9 {
					t.Errorf("Ints: expected [7,8,9], got %v", params.Ints)
				}
				if len(params.F64s) != 2 || params.F64s[0] != 1.1 || params.F64s[1] != 2.2 {
					t.Errorf("F64s: expected [1.1,2.2], got %v", params.F64s)
				}
			},
		}.RunArgs([]string{})

		if !ran {
			t.Fatal("expected command to run")
		}
	})
}

// Test type alias slices as plain fields
func TestTypeAlias_Slice(t *testing.T) {
	type MyStringSliceLocal []string
	type MyIntSliceLocal []int

	type Config struct {
		Names  MyStringSliceLocal `descr:"List of names"`
		Values MyIntSliceLocal    `descr:"List of values"`
	}

	ran := false

	CmdT[Config]{
		Use: "test",
		RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
			ran = true
			names := params.Names
			if len(names) != 2 || names[0] != "a" || names[1] != "b" {
				t.Fatalf("expected Names to be [a, b] but got %v", names)
			}
			values := params.Values
			if len(values) != 3 || values[0] != 1 || values[1] != 2 || values[2] != 3 {
				t.Fatalf("expected Values to be [1, 2, 3] but got %v", values)
			}
		},
	}.RunArgs([]string{
		"--names", "a,b",
		"--values", "1,2,3",
	})

	if !ran {
		t.Fatal("expected command to run")
	}
}

// Slice type aliases
type MyStringSlice []string
type MyIntSlice []int
type MyInt32Slice []int32
type MyInt64Slice []int64
type MyFloat32Slice []float32
type MyFloat64Slice []float64

// TestTypeAlias_AllPrimitiveTypes tests all primitive type aliases with optional tag and defaults
func TestTypeAlias_AllPrimitiveTypes(t *testing.T) {
	type Config struct {
		Str   MyString  `descr:"string alias" default:"default-str" optional:"true"`
		Int   MyInt     `descr:"int alias" default:"42" optional:"true"`
		Int32 MyInt32   `descr:"int32 alias" default:"32" optional:"true"`
		Int64 MyInt64   `descr:"int64 alias" default:"64" optional:"true"`
		F32   MyFloat32 `descr:"float32 alias" default:"3.2" optional:"true"`
		F64   MyFloat64 `descr:"float64 alias" default:"6.4" optional:"true"`
		Bool  MyBool    `descr:"bool alias" default:"true" optional:"true"`
	}

	t.Run("with defaults only", func(t *testing.T) {
		ran := false

		CmdT[Config]{
			Use: "test",
			RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
				ran = true
				if params.Str != "default-str" {
					t.Errorf("Str: expected 'default-str', got '%s'", params.Str)
				}
				if params.Int != 42 {
					t.Errorf("Int: expected 42, got %d", params.Int)
				}
				if params.Int32 != 32 {
					t.Errorf("Int32: expected 32, got %d", params.Int32)
				}
				if params.Int64 != 64 {
					t.Errorf("Int64: expected 64, got %d", params.Int64)
				}
				if params.F32 != 3.2 {
					t.Errorf("F32: expected 3.2, got %f", params.F32)
				}
				if params.F64 != 6.4 {
					t.Errorf("F64: expected 6.4, got %f", params.F64)
				}
				if params.Bool != true {
					t.Errorf("Bool: expected true, got %v", params.Bool)
				}
			},
		}.RunArgs([]string{})

		if !ran {
			t.Fatal("expected command to run")
		}
	})

	t.Run("with CLI values", func(t *testing.T) {
		ran := false

		CmdT[Config]{
			Use: "test",
			RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
				ran = true
				if params.Str != "cli-str" {
					t.Errorf("Str: expected 'cli-str', got '%s'", params.Str)
				}
				if params.Int != 100 {
					t.Errorf("Int: expected 100, got %d", params.Int)
				}
				if params.Int32 != 200 {
					t.Errorf("Int32: expected 200, got %d", params.Int32)
				}
				if params.Int64 != 300 {
					t.Errorf("Int64: expected 300, got %d", params.Int64)
				}
				if params.F32 != 1.5 {
					t.Errorf("F32: expected 1.5, got %f", params.F32)
				}
				if params.F64 != 2.5 {
					t.Errorf("F64: expected 2.5, got %f", params.F64)
				}
				if params.Bool != false {
					t.Errorf("Bool: expected false, got %v", params.Bool)
				}
			},
		}.RunArgs([]string{
			"--str", "cli-str",
			"--int", "100",
			"--int32", "200",
			"--int64", "300",
			"--f32", "1.5",
			"--f64", "2.5",
			"--bool=false",
		})

		if !ran {
			t.Fatal("expected command to run")
		}
	})
}

// TestTypeAlias_AllPrimitiveTypesRequired tests all primitive type aliases as required fields
func TestTypeAlias_AllPrimitiveTypesRequired(t *testing.T) {
	type Config struct {
		Str   MyString  `descr:"string alias"`
		Int   MyInt     `descr:"int alias"`
		Int32 MyInt32   `descr:"int32 alias"`
		Int64 MyInt64   `descr:"int64 alias"`
		F32   MyFloat32 `descr:"float32 alias"`
		F64   MyFloat64 `descr:"float64 alias"`
		Bool  MyBool    `descr:"bool alias"`
	}

	ran := false

	CmdT[Config]{
		Use: "test",
		RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
			ran = true
			if params.Str != "test-str" {
				t.Errorf("Str: expected 'test-str', got '%s'", params.Str)
			}
			if params.Int != 111 {
				t.Errorf("Int: expected 111, got %d", params.Int)
			}
			if params.Int32 != 222 {
				t.Errorf("Int32: expected 222, got %d", params.Int32)
			}
			if params.Int64 != 333 {
				t.Errorf("Int64: expected 333, got %d", params.Int64)
			}
			if params.F32 != 4.44 {
				t.Errorf("F32: expected 4.44, got %f", params.F32)
			}
			if params.F64 != 5.55 {
				t.Errorf("F64: expected 5.55, got %f", params.F64)
			}
			if params.Bool != true {
				t.Errorf("Bool: expected true, got %v", params.Bool)
			}
		},
	}.RunArgs([]string{
		"--str", "test-str",
		"--int", "111",
		"--int32", "222",
		"--int64", "333",
		"--f32", "4.44",
		"--f64", "5.55",
		"--bool",
	})

	if !ran {
		t.Fatal("expected command to run")
	}
}

// TestTypeAlias_AllSliceTypes tests all slice type aliases with defaults
func TestTypeAlias_AllSliceTypes(t *testing.T) {
	type Config struct {
		Strings  MyStringSlice  `descr:"string slice alias" default:"a,b,c" optional:"true"`
		Ints     MyIntSlice     `descr:"int slice alias" default:"1,2,3" optional:"true"`
		Int32s   MyInt32Slice   `descr:"int32 slice alias" default:"10,20,30" optional:"true"`
		Int64s   MyInt64Slice   `descr:"int64 slice alias" default:"100,200,300" optional:"true"`
		Float32s MyFloat32Slice `descr:"float32 slice alias" default:"1.1,2.2,3.3" optional:"true"`
		Float64s MyFloat64Slice `descr:"float64 slice alias" default:"11.1,22.2,33.3" optional:"true"`
	}

	t.Run("with defaults only", func(t *testing.T) {
		ran := false

		CmdT[Config]{
			Use: "test",
			RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
				ran = true
				strs := params.Strings
				if len(strs) != 3 || strs[0] != "a" || strs[1] != "b" || strs[2] != "c" {
					t.Errorf("Strings: expected [a,b,c], got %v", strs)
				}
				ints := params.Ints
				if len(ints) != 3 || ints[0] != 1 || ints[1] != 2 || ints[2] != 3 {
					t.Errorf("Ints: expected [1,2,3], got %v", ints)
				}
				int32s := params.Int32s
				if len(int32s) != 3 || int32s[0] != 10 || int32s[1] != 20 || int32s[2] != 30 {
					t.Errorf("Int32s: expected [10,20,30], got %v", int32s)
				}
				int64s := params.Int64s
				if len(int64s) != 3 || int64s[0] != 100 || int64s[1] != 200 || int64s[2] != 300 {
					t.Errorf("Int64s: expected [100,200,300], got %v", int64s)
				}
				f32s := params.Float32s
				if len(f32s) != 3 || f32s[0] != 1.1 || f32s[1] != 2.2 || f32s[2] != 3.3 {
					t.Errorf("Float32s: expected [1.1,2.2,3.3], got %v", f32s)
				}
				f64s := params.Float64s
				if len(f64s) != 3 || f64s[0] != 11.1 || f64s[1] != 22.2 || f64s[2] != 33.3 {
					t.Errorf("Float64s: expected [11.1,22.2,33.3], got %v", f64s)
				}
			},
		}.RunArgs([]string{})

		if !ran {
			t.Fatal("expected command to run")
		}
	})

	t.Run("with CLI values", func(t *testing.T) {
		ran := false

		CmdT[Config]{
			Use: "test",
			RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
				ran = true
				strs := params.Strings
				if len(strs) != 2 || strs[0] != "x" || strs[1] != "y" {
					t.Errorf("Strings: expected [x,y], got %v", strs)
				}
				ints := params.Ints
				if len(ints) != 2 || ints[0] != 9 || ints[1] != 8 {
					t.Errorf("Ints: expected [9,8], got %v", ints)
				}
			},
		}.RunArgs([]string{
			"--strings", "x,y",
			"--ints", "9,8",
		})

		if !ran {
			t.Fatal("expected command to run")
		}
	})
}

// TestTypeAlias_BoolEnricherAllWrappers tests bool type alias with enricher
func TestTypeAlias_BoolEnricherAllWrappers(t *testing.T) {
	t.Run("Optional without default uses enricher", func(t *testing.T) {
		type Config struct {
			Flag MyBool `descr:"bool flag without default" optional:"true"`
		}

		ran := false

		CmdT[Config]{
			Use: "test",
			RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
				ran = true
				// When --flag is passed, it becomes true
				if params.Flag != true {
					t.Errorf("expected Flag to be true, got %v", params.Flag)
				}
			},
		}.RunArgs([]string{"--flag"})

		if !ran {
			t.Fatal("expected command to run")
		}
	})

	t.Run("Optional without default - default value check", func(t *testing.T) {
		type Config struct {
			Flag MyBool `descr:"bool flag without default" optional:"true"`
		}

		ran := false

		CmdT[Config]{
			Use: "test",
			RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
				ran = true
				// Enricher should have set default to false
				if params.Flag != false {
					t.Errorf("expected Flag default to be false, got %v", params.Flag)
				}
			},
		}.RunArgs([]string{})

		if !ran {
			t.Fatal("expected command to run")
		}
	})

	t.Run("Required without default uses enricher", func(t *testing.T) {
		type Config struct {
			Flag MyBool `descr:"bool flag without default"`
		}

		ran := false

		CmdT[Config]{
			Use: "test",
			RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
				ran = true
				// When --flag is passed, it becomes true
				if params.Flag != true {
					t.Errorf("expected Flag to be true, got %v", params.Flag)
				}
			},
		}.RunArgs([]string{"--flag"})

		if !ran {
			t.Fatal("expected command to run")
		}
	})
}

// TestTypeAlias_SetDefaultExplicit tests calling SetDefault explicitly with underlying type pointers
// via HookContext (since the types are now unexported mirrors, we test through the ctx API)
func TestTypeAlias_SetDefaultExplicit(t *testing.T) {
	t.Run("SetDefault via HookContext with underlying type pointers", func(t *testing.T) {
		type Config struct {
			Str  MyString  `descr:"string alias" optional:"true"`
			Int  MyInt     `descr:"int alias" optional:"true"`
			Bool MyBool    `descr:"bool alias" optional:"true"`
			F64  MyFloat64 `descr:"float64 alias" optional:"true"`
		}

		ran := false

		CmdT[Config]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			InitFuncCtx: func(ctx *HookContext, params *Config, cmd *cobra.Command) error {
				strVal := "test-string"
				ctx.GetParam(&params.Str).SetDefault(&strVal)

				intVal := 42
				ctx.GetParam(&params.Int).SetDefault(&intVal)

				boolVal := true
				ctx.GetParam(&params.Bool).SetDefault(&boolVal)

				f64Val := 6.4
				ctx.GetParam(&params.F64).SetDefault(&f64Val)

				return nil
			},
			RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
				ran = true
				if params.Str != "test-string" {
					t.Errorf("strParam: expected 'test-string', got %v", params.Str)
				}
				if params.Int != 42 {
					t.Errorf("intParam: expected 42, got %v", params.Int)
				}
				if params.Bool != true {
					t.Errorf("boolParam: expected true, got %v", params.Bool)
				}
				if params.F64 != 6.4 {
					t.Errorf("f64Param: expected 6.4, got %v", params.F64)
				}
			},
		}.RunArgs([]string{})

		if !ran {
			t.Fatal("expected command to run")
		}
	})
}

// TestTypeAlias_SetDefaultWithDefaultHelper tests using the Default() helper function
func TestTypeAlias_SetDefaultWithDefaultHelper(t *testing.T) {
	t.Run("SetDefault via HookContext with Default helper", func(t *testing.T) {
		type Config struct {
			Bool MyBool   `descr:"bool" optional:"true"`
			Str  MyString `descr:"string" optional:"true"`
			Int  MyInt    `descr:"int" optional:"true"`
		}

		ran := false

		CmdT[Config]{
			Use:         "test",
			ParamEnrich: ParamEnricherName,
			InitFuncCtx: func(ctx *HookContext, params *Config, cmd *cobra.Command) error {
				ctx.GetParam(&params.Bool).SetDefault(Default(false))
				ctx.GetParam(&params.Str).SetDefault(Default("default-via-helper"))
				ctx.GetParam(&params.Int).SetDefault(Default(999))
				return nil
			},
			RunFunc: func(params *Config, cmd *cobra.Command, args []string) {
				ran = true
				if params.Bool != false {
					t.Errorf("boolParam: expected false, got %v", params.Bool)
				}
				if params.Str != "default-via-helper" {
					t.Errorf("strParam: expected 'default-via-helper', got %v", params.Str)
				}
				if params.Int != 999 {
					t.Errorf("intParam: expected 999, got %v", params.Int)
				}
			},
		}.RunArgs([]string{})

		if !ran {
			t.Fatal("expected command to run")
		}
	})
}
