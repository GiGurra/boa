package boa

import (
	"net"
	"testing"

	"github.com/spf13/cobra"
)

// Tests for net.IP support

func TestNetIP_Required(t *testing.T) {
	type Params struct {
		Host net.IP `descr:"host IP address"`
	}

	wasRun := false

	CmdT[Params]{
		Use: "test",
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			wasRun = true
			expected := net.ParseIP("192.168.1.1")
			if !p.Host.Equal(expected) {
				t.Errorf("expected %v, got %v", expected, p.Host)
			}
		},
	}.RunArgs([]string{"--host", "192.168.1.1"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestNetIP_Optional(t *testing.T) {
	type Params struct {
		Host net.IP `descr:"host IP address" optional:"true"`
	}

	wasRun := false

	CmdT[Params]{
		Use: "test",
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			wasRun = true
			if !ctx.HasValue(&p.Host) {
				t.Error("expected host to have value")
			}
			expected := net.ParseIP("10.0.0.1")
			if !p.Host.Equal(expected) {
				t.Errorf("expected %v, got %v", expected, p.Host)
			}
		},
	}.RunArgs([]string{"--host", "10.0.0.1"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestNetIP_IPv6(t *testing.T) {
	type Params struct {
		Host net.IP `descr:"host IP address"`
	}

	wasRun := false

	CmdT[Params]{
		Use: "test",
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			wasRun = true
			expected := net.ParseIP("::1")
			if !p.Host.Equal(expected) {
				t.Errorf("expected %v, got %v", expected, p.Host)
			}
		},
	}.RunArgs([]string{"--host", "::1"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestNetIP_IPv6Full(t *testing.T) {
	type Params struct {
		Host net.IP `descr:"host IP address"`
	}

	wasRun := false

	CmdT[Params]{
		Use: "test",
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			wasRun = true
			expected := net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334")
			if !p.Host.Equal(expected) {
				t.Errorf("expected %v, got %v", expected, p.Host)
			}
		},
	}.RunArgs([]string{"--host", "2001:0db8:85a3:0000:0000:8a2e:0370:7334"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestNetIP_Raw(t *testing.T) {
	type Params struct {
		Host net.IP `descr:"host IP address" optional:"true"`
	}

	wasRun := false

	CmdT[Params]{
		Use: "test",
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			wasRun = true
			expected := net.ParseIP("172.16.0.1")
			if !p.Host.Equal(expected) {
				t.Errorf("expected %v, got %v", expected, p.Host)
			}
		},
	}.RunArgs([]string{"--host", "172.16.0.1"})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestNetIP_EnvVar(t *testing.T) {
	type Params struct {
		Host net.IP `descr:"host IP address" env:"TEST_HOST_IP"`
	}

	t.Setenv("TEST_HOST_IP", "8.8.8.8")

	wasRun := false

	CmdT[Params]{
		Use: "test",
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			wasRun = true
			expected := net.ParseIP("8.8.8.8")
			if !p.Host.Equal(expected) {
				t.Errorf("expected %v, got %v", expected, p.Host)
			}
		},
	}.RunArgs([]string{})

	if !wasRun {
		t.Fatal("run func was not called")
	}
}

func TestNetIP_ParseFormats(t *testing.T) {
	type Params struct {
		Addr net.IP `descr:"IP address"`
	}

	testCases := []struct {
		input    string
		expected net.IP
	}{
		{"127.0.0.1", net.ParseIP("127.0.0.1")},
		{"0.0.0.0", net.ParseIP("0.0.0.0")},
		{"255.255.255.255", net.ParseIP("255.255.255.255")},
		{"::1", net.ParseIP("::1")},
		{"::", net.ParseIP("::")},
		{"fe80::1", net.ParseIP("fe80::1")},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			wasRun := false

			CmdT[Params]{
				Use: "test",
				RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
					wasRun = true
					if !p.Addr.Equal(tc.expected) {
						t.Errorf("expected %v, got %v", tc.expected, p.Addr)
					}
				},
			}.RunArgs([]string{"--addr", tc.input})

			if !wasRun {
				t.Fatal("run func was not called")
			}
		})
	}
}
