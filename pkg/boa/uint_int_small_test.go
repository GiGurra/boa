package boa

import (
	"math"
	"testing"

	"github.com/spf13/cobra"
)

// Tests for unsigned integer types (uint, uint8, uint16, uint32, uint64) and the
// small signed integer types (int8, int16). Prior to support being added these
// fields were silently dropped by isSupportedType, so the corresponding CLI flag
// was never registered — every test below would fail with "unknown flag: --..."
// or with the field staying at its zero value.

// ==================== uint ====================

func TestUint_CLI(t *testing.T) {
	type P struct {
		Count uint `descr:"count"`
	}
	wasRun := false
	if err := (CmdT[P]{
		Use: "test",
		RunFunc: func(p *P, cmd *cobra.Command, args []string) {
			wasRun = true
			if p.Count != 42 {
				t.Errorf("expected 42 got %d", p.Count)
			}
		},
	}).RunArgsE([]string{"--count", "42"}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestUint_Default(t *testing.T) {
	type P struct {
		Count uint `descr:"count" default:"7" optional:"true"`
	}
	wasRun := false
	if err := (CmdT[P]{
		Use: "test",
		RunFunc: func(p *P, cmd *cobra.Command, args []string) {
			wasRun = true
			if p.Count != 7 {
				t.Errorf("expected 7 got %d", p.Count)
			}
		},
	}).RunArgsE([]string{}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !wasRun {
		t.Fatal("run func was not called")
	}
}

// ==================== uint8 ====================

func TestUint8_CLI(t *testing.T) {
	type P struct {
		Byte uint8 `descr:"byte"`
	}
	wasRun := false
	if err := (CmdT[P]{
		Use: "test",
		RunFunc: func(p *P, cmd *cobra.Command, args []string) {
			wasRun = true
			if p.Byte != 255 {
				t.Errorf("expected 255 got %d", p.Byte)
			}
		},
	}).RunArgsE([]string{"--byte", "255"}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestUint8_OutOfRange(t *testing.T) {
	type P struct {
		Byte uint8 `descr:"byte"`
	}
	err := (CmdT[P]{
		Use:     "test",
		RunFunc: func(p *P, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--byte", "256"})
	if err == nil {
		t.Fatal("expected error for value out of uint8 range")
	}
}

// ==================== uint16 ====================

func TestUint16_CLI(t *testing.T) {
	type P struct {
		Port uint16 `descr:"port"`
	}
	wasRun := false
	if err := (CmdT[P]{
		Use: "test",
		RunFunc: func(p *P, cmd *cobra.Command, args []string) {
			wasRun = true
			if p.Port != 65535 {
				t.Errorf("expected 65535 got %d", p.Port)
			}
		},
	}).RunArgsE([]string{"--port", "65535"}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !wasRun {
		t.Fatal("run func was not called")
	}
}

// ==================== uint32 ====================

func TestUint32_CLI(t *testing.T) {
	type P struct {
		ID uint32 `descr:"id"`
	}
	wasRun := false
	if err := (CmdT[P]{
		Use: "test",
		RunFunc: func(p *P, cmd *cobra.Command, args []string) {
			wasRun = true
			if p.ID != math.MaxUint32 {
				t.Errorf("expected %d got %d", uint32(math.MaxUint32), p.ID)
			}
		},
	}).RunArgsE([]string{"--id", "4294967295"}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !wasRun {
		t.Fatal("run func was not called")
	}
}

// ==================== uint64 ====================

func TestUint64_CLI(t *testing.T) {
	type P struct {
		Port uint64 `descr:"port"`
	}
	wasRun := false
	if err := (CmdT[P]{
		Use: "test",
		RunFunc: func(p *P, cmd *cobra.Command, args []string) {
			wasRun = true
			if p.Port != 8080 {
				t.Errorf("expected 8080 got %d", p.Port)
			}
		},
	}).RunArgsE([]string{"--port", "8080"}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestUint64_LargeValue(t *testing.T) {
	type P struct {
		N uint64 `descr:"n"`
	}
	wasRun := false
	if err := (CmdT[P]{
		Use: "test",
		RunFunc: func(p *P, cmd *cobra.Command, args []string) {
			wasRun = true
			if p.N != math.MaxUint64 {
				t.Errorf("expected MaxUint64 got %d", p.N)
			}
		},
	}).RunArgsE([]string{"--n", "18446744073709551615"}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestUint64_EnvVar(t *testing.T) {
	type P struct {
		N uint64 `descr:"n" env:"TEST_UINT64_N"`
	}
	t.Setenv("TEST_UINT64_N", "12345")
	wasRun := false
	if err := (CmdT[P]{
		Use: "test",
		RunFunc: func(p *P, cmd *cobra.Command, args []string) {
			wasRun = true
			if p.N != 12345 {
				t.Errorf("expected 12345 got %d", p.N)
			}
		},
	}).RunArgsE([]string{}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !wasRun {
		t.Fatal("run func was not called")
	}
}

// ==================== int8 ====================

func TestInt8_CLI(t *testing.T) {
	type P struct {
		N int8 `descr:"n"`
	}
	wasRun := false
	if err := (CmdT[P]{
		Use: "test",
		RunFunc: func(p *P, cmd *cobra.Command, args []string) {
			wasRun = true
			if p.N != -128 {
				t.Errorf("expected -128 got %d", p.N)
			}
		},
	}).RunArgsE([]string{"--n", "-128"}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !wasRun {
		t.Fatal("run func was not called")
	}
}

// ==================== int16 ====================

func TestInt16_CLI(t *testing.T) {
	type P struct {
		N int16 `descr:"n"`
	}
	wasRun := false
	if err := (CmdT[P]{
		Use: "test",
		RunFunc: func(p *P, cmd *cobra.Command, args []string) {
			wasRun = true
			if p.N != 32767 {
				t.Errorf("expected 32767 got %d", p.N)
			}
		},
	}).RunArgsE([]string{"--n", "32767"}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !wasRun {
		t.Fatal("run func was not called")
	}
}

// ==================== slice variants ====================

func TestSliceUint_CLI(t *testing.T) {
	type P struct {
		Vals []uint `descr:"vals"`
	}
	wasRun := false
	if err := (CmdT[P]{
		Use: "test",
		RunFunc: func(p *P, cmd *cobra.Command, args []string) {
			wasRun = true
			if len(p.Vals) != 3 || p.Vals[0] != 1 || p.Vals[1] != 2 || p.Vals[2] != 3 {
				t.Errorf("unexpected vals: %v", p.Vals)
			}
		},
	}).RunArgsE([]string{"--vals", "1", "--vals", "2", "--vals", "3"}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceUint8_CLI(t *testing.T) {
	type P struct {
		Bytes []uint8 `descr:"bytes"`
	}
	wasRun := false
	if err := (CmdT[P]{
		Use: "test",
		RunFunc: func(p *P, cmd *cobra.Command, args []string) {
			wasRun = true
			if len(p.Bytes) != 2 || p.Bytes[0] != 0 || p.Bytes[1] != 255 {
				t.Errorf("unexpected bytes: %v", p.Bytes)
			}
		},
	}).RunArgsE([]string{"--bytes", "0", "--bytes", "255"}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceUint16_CLI(t *testing.T) {
	type P struct {
		Ports []uint16 `descr:"ports"`
	}
	wasRun := false
	if err := (CmdT[P]{
		Use: "test",
		RunFunc: func(p *P, cmd *cobra.Command, args []string) {
			wasRun = true
			if len(p.Ports) != 2 || p.Ports[0] != 80 || p.Ports[1] != 65535 {
				t.Errorf("unexpected ports: %v", p.Ports)
			}
		},
	}).RunArgsE([]string{"--ports", "80", "--ports", "65535"}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceUint32_CLI(t *testing.T) {
	type P struct {
		Vals []uint32 `descr:"vals"`
	}
	wasRun := false
	if err := (CmdT[P]{
		Use: "test",
		RunFunc: func(p *P, cmd *cobra.Command, args []string) {
			wasRun = true
			if len(p.Vals) != 2 || p.Vals[0] != 1 || p.Vals[1] != math.MaxUint32 {
				t.Errorf("unexpected vals: %v", p.Vals)
			}
		},
	}).RunArgsE([]string{"--vals", "1", "--vals", "4294967295"}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceUint64_CLI(t *testing.T) {
	type P struct {
		Vals []uint64 `descr:"vals"`
	}
	wasRun := false
	if err := (CmdT[P]{
		Use: "test",
		RunFunc: func(p *P, cmd *cobra.Command, args []string) {
			wasRun = true
			if len(p.Vals) != 2 || p.Vals[0] != 1 || p.Vals[1] != math.MaxUint64 {
				t.Errorf("unexpected vals: %v", p.Vals)
			}
		},
	}).RunArgsE([]string{"--vals", "1", "--vals", "18446744073709551615"}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceInt8_CLI(t *testing.T) {
	type P struct {
		Vals []int8 `descr:"vals"`
	}
	wasRun := false
	if err := (CmdT[P]{
		Use: "test",
		RunFunc: func(p *P, cmd *cobra.Command, args []string) {
			wasRun = true
			if len(p.Vals) != 2 || p.Vals[0] != -128 || p.Vals[1] != 127 {
				t.Errorf("unexpected vals: %v", p.Vals)
			}
		},
	}).RunArgsE([]string{"--vals", "-128", "--vals", "127"}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceInt16_CLI(t *testing.T) {
	type P struct {
		Vals []int16 `descr:"vals"`
	}
	wasRun := false
	if err := (CmdT[P]{
		Use: "test",
		RunFunc: func(p *P, cmd *cobra.Command, args []string) {
			wasRun = true
			if len(p.Vals) != 2 || p.Vals[0] != -32768 || p.Vals[1] != 32767 {
				t.Errorf("unexpected vals: %v", p.Vals)
			}
		},
	}).RunArgsE([]string{"--vals", "-32768", "--vals", "32767"}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !wasRun {
		t.Fatal("run func was not called")
	}
}

// ==================== type aliases ====================

type Port uint16
type UserID uint64

type CustomUintParams struct {
	Port   Port   `descr:"port"`
	UserID UserID `descr:"user id"`
}

func TestCustomUintTypeAliases(t *testing.T) {
	wasRun := false
	if err := (CmdT[CustomUintParams]{
		Use: "test",
		RunFunc: func(p *CustomUintParams, cmd *cobra.Command, args []string) {
			wasRun = true
			if p.Port != 8443 {
				t.Errorf("expected 8443 got %d", p.Port)
			}
			if p.UserID != 999999 {
				t.Errorf("expected 999999 got %d", p.UserID)
			}
		},
	}).RunArgsE([]string{"--port", "8443", "--user-id", "999999"}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !wasRun {
		t.Fatal("run func was not called")
	}
}

// ==================== slice: CSV / env / default coverage ====================

// Locks in typedIntSliceValue.Set's CSV-split path when a single --flag carries
// multiple comma-separated values (the repeated-flag cases above only hit the
// "first value" branch).
func TestSliceUint16_CSVSingleFlag(t *testing.T) {
	type P struct {
		Vals []uint16 `descr:"vals"`
	}
	wasRun := false
	if err := (CmdT[P]{
		Use: "test",
		RunFunc: func(p *P, cmd *cobra.Command, args []string) {
			wasRun = true
			if len(p.Vals) != 3 || p.Vals[0] != 1 || p.Vals[1] != 2 || p.Vals[2] != 3 {
				t.Errorf("unexpected vals: %v", p.Vals)
			}
		},
	}).RunArgsE([]string{"--vals", "1,2,3"}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !wasRun {
		t.Fatal("run func was not called")
	}
}

// Locks in the parseSliceWith path used by env-var ingestion for slice types
// without a native pflag slice flag.
func TestSliceUint16_EnvVar(t *testing.T) {
	type P struct {
		Vals []uint16 `descr:"vals" env:"TEST_UINT16_VALS"`
	}
	t.Setenv("TEST_UINT16_VALS", "10,20,30")
	wasRun := false
	if err := (CmdT[P]{
		Use: "test",
		RunFunc: func(p *P, cmd *cobra.Command, args []string) {
			wasRun = true
			if len(p.Vals) != 3 || p.Vals[0] != 10 || p.Vals[1] != 20 || p.Vals[2] != 30 {
				t.Errorf("unexpected vals: %v", p.Vals)
			}
		},
	}).RunArgsE([]string{}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !wasRun {
		t.Fatal("run func was not called")
	}
}

// Locks in the default-tag path through parseSliceWith.
func TestSliceUint8_Default(t *testing.T) {
	type P struct {
		Bytes []uint8 `descr:"bytes" default:"0,127,255" optional:"true"`
	}
	wasRun := false
	if err := (CmdT[P]{
		Use: "test",
		RunFunc: func(p *P, cmd *cobra.Command, args []string) {
			wasRun = true
			if len(p.Bytes) != 3 || p.Bytes[0] != 0 || p.Bytes[1] != 127 || p.Bytes[2] != 255 {
				t.Errorf("unexpected bytes: %v", p.Bytes)
			}
		},
	}).RunArgsE([]string{}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !wasRun {
		t.Fatal("run func was not called")
	}
}

// Same as above but with the bracketed [a,b,c] form parseSliceWith also accepts.
func TestSliceInt16_DefaultBracketed(t *testing.T) {
	type P struct {
		Vals []int16 `descr:"vals" default:"[-1,0,1]" optional:"true"`
	}
	wasRun := false
	if err := (CmdT[P]{
		Use: "test",
		RunFunc: func(p *P, cmd *cobra.Command, args []string) {
			wasRun = true
			if len(p.Vals) != 3 || p.Vals[0] != -1 || p.Vals[1] != 0 || p.Vals[2] != 1 {
				t.Errorf("unexpected vals: %v", p.Vals)
			}
		},
	}).RunArgsE([]string{}); err != nil {
		t.Fatalf("RunArgsE: %v", err)
	}
	if !wasRun {
		t.Fatal("run func was not called")
	}
}
