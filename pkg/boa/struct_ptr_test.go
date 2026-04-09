package boa

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

// Shared struct types for struct pointer tests

type spDBConfig struct {
	Host string `descr:"database host" default:"localhost"`
	Port int    `descr:"database port" default:"5432"`
}

type spCacheConfig struct {
	TTL     int    `descr:"cache ttl seconds" default:"300"`
	Backend string `descr:"cache backend" default:"memory"`
}

// --- Basic: struct pointer nil when no flags set ---

func TestStructPtr_NilWhenNoFlagsSet(t *testing.T) {
	type Params struct {
		Name string     `descr:"app name"`
		DB   *spDBConfig // pointer — should be nil if no --db-* flags given
	}

	var gotDB *spDBConfig
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotDB = p.DB
		},
	}).RunArgsE([]string{"--name", "myapp"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotDB != nil {
		t.Errorf("expected DB to be nil when no --db-* flags set, got %+v", gotDB)
	}
}

// --- CLI arg triggers instantiation ---

func TestStructPtr_SetViaCLI_Host(t *testing.T) {
	type Params struct {
		Name string      `descr:"app name"`
		DB   *spDBConfig
	}

	var gotDB *spDBConfig
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotDB = p.DB
		},
	}).RunArgsE([]string{"--name", "myapp", "--db-host", "db.example.com"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotDB == nil {
		t.Fatal("expected DB to be non-nil when --db-host was set")
	}
	if gotDB.Host != "db.example.com" {
		t.Errorf("expected host='db.example.com', got %q", gotDB.Host)
	}
	// Port should get its default
	if gotDB.Port != 5432 {
		t.Errorf("expected port=5432 (default), got %d", gotDB.Port)
	}
}

func TestStructPtr_SetViaCLI_Port(t *testing.T) {
	type Params struct {
		DB *spDBConfig
	}

	var gotDB *spDBConfig
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotDB = p.DB
		},
	}).RunArgsE([]string{"--db-port", "3306"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotDB == nil {
		t.Fatal("expected DB to be non-nil when --db-port was set")
	}
	if gotDB.Port != 3306 {
		t.Errorf("expected port=3306, got %d", gotDB.Port)
	}
	if gotDB.Host != "localhost" {
		t.Errorf("expected host='localhost' (default), got %q", gotDB.Host)
	}
}

// --- Env var triggers instantiation ---

func TestStructPtr_SetViaEnv(t *testing.T) {
	type Inner struct {
		Host string `descr:"host" default:"localhost" env:"SP_DB_HOST"`
		Port int    `descr:"port" default:"5432"`
	}
	type Params struct {
		DB *Inner
	}

	t.Setenv("DB_SP_DB_HOST", "env-host.example.com")

	var gotDB *Inner
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotDB = p.DB
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotDB == nil {
		t.Fatal("expected DB to be non-nil when env was set")
	}
	if gotDB.Host != "env-host.example.com" {
		t.Errorf("expected host='env-host.example.com', got %q", gotDB.Host)
	}
	if gotDB.Port != 5432 {
		t.Errorf("expected port=5432 (default), got %d", gotDB.Port)
	}
}

func TestStructPtr_SetViaEnv_ExplicitEnvTag(t *testing.T) {
	type Inner struct {
		Value string `descr:"a value" env:"MY_CUSTOM_VAR"`
	}
	type Params struct {
		Thing *Inner
	}

	t.Setenv("THING_MY_CUSTOM_VAR", "from-env")

	var gotThing *Inner
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherDefault,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotThing = p.Thing
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotThing == nil {
		t.Fatal("expected Thing to be non-nil when env was set")
	}
	if gotThing.Value != "from-env" {
		t.Errorf("expected value='from-env', got %q", gotThing.Value)
	}
}

// --- Config file triggers instantiation ---

func TestStructPtr_SetViaConfigFile(t *testing.T) {
	type Inner struct {
		Host string `descr:"host" default:"localhost"`
		Port int    `descr:"port" default:"5432"`
	}
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		DB         *Inner
	}

	cfgData, _ := json.Marshal(map[string]any{
		"DB": map[string]any{"Host": "config-host", "Port": 9999},
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "app.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	var gotDB *Inner
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotDB = p.DB
		},
	}).RunArgsE([]string{"--config-file", cfgPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotDB == nil {
		t.Fatal("expected DB to be non-nil when set via config file")
	}
	if gotDB.Host != "config-host" {
		t.Errorf("expected host='config-host', got %q", gotDB.Host)
	}
	if gotDB.Port != 9999 {
		t.Errorf("expected port=9999, got %d", gotDB.Port)
	}
}

// --- CLI + Env combination ---

func TestStructPtr_CLIOverridesEnv(t *testing.T) {
	type Inner struct {
		Host string `descr:"host" env:"SP_CLI_DB_HOST"`
	}
	type Params struct {
		DB *Inner
	}

	t.Setenv("DB_SP_CLI_DB_HOST", "env-host")

	var gotDB *Inner
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotDB = p.DB
		},
	}).RunArgsE([]string{"--db-host", "cli-host"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotDB == nil {
		t.Fatal("expected DB to be non-nil")
	}
	if gotDB.Host != "cli-host" {
		t.Errorf("expected host='cli-host' (CLI overrides env), got %q", gotDB.Host)
	}
}

// --- CLI + Config file combination ---

func TestStructPtr_CLIOverridesConfigFile(t *testing.T) {
	type Inner struct {
		Host string `descr:"host" default:"localhost"`
		Port int    `descr:"port" default:"5432"`
	}
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		DB         *Inner
	}

	cfgData, _ := json.Marshal(map[string]any{
		"DB": map[string]any{"Host": "config-host", "Port": 3306},
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "app.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	var gotDB *Inner
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotDB = p.DB
		},
	}).RunArgsE([]string{"--config-file", cfgPath, "--db-host", "cli-host"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotDB == nil {
		t.Fatal("expected DB to be non-nil")
	}
	if gotDB.Host != "cli-host" {
		t.Errorf("expected host='cli-host' (CLI overrides config), got %q", gotDB.Host)
	}
	if gotDB.Port != 3306 {
		t.Errorf("expected port=3306 (from config), got %d", gotDB.Port)
	}
}

// --- Two struct pointers, only one set ---

func TestStructPtr_TwoPointers_OnlyOneSet(t *testing.T) {
	type Params struct {
		DB    *spDBConfig
		Cache *spCacheConfig
	}

	var gotDB *spDBConfig
	var gotCache *spCacheConfig
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotDB = p.DB
			gotCache = p.Cache
		},
	}).RunArgsE([]string{"--db-host", "mydb"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotDB == nil {
		t.Fatal("expected DB to be non-nil")
	}
	if gotDB.Host != "mydb" {
		t.Errorf("expected host='mydb', got %q", gotDB.Host)
	}
	if gotCache != nil {
		t.Errorf("expected Cache to be nil, got %+v", gotCache)
	}
}

// --- Both struct pointers set ---

func TestStructPtr_TwoPointers_BothSet(t *testing.T) {
	type Params struct {
		DB    *spDBConfig
		Cache *spCacheConfig
	}

	var gotDB *spDBConfig
	var gotCache *spCacheConfig
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotDB = p.DB
			gotCache = p.Cache
		},
	}).RunArgsE([]string{"--db-host", "mydb", "--cache-ttl", "60"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotDB == nil {
		t.Fatal("expected DB to be non-nil")
	}
	if gotCache == nil {
		t.Fatal("expected Cache to be non-nil")
	}
	if gotDB.Host != "mydb" {
		t.Errorf("expected db host='mydb', got %q", gotDB.Host)
	}
	if gotCache.TTL != 60 {
		t.Errorf("expected cache ttl=60, got %d", gotCache.TTL)
	}
	if gotCache.Backend != "memory" {
		t.Errorf("expected cache backend='memory' (default), got %q", gotCache.Backend)
	}
}

// --- Nested struct pointer inside struct pointer ---

func TestStructPtr_NestedPtrInPtr(t *testing.T) {
	type Inner struct {
		Value string `descr:"inner value"`
	}
	type Outer struct {
		Name  string `descr:"outer name" default:"outer-default"`
		Inner *Inner
	}
	type Params struct {
		Wrapper *Outer
	}

	var gotWrapper *Outer
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotWrapper = p.Wrapper
		},
	}).RunArgsE([]string{"--wrapper-inner-value", "deep"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotWrapper == nil {
		t.Fatal("expected Wrapper to be non-nil")
	}
	if gotWrapper.Inner == nil {
		t.Fatal("expected Wrapper.Inner to be non-nil")
	}
	if gotWrapper.Inner.Value != "deep" {
		t.Errorf("expected inner value='deep', got %q", gotWrapper.Inner.Value)
	}
	if gotWrapper.Name != "outer-default" {
		t.Errorf("expected outer name='outer-default', got %q", gotWrapper.Name)
	}
}

func TestStructPtr_NestedPtrInPtr_NoneSet(t *testing.T) {
	type Inner struct {
		Value string `descr:"inner value" optional:"true"`
	}
	type Outer struct {
		Name  string `descr:"outer name" default:"x" optional:"true"`
		Inner *Inner
	}
	type Params struct {
		Other   string `descr:"other" optional:"true"`
		Wrapper *Outer
	}

	var gotWrapper *Outer
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotWrapper = p.Wrapper
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotWrapper != nil {
		t.Errorf("expected Wrapper to be nil, got %+v", gotWrapper)
	}
}

func TestStructPtr_NestedPtrInPtr_OnlyInnerSet(t *testing.T) {
	// Setting only the deeply nested field should instantiate the full chain
	type Inner struct {
		Value string `descr:"inner value"`
	}
	type Outer struct {
		Name  string `descr:"outer name" optional:"true"`
		Inner *Inner
	}
	type Params struct {
		Wrapper *Outer
	}

	var gotWrapper *Outer
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotWrapper = p.Wrapper
		},
	}).RunArgsE([]string{"--wrapper-inner-value", "deep"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotWrapper == nil {
		t.Fatal("expected Wrapper to be non-nil (inner field was set)")
	}
	if gotWrapper.Inner == nil {
		t.Fatal("expected Wrapper.Inner to be non-nil")
	}
	if gotWrapper.Inner.Value != "deep" {
		t.Errorf("expected inner value='deep', got %q", gotWrapper.Inner.Value)
	}
}

// --- Embedded (anonymous) struct pointer ---

func TestStructPtr_Embedded_Anonymous(t *testing.T) {
	type CommonOpts struct {
		Verbose bool `descr:"verbose" optional:"true"`
	}
	type Params struct {
		*CommonOpts
		Name string `descr:"name"`
	}

	var gotParams *Params
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotParams = p
		},
	}).RunArgsE([]string{"--name", "test", "--verbose"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotParams.CommonOpts == nil {
		t.Fatal("expected embedded CommonOpts to be non-nil when --verbose was set")
	}
	if !gotParams.Verbose {
		t.Error("expected verbose=true")
	}
}

func TestStructPtr_Embedded_Anonymous_NilWhenNotSet(t *testing.T) {
	type CommonOpts struct {
		Verbose bool `descr:"verbose" optional:"true"`
	}
	type Params struct {
		*CommonOpts
		Name string `descr:"name"`
	}

	var gotParams *Params
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotParams = p
		},
	}).RunArgsE([]string{"--name", "test"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotParams.CommonOpts != nil {
		t.Errorf("expected embedded CommonOpts to be nil when --verbose was not set, got %+v", gotParams.CommonOpts)
	}
}

// --- Custom validator on struct pointer fields ---

func TestStructPtr_CustomValidator(t *testing.T) {
	type Params struct {
		DB *spDBConfig
	}

	validatorCalled := false
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		InitFunc: func(p *Params, cmd *cobra.Command) error {
			// We'd need HookContext to set a validator on db-port...
			// but this tests that the struct pointer fields are accessible for validation
			return nil
		},
		PreValidateFunc: func(p *Params, cmd *cobra.Command, args []string) error {
			if p.DB != nil {
				validatorCalled = true
				if p.DB.Port < 1 || p.DB.Port > 65535 {
					return NewUserInputErrorf("port must be between 1 and 65535")
				}
			}
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--db-port", "3306"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !validatorCalled {
		t.Error("expected custom validator to be called when DB was set")
	}
}

func TestStructPtr_CustomValidator_NotCalledWhenNil(t *testing.T) {
	type Params struct {
		Name string     `descr:"name"`
		DB   *spDBConfig
	}

	validatorCalled := false
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		PreValidateFunc: func(p *Params, cmd *cobra.Command, args []string) error {
			if p.DB != nil {
				validatorCalled = true
			}
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--name", "test"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if validatorCalled {
		t.Error("expected validator NOT to be called when DB is nil")
	}
}

func TestStructPtr_CustomValidatorViaHookCtx(t *testing.T) {
	type Params struct {
		DB *spDBConfig
	}

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		PostCreateFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command) error {
			// p.DB should be non-nil here (preallocated) so we can get the param
			if p.DB == nil {
				t.Error("expected DB to be preallocated in PostCreateFuncCtx")
				return nil
			}
			portParam := ctx.GetParam(&p.DB.Port)
			if portParam == nil {
				t.Error("expected to find port param via HookContext")
				return nil
			}
			portParam.SetCustomValidator(func(val any) error {
				v := val.(int)
				if v < 1024 {
					return NewUserInputErrorf("port must be >= 1024, got %d", v)
				}
				return nil
			})
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--db-port", "80"})

	if err == nil {
		t.Error("expected validation error for port < 1024")
	}
}

// --- AlternativesFunc on struct pointer fields ---

func TestStructPtr_AlternativesFunc(t *testing.T) {
	type Params struct {
		DB *spDBConfig
	}

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		PostCreateFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command) error {
			if p.DB == nil {
				t.Error("expected DB to be preallocated in PostCreateFuncCtx")
				return nil
			}
			hostParam := ctx.GetParam(&p.DB.Host)
			if hostParam == nil {
				t.Error("expected to find host param via HookContext")
				return nil
			}
			hostParam.SetAlternativesFunc(func(cmd *cobra.Command, args []string, toComplete string) []string {
				return []string{"localhost", "db.prod", "db.staging"}
			})
			return nil
		},
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--db-host", "localhost"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStructPtr_Alternatives_Strict(t *testing.T) {
	type Inner struct {
		Mode string `descr:"mode" alts:"fast,slow" strict:"true"`
	}
	type Params struct {
		Config *Inner
	}

	// Valid value
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--config-mode", "fast"})

	if err != nil {
		t.Fatalf("unexpected error for valid alt: %v", err)
	}

	// Invalid value
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--config-mode", "invalid"})

	if err == nil {
		t.Error("expected error for invalid alt value")
	}
}

