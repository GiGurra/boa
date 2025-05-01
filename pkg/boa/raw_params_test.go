package boa

import (
	"strconv"
	"testing"
)

type RawConfig struct {
	Host   Required[string] `long:"host" env:"HOST"`
	Port   int              `long:"port" env:"PORT" default:"8080"`
	Extra1 string           `long:"extra1" env:"EXTRA1" required:"false"`
	Extra2 string           `long:"extra2" env:"EXTRA2" optional:"true"`
}

func TestRawConfig(t *testing.T) {

	expected := RawConfig{
		Host:   Req("someHost"),
		Port:   12345,
		Extra1: "ex1",
		Extra2: "ex2",
	}

	config := RawConfig{}

	err := NewCmdT2("root", &config).
		WithRawArgs([]string{
			"--host", expected.Host.Value(),
			"--port", strconv.Itoa(expected.Port),
			"--extra1", expected.Extra1,
			"--extra2", expected.Extra2,
		}).
		Validate()

	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	if config.Host.Value() != expected.Host.Value() {
		t.Fatalf("Expected Host: %v, got: %v", expected.Host.Value(), config.Host.Value())
	}
}
