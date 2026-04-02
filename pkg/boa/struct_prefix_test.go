package boa

import (
	"os"
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

func TestNamedStruct_EnvVarAutoPrefix(t *testing.T) {
	// Env var names should also be prefixed: DB.Host → DB_HOST
	type Params struct {
		DB DBConfig
	}

	_ = os.Setenv("DB_HOST", "env-host")
	_ = os.Setenv("DB_PORT", "9999")
	defer func() { _ = os.Unsetenv("DB_HOST") }()
	defer func() { _ = os.Unsetenv("DB_PORT") }()

	var gotHost string
	var gotPort int
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherCombine(ParamEnricherDefault, ParamEnricherEnv),
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotHost = p.DB.Host
			gotPort = p.DB.Port
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHost != "env-host" {
		t.Errorf("expected host='env-host', got %q", gotHost)
	}
	if gotPort != 9999 {
		t.Errorf("expected port=9999, got %d", gotPort)
	}
}

func TestEmbeddedStruct_EnvVarNoPrefix(t *testing.T) {
	// Embedded struct env vars should NOT be prefixed
	type Params struct {
		CommonOpts // embedded
	}

	_ = os.Setenv("VERBOSE", "true")
	defer func() { _ = os.Unsetenv("VERBOSE") }()

	var gotVerbose bool
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherCombine(ParamEnricherDefault, ParamEnricherEnv),
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotVerbose = p.Verbose
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gotVerbose {
		t.Error("expected verbose=true from env var")
	}
}

func TestNamedStruct_DeepNesting_EnvVar(t *testing.T) {
	// 3 levels deep: Infra.Primary.Host → INFRA_PRIMARY_HOST
	type ConnectionConfig struct {
		Host string `descr:"host" default:"localhost"`
	}
	type ClusterConfig struct {
		Primary ConnectionConfig
	}
	type Params struct {
		Infra ClusterConfig
	}

	_ = os.Setenv("INFRA_PRIMARY_HOST", "deep-env-host")
	defer func() { _ = os.Unsetenv("INFRA_PRIMARY_HOST") }()

	var gotHost string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherCombine(ParamEnricherDefault, ParamEnricherEnv),
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotHost = p.Infra.Primary.Host
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHost != "deep-env-host" {
		t.Errorf("expected host='deep-env-host', got %q", gotHost)
	}
}

func TestNamedStruct_ExplicitEnvTag(t *testing.T) {
	// Explicit env tags get prefixed when inside a named struct field.
	// API.Host with env:"SERVER_HOST" becomes API_SERVER_HOST.
	type ServerConfig struct {
		Host string `descr:"host" env:"SERVER_HOST" default:"localhost"`
		Port int    `descr:"port" env:"SERVER_PORT" default:"8080"`
	}
	type Params struct {
		API ServerConfig
	}

	_ = os.Setenv("API_SERVER_HOST", "api.example.com")
	_ = os.Setenv("API_SERVER_PORT", "9090")
	defer func() { _ = os.Unsetenv("API_SERVER_HOST") }()
	defer func() { _ = os.Unsetenv("API_SERVER_PORT") }()

	var gotHost string
	var gotPort int
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName, // no env enricher — rely on explicit tags
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotHost = p.API.Host
			gotPort = p.API.Port
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHost != "api.example.com" {
		t.Errorf("expected host='api.example.com', got %q", gotHost)
	}
	if gotPort != 9090 {
		t.Errorf("expected port=9090, got %d", gotPort)
	}
}

func TestNamedStruct_ExplicitNameGetsPrefixed(t *testing.T) {
	// Explicit name:"..." on a child field should ALSO be prefixed
	// when inside a named struct field. The parent prefix always applies.
	// This avoids collisions when the same struct is used in multiple fields.
	type Config struct {
		Host string `descr:"host" name:"host"` // explicit name
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
	}).RunArgsE([]string{"--api-host", "api.example.com"})

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

func TestNamedStruct_ExplicitEnvGetsPrefixed(t *testing.T) {
	// Explicit env:"..." on a child field should ALSO be prefixed
	// when inside a named struct field.
	type Config struct {
		Host string `descr:"host" env:"HOST" default:"localhost"`
	}
	type Params struct {
		Primary Config
		Replica Config
	}

	_ = os.Setenv("PRIMARY_HOST", "primary.db")
	_ = os.Setenv("REPLICA_HOST", "replica.db")
	defer func() { _ = os.Unsetenv("PRIMARY_HOST") }()
	defer func() { _ = os.Unsetenv("REPLICA_HOST") }()

	var gotPrimary, gotReplica string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotPrimary = p.Primary.Host
			gotReplica = p.Replica.Host
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPrimary != "primary.db" {
		t.Errorf("expected primary='primary.db', got %q", gotPrimary)
	}
	if gotReplica != "replica.db" {
		t.Errorf("expected replica='replica.db', got %q", gotReplica)
	}
}

func TestNamedStruct_ExplicitEnvNoPrefixWhenEmbedded(t *testing.T) {
	// Embedded (anonymous) structs should NOT prefix explicit env tags
	type Config struct {
		Host string `descr:"host" env:"MY_HOST" default:"localhost"`
	}
	type Params struct {
		Config // embedded — env stays MY_HOST
	}

	_ = os.Setenv("MY_HOST", "embedded.host")
	defer func() { _ = os.Unsetenv("MY_HOST") }()

	var gotHost string
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotHost = p.Host
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHost != "embedded.host" {
		t.Errorf("expected host='embedded.host', got %q", gotHost)
	}
}
