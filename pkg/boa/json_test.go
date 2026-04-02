package boa

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

type AppConfig struct {
	Host                string `env:"HOST"`
	Port                int    `env:"PORT" default:"8080"`
	KafkaCredentials    string `env:"KAFKA_CREDENTIALS" default:"" optional:"true"`
	KafkaNilCredentials string `env:"KAFKA_NIL_CREDENTIALS" optional:"true"`
}

func TestJsonSerialization(t *testing.T) {

	data := AppConfig{
		Host:                "someHost",
		Port:                12345,
		KafkaCredentials:    "someCredentials",
		KafkaNilCredentials: "",
	}

	// Serialize to JSON
	serialized, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Errorf("json.MarshalIndent() error = %v", err)
	}
	t.Logf("Serialized: %s", serialized)

	// Deserialize from JSON
	var deserialized AppConfig
	if err := json.Unmarshal(serialized, &deserialized); err != nil {
		t.Errorf("json.Unmarshal() error = %v", err)
	}

	// Check if the deserialized data matches the original
	if data.Port != deserialized.Port {
		t.Errorf("Port mismatch: got %d, want %d", deserialized.Port, data.Port)
	}

	if data.Host != deserialized.Host {
		t.Errorf("Host mismatch: got %s, want %s", deserialized.Host, data.Host)
	}

	if data.KafkaCredentials != deserialized.KafkaCredentials {
		t.Errorf("KafkaCredentials mismatch: got %s, want %s", deserialized.KafkaCredentials, data.KafkaCredentials)
	}
}

type EmbeddedConfigStruct struct {
	Foobar string `default:"foobar"`
	AppConfig
}

type EmbeddedConfigStructPointer struct {
	Foobar string `default:"foobar"`
	*AppConfig
}

func TestJsonSerializationEmbeddedStruct(t *testing.T) {

	data :=
		EmbeddedConfigStruct{
			Foobar: "foobar",
			AppConfig: AppConfig{
				Host:                "someHost",
				Port:                12345,
				KafkaCredentials:    "someCredentials",
				KafkaNilCredentials: "",
			},
		}

	// Serialize to JSON
	serialized, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Errorf("json.MarshalIndent() error = %v", err)
	}
	t.Logf("Serialized: %s", serialized)

	expSerialized :=
		`{
  "Foobar": "foobar",
  "Host": "someHost",
  "Port": 12345,
  "KafkaCredentials": "someCredentials",
  "KafkaNilCredentials": ""
}`

	if string(serialized) != expSerialized {
		t.Errorf("Serialized data mismatch: got %s, want %s", string(serialized), expSerialized)
	}

	// Deserialize from JSON
	var deserialized EmbeddedConfigStruct
	if err := json.Unmarshal(serialized, &deserialized); err != nil {
		t.Errorf("json.Unmarshal() error = %v", err)
	}

	// Check if the deserialized data matches the original
	if data.Port != deserialized.Port {
		t.Errorf("Port mismatch: got %d, want %d", deserialized.Port, data.Port)
	}

	if data.Host != deserialized.Host {
		t.Errorf("Host mismatch: got %s, want %s", deserialized.Host, data.Host)
	}

	if data.KafkaCredentials != deserialized.KafkaCredentials {
		t.Errorf("KafkaCredentials mismatch: got %s, want %s", deserialized.KafkaCredentials, data.KafkaCredentials)
	}

	if deserialized.Foobar != "foobar" {
		t.Errorf("Foobar mismatch: got %s, want %s", deserialized.Foobar, "foobar")
	}
}

func TestJsonSerializationEmbeddedStructPointerValid(t *testing.T) {

	err := Cmd{
		Params: &EmbeddedConfigStructPointer{
			Foobar: "foobar",
			AppConfig: &AppConfig{
				Host:                "someHost",
				Port:                12345,
				KafkaCredentials:    "someCredentials",
				KafkaNilCredentials: "",
			},
		},
		ParamEnrich: ParamEnricherName, RawArgs: []string{},
	}.Validate()
	if err != nil {
		t.Errorf("Validation error: %v", err)
	}

	err = Cmd{
		Params: &EmbeddedConfigStructPointer{
			Foobar: "foobar",
			AppConfig: &AppConfig{
				Port:                12345,
				KafkaCredentials:    "someCredentials",
				KafkaNilCredentials: "",
			},
		},
		ParamEnrich: ParamEnricherName, RawArgs: []string{},
	}.Validate()
	if err == nil {
		t.Errorf("Expected validation error, got nil")
	} else {
		if strings.Contains(err.Error(), "missing required param 'host'") {
			t.Logf("Validation error as expected: %v", err)
		} else {
			t.Errorf("Unexpected validation error: %v", err)
		}
	}
}

