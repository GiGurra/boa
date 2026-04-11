# Live Config Reload

Long-running programs — servers, daemons, background workers — often want to re-read config without restarting. BOA ships a primitive for this: `boa.Reload[T](ctx) (*T, error)`. It re-runs the entire post-flag-parse pipeline — CLI → env → config files → defaults → validation — on a **freshly allocated** `*T` and returns it.

## Quick Start

```go
package main

import (
    "log"
    "os"
    "os/signal"
    "sync/atomic"
    "syscall"

    "github.com/GiGurra/boa/pkg/boa"
    "github.com/spf13/cobra"
)

type Params struct {
    ConfigFile string `configfile:"true" optional:"true"`
    Host       string `optional:"true"`
    Port       int    `optional:"true" default:"8080"`
}

var active atomic.Pointer[Params]

func main() {
    boa.CmdT[Params]{
        Use: "server",
        RunFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command, args []string) {
            active.Store(p)

            // Wire a trigger — here, SIGHUP.
            sighup := make(chan os.Signal, 1)
            signal.Notify(sighup, syscall.SIGHUP)
            go func() {
                for range sighup {
                    fresh, err := boa.Reload[Params](ctx)
                    if err != nil {
                        log.Printf("config reload rejected: %v", err) // old state preserved
                        continue
                    }
                    active.Store(fresh)
                    log.Println("config reloaded")
                }
            }()

            startServer()
        },
    }.Run()
}
```

`kill -HUP <pid>` now re-reads every file BOA loaded at startup, re-applies precedence (CLI still wins), re-validates, and hands you a fresh `*Params` to atomically swap. A reader goroutine that does `cfg := active.Load()` always sees a consistent snapshot.

## What Reload does

1. **Allocates a fresh `*T`.** The struct you were handed in `RunFunc` is **not mutated**. Callers decide what to do with the new snapshot: atomic pointer swap, diff for "did the field I care about actually change?", notify subscribers, or discard entirely. BOA doesn't dictate a concurrency model.
2. **Re-runs the full pipeline.** Defaults → env (re-read from the current process environment) → config files (re-read from disk) → CLI precedence (the original startup args still win) → validation → `PreValidate` hooks.
3. **Skips `PreExecuteFunc` and the command's `RunFunc`** — a reload is value-sourcing, not command execution.

## Error Handling: Reload is All-or-Nothing

`Reload` never mutates anything — every call is either a clean success (fresh `*T` returned) or a clean failure (`(nil, err)` returned and nothing else happens). The struct you're holding is a completely separate allocation that Reload can't see; your atomic swap target keeps pointing at whatever it was pointing at before.

| Failure | What the caller sees |
|---|---|
| **File parse error** (malformed JSON/YAML/TOML, truncated mid-write) | Error names the offending file. Nothing allocated, nothing swapped — the previous snapshot is still the live one. |
| **Validation failure** (`min` / `max` / `pattern` / custom validator) | Error describes which field failed. Fresh struct is discarded before it ever leaves Reload. |
| **File disappeared** | Clean read error naming the path. |
| **PreValidate hook error** | Propagated as-is. |

This is deliberate so you can wire `Reload` to a noisy trigger — fsnotify fires 2–5 times per save on most editors — and safely ignore every error. Each failed attempt just logs and keeps serving the existing config:

```go
for range fileChanges {
    fresh, err := boa.Reload[Params](ctx)
    if err != nil {
        log.Printf("reload failed (keeping current config): %v", err)
        continue
    }
    active.Store(fresh)
}
```

## What Reload does NOT do

- **No fsnotify, no SIGHUP handler, no HTTP endpoint.** The primitive just answers "give me a fresh validated config now". Wire whatever trigger makes sense — `signal.Notify(syscall.SIGHUP)`, a timer, a tiny admin endpoint, fsnotify, a test harness. A higher-level watcher subpackage that wraps fsnotify with sane debouncing is planned as a follow-up.
- **No concurrency coordination.** If your goroutines read from a shared `*Params`, you have to coordinate reads against whatever swap model you pick. `atomic.Pointer[T]` is the cleanest, but `sync.RWMutex` works too. BOA refuses to dictate sync for you.
- **No deep merging**, no partial reload of a single file from a chain — the whole pipeline re-runs against the whole input set. Simplest semantics, easiest to reason about.

## Which Files Get Watched?

`ctx.WatchedConfigFiles()` returns the paths a live-reload watcher should listen on. Use this to hand the file set to fsnotify / your custom watcher of choice.