// --- Validation tags on struct pointer fields ---

func TestStructPtr_ValidationTags_MinMax(t *testing.T) {
	type Inner struct {
		Port int `descr:"port" min:"1" max:"65535"`
	}
	type Params struct {
		Server *Inner
	}

	// Valid
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--server-port", "8080"})

	if err != nil {
		t.Fatalf("unexpected error for valid port: %v", err)
	}

	// Invalid (too high)
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--server-port", "99999"})

	if err == nil {
		t.Error("expected validation error for port > 65535")
	}
}

func TestStructPtr_ValidationTags_Pattern(t *testing.T) {
	type Inner struct {
		Tag string `descr:"tag" pattern:"^v[0-9]+\\.[0-9]+$"`
	}
	type Params struct {
		Release *Inner
	}

	// Valid
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--release-tag", "v1.0"})

	if err != nil {
		t.Fatalf("unexpected error for valid tag: %v", err)
	}

	// Invalid
	err = (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--release-tag", "bad"})

	if err == nil {
		t.Error("expected validation error for tag not matching pattern")
	}
}

// --- Required fields inside struct pointer ---

func TestStructPtr_RequiredFieldInsidePtr_NoErrorWhenStructNil(t *testing.T) {
	// If the struct pointer is nil (nothing set), required fields inside
	// should NOT trigger validation errors.
	type Inner struct {
		Host string `descr:"host"` // required by default
	}
	type Params struct {
		Other  string `descr:"other" optional:"true"`
		Server *Inner
	}

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("expected no error when struct ptr is nil, but got: %v", err)
	}
}

