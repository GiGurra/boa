package boa

import (
	"testing"
	"time"
)

// Tests for time.Duration support

func TestDuration_Required(t *testing.T) {
	type Params struct {
		Timeout Required[time.Duration] `descr:"operation timeout"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if p.Timeout.Value() != 5*time.Second {
				t.Errorf("expected 5s, got %v", p.Timeout.Value())
			}
		}).
		RunArgs([]string{"--timeout", "5s"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestDuration_Optional(t *testing.T) {
	type Params struct {
		Timeout Optional[time.Duration] `descr:"operation timeout"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if !p.Timeout.HasValue() {
				t.Error("expected timeout to have value")
			}
			if *p.Timeout.Value() != 1*time.Hour+30*time.Minute {
				t.Errorf("expected 1h30m, got %v", *p.Timeout.Value())
			}
		}).
		RunArgs([]string{"--timeout", "1h30m"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestDuration_WithDefault(t *testing.T) {
	t.Run("default from struct tag", func(t *testing.T) {
		type Params struct {
			Timeout Required[time.Duration] `descr:"operation timeout" default:"10s"`
		}

		params := Params{}
		wasRun := false

		NewCmdT2("test", &params).
			WithRunFunc(func(p *Params) {
				wasRun = true
				if p.Timeout.Value() != 10*time.Second {
					t.Errorf("expected 10s, got %v", p.Timeout.Value())
				}
			}).
			RunArgs([]string{})

		if !wasRun {
			t.Fatal("run func was not called")
		}
	})

	t.Run("CLI overrides default", func(t *testing.T) {
		type Params struct {
			Timeout Required[time.Duration] `descr:"operation timeout" default:"10s"`
		}

		params := Params{}
		wasRun := false

		NewCmdT2("test", &params).
			WithRunFunc(func(p *Params) {
				wasRun = true
				if p.Timeout.Value() != 30*time.Second {
					t.Errorf("expected 30s, got %v", p.Timeout.Value())
				}
			}).
			RunArgs([]string{"--timeout", "30s"})

		if !wasRun {
			t.Fatal("run func was not called")
		}
	})
}

func TestDuration_ParseFormats(t *testing.T) {
	type Params struct {
		D Required[time.Duration] `descr:"duration"`
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
			params := Params{}
			wasRun := false

			NewCmdT2("test", &params).
				WithRunFunc(func(p *Params) {
					wasRun = true
					if p.D.Value() != tc.expected {
						t.Errorf("expected %v, got %v", tc.expected, p.D.Value())
					}
				}).
				RunArgs([]string{"--d", tc.input})

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

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if p.Timeout != 2*time.Minute {
				t.Errorf("expected 2m, got %v", p.Timeout)
			}
		}).
		RunArgs([]string{"--timeout", "2m"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestDuration_RawWithDefault(t *testing.T) {
	type Params struct {
		Timeout time.Duration `descr:"operation timeout" default:"30s"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if p.Timeout != 30*time.Second {
				t.Errorf("expected 30s, got %v", p.Timeout)
			}
		}).
		RunArgs([]string{})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestDuration_EnvVar(t *testing.T) {
	type Params struct {
		Timeout Required[time.Duration] `descr:"operation timeout" env:"TEST_TIMEOUT_DUR"`
	}

	t.Setenv("TEST_TIMEOUT_DUR", "45s")

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if p.Timeout.Value() != 45*time.Second {
				t.Errorf("expected 45s, got %v", p.Timeout.Value())
			}
		}).
		RunArgs([]string{})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}