### Auto-tracked

- Every `configfile:"true"` tagged field (single path or `[]string` overlay chain)
- `Cmd.ConfigFormat` / `Cmd.ConfigUnmarshal` per-command escape hatches

### Not auto-tracked

`boa.LoadConfigFile` / `LoadConfigFiles` / `LoadConfigBytes` called from inside a user hook — these are public helpers outside BOA's internal pipeline. Register those explicitly with `ctx.WatchConfigFile(path)` inside the same hook. The registration persists across reloads because the hook re-runs during the replay:

```go
PreValidateFuncCtx: func(ctx *boa.HookContext, p *Params, cmd *cobra.Command, args []string) error {
    if err := boa.LoadConfigFile("/etc/myapp/overrides.json", p, nil); err != nil {
        return err
    }
    ctx.WatchConfigFile("/etc/myapp/overrides.json") // opt in to watching
    return nil
},
```

## Hook Behavior on Reload

| Hook | Runs on reload? |
|---|---|
| `InitFunc` / `InitFuncCtx` | ✅ |
| `PostCreateFunc` / `PostCreateFuncCtx` | ✅ |
| `PreValidateFunc` / `PreValidateFuncCtx` | ✅ |
| `PreExecuteFunc` / `PreExecuteFuncCtx` | ❌ (no main action to run) |
| `RunFunc` / `RunFuncCtx` / `RunFuncE` / `RunFuncCtxE` | ❌ (no main action to run) |
| `CfgStructInit` / `CfgStructPreValidate` interface methods | ✅ |
| `CfgStructPreExecute` interface methods | ❌ |

If you have state-heavy init you don't want re-run on reload, guard with a `sync.Once` or an "already initialized" sentinel inside the hook.

## Typical Triggers

### SIGHUP (POSIX convention)

```go
sighup := make(chan os.Signal, 1)
signal.Notify(sighup, syscall.SIGHUP)
go func() {
    for range sighup {
        if fresh, err := boa.Reload[Params](ctx); err == nil {
            active.Store(fresh)
        }
    }
}()
```

### Admin HTTP endpoint

```go
http.HandleFunc("/admin/reload", func(w http.ResponseWriter, r *http.Request) {
    fresh, err := boa.Reload[Params](ctx)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    active.Store(fresh)
    w.WriteHeader(http.StatusNoContent)
})
```

### fsnotify (watch the directory, not the file, to survive atomic-rename-saves)

```go
watcher, _ := fsnotify.NewWatcher()
defer watcher.Close()
watched := ctx.WatchedConfigFiles()
dirs := map[string]bool{}
for _, p := range watched {
    dirs[filepath.Dir(p)] = true
}
for d := range dirs {
    watcher.Add(d)
}
targets := map[string]bool{}
for _, p := range watched {
    targets[p] = true
}
debounce := time.NewTimer(time.Hour)
debounce.Stop()
for {
    select {
    case ev := <-watcher.Events:
        if targets[ev.Name] {
            debounce.Reset(200 * time.Millisecond)
        }
    case <-debounce.C:
        if fresh, err := boa.Reload[Params](ctx); err == nil {
            active.Store(fresh)
        }
    }
}
```

### Timer (poll every N seconds)

```go
ticker := time.NewTicker(30 * time.Second)
defer ticker.Stop()
for range ticker.C {
    if fresh, err := boa.Reload[Params](ctx); err == nil {
        active.Store(fresh)
    }
}
```

## Reading the Active Config from Worker Goroutines

The atomic-pointer pattern keeps readers lock-free and always-consistent:

```go
var active atomic.Pointer[Params]

func handleRequest(w http.ResponseWriter, r *http.Request) {
    cfg := active.Load() // always points at a fully-validated, immutable snapshot
    fmt.Fprintf(w, "host=%s port=%d\n", cfg.Host, cfg.Port)
}
```

Each `active.Load()` returns the pointer that was current when the load began. A concurrent `Store` from the reload goroutine can swap it — in-flight readers continue with the old snapshot, new readers see the new one. No torn reads, no locks.

If you need to react to specific changes — "port changed, restart the listener" — diff the old and new snapshots after the swap:

```go
old := active.Load()
fresh, err := boa.Reload[Params](ctx)
if err != nil {
    return
}
active.Store(fresh)
if fresh.Port != old.Port {
    go restartListener(fresh.Port)
}
```