func TestStructPtr_RequiredFieldInsidePtr_ErrorWhenPartiallySet(t *testing.T) {
	// If one field is set but another required field in the same struct isn't,
	// that should be a validation error.
	type Inner struct {
		Host string `descr:"host"`              // required
		Port int    `descr:"port" optional:"true"` // optional
		Name string `descr:"name"`              // required
	}
	type Params struct {
		Server *Inner
	}

	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{"--server-host", "example.com"})
	// Host is set but Name is required and not set — should error

	if err == nil {
		t.Error("expected validation error for missing required --server-name when --server-host was set")
	}
}

// --- Mixed pointer and non-pointer structs ---

func TestStructPtr_MixedPointerAndValue(t *testing.T) {
	type Params struct {
		DB    spDBConfig    // value struct — always present
		Cache *spCacheConfig // pointer struct — nil if not set
	}

	var gotDB spDBConfig
	var gotCache *spCacheConfig
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotDB = p.DB
			gotCache = p.Cache
		},
	}).RunArgsE([]string{"--db-host", "mydb"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotDB.Host != "mydb" {
		t.Errorf("expected db host='mydb', got %q", gotDB.Host)
	}
	if gotCache != nil {
		t.Errorf("expected Cache to be nil, got %+v", gotCache)
	}
}

// --- Struct pointer with all-optional fields ---

func TestStructPtr_AllOptionalFields_NilWhenNotSet(t *testing.T) {
	type Inner struct {
		Debug   bool   `descr:"debug" optional:"true"`
		LogFile string `descr:"log file" optional:"true"`
	}
	type Params struct {
		Logging *Inner
	}

	var gotLogging *Inner
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotLogging = p.Logging
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotLogging != nil {
		t.Errorf("expected Logging to be nil, got %+v", gotLogging)
	}
}

func TestStructPtr_AllOptionalFields_SetWhenOneProvided(t *testing.T) {
	type Inner struct {
		Debug   bool   `descr:"debug" optional:"true"`
		LogFile string `descr:"log file" optional:"true"`
	}
	type Params struct {
		Logging *Inner
	}

	var gotLogging *Inner
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotLogging = p.Logging
		},
	}).RunArgsE([]string{"--logging-debug"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotLogging == nil {
		t.Fatal("expected Logging to be non-nil when --logging-debug was set")
	}
	if !gotLogging.Debug {
		t.Error("expected debug=true")
	}
}

// --- Defaults inside struct pointer should not trigger instantiation ---

func TestStructPtr_DefaultsAloneDontInstantiate(t *testing.T) {
	type Inner struct {
		Host string `descr:"host" default:"localhost" optional:"true"`
		Port int    `descr:"port" default:"5432" optional:"true"`
	}
	type Params struct {
		Other string `descr:"other" optional:"true"`
		DB    *Inner
	}

	var gotDB *Inner
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotDB = p.DB
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotDB != nil {
		t.Errorf("expected DB to be nil (defaults alone shouldn't instantiate), got %+v", gotDB)
	}
}

// --- Env + Config file combination with struct pointer ---

func TestStructPtr_EnvAndConfigFile(t *testing.T) {
	type Inner struct {
		Host string `descr:"host" default:"localhost" env:"SP_ENV_CFG_HOST"`
		Port int    `descr:"port" default:"5432"`
	}
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		DB         *Inner
	}

	// Config sets port
	cfgData, _ := json.Marshal(map[string]any{
		"DB": map[string]any{"Port": 3306},
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "app.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	// Env sets host (prefixed by named struct "DB")
	t.Setenv("DB_SP_ENV_CFG_HOST", "env-host")

	var gotDB *Inner
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotDB = p.DB
		},
	}).RunArgsE([]string{"--config-file", cfgPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotDB == nil {
		t.Fatal("expected DB to be non-nil")
	}
	if gotDB.Host != "env-host" {
		t.Errorf("expected host='env-host', got %q", gotDB.Host)
	}
	if gotDB.Port != 3306 {
		t.Errorf("expected port=3306 (from config), got %d", gotDB.Port)
	}
}

// --- Config file should not erase struct pointer ---

func TestStructPtr_ConfigFileKeepsStructAlive(t *testing.T) {
	// Values from config file should keep the struct pointer non-nil
	type Inner struct {
		Host string `descr:"host" default:"localhost" optional:"true"`
		Port int    `descr:"port" default:"5432" optional:"true"`
	}
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		DB         *Inner
	}

	cfgData, _ := json.Marshal(map[string]any{
		"DB": map[string]any{"Host": "config-host"},
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "app.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	var gotDB *Inner
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotDB = p.DB
		},
	}).RunArgsE([]string{"--config-file", cfgPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotDB == nil {
		t.Fatal("expected DB to be non-nil when set via config file")
	}
	if gotDB.Host != "config-host" {
		t.Errorf("expected host='config-host', got %q", gotDB.Host)
	}
}

func TestStructPtr_ConfigFileSetsZeroValue_StillKeepsStruct(t *testing.T) {
	// Config file explicitly sets a field to the Go zero value (0 for int).
	// The struct pointer should remain non-nil because the user explicitly
	// provided the value in the config file.
	type Inner struct {
		Count int `descr:"count" optional:"true"`
	}
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Stats      *Inner
	}

	cfgData, _ := json.Marshal(map[string]any{
		"Stats": map[string]any{"Count": 0},
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "cfg.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	var gotStats *Inner
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotStats = p.Stats
		},
	}).RunArgsE([]string{"--config-file", cfgPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotStats == nil {
		t.Fatal("expected Stats to be non-nil when config file explicitly set Count=0")
	}
}

func TestStructPtr_ConfigFileSetsDefaultValue_StillKeepsStruct(t *testing.T) {
	// Even if config sets same value as default, the struct should stay alive
	// because the user explicitly provided it in a config file.
	// Key-presence detection handles this — we detect the key in JSON, not
	// the value.
	type Inner struct {
		Host string `descr:"host" default:"localhost" optional:"true"`
	}
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		DB         *Inner
	}

	cfgData, _ := json.Marshal(map[string]any{
		"DB": map[string]any{"Host": "localhost"}, // same as default!
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "cfg.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	var gotDB *Inner
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotDB = p.DB
		},
	}).RunArgsE([]string{"--config-file", cfgPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotDB == nil {
		t.Fatal("expected DB to be non-nil when config file explicitly set the field (even to default value)")
	}
}

func TestStructPtr_SubstructOwnConfigFile(t *testing.T) {
	// The inner struct has its OWN configfile:"true" field, not the root.
	// This tests that substruct config file loading preserves the struct pointer.
	type Inner struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Host       string `descr:"host" default:"localhost" optional:"true"`
		Port       int    `descr:"port" default:"5432" optional:"true"`
	}
	type Params struct {
		Name string `descr:"name" optional:"true"`
		DB   *Inner
	}

	cfgData, _ := json.Marshal(map[string]any{
		"Host": "substruct-host",
		"Port": 9999,
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "db.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	var gotDB *Inner
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotDB = p.DB
		},
	}).RunArgsE([]string{"--db-config-file", cfgPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotDB == nil {
		t.Fatal("expected DB to be non-nil when loaded via its own config file")
	}
	if gotDB.Host != "substruct-host" {
		t.Errorf("expected host='substruct-host', got %q", gotDB.Host)
	}
	if gotDB.Port != 9999 {
		t.Errorf("expected port=9999, got %d", gotDB.Port)
	}
}

func TestStructPtr_SubstructOwnConfigFile_ZeroValue(t *testing.T) {
	// Substruct config file sets a field to the Go zero value.
	type Inner struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Count      int    `descr:"count" optional:"true"`
	}
	type Params struct {
		Stats *Inner
	}

	cfgData, _ := json.Marshal(map[string]any{
		"Count": 0,
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "stats.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	var gotStats *Inner
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotStats = p.Stats
		},
	}).RunArgsE([]string{"--stats-config-file", cfgPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotStats == nil {
		t.Fatal("expected Stats to be non-nil when loaded via its own config file")
	}
}

func TestStructPtr_TwoLevelNestedPtrs_ConfigFile(t *testing.T) {
	// X -> *Y -> *Z, config file sets a field in Z.
	type Z struct {
		Value string `descr:"value" optional:"true"`
	}
	type Y struct {
		Label string `descr:"label" optional:"true"`
		Inner *Z
	}
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Outer      *Y
	}

	cfgData, _ := json.Marshal(map[string]any{
		"Outer": map[string]any{
			"Inner": map[string]any{"Value": "deep-config"},
		},
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "app.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	var gotOuter *Y
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotOuter = p.Outer
		},
	}).RunArgsE([]string{"--config-file", cfgPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotOuter == nil {
		t.Fatal("expected Outer to be non-nil")
	}
	if gotOuter.Inner == nil {
		t.Fatal("expected Outer.Inner to be non-nil")
	}
	if gotOuter.Inner.Value != "deep-config" {
		t.Errorf("expected value='deep-config', got %q", gotOuter.Inner.Value)
	}
}

func TestStructPtr_TwoLevelNestedPtrs_ConfigFile_ZeroValue(t *testing.T) {
	// X -> *Y -> *Z, config file sets a field in Z to zero value.
	type Z struct {
		Count int `descr:"count" optional:"true"`
	}
	type Y struct {
		Inner *Z
	}
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Outer      *Y
	}

	cfgData, _ := json.Marshal(map[string]any{
		"Outer": map[string]any{
			"Inner": map[string]any{"Count": 0},
		},
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "app.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	var gotOuter *Y
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotOuter = p.Outer
		},
	}).RunArgsE([]string{"--config-file", cfgPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotOuter == nil {
		t.Fatal("expected Outer to be non-nil")
	}
	if gotOuter.Inner == nil {
		t.Fatal("expected Outer.Inner to be non-nil (config set Count=0)")
	}
}

func TestStructPtr_TwoLevelNestedPtrs_OnlyMiddleSetViaConfig(t *testing.T) {
	// Config sets a field on Y but not Z — Y should be non-nil, Z should be nil.
	type Z struct {
		Value string `descr:"value" optional:"true"`
	}
	type Y struct {
		Label string `descr:"label" optional:"true"`
		Inner *Z
	}
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		Outer      *Y
	}

	cfgData, _ := json.Marshal(map[string]any{
		"Outer": map[string]any{
			"Label": "from-config",
		},
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "app.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	var gotOuter *Y
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc: func(p *Params, cmd *cobra.Command, args []string) {
			gotOuter = p.Outer
		},
	}).RunArgsE([]string{"--config-file", cfgPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotOuter == nil {
		t.Fatal("expected Outer to be non-nil")
	}
	if gotOuter.Label != "from-config" {
		t.Errorf("expected label='from-config', got %q", gotOuter.Label)
	}
	if gotOuter.Inner != nil {
		t.Errorf("expected Outer.Inner to be nil (not mentioned in config), got %+v", gotOuter.Inner)
	}
}

// --- Pre-initialized struct pointer should be kept ---

func TestStructPtr_PreInitialized_KeptEvenIfNoFlagsSet(t *testing.T) {
	type Inner struct {
		Host string `descr:"host" default:"localhost" optional:"true"`
	}
	type Params struct {
		DB *Inner
	}

	params := &Params{DB: &Inner{Host: "pre-init"}}

	err := (CmdT[Params]{
		Use:         "test",
		Params:      params,
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.DB == nil {
		t.Fatal("expected pre-initialized DB to remain non-nil")
	}
	if params.DB.Host != "pre-init" {
		t.Errorf("expected host='pre-init', got %q", params.DB.Host)
	}
}

// --- HasValue works correctly for fields inside struct pointers ---

func TestStructPtr_HasValue_SetField(t *testing.T) {
	// HasValue should return true for a field that was explicitly set via CLI
	type Params struct {
		DB *spDBConfig
	}

	hostHasValue := false
	portHasValue := false
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			if p.DB != nil {
				hostHasValue = ctx.HasValue(&p.DB.Host)
				portHasValue = ctx.HasValue(&p.DB.Port)
			}
		},
	}).RunArgsE([]string{"--db-host", "myhost"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hostHasValue {
		t.Error("expected HasValue(&p.DB.Host) to be true when --db-host was set")
	}
	// Port was not explicitly set, but has a default — HasValue should still be true
	// (consistent with non-pointer struct behavior)
	if !portHasValue {
		t.Error("expected HasValue(&p.DB.Port) to be true (has default value)")
	}
}

func TestStructPtr_HasValue_NoFieldsSet(t *testing.T) {
	// When struct pointer is nil (nothing set), user should nil-check before HasValue.
	// This test verifies the nil-check pattern works and mirrors are cleaned up properly.
	type Params struct {
		Name string     `descr:"name"`
		DB   *spDBConfig
	}

	dbWasNil := false
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			dbWasNil = (p.DB == nil)
		},
	}).RunArgsE([]string{"--name", "test"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dbWasNil {
		t.Error("expected p.DB to be nil when no --db-* flags were set")
	}
}

func TestStructPtr_HasValue_SetViaEnv(t *testing.T) {
	type Inner struct {
		Host string `descr:"host" env:"SP_HV_HOST" optional:"true"`
		Port int    `descr:"port" default:"5432" optional:"true"`
	}
	type Params struct {
		DB *Inner
	}

	t.Setenv("DB_SP_HV_HOST", "env-host")

	hostHasValue := false
	portHasValue := false
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			if p.DB != nil {
				hostHasValue = ctx.HasValue(&p.DB.Host)
				portHasValue = ctx.HasValue(&p.DB.Port)
			}
		},
	}).RunArgsE([]string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hostHasValue {
		t.Error("expected HasValue(&p.DB.Host) to be true when set via env")
	}
	if !portHasValue {
		t.Error("expected HasValue(&p.DB.Port) to be true (has default)")
	}
}

