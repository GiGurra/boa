// Example: registering a custom config file format (unmarshal + KeyTree) so
// that the *same compiled binary* can load config files in any registered
// format — JSON (built in), a custom "KV" format (registered here), and by
// extension anything else you plug in.
//
// The command does not lock itself to a single format. boa dispatches at
// runtime based on the file extension of the --config-file argument, so:
//
//	./server --config-file prod.json   # uses the built-in JSON handler
//	./server --config-file prod.kv     # uses the handler registered here
//
// Switching formats in production is just a different --config-file; no
// code change, no rebuild.
//
// This example uses a trivially small "KV" format for illustration only. In a
// real application you would substitute the unmarshal/KeyTree functions for
// your favourite parser (gopkg.in/yaml.v3, github.com/BurntSushi/toml, …) and
// keep boa itself free of that dependency.
//
// The config file looks like:
//
//	host = db.internal
//	port = 5432
//	db.host =
//	db.port = 5432
//
// Lines are "key = value", with "." introducing nesting. Quotes/escapes are
// intentionally ignored; the point is to show the ConfigFormat integration,
// not to ship a production parser.
package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/GiGurra/boa/pkg/boa"
	"github.com/spf13/cobra"
)

type DBConfig struct {
	Host string `descr:"database host" default:"localhost"`
	Port int    `descr:"database port" default:"5432"`
}

type Params struct {
	ConfigFile string `configfile:"true" optional:"true"`
	Host       string `descr:"app host" default:"localhost"`
	Port       int    `descr:"app port" default:"8080"`
	// Optional parameter group. The .kv format's KeyTree probe lets boa
	// see that "db.host" and "db.port" are written even when they equal
	// the zero value / the default — so this pointer survives cleanup.
	DB *DBConfig
}

// parseKV returns the key/value pairs and a nested key tree that mirrors
// the dotted-path structure. It deliberately shares work between Unmarshal
// and KeyTree in real code you would split them if the parser is expensive.
func parseKV(data []byte) (flat map[string]string, tree map[string]any, err error) {
	flat = map[string]string{}
	tree = map[string]any{}
	for lineNo, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			return nil, nil, fmt.Errorf("line %d: missing '='", lineNo+1)
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		flat[key] = val

		// Walk the dotted path in the tree, creating sub-maps on the way.
		segments := strings.Split(key, ".")
		cursor := tree
		for i, seg := range segments {
			if i == len(segments)-1 {
				cursor[seg] = val
				break
			}
			next, ok := cursor[seg].(map[string]any)
			if !ok {
				next = map[string]any{}
				cursor[seg] = next
			}
			cursor = next
		}
	}
	return flat, tree, nil
}

// kvUnmarshal populates `target` from the parsed KV pairs using reflection.
// Top-level keys map onto struct fields; dotted keys descend into sub-structs.
func kvUnmarshal(data []byte, target any) error {
	flat, _, err := parseKV(data)
	if err != nil {
		return err
	}
	root := reflect.ValueOf(target).Elem()
	for key, val := range flat {
		if err := assignDotted(root, strings.Split(key, "."), val); err != nil {
			return fmt.Errorf("key %q: %w", key, err)
		}
	}
	return nil
}

func assignDotted(v reflect.Value, segments []string, val string) error {
	if len(segments) == 0 {
		return nil
	}
	field := lookupField(v, segments[0])
	if !field.IsValid() {
		return nil // unknown key — ignore
	}
	// Follow a pointer-to-struct if present (boa preallocates these for
	// optional parameter groups, so deref is always safe here).
	if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}
	if len(segments) > 1 {
		return assignDotted(field, segments[1:], val)
	}
	return setScalar(field, val)
}

func lookupField(v reflect.Value, name string) reflect.Value {
	if v.Kind() != reflect.Struct {
		return reflect.Value{}
	}
	if f := v.FieldByName(name); f.IsValid() {
		return f
	}
	// Case-insensitive fallback, matching encoding/json behaviour.
	for i := 0; i < v.NumField(); i++ {
		if strings.EqualFold(v.Type().Field(i).Name, name) {
			return v.Field(i)
		}
	}
	return reflect.Value{}
}

func setScalar(f reflect.Value, val string) error {
	switch f.Kind() {
	case reflect.String:
		f.SetString(val)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return err
		}
		f.SetInt(n)
	case reflect.Bool:
		b, err := strconv.ParseBool(val)
		if err != nil {
			return err
		}
		f.SetBool(b)
	}
	return nil
}

// kvKeyTree returns the nested key tree boa needs for set-by-config detection.
// Scalar leaf values come through as plain strings; boa only inspects presence.
func kvKeyTree(data []byte) (map[string]any, error) {
	_, tree, err := parseKV(data)
	return tree, err
}

// Observed records what each RunFuncCtx invocation saw — used by the tests to
// assert on the actual parsed state instead of just exit status.
type Observed struct {
	Host         string
	Port         int
	DBPresent    bool
	DBHost       string
	DBPort       int
	DBHostViaCfg bool
	DBPortViaCfg bool
}

// newServerCmd builds the command and wires it to capture the parsed state.
// It is the testable entry point; main() just runs the same builder.
func newServerCmd(obs *Observed) boa.CmdT[Params] {
	return boa.CmdT[Params]{
		Use:         "server",
		Short:       "Server with a custom config file format",
		ParamEnrich: boa.ParamEnricherName,
		RunFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command, args []string) {
			obs.Host, obs.Port = p.Host, p.Port
			fmt.Printf("Host: %s, Port: %d\n", p.Host, p.Port)
			if p.DB == nil {
				fmt.Println("DB: <unset>")
				return
			}
			obs.DBPresent = true
			obs.DBHost, obs.DBPort = p.DB.Host, p.DB.Port
			obs.DBHostViaCfg = ctx.HasValue(&p.DB.Host)
			obs.DBPortViaCfg = ctx.HasValue(&p.DB.Port)
			fmt.Printf("DB.Host: %q (set by config: %t)\n", p.DB.Host, obs.DBHostViaCfg)
			fmt.Printf("DB.Port: %d (set by config: %t)\n", p.DB.Port, obs.DBPortViaCfg)
		},
	}
}

// registerKVFormat registers the custom .kv format. Separated from main() so
// tests can call it exactly once via sync.Once semantics at package level.
func registerKVFormat() {
	boa.RegisterConfigFormatFull(".kv", boa.ConfigFormat{
		Unmarshal: kvUnmarshal,
		KeyTree:   kvKeyTree,
	})
}

func main() {
	registerKVFormat()
	var obs Observed
	newServerCmd(&obs).Run()
}
