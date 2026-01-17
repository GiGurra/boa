package boa

import (
	"testing"
)

// Type aliases for basic types
type MyString string
type MyInt int
type MyInt32 int32
type MyInt64 int64
type MyFloat32 float32
type MyFloat64 float64
type MyBool bool

// Test that type aliases work with Required wrapper
func TestTypeAlias_Required(t *testing.T) {
	type Config struct {
		Name  Required[MyString]  `descr:"A custom string type"`
		Count Required[MyInt]     `descr:"A custom int type"`
		Big   Required[MyInt64]   `descr:"A custom int64 type"`
		Ratio Required[MyFloat64] `descr:"A custom float64 type"`
	}

	config := Config{}
	ran := false

	NewCmdT2("test", &config).
		WithRunFunc(func(params *Config) {
			ran = true
			if params.Name.Value() != "hello" {
				t.Fatalf("expected Name to be 'hello' but got '%s'", params.Name.Value())
			}
			if params.Count.Value() != 42 {
				t.Fatalf("expected Count to be 42 but got %d", params.Count.Value())
			}
			if params.Big.Value() != 9999999999 {
				t.Fatalf("expected Big to be 9999999999 but got %d", params.Big.Value())
			}
			if params.Ratio.Value() != 3.14 {
				t.Fatalf("expected Ratio to be 3.14 but got %f", params.Ratio.Value())
			}
		}).
		RunArgs([]string{
			"--name", "hello",
			"--count", "42",
			"--big", "9999999999",
			"--ratio", "3.14",
		})

	if !ran {
		t.Fatal("expected command to run")
	}
}