func TestStructPtr_HasValue_SetViaConfigFile(t *testing.T) {
	type Inner struct {
		Host string `descr:"host" optional:"true"`
		Port int    `descr:"port" optional:"true"`
	}
	type Params struct {
		ConfigFile string `configfile:"true" optional:"true"`
		DB         *Inner
	}

	cfgData, _ := json.Marshal(map[string]any{
		"DB": map[string]any{"Host": "cfg-host"},
	})
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "app.json")
	_ = os.WriteFile(cfgPath, cfgData, 0644)

	hostHasValue := false
	portHasValue := false
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			if p.DB != nil {
				hostHasValue = ctx.HasValue(&p.DB.Host)
				portHasValue = ctx.HasValue(&p.DB.Port)
			}
		},
	}).RunArgsE([]string{"--config-file", cfgPath})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hostHasValue {
		t.Error("expected HasValue(&p.DB.Host) to be true when set via config")
	}
	// Port was not in config and has no default — HasValue should be false
	if portHasValue {
		t.Error("expected HasValue(&p.DB.Port) to be false (not set, no default)")
	}
}

func TestStructPtr_HasValue_NestedPtr(t *testing.T) {
	// HasValue should work through nested pointer structs
	type Inner struct {
		Value string `descr:"value" optional:"true"`
	}
	type Outer struct {
		Name  string `descr:"name" optional:"true"`
		Inner *Inner
	}
	type Params struct {
		Wrapper *Outer
	}

	valueHasValue := false
	nameHasValue := false
	err := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFuncCtx: func(ctx *HookContext, p *Params, cmd *cobra.Command, args []string) {
			if p.Wrapper != nil && p.Wrapper.Inner != nil {
				valueHasValue = ctx.HasValue(&p.Wrapper.Inner.Value)
			}
			if p.Wrapper != nil {
				nameHasValue = ctx.HasValue(&p.Wrapper.Name)
			}
		},
	}).RunArgsE([]string{"--wrapper-inner-value", "hello"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valueHasValue {
		t.Error("expected HasValue for nested value to be true")
	}
	if nameHasValue {
		t.Error("expected HasValue for wrapper name to be false (not set, no default)")
	}
}

// --- Flags from struct pointers show up in help ---

func TestStructPtr_FlagsVisibleInHelp(t *testing.T) {
	type Params struct {
		Name string     `descr:"app name"`
		DB   *spDBConfig
	}

	cmd := (CmdT[Params]{
		Use:         "test",
		ParamEnrich: ParamEnricherName,
		RunFunc:     func(p *Params, cmd *cobra.Command, args []string) {},
	}).ToCobra()

	// Check that db-host and db-port flags exist
	hostFlag := cmd.Flags().Lookup("db-host")
	if hostFlag == nil {
		t.Error("expected --db-host flag to be registered")
	}

	portFlag := cmd.Flags().Lookup("db-port")
	if portFlag == nil {
		t.Error("expected --db-port flag to be registered")
	}
}
