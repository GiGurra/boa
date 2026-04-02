package boa

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// Tests for time.Duration support

func TestDuration_Required(t *testing.T) {
	type Params struct {
		Timeout time.Duration `descr:"operation timeout"`
	}

	wasRun := false

	CmdT[Params]{
		Use: "test",
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			wasRun = true
			if p.Timeout != 5*time.Second {
				t.Errorf("expected 5s, got %v", p.Timeout)
			}
		},
	}.RunArgs([]string{"--timeout", "5s"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestDuration_Optional(t *testing.T) {
	type Params struct {
		Timeout time.Duration `descr:"operation timeout" optional:"true"`
	}

	wasRun := false

	CmdT[Params]{
		Use: "test",
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			wasRun = true
			if !ctx.HasValue(&p.Timeout) {
				t.Error("expected timeout to have value")
			}
			if p.Timeout != 1*time.Hour+30*time.Minute {
				t.Errorf("expected 1h30m, got %v", p.Timeout)
			}
		},
	}.RunArgs([]string{"--timeout", "1h30m"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestDuration_WithDefault(t *testing.T) {
	t.Run("default from struct tag", func(t *testing.T) {
		type Params struct {
			Timeout time.Duration `descr:"operation timeout" default:"10s"`
		}

		wasRun := false

		CmdT[Params]{
			Use: "test",
			RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
				wasRun = true
				if p.Timeout != 10*time.Second {
					t.Errorf("expected 10s, got %v", p.Timeout)
				}
			},
		}.RunArgs([]string{})

		if !wasRun {
			t.Fatal("run func was not called")
		}
	})

	t.Run("CLI overrides default", func(t *testing.T) {
		type Params struct {
			Timeout time.Duration `descr:"operation timeout" default:"10s"`
		}

		wasRun := false

		CmdT[Params]{
			Use: "test",
			RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
				wasRun = true
				if p.Timeout != 30*time.Second {
					t.Errorf("expected 30s, got %v", p.Timeout)
				}
			},
		}.RunArgs([]string{"--timeout", "30s"})

		if !wasRun {
			t.Fatal("run func was not called")
		}
	})
}

func TestDuration_ParseFormats(t *testing.T) {
	type Params struct {
		D time.Duration `descr:"duration"`
	}

	testCases := []struct {
		input    string
		expected time.Duration
	}{
		{"1ns", time.Nanosecond},
		{"1us", time.Microsecond},
		{"1ms", time.Millisecond},
		{"1s", time.Second},
		{"1m", time.Minute},
		{"1h", time.Hour},
		{"1h30m", time.Hour + 30*time.Minute},
		{"2h45m30s", 2*time.Hour + 45*time.Minute + 30*time.Second},
		{"500ms", 500 * time.Millisecond},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			wasRun := false

			CmdT[Params]{
				Use: "test",
				RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
					wasRun = true
					if p.D != tc.expected {
						t.Errorf("expected %v, got %v", tc.expected, p.D)
					}
				},
			}.RunArgs([]string{"--d", tc.input})

			if !wasRun {
				t.Fatal("run func was not called")
			}
		})
	}
}

func TestDuration_Raw(t *testing.T) {
	type Params struct {
		Timeout time.Duration `descr:"operation timeout" optional:"true"`
	}

	wasRun := false

	CmdT[Params]{
		Use: "test",
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			wasRun = true
			if p.Timeout != 2*time.Minute {
				t.Errorf("expected 2m, got %v", p.Timeout)
			}
		},
	}.RunArgs([]string{"--timeout", "2m"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestDuration_RawWithDefault(t *testing.T) {
	type Params struct {
		Timeout time.Duration `descr:"operation timeout" default:"30s"`
	}

	wasRun := false

	CmdT[Params]{
		Use: "test",
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			wasRun = true
			if p.Timeout != 30*time.Second {
				t.Errorf("expected 30s, got %v", p.Timeout)
			}
		},
	}.RunArgs([]string{})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestDuration_EnvVar(t *testing.T) {
	type Params struct {
		Timeout time.Duration `descr:"operation timeout" env:"TEST_TIMEOUT_DUR"`
	}

	t.Setenv("TEST_TIMEOUT_DUR", "45s")

	wasRun := false

	CmdT[Params]{
		Use: "test",
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			wasRun = true
			if p.Timeout != 45*time.Second {
				t.Errorf("expected 45s, got %v", p.Timeout)
			}
		},
	}.RunArgs([]string{})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}