func TestJsonSerializationEmbeddedStructPointer(t *testing.T) {

	data :=
		EmbeddedConfigStructPointer{
			Foobar: "foobar",
			AppConfig: &AppConfig{
				Host:                "someHost",
				Port:                12345,
				KafkaCredentials:    "someCredentials",
				KafkaNilCredentials: "",
			},
		}

	// Serialize to JSON
	serialized, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Errorf("json.MarshalIndent() error = %v", err)
	}
	t.Logf("Serialized: %s", serialized)

	expSerialized :=
		`{
  "Foobar": "foobar",
  "Host": "someHost",
  "Port": 12345,
  "KafkaCredentials": "someCredentials",
  "KafkaNilCredentials": ""
}`

	if string(serialized) != expSerialized {
		t.Errorf("Serialized data mismatch: got %s, want %s", string(serialized), expSerialized)
	}

	// Deserialize from JSON
	var deserialized EmbeddedConfigStruct
	if err := json.Unmarshal(serialized, &deserialized); err != nil {
		t.Errorf("json.Unmarshal() error = %v", err)
	}

	// Check if the deserialized data matches the original
	if data.Port != deserialized.Port {
		t.Errorf("Port mismatch: got %d, want %d", deserialized.Port, data.Port)
	}

	if data.Host != deserialized.Host {
		t.Errorf("Host mismatch: got %s, want %s", deserialized.Host, data.Host)
	}

	if data.KafkaCredentials != deserialized.KafkaCredentials {
		t.Errorf("KafkaCredentials mismatch: got %s, want %s", deserialized.KafkaCredentials, data.KafkaCredentials)
	}

	if deserialized.Foobar != "foobar" {
		t.Errorf("Foobar mismatch: got %s, want %s", deserialized.Foobar, "foobar")
	}
}

type AppConfigFromFile struct {
	File                string `optional:"true"`
	Host                string
	Port                int    `default:"8080"`
	KafkaCredentials    string `optional:"true"`
	KafkaNilCredentials string `optional:"true"`
}

func TestWriteJsonToFileAndTreatAsConfig(t *testing.T) {
	origCfg := AppConfig{
		Host:             "someHost",
		Port:             12345,
		KafkaCredentials: "someCredentials",
	}

	// Serialize to JSON
	serialized, err := json.MarshalIndent(origCfg, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}

	// temp file
	file, err := os.CreateTemp("", "config.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		_ = file.Close()
		if err := os.Remove(file.Name()); err != nil {
			t.Errorf("Failed to remove temp file: %v", err)
		}
	}()

	// Write JSON to file
	if _, err := file.Write(serialized); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	CmdT[AppConfigFromFile]{
		Use: "root",
		PreValidateFunc: func(params *AppConfigFromFile, cmd *cobra.Command, args []string) error {
			if params.File != "" {
				return LoadConfigFile(params.File, params, nil)
			}
			return nil
		},
		RunFunc: func(params *AppConfigFromFile, cmd *cobra.Command, args []string) {
			if params.Host != origCfg.Host {
				t.Fatalf("Host mismatch: got %s, want %s", params.Host, origCfg.Host)
			}
		},
	}.RunArgs([]string{"-f", file.Name()})

}

func TestWriteJsonToFileAndTreatAsConfigCliOvrd(t *testing.T) {
	origCfg := AppConfig{
		Host:             "someHost",
		Port:             12345,
		KafkaCredentials: "someCredentials",
	}

	// Serialize to JSON
	serialized, err := json.MarshalIndent(origCfg, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}

	// temp file
	file, err := os.CreateTemp("", "config.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		_ = file.Close()
		if err := os.Remove(file.Name()); err != nil {
			t.Errorf("Failed to remove temp file: %v", err)
		}
	}()

	// Write JSON to file
	if _, err := file.Write(serialized); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	CmdT[AppConfigFromFile]{
		Use: "root",
		PreValidateFunc: func(params *AppConfigFromFile, cmd *cobra.Command, args []string) error {
			if params.File != "" {
				return LoadConfigFile(params.File, params, nil)
			}
			return nil
		},
		RunFunc: func(params *AppConfigFromFile, cmd *cobra.Command, args []string) {
			if params.Host != "cliHost" {
				t.Fatalf("Host mismatch: got %s, want %s", params.Host, "cliHost")
			}
			if params.Port != origCfg.Port {
				t.Fatalf("Port mismatch: got %d, want %d", params.Port, origCfg.Port)
			}
		},
	}.RunArgs([]string{"-f", file.Name(), "--host", "cliHost"})

}
