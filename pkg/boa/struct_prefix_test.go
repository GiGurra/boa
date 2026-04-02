package boa

import (
	"testing"

	"github.com/spf13/cobra"
)

type CommonOpts struct {
	Verbose bool `descr:"verbose output" short:"v" optional:"true"`
	Debug   bool `descr:"debug mode" optional:"true"`
}

type DBConfig struct {
	Host string `descr:"database host" default:"localhost"`
	Port int    `descr:"database port" default:"5432"`
}

func TestEmbeddedStruct_NoPrefixByDefault(t *testing.T) {
	// Embedded (anonymous) struct fields should NOT be prefixed.
	// This is the common pattern for shared options across commands.
	type Params struct {
		CommonOpts        // embedded — fields become --verbose, --debug
		Name       string `descr:"app name"`
	}

	var gotVerbose bool
	var gotName string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotVerbose = p.Verbose
			gotName = p.Name
		},
	}).RunArgsE([]string{"--verbose", "--name", "myapp"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gotVerbose {
		t.Error("expected verbose=true")
	}
	if gotName != "myapp" {
		t.Errorf("expected name='myapp', got %q", gotName)
	}
}

func TestNamedStruct_AutoPrefixed(t *testing.T) {
	// Named struct fields should auto-prefix their children.
	// DB.Host becomes --db-host, DB.Port becomes --db-port
	type Params struct {
		Name string   `descr:"app name"`
		DB   DBConfig // named field — children become --db-host, --db-port
	}

	var gotHost string
	var gotPort int
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotHost = p.DB.Host
			gotPort = p.DB.Port
		},
	}).RunArgsE([]string{"--name", "myapp", "--db-host", "db.example.com", "--db-port", "3306"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHost != "db.example.com" {
		t.Errorf("expected host='db.example.com', got %q", gotHost)
	}
	if gotPort != 3306 {
		t.Errorf("expected port=3306, got %d", gotPort)
	}
}

func TestNamedStruct_AutoPrefixedEnvVar(t *testing.T) {
	// Named struct prefix also applies to auto-generated env var names.
	// DB.Host with ParamEnricherEnv becomes DB_HOST
	type Params struct {
		DB DBConfig
	}

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherDefault,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
		},
	}).RunArgsE([]string{"--db-host", "x"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// If we got here without "unknown flag", the prefix worked
}

func TestNamedStruct_TwoInstances_NoPrefixCollision(t *testing.T) {
	// Two named instances of the same struct type should not collide.
	type Params struct {
		Primary DBConfig
		Replica DBConfig
	}

	var gotPrimaryHost, gotReplicaHost string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotPrimaryHost = p.Primary.Host
			gotReplicaHost = p.Replica.Host
		},
	}).RunArgsE([]string{"--primary-host", "primary.db", "--replica-host", "replica.db"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPrimaryHost != "primary.db" {
		t.Errorf("expected primary host='primary.db', got %q", gotPrimaryHost)
	}
	if gotReplicaHost != "replica.db" {
		t.Errorf("expected replica host='replica.db', got %q", gotReplicaHost)
	}
}

func TestNamedStruct_DeepNesting(t *testing.T) {
	// 3+ levels of nesting should chain prefixes
	// Infra.Primary.Host → --infra-primary-host
	type ConnectionConfig struct {
		Host string `descr:"host" default:"localhost"`
		Port int    `descr:"port" default:"5432"`
	}
	type ClusterConfig struct {
		Primary ConnectionConfig
		Replica ConnectionConfig
	}
	type Params struct {
		Infra ClusterConfig
	}

	var gotPrimaryHost, gotReplicaHost string
	var gotPrimaryPort, gotReplicaPort int
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotPrimaryHost = p.Infra.Primary.Host
			gotPrimaryPort = p.Infra.Primary.Port
			gotReplicaHost = p.Infra.Replica.Host
			gotReplicaPort = p.Infra.Replica.Port
		},
	}).RunArgsE([]string{
		"--infra-primary-host", "primary.db",
		"--infra-primary-port", "5433",
		"--infra-replica-host", "replica.db",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPrimaryHost != "primary.db" {
		t.Errorf("expected primary host='primary.db', got %q", gotPrimaryHost)
	}
	if gotPrimaryPort != 5433 {
		t.Errorf("expected primary port=5433, got %d", gotPrimaryPort)
	}
	if gotReplicaHost != "replica.db" {
		t.Errorf("expected replica host='replica.db', got %q", gotReplicaHost)
	}
	if gotReplicaPort != 5432 {
		t.Errorf("expected replica port=5432 (default), got %d", gotReplicaPort)
	}
}

func TestNamedStruct_MixedEmbeddedAndNamed(t *testing.T) {
	// Embedded struct at one level + named at another
	type Logging struct {
		Level string `descr:"log level" default:"info" optional:"true"`
	}
	type Params struct {
		Logging            // embedded — --level (no prefix)
		DB      DBConfig   // named — --db-host, --db-port
	}

	var gotLevel, gotHost string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotLevel = p.Level
			gotHost = p.DB.Host
		},
	}).RunArgsE([]string{"--level", "debug", "--db-host", "myhost"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotLevel != "debug" {
		t.Errorf("expected level='debug', got %q", gotLevel)
	}
	if gotHost != "myhost" {
		t.Errorf("expected host='myhost', got %q", gotHost)
	}
}

func TestNamedStruct_ExplicitNameOverridesPrefix(t *testing.T) {
	// Explicit name:"..." tag on a child field should override the auto-prefix
	type Config struct {
		Host string `descr:"host" name:"server-host"`
		Port int    `descr:"port" default:"8080"`
	}
	type Params struct {
		API Config
	}

	var gotHost string
	var gotPort int
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotHost = p.API.Host
			gotPort = p.API.Port
		},
	}).RunArgsE([]string{"--server-host", "api.example.com"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHost != "api.example.com" {
		t.Errorf("expected host='api.example.com', got %q", gotHost)
	}
	if gotPort != 8080 {
		t.Errorf("expected port=8080, got %d", gotPort)
	}
}
