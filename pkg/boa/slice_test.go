package boa

import (
	"testing"
)

// Tests for slice types: []string, []int, []int32, []int64, []float32, []float64

// ==================== []string tests ====================

func TestSliceString_Required(t *testing.T) {
	type Params struct {
		Names Required[[]string] `descr:"List of names"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			names := p.Names.Value()
			if len(names) != 3 {
				t.Errorf("expected 3 names, got %d", len(names))
			}
			if names[0] != "alice" || names[1] != "bob" || names[2] != "carol" {
				t.Errorf("unexpected names: %v", names)
			}
		}).
		RunArgs([]string{"--names", "alice", "--names", "bob", "--names", "carol"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceString_Optional(t *testing.T) {
	type Params struct {
		Tags Optional[[]string] `descr:"List of tags"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if !p.Tags.HasValue() {
				t.Error("expected tags to have value")
			}
			tags := *p.Tags.Value()
			if len(tags) != 2 {
				t.Errorf("expected 2 tags, got %d", len(tags))
			}
		}).
		RunArgs([]string{"--tags", "v1", "--tags", "latest"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceString_Raw(t *testing.T) {
	type Params struct {
		Items []string `descr:"List of items" optional:"true"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if len(p.Items) != 2 {
				t.Errorf("expected 2 items, got %d", len(p.Items))
			}
			if p.Items[0] != "foo" || p.Items[1] != "bar" {
				t.Errorf("unexpected items: %v", p.Items)
			}
		}).
		RunArgs([]string{"--items", "foo", "--items", "bar"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceString_Default(t *testing.T) {
	type Params struct {
		Modes Optional[[]string] `descr:"Operation modes" default:"read,write"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			modes := *p.Modes.Value()
			if len(modes) != 2 {
				t.Errorf("expected 2 modes, got %d", len(modes))
			}
			if modes[0] != "read" || modes[1] != "write" {
				t.Errorf("unexpected modes: %v", modes)
			}
		}).
		RunArgs([]string{})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceString_EnvVar(t *testing.T) {
	type Params struct {
		Hosts Required[[]string] `descr:"List of hosts" env:"TEST_HOSTS"`
	}

	t.Setenv("TEST_HOSTS", "host1,host2,host3")

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			hosts := p.Hosts.Value()
			if len(hosts) != 3 {
				t.Errorf("expected 3 hosts, got %d", len(hosts))
			}
		}).
		RunArgs([]string{})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

// ==================== []int tests ====================

func TestSliceInt_Required(t *testing.T) {
	type Params struct {
		Ports Required[[]int] `descr:"List of ports"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			ports := p.Ports.Value()
			if len(ports) != 3 {
				t.Errorf("expected 3 ports, got %d", len(ports))
			}
			if ports[0] != 80 || ports[1] != 443 || ports[2] != 8080 {
				t.Errorf("unexpected ports: %v", ports)
			}
		}).
		RunArgs([]string{"--ports", "80", "--ports", "443", "--ports", "8080"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceInt_Optional(t *testing.T) {
	type Params struct {
		Numbers Optional[[]int] `descr:"List of numbers"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if !p.Numbers.HasValue() {
				t.Error("expected numbers to have value")
			}
			nums := *p.Numbers.Value()
			if len(nums) != 2 || nums[0] != 1 || nums[1] != 2 {
				t.Errorf("unexpected numbers: %v", nums)
			}
		}).
		RunArgs([]string{"--numbers", "1", "--numbers", "2"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceInt_Raw(t *testing.T) {
	type Params struct {
		Counts []int `descr:"List of counts" optional:"true"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if len(p.Counts) != 3 || p.Counts[0] != 10 || p.Counts[1] != 20 || p.Counts[2] != 30 {
				t.Errorf("unexpected counts: %v", p.Counts)
			}
		}).
		RunArgs([]string{"--counts", "10", "--counts", "20", "--counts", "30"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceInt_Default(t *testing.T) {
	type Params struct {
		Limits Optional[[]int] `descr:"Limits" default:"100,200"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			limits := *p.Limits.Value()
			if len(limits) != 2 || limits[0] != 100 || limits[1] != 200 {
				t.Errorf("unexpected limits: %v", limits)
			}
		}).
		RunArgs([]string{})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

// ==================== []int32 tests ====================

func TestSliceInt32_Required(t *testing.T) {
	type Params struct {
		Values Required[[]int32] `descr:"List of values"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			vals := p.Values.Value()
			if len(vals) != 2 || vals[0] != 100 || vals[1] != 200 {
				t.Errorf("unexpected values: %v", vals)
			}
		}).
		RunArgs([]string{"--values", "100", "--values", "200"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceInt32_Optional(t *testing.T) {
	type Params struct {
		Nums Optional[[]int32] `descr:"List of numbers"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if !p.Nums.HasValue() {
				t.Error("expected nums to have value")
			}
			nums := *p.Nums.Value()
			if len(nums) != 2 || nums[0] != 32 || nums[1] != 64 {
				t.Errorf("unexpected nums: %v", nums)
			}
		}).
		RunArgs([]string{"--nums", "32", "--nums", "64"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceInt32_Raw(t *testing.T) {
	type Params struct {
		Codes []int32 `descr:"List of codes" optional:"true"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if len(p.Codes) != 2 || p.Codes[0] != 1 || p.Codes[1] != 2 {
				t.Errorf("unexpected codes: %v", p.Codes)
			}
		}).
		RunArgs([]string{"--codes", "1", "--codes", "2"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

// ==================== []int64 tests ====================

func TestSliceInt64_Required(t *testing.T) {
	type Params struct {
		Timestamps Required[[]int64] `descr:"List of timestamps"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			ts := p.Timestamps.Value()
			if len(ts) != 2 || ts[0] != 1700000000 || ts[1] != 1700000001 {
				t.Errorf("unexpected timestamps: %v", ts)
			}
		}).
		RunArgs([]string{"--timestamps", "1700000000", "--timestamps", "1700000001"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceInt64_Optional(t *testing.T) {
	type Params struct {
		BigNums Optional[[]int64] `descr:"List of big numbers"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if !p.BigNums.HasValue() {
				t.Error("expected bignums to have value")
			}
			nums := *p.BigNums.Value()
			if len(nums) != 2 || nums[0] != 9223372036854775807 || nums[1] != -9223372036854775808 {
				t.Errorf("unexpected bignums: %v", nums)
			}
		}).
		RunArgs([]string{"--big-nums", "9223372036854775807", "--big-nums", "-9223372036854775808"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceInt64_Raw(t *testing.T) {
	type Params struct {
		Offsets []int64 `descr:"List of offsets" optional:"true"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if len(p.Offsets) != 2 || p.Offsets[0] != 1000 || p.Offsets[1] != 2000 {
				t.Errorf("unexpected offsets: %v", p.Offsets)
			}
		}).
		RunArgs([]string{"--offsets", "1000", "--offsets", "2000"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

// ==================== []float32 tests ====================

func TestSliceFloat32_Required(t *testing.T) {
	type Params struct {
		Rates Required[[]float32] `descr:"List of rates"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			rates := p.Rates.Value()
			if len(rates) != 2 {
				t.Errorf("expected 2 rates, got %d", len(rates))
			}
			if rates[0] != 1.5 || rates[1] != 2.5 {
				t.Errorf("unexpected rates: %v", rates)
			}
		}).
		RunArgs([]string{"--rates", "1.5", "--rates", "2.5"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceFloat32_Optional(t *testing.T) {
	type Params struct {
		Factors Optional[[]float32] `descr:"List of factors"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if !p.Factors.HasValue() {
				t.Error("expected factors to have value")
			}
			factors := *p.Factors.Value()
			if len(factors) != 2 || factors[0] != 0.5 || factors[1] != 1.5 {
				t.Errorf("unexpected factors: %v", factors)
			}
		}).
		RunArgs([]string{"--factors", "0.5", "--factors", "1.5"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceFloat32_Raw(t *testing.T) {
	type Params struct {
		Weights []float32 `descr:"List of weights" optional:"true"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if len(p.Weights) != 2 || p.Weights[0] != 0.1 || p.Weights[1] != 0.9 {
				t.Errorf("unexpected weights: %v", p.Weights)
			}
		}).
		RunArgs([]string{"--weights", "0.1", "--weights", "0.9"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

// ==================== []float64 tests ====================

func TestSliceFloat64_Required(t *testing.T) {
	type Params struct {
		Coords Required[[]float64] `descr:"List of coordinates"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			coords := p.Coords.Value()
			if len(coords) != 2 {
				t.Errorf("expected 2 coords, got %d", len(coords))
			}
			if coords[0] != 59.334591 || coords[1] != 18.063240 {
				t.Errorf("unexpected coords: %v", coords)
			}
		}).
		RunArgs([]string{"--coords", "59.334591", "--coords", "18.063240"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceFloat64_Optional(t *testing.T) {
	type Params struct {
		Prices Optional[[]float64] `descr:"List of prices"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if !p.Prices.HasValue() {
				t.Error("expected prices to have value")
			}
			prices := *p.Prices.Value()
			if len(prices) != 3 || prices[0] != 9.99 || prices[1] != 19.99 || prices[2] != 29.99 {
				t.Errorf("unexpected prices: %v", prices)
			}
		}).
		RunArgs([]string{"--prices", "9.99", "--prices", "19.99", "--prices", "29.99"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceFloat64_Raw(t *testing.T) {
	type Params struct {
		Temps []float64 `descr:"List of temperatures" optional:"true"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if len(p.Temps) != 2 || p.Temps[0] != -40.0 || p.Temps[1] != 100.0 {
				t.Errorf("unexpected temps: %v", p.Temps)
			}
		}).
		RunArgs([]string{"--temps", "-40.0", "--temps", "100.0"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceFloat64_Default(t *testing.T) {
	type Params struct {
		Thresholds Optional[[]float64] `descr:"Thresholds" default:"0.5,0.75,0.9"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			th := *p.Thresholds.Value()
			if len(th) != 3 || th[0] != 0.5 || th[1] != 0.75 || th[2] != 0.9 {
				t.Errorf("unexpected thresholds: %v", th)
			}
		}).
		RunArgs([]string{})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

// ==================== Empty slice tests ====================

func TestSlice_Empty(t *testing.T) {
	type Params struct {
		Items Optional[[]string] `descr:"List of items"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if p.Items.HasValue() {
				t.Error("expected items to not have value when not provided")
			}
		}).
		RunArgs([]string{})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

// ==================== Single element tests ====================

func TestSlice_SingleElement(t *testing.T) {
	type Params struct {
		Values Required[[]int] `descr:"List of values"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			vals := p.Values.Value()
			if len(vals) != 1 || vals[0] != 42 {
				t.Errorf("expected single value [42], got %v", vals)
			}
		}).
		RunArgs([]string{"--values", "42"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}
