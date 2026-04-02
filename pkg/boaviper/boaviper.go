// Package boaviper provides Viper-like automatic config file discovery for boa.
//
// It searches standard paths for config files and integrates with boa's config
// file loading via the configfile:"true" struct tag.
//
// Usage:
//
//	type Params struct {
//	    ConfigFile string `configfile:"true" optional:"true"`
//	    Port       int    `descr:"server port" default:"8080"`
//	}
//
//	boa.CmdT[Params]{
//	    Use:      "myapp",
//	    InitFunc: boaviper.AutoConfig("myapp"),
//	    RunFunc:  func(p *Params, cmd *cobra.Command, args []string) { ... },
//	}.Run()
//
// This will automatically search for config files in:
//   - ./myapp.json (current directory)
//   - $HOME/.config/myapp/config.json
//   - /etc/myapp/config.json
//
// The first file found is used. All registered config format extensions
// (via boa.RegisterConfigFormat) are tried at each path.
package boaviper

import (
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

// DefaultSearchPaths returns the standard Viper-like search paths for an app.
// Paths are returned in priority order (first match wins):
//  1. Current working directory: ./<appName>.<ext>
//  2. XDG config: $HOME/.config/<appName>/config.<ext>
//  3. System config: /etc/<appName>/config.<ext>
func DefaultSearchPaths(appName string) []string {
	paths := []string{"."}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", appName))
	}
	paths = append(paths, filepath.Join("/etc", appName))
	return paths
}

// FindConfig searches for a config file in the given paths (or default paths
// if none provided). Tries all registered config format extensions at each path.
//
// For each search path, it tries:
//   - <path>/<appName>.<ext> (in the root search path, e.g., ./myapp.json)
//   - <path>/config.<ext> (in named directories, e.g., ~/.config/myapp/config.json)
//
// Returns the path to the first file found, or empty string if none found.
func FindConfig(appName string, searchPaths ...string) string {
	if len(searchPaths) == 0 {
		searchPaths = DefaultSearchPaths(appName)
	}

	exts := boa.ConfigFormatExtensions()

	var tried []string

	for i, dir := range searchPaths {
		// First search path: try <appName>.<ext> (e.g., ./myapp.json)
		// Other paths: try config.<ext> (e.g., ~/.config/myapp/config.json)
		var baseName string
		if i == 0 {
			baseName = appName
		} else {
			baseName = "config"
		}

		for _, ext := range exts {
			candidate := filepath.Join(dir, baseName+ext)
			if _, err := os.Stat(candidate); err == nil {
				slog.Debug("boaviper: found config file", "path", candidate)
				return candidate
			}
			tried = append(tried, candidate)
		}

		// Also try <appName>.<ext> in all directories (not just first)
		if i > 0 {
			for _, ext := range exts {
				candidate := filepath.Join(dir, appName+ext)
				if _, err := os.Stat(candidate); err == nil {
					slog.Debug("boaviper: found config file", "path", candidate)
					return candidate
				}
				tried = append(tried, candidate)
			}
		}
	}

	slog.Debug("boaviper: no config file found", "app", appName, "tried", tried)
	return ""
}

// AutoConfig returns an InitFunc that automatically discovers and sets the
// config file path if the user hasn't explicitly provided one via CLI flag.
//
// It finds the first field tagged with configfile:"true" in the params struct,
// and if its value is empty (not set by CLI), sets it to the discovered path.
//
// Usage:
//
//	boa.CmdT[Params]{
//	    Use:      "myapp",
//	    InitFunc: boaviper.AutoConfig("myapp"),
//	    // ...
//	}
//
// With custom search paths:
//
//	boa.CmdT[Params]{
//	    Use:      "myapp",
//	    InitFunc: boaviper.AutoConfig("myapp", "./config", "/opt/myapp"),
//	    // ...
//	}
func AutoConfig[T any](appName string, searchPaths ...string) func(params *T, cmd *cobra.Command) error {
	return func(params *T, cmd *cobra.Command) error {
		// Find the configfile field and set it if empty
		v := reflect.ValueOf(params).Elem()
		t := v.Type()

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if tag := field.Tag.Get("configfile"); tag == "true" {
				fieldVal := v.Field(i)
				if fieldVal.Kind() == reflect.String && fieldVal.String() == "" {
					path := FindConfig(appName, searchPaths...)
					if path != "" {
						fieldVal.SetString(path)
					}
				}
				break
			}
		}

		return nil
	}
}

// AutoConfigCtx is like AutoConfig but returns an InitFuncCtx (with HookContext).
func AutoConfigCtx[T any](appName string, searchPaths ...string) func(ctx *boa.HookContext, params *T, cmd *cobra.Command) error {
	initFunc := AutoConfig[T](appName, searchPaths...)
	return func(ctx *boa.HookContext, params *T, cmd *cobra.Command) error {
		return initFunc(params, cmd)
	}
}

// SetEnvPrefix creates a combined enricher that auto-generates env var names
// AND prefixes them, similar to Viper's SetEnvPrefix.
//
// This combines ParamEnricherEnv (to generate env names from flag names)
// with ParamEnricherEnvPrefix (to add the prefix).
//
// Usage:
//
//	boa.CmdT[Params]{
//	    ParamEnrich: boa.ParamEnricherCombine(
//	        boa.ParamEnricherDefault,
//	        boaviper.SetEnvPrefix("MYAPP"),
//	    ),
//	}
//
// A field `Port int` becomes env var `MYAPP_PORT`.
func SetEnvPrefix(prefix string) boa.ParamEnricher {
	return boa.ParamEnricherCombine(
		boa.ParamEnricherEnv,
		boa.ParamEnricherEnvPrefix(strings.ToUpper(prefix)),
	)
}
