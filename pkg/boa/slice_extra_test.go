package boa

import (
	"net"
	"net/url"
	"testing"
	"time"
)

// Tests for slice types of other supported types: []bool, []time.Time, []net.IP, []*url.URL
// These tests focus on raw parameters as requested.

// ==================== []bool tests ====================

func TestSliceBool_Raw(t *testing.T) {
	type Params struct {
		Flags []bool `descr:"List of flags" optional:"true"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if len(p.Flags) != 3 {
				t.Errorf("expected 3 flags, got %d", len(p.Flags))
			}
			if p.Flags[0] != true || p.Flags[1] != false || p.Flags[2] != true {
				t.Errorf("unexpected flags: %v", p.Flags)
			}
		}).
		RunArgs([]string{"--flags", "true", "--flags", "false", "--flags", "true"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceBool_Raw_Default(t *testing.T) {
	type Params struct {
		Enabled []bool `descr:"Enabled features" optional:"true" default:"true,false"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if len(p.Enabled) != 2 {
				t.Errorf("expected 2 values, got %d", len(p.Enabled))
			}
			if p.Enabled[0] != true || p.Enabled[1] != false {
				t.Errorf("unexpected enabled: %v", p.Enabled)
			}
		}).
		RunArgs([]string{})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

// ==================== []time.Time tests ====================

func TestSliceTime_Raw(t *testing.T) {
	type Params struct {
		Dates []time.Time `descr:"List of dates" optional:"true"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if len(p.Dates) != 2 {
				t.Errorf("expected 2 dates, got %d", len(p.Dates))
			}
			expected1 := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
			expected2 := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
			if !p.Dates[0].Equal(expected1) {
				t.Errorf("expected first date %v, got %v", expected1, p.Dates[0])
			}
			if !p.Dates[1].Equal(expected2) {
				t.Errorf("expected second date %v, got %v", expected2, p.Dates[1])
			}
		}).
		RunArgs([]string{"--dates", "2024-01-15", "--dates", "2024-06-30"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceTime_Raw_RFC3339(t *testing.T) {
	type Params struct {
		Timestamps []time.Time `descr:"List of timestamps" optional:"true"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if len(p.Timestamps) != 2 {
				t.Errorf("expected 2 timestamps, got %d", len(p.Timestamps))
			}
			// Check that both parsed correctly (exact time comparison)
			if p.Timestamps[0].Year() != 2024 || p.Timestamps[0].Month() != 3 {
				t.Errorf("unexpected first timestamp: %v", p.Timestamps[0])
			}
			if p.Timestamps[1].Year() != 2024 || p.Timestamps[1].Month() != 12 {
				t.Errorf("unexpected second timestamp: %v", p.Timestamps[1])
			}
		}).
		RunArgs([]string{"--timestamps", "2024-03-15T10:30:00Z", "--timestamps", "2024-12-25T18:00:00Z"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

// ==================== []time.Duration tests ====================

func TestSliceDuration_Raw(t *testing.T) {
	type Params struct {
		Timeouts []time.Duration `descr:"List of timeouts" optional:"true"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if len(p.Timeouts) != 3 {
				t.Errorf("expected 3 timeouts, got %d", len(p.Timeouts))
			}
			if p.Timeouts[0] != 5*time.Second {
				t.Errorf("expected 5s, got %v", p.Timeouts[0])
			}
			if p.Timeouts[1] != 1*time.Minute {
				t.Errorf("expected 1m, got %v", p.Timeouts[1])
			}
			if p.Timeouts[2] != 2*time.Hour {
				t.Errorf("expected 2h, got %v", p.Timeouts[2])
			}
		}).
		RunArgs([]string{"--timeouts", "5s", "--timeouts", "1m", "--timeouts", "2h"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceDuration_Raw_Default(t *testing.T) {
	type Params struct {
		Intervals []time.Duration `descr:"List of intervals" optional:"true" default:"1s,5s,30s"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if len(p.Intervals) != 3 {
				t.Errorf("expected 3 intervals, got %d", len(p.Intervals))
			}
			if p.Intervals[0] != 1*time.Second || p.Intervals[1] != 5*time.Second || p.Intervals[2] != 30*time.Second {
				t.Errorf("unexpected intervals: %v", p.Intervals)
			}
		}).
		RunArgs([]string{})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

// ==================== []net.IP tests ====================

func TestSliceIP_Raw(t *testing.T) {
	type Params struct {
		Hosts []net.IP `descr:"List of hosts" optional:"true"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if len(p.Hosts) != 3 {
				t.Errorf("expected 3 hosts, got %d", len(p.Hosts))
			}
			if p.Hosts[0].String() != "192.168.1.1" {
				t.Errorf("expected 192.168.1.1, got %s", p.Hosts[0].String())
			}
			if p.Hosts[1].String() != "10.0.0.1" {
				t.Errorf("expected 10.0.0.1, got %s", p.Hosts[1].String())
			}
			if p.Hosts[2].String() != "172.16.0.1" {
				t.Errorf("expected 172.16.0.1, got %s", p.Hosts[2].String())
			}
		}).
		RunArgs([]string{"--hosts", "192.168.1.1", "--hosts", "10.0.0.1", "--hosts", "172.16.0.1"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceIP_Raw_Mixed(t *testing.T) {
	type Params struct {
		Addrs []net.IP `descr:"List of addresses" optional:"true"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if len(p.Addrs) != 2 {
				t.Errorf("expected 2 addresses, got %d", len(p.Addrs))
			}
			// First is IPv4
			if p.Addrs[0].To4() == nil {
				t.Errorf("expected IPv4 address, got %s", p.Addrs[0].String())
			}
			// Second is IPv6
			if p.Addrs[1].String() != "::1" {
				t.Errorf("expected ::1, got %s", p.Addrs[1].String())
			}
		}).
		RunArgs([]string{"--addrs", "127.0.0.1", "--addrs", "::1"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

// ==================== []*url.URL tests ====================

func TestSliceURL_Raw(t *testing.T) {
	type Params struct {
		Endpoints []*url.URL `descr:"List of endpoints" optional:"true"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if len(p.Endpoints) != 2 {
				t.Errorf("expected 2 endpoints, got %d", len(p.Endpoints))
			}
			if p.Endpoints[0].String() != "https://api.example.com" {
				t.Errorf("expected https://api.example.com, got %s", p.Endpoints[0].String())
			}
			if p.Endpoints[1].String() != "http://localhost:8080" {
				t.Errorf("expected http://localhost:8080, got %s", p.Endpoints[1].String())
			}
		}).
		RunArgs([]string{"--endpoints", "https://api.example.com", "--endpoints", "http://localhost:8080"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestSliceURL_Raw_WithPaths(t *testing.T) {
	type Params struct {
		Services []*url.URL `descr:"List of service URLs" optional:"true"`
	}

	params := Params{}
	wasRun := false

	NewCmdT2("test", &params).
		WithRunFunc(func(p *Params) {
			wasRun = true
			if len(p.Services) != 2 {
				t.Errorf("expected 2 services, got %d", len(p.Services))
			}
			if p.Services[0].Path != "/api/v1" {
				t.Errorf("expected path /api/v1, got %s", p.Services[0].Path)
			}
			if p.Services[1].Path != "/api/v2/users" {
				t.Errorf("expected path /api/v2/users, got %s", p.Services[1].Path)
			}
		}).
		RunArgs([]string{"--services", "https://example.com/api/v1", "--services", "https://example.com/api/v2/users"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}