// Test that type aliases work with Optional wrapper
func TestTypeAlias_Optional(t *testing.T) {
	type Config struct {
		Name  Optional[MyString]  `descr:"A custom string type"`
		Count Optional[MyInt]     `descr:"A custom int type"`
		Small Optional[MyInt32]   `descr:"A custom int32 type"`
		Ratio Optional[MyFloat32] `descr:"A custom float32 type"`
	}

	config := Config{}
	ran := false

	NewCmdT2("test", &config).
		WithRunFunc(func(params *Config) {
			ran = true
			if !params.Name.HasValue() || *params.Name.Value() != "test" {
				t.Fatalf("expected Name to be 'test'")
			}
			if !params.Count.HasValue() || *params.Count.Value() != 100 {
				t.Fatalf("expected Count to be 100")
			}
			if !params.Small.HasValue() || *params.Small.Value() != 32 {
				t.Fatalf("expected Small to be 32")
			}
			if !params.Ratio.HasValue() || *params.Ratio.Value() != 1.5 {
				t.Fatalf("expected Ratio to be 1.5")
			}
		}).
		RunArgs([]string{
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
		Flag Optional[MyBool] `descr:"A custom bool type" default:"false"`
	}

	config := Config{}
	ran := false

	NewCmdT2("test", &config).
		WithRunFunc(func(params *Config) {
			ran = true
			if !params.Flag.HasValue() || *params.Flag.Value() != true {
				t.Fatalf("expected Flag to be true")
			}
		}).
		RunArgs([]string{"--flag"})

	if !ran {
		t.Fatal("expected command to run")
	}
}

// Test that bool type aliases work WITHOUT explicit default (tests enricher path)
func TestTypeAlias_BoolWithEnricher(t *testing.T) {
	type Config struct {
		Flag Optional[MyBool] `descr:"A custom bool type"`
	}

	config := Config{}
	ran := false

	NewCmdT2("test", &config).
		WithRunFunc(func(params *Config) {
			ran = true
			if !params.Flag.HasValue() || *params.Flag.Value() != true {
				t.Fatalf("expected Flag to be true")
			}
		}).
		RunArgs([]string{"--flag"})

	if !ran {
		t.Fatal("expected command to run")
	}
}

// Test that type aliases work with defaults
func TestTypeAlias_WithDefaults(t *testing.T) {
	type Config struct {
		Name  Optional[MyString] `default:"default-value"`
		Count Optional[MyInt]    `default:"123"`
	}

	config := Config{}
	ran := false

	NewCmdT2("test", &config).
		WithRunFunc(func(params *Config) {
			ran = true
			if !params.Name.HasValue() || *params.Name.Value() != "default-value" {
				t.Fatalf("expected Name to be 'default-value' but got '%v'", params.Name.Value())
			}
			if !params.Count.HasValue() || *params.Count.Value() != 123 {
				t.Fatalf("expected Count to be 123 but got '%v'", params.Count.Value())
			}
		}).
		RunArgs([]string{})

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

	config := Config{}
	ran := false

	NewCmdT2("test", &config).
		WithRunFunc(func(params *Config) {
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
		}).
		RunArgs([]string{
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

		config := Config{}
		ran := false

		NewCmdT2("test", &config).
			WithRunFunc(func(params *Config) {
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
			}).
			RunArgs([]string{
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

		config := Config{}
		ran := false

		NewCmdT2("test", &config).
			WithRunFunc(func(params *Config) {
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
			}).
			RunArgs([]string{})

		if !ran {
			t.Fatal("expected command to run")
		}
	})

	t.Run("bool without explicit default uses enricher", func(t *testing.T) {
		type Config struct {
			Flag MyBool `descr:"bool flag without default" optional:"true"`
		}

		config := Config{}
		ran := false

		NewCmdT2("test", &config).
			WithRunFunc(func(params *Config) {
				ran = true
				// When --flag is passed, it becomes true
				if params.Flag != true {
					t.Errorf("expected Flag to be true, got %v", params.Flag)
				}
			}).
			RunArgs([]string{"--flag"})

		if !ran {
			t.Fatal("expected command to run")
		}
	})

	t.Run("bool without explicit default - default value check", func(t *testing.T) {
		type Config struct {
			Flag MyBool `descr:"bool flag without default" optional:"true"`
		}

		config := Config{}
		ran := false

		NewCmdT2("test", &config).
			WithRunFunc(func(params *Config) {
				ran = true
				// Enricher should have set default to false
				if params.Flag != false {
					t.Errorf("expected Flag default to be false, got %v", params.Flag)
				}
			}).
			RunArgs([]string{})

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

		config := Config{}
		ran := false

		NewCmdT2("test", &config).
			WithRunFunc(func(params *Config) {
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
			}).
			RunArgs([]string{
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

		config := Config{}
		ran := false

		NewCmdT2("test", &config).
			WithRunFunc(func(params *Config) {
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
			}).
			RunArgs([]string{})

		if !ran {
			t.Fatal("expected command to run")
		}
	})
}

// Test type alias slices
func TestTypeAlias_Slice(t *testing.T) {
	type MyStringSlice []string
	type MyIntSlice []int

	type Config struct {
		Names  Required[MyStringSlice] `descr:"List of names"`
		Values Required[MyIntSlice]    `descr:"List of values"`
	}

	config := Config{}
	ran := false

	NewCmdT2("test", &config).
		WithRunFunc(func(params *Config) {
			ran = true
			names := params.Names.Value()
			if len(names) != 2 || names[0] != "a" || names[1] != "b" {
				t.Fatalf("expected Names to be [a, b] but got %v", names)
			}
			values := params.Values.Value()
			if len(values) != 3 || values[0] != 1 || values[1] != 2 || values[2] != 3 {
				t.Fatalf("expected Values to be [1, 2, 3] but got %v", values)
			}
		}).
		RunArgs([]string{
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

// TestTypeAlias_AllPrimitiveTypes tests all primitive type aliases with Optional wrapper
// This ensures the connect function properly handles type alias defaults via reflection
func TestTypeAlias_AllPrimitiveTypes(t *testing.T) {
	type Config struct {
		Str   Optional[MyString]  `descr:"string alias" default:"default-str"`
		Int   Optional[MyInt]     `descr:"int alias" default:"42"`
		Int32 Optional[MyInt32]   `descr:"int32 alias" default:"32"`
		Int64 Optional[MyInt64]   `descr:"int64 alias" default:"64"`
		F32   Optional[MyFloat32] `descr:"float32 alias" default:"3.2"`
		F64   Optional[MyFloat64] `descr:"float64 alias" default:"6.4"`
		Bool  Optional[MyBool]    `descr:"bool alias" default:"true"`
	}

	t.Run("with defaults only", func(t *testing.T) {
		config := Config{}
		ran := false

		NewCmdT2("test", &config).
			WithRunFunc(func(params *Config) {
				ran = true
				if *params.Str.Value() != "default-str" {
					t.Errorf("Str: expected 'default-str', got '%s'", *params.Str.Value())
				}
				if *params.Int.Value() != 42 {
					t.Errorf("Int: expected 42, got %d", *params.Int.Value())
				}
				if *params.Int32.Value() != 32 {
					t.Errorf("Int32: expected 32, got %d", *params.Int32.Value())
				}
				if *params.Int64.Value() != 64 {
					t.Errorf("Int64: expected 64, got %d", *params.Int64.Value())
				}
				if *params.F32.Value() != 3.2 {
					t.Errorf("F32: expected 3.2, got %f", *params.F32.Value())
				}
				if *params.F64.Value() != 6.4 {
					t.Errorf("F64: expected 6.4, got %f", *params.F64.Value())
				}
				if *params.Bool.Value() != true {
					t.Errorf("Bool: expected true, got %v", *params.Bool.Value())
				}
			}).
			RunArgs([]string{})

		if !ran {
			t.Fatal("expected command to run")
		}
	})

	t.Run("with CLI values", func(t *testing.T) {
		config := Config{}
		ran := false

		NewCmdT2("test", &config).
			WithRunFunc(func(params *Config) {
				ran = true
				if *params.Str.Value() != "cli-str" {
					t.Errorf("Str: expected 'cli-str', got '%s'", *params.Str.Value())
				}
				if *params.Int.Value() != 100 {
					t.Errorf("Int: expected 100, got %d", *params.Int.Value())
				}
				if *params.Int32.Value() != 200 {
					t.Errorf("Int32: expected 200, got %d", *params.Int32.Value())
				}
				if *params.Int64.Value() != 300 {
					t.Errorf("Int64: expected 300, got %d", *params.Int64.Value())
				}
				if *params.F32.Value() != 1.5 {
					t.Errorf("F32: expected 1.5, got %f", *params.F32.Value())
				}
				if *params.F64.Value() != 2.5 {
					t.Errorf("F64: expected 2.5, got %f", *params.F64.Value())
				}
				if *params.Bool.Value() != false {
					t.Errorf("Bool: expected false, got %v", *params.Bool.Value())
				}
			}).
			RunArgs([]string{
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

// TestTypeAlias_AllPrimitiveTypesRequired tests all primitive type aliases with Required wrapper
func TestTypeAlias_AllPrimitiveTypesRequired(t *testing.T) {
	type Config struct {
		Str   Required[MyString]  `descr:"string alias"`
		Int   Required[MyInt]     `descr:"int alias"`
		Int32 Required[MyInt32]   `descr:"int32 alias"`
		Int64 Required[MyInt64]   `descr:"int64 alias"`
		F32   Required[MyFloat32] `descr:"float32 alias"`
		F64   Required[MyFloat64] `descr:"float64 alias"`
		Bool  Required[MyBool]    `descr:"bool alias"`
	}

	config := Config{}
	ran := false

	NewCmdT2("test", &config).
		WithRunFunc(func(params *Config) {
			ran = true
			if params.Str.Value() != "test-str" {
				t.Errorf("Str: expected 'test-str', got '%s'", params.Str.Value())
			}
			if params.Int.Value() != 111 {
				t.Errorf("Int: expected 111, got %d", params.Int.Value())
			}
			if params.Int32.Value() != 222 {
				t.Errorf("Int32: expected 222, got %d", params.Int32.Value())
			}
			if params.Int64.Value() != 333 {
				t.Errorf("Int64: expected 333, got %d", params.Int64.Value())
			}
			if params.F32.Value() != 4.44 {
				t.Errorf("F32: expected 4.44, got %f", params.F32.Value())
			}
			if params.F64.Value() != 5.55 {
				t.Errorf("F64: expected 5.55, got %f", params.F64.Value())
			}
			if params.Bool.Value() != true {
				t.Errorf("Bool: expected true, got %v", params.Bool.Value())
			}
		}).
		RunArgs([]string{
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

// TestTypeAlias_AllSliceTypes tests all slice type aliases
func TestTypeAlias_AllSliceTypes(t *testing.T) {
	type Config struct {
		Strings  Optional[MyStringSlice]  `descr:"string slice alias" default:"a,b,c"`
		Ints     Optional[MyIntSlice]     `descr:"int slice alias" default:"1,2,3"`
		Int32s   Optional[MyInt32Slice]   `descr:"int32 slice alias" default:"10,20,30"`
		Int64s   Optional[MyInt64Slice]   `descr:"int64 slice alias" default:"100,200,300"`
		Float32s Optional[MyFloat32Slice] `descr:"float32 slice alias" default:"1.1,2.2,3.3"`
		Float64s Optional[MyFloat64Slice] `descr:"float64 slice alias" default:"11.1,22.2,33.3"`
	}

	t.Run("with defaults only", func(t *testing.T) {
		config := Config{}
		ran := false

		NewCmdT2("test", &config).
			WithRunFunc(func(params *Config) {
				ran = true
				strs := *params.Strings.Value()
				if len(strs) != 3 || strs[0] != "a" || strs[1] != "b" || strs[2] != "c" {
					t.Errorf("Strings: expected [a,b,c], got %v", strs)
				}
				ints := *params.Ints.Value()
				if len(ints) != 3 || ints[0] != 1 || ints[1] != 2 || ints[2] != 3 {
					t.Errorf("Ints: expected [1,2,3], got %v", ints)
				}
				int32s := *params.Int32s.Value()
				if len(int32s) != 3 || int32s[0] != 10 || int32s[1] != 20 || int32s[2] != 30 {
					t.Errorf("Int32s: expected [10,20,30], got %v", int32s)
				}
				int64s := *params.Int64s.Value()
				if len(int64s) != 3 || int64s[0] != 100 || int64s[1] != 200 || int64s[2] != 300 {
					t.Errorf("Int64s: expected [100,200,300], got %v", int64s)
				}
				f32s := *params.Float32s.Value()
				if len(f32s) != 3 || f32s[0] != 1.1 || f32s[1] != 2.2 || f32s[2] != 3.3 {
					t.Errorf("Float32s: expected [1.1,2.2,3.3], got %v", f32s)
				}
				f64s := *params.Float64s.Value()
				if len(f64s) != 3 || f64s[0] != 11.1 || f64s[1] != 22.2 || f64s[2] != 33.3 {
					t.Errorf("Float64s: expected [11.1,22.2,33.3], got %v", f64s)
				}
			}).
			RunArgs([]string{})

		if !ran {
			t.Fatal("expected command to run")
		}
	})

	t.Run("with CLI values", func(t *testing.T) {
		config := Config{}
		ran := false

		NewCmdT2("test", &config).
			WithRunFunc(func(params *Config) {
				ran = true
				strs := *params.Strings.Value()
				if len(strs) != 2 || strs[0] != "x" || strs[1] != "y" {
					t.Errorf("Strings: expected [x,y], got %v", strs)
				}
				ints := *params.Ints.Value()
				if len(ints) != 2 || ints[0] != 9 || ints[1] != 8 {
					t.Errorf("Ints: expected [9,8], got %v", ints)
				}
			}).
			RunArgs([]string{
				"--strings", "x,y",
				"--ints", "9,8",
			})

		if !ran {
			t.Fatal("expected command to run")
		}
	})
}

// TestTypeAlias_BoolEnricherAllWrappers tests bool type alias with enricher for both Optional and Required
func TestTypeAlias_BoolEnricherAllWrappers(t *testing.T) {
	t.Run("Optional without default uses enricher", func(t *testing.T) {
		type Config struct {
			Flag Optional[MyBool] `descr:"bool flag without default"`
		}

		config := Config{}
		ran := false

		NewCmdT2("test", &config).
			WithRunFunc(func(params *Config) {
				ran = true
				// Enricher should have set default to false, so HasValue is true
				if !params.Flag.HasValue() {
					t.Error("expected Flag to have value from enricher default")
				}
				// When --flag is passed, it becomes true
				if *params.Flag.Value() != true {
					t.Errorf("expected Flag to be true, got %v", *params.Flag.Value())
				}
			}).
			RunArgs([]string{"--flag"})

		if !ran {
			t.Fatal("expected command to run")
		}
	})

	t.Run("Optional without default - default value check", func(t *testing.T) {
		type Config struct {
			Flag Optional[MyBool] `descr:"bool flag without default"`
		}

		config := Config{}
		ran := false

		NewCmdT2("test", &config).
			WithRunFunc(func(params *Config) {
				ran = true
				// Enricher should have set default to false
				if !params.Flag.HasValue() {
					t.Error("expected Flag to have value from enricher default")
				}
				if *params.Flag.Value() != false {
					t.Errorf("expected Flag default to be false, got %v", *params.Flag.Value())
				}
			}).
			RunArgs([]string{})

		if !ran {
			t.Fatal("expected command to run")
		}
	})

	t.Run("Required without default uses enricher", func(t *testing.T) {
		type Config struct {
			Flag Required[MyBool] `descr:"bool flag without default"`
		}

		config := Config{}
		ran := false

		NewCmdT2("test", &config).
			WithRunFunc(func(params *Config) {
				ran = true
				// When --flag is passed, it becomes true
				if params.Flag.Value() != true {
					t.Errorf("expected Flag to be true, got %v", params.Flag.Value())
				}
			}).
			RunArgs([]string{"--flag"})

		if !ran {
			t.Fatal("expected command to run")
		}
	})
}

// TestTypeAlias_SetDefaultExplicit tests calling SetDefault explicitly with underlying type pointers
// This verifies the reflection-based conversion in SetDefault works for all supported types
func TestTypeAlias_SetDefaultExplicit(t *testing.T) {
	t.Run("Optional SetDefault with underlying type pointers", func(t *testing.T) {
		// Create parameters with type aliases
		var strParam Optional[MyString]
		var intParam Optional[MyInt]
		var int32Param Optional[MyInt32]
		var int64Param Optional[MyInt64]
		var f32Param Optional[MyFloat32]
		var f64Param Optional[MyFloat64]
		var boolParam Optional[MyBool]

		// Call SetDefault with pointers to underlying types (simulating what enrichers do)
		strVal := "test-string"
		strParam.SetDefault(&strVal)

		intVal := 42
		intParam.SetDefault(&intVal)

		int32Val := int32(32)
		int32Param.SetDefault(&int32Val)

		int64Val := int64(64)
		int64Param.SetDefault(&int64Val)

		f32Val := float32(3.2)
		f32Param.SetDefault(&f32Val)

		f64Val := 6.4
		f64Param.SetDefault(&f64Val)

		boolVal := true
		boolParam.SetDefault(&boolVal)

		// Verify defaults were set correctly
		if strParam.Default == nil || *strParam.Default != "test-string" {
			t.Errorf("strParam.Default: expected 'test-string', got %v", strParam.Default)
		}
		if intParam.Default == nil || *intParam.Default != 42 {
			t.Errorf("intParam.Default: expected 42, got %v", intParam.Default)
		}
		if int32Param.Default == nil || *int32Param.Default != 32 {
			t.Errorf("int32Param.Default: expected 32, got %v", int32Param.Default)
		}
		if int64Param.Default == nil || *int64Param.Default != 64 {
			t.Errorf("int64Param.Default: expected 64, got %v", int64Param.Default)
		}
		if f32Param.Default == nil || *f32Param.Default != 3.2 {
			t.Errorf("f32Param.Default: expected 3.2, got %v", f32Param.Default)
		}
		if f64Param.Default == nil || *f64Param.Default != 6.4 {
			t.Errorf("f64Param.Default: expected 6.4, got %v", f64Param.Default)
		}
		if boolParam.Default == nil || *boolParam.Default != true {
			t.Errorf("boolParam.Default: expected true, got %v", boolParam.Default)
		}
	})

	t.Run("Required SetDefault with underlying type pointers", func(t *testing.T) {
		// Create parameters with type aliases
		var strParam Required[MyString]
		var intParam Required[MyInt]
		var int32Param Required[MyInt32]
		var int64Param Required[MyInt64]
		var f32Param Required[MyFloat32]
		var f64Param Required[MyFloat64]
		var boolParam Required[MyBool]

		// Call SetDefault with pointers to underlying types
		strVal := "required-string"
		strParam.SetDefault(&strVal)

		intVal := 100
		intParam.SetDefault(&intVal)

		int32Val := int32(320)
		int32Param.SetDefault(&int32Val)

		int64Val := int64(640)
		int64Param.SetDefault(&int64Val)

		f32Val := float32(32.0)
		f32Param.SetDefault(&f32Val)

		f64Val := 64.0
		f64Param.SetDefault(&f64Val)

		boolVal := false
		boolParam.SetDefault(&boolVal)

		// Verify defaults were set correctly
		if strParam.Default == nil || *strParam.Default != "required-string" {
			t.Errorf("strParam.Default: expected 'required-string', got %v", strParam.Default)
		}
		if intParam.Default == nil || *intParam.Default != 100 {
			t.Errorf("intParam.Default: expected 100, got %v", intParam.Default)
		}
		if int32Param.Default == nil || *int32Param.Default != 320 {
			t.Errorf("int32Param.Default: expected 320, got %v", int32Param.Default)
		}
		if int64Param.Default == nil || *int64Param.Default != 640 {
			t.Errorf("int64Param.Default: expected 640, got %v", int64Param.Default)
		}
		if f32Param.Default == nil || *f32Param.Default != 32.0 {
			t.Errorf("f32Param.Default: expected 32.0, got %v", f32Param.Default)
		}
		if f64Param.Default == nil || *f64Param.Default != 64.0 {
			t.Errorf("f64Param.Default: expected 64.0, got %v", f64Param.Default)
		}
		if boolParam.Default == nil || *boolParam.Default != false {
			t.Errorf("boolParam.Default: expected false, got %v", boolParam.Default)
		}
	})

	t.Run("SetDefault with matching type alias pointers (direct path)", func(t *testing.T) {
		// This tests the direct type assertion path (when types match exactly)
		var strParam Optional[MyString]
		var boolParam Optional[MyBool]

		// Call SetDefault with pointers to the exact type alias
		strVal := MyString("alias-string")
		strParam.SetDefault(&strVal)

		boolVal := MyBool(true)
		boolParam.SetDefault(&boolVal)

		// Verify defaults were set correctly
		if strParam.Default == nil || *strParam.Default != "alias-string" {
			t.Errorf("strParam.Default: expected 'alias-string', got %v", strParam.Default)
		}
		if boolParam.Default == nil || *boolParam.Default != true {
			t.Errorf("boolParam.Default: expected true, got %v", boolParam.Default)
		}
	})
}

// TestTypeAlias_SetDefaultWithDefaultHelper tests using the Default() helper function
// which returns a pointer to the underlying type
func TestTypeAlias_SetDefaultWithDefaultHelper(t *testing.T) {
	t.Run("Optional with Default helper", func(t *testing.T) {
		var boolParam Optional[MyBool]
		var strParam Optional[MyString]
		var intParam Optional[MyInt]

		// Default() returns *T where T is the underlying type
		boolParam.SetDefault(Default(false))
		strParam.SetDefault(Default("default-via-helper"))
		intParam.SetDefault(Default(999))

		if boolParam.Default == nil || *boolParam.Default != false {
			t.Errorf("boolParam.Default: expected false, got %v", boolParam.Default)
		}
		if strParam.Default == nil || *strParam.Default != "default-via-helper" {
			t.Errorf("strParam.Default: expected 'default-via-helper', got %v", strParam.Default)
		}
		if intParam.Default == nil || *intParam.Default != 999 {
			t.Errorf("intParam.Default: expected 999, got %v", intParam.Default)
		}
	})

	t.Run("Required with Default helper", func(t *testing.T) {
		var boolParam Required[MyBool]
		var strParam Required[MyString]
		var intParam Required[MyInt]

		boolParam.SetDefault(Default(true))
		strParam.SetDefault(Default("required-via-helper"))
		intParam.SetDefault(Default(888))

		if boolParam.Default == nil || *boolParam.Default != true {
			t.Errorf("boolParam.Default: expected true, got %v", boolParam.Default)
		}
		if strParam.Default == nil || *strParam.Default != "required-via-helper" {
			t.Errorf("strParam.Default: expected 'required-via-helper', got %v", strParam.Default)
		}
		if intParam.Default == nil || *intParam.Default != 888 {
			t.Errorf("intParam.Default: expected 888, got %v", intParam.Default)
		}
	})
}
