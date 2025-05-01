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
	Extra3 string           `long:"extra2" env:"EXTRA2" required:"true" default:"blah"`
}

func TestRawConfig(t *testing.T) {

	expected := RawConfig{
		Host:   Req("someHost"),
		Port:   12345,
		Extra1: "ex1",
		Extra2: "ex2",
		Extra3: "blah",
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

	if config.Port != expected.Port {
		t.Fatalf("Expected Port: %v, got: %v", expected.Port, config.Port)
	}

	if config.Extra1 != expected.Extra1 {
		t.Fatalf("Expected Extra1: %v, got: %v", expected.Extra1, config.Extra1)
	}

	if config.Extra2 != expected.Extra2 {
		t.Fatalf("Expected Extra2: %v, got: %v", expected.Extra2, config.Extra2)
	}

	if config.Extra3 != expected.Extra3 {
		t.Fatalf("Expected Extra3: %v, got: %v", expected.Extra3, config.Extra3)
	}
}
