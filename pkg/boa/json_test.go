package boa

import (
	"encoding/json"
	"testing"
)

type AppConfig struct {
	Host                Required[string] `long:"host" env:"HOST" default:"localhost"`
	Port                Required[int]    `long:"port" env:"PORT" default:"8080"`
	KafkaCredentials    Optional[string] `long:"kafka-credentials" env:"KAFKA_CREDENTIALS" default:""`
	KafkaNilCredentials Optional[string] `long:"kafka-nil-credentials" env:"KAFKA_NIL_CREDENTIALS"`
}

func TestJsonSerialization(t *testing.T) {

	data := AppConfig{
		Host:                Req("someHost"),
		Port:                Req(12345),
		KafkaCredentials:    Opt("someCredentials"),
		KafkaNilCredentials: OptP[string](nil),
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
	if data.Port.Value() != deserialized.Port.Value() {
		t.Errorf("Port mismatch: got %d, want %d", deserialized.Port.Value(), data.Port.Value())
	}

	if data.Host.Value() != deserialized.Host.Value() {
		t.Errorf("Host mismatch: got %s, want %s", deserialized.Host.Value(), data.Host.Value())
	}

	if data.KafkaNilCredentials.HasValue() {
		t.Errorf("KafkaNilCredentials should not have value")
	}

	if deserialized.KafkaNilCredentials.HasValue() {
		t.Errorf("KafkaNilCredentials should not have value")
	}

	if !data.KafkaCredentials.HasValue() {
		t.Errorf("KafkaCredentials should have value")
	}
	if !deserialized.KafkaCredentials.HasValue() {
		t.Errorf("KafkaCredentials should have value")
	}
	if data.KafkaCredentials.GetOrElse("") != deserialized.KafkaCredentials.GetOrElse("") {
		t.Errorf("KafkaCredentials mismatch: got %s, want %s", deserialized.KafkaCredentials.GetOrElse(""), data.KafkaCredentials.GetOrElse(""))
	}
}
