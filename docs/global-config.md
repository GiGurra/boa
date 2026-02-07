# Global Configuration

Boa supports global configuration via `boa.Init()` with functional options. Call `Init` before creating any commands to change default behavior.

## Usage

```go
package main

import "github.com/GiGurra/boa/pkg/boa"

func main() {
    boa.Init(
        boa.WithDefaultOptional(),
    )

    // Create commands as usual...
}
```

## Available Options

### `WithDefaultOptional()`

By default, raw Go type fields (`string`, `int`, `bool`, etc.) in parameter structs are **required**. This means users must provide a value or the command will fail with a validation error.

`WithDefaultOptional()` changes this default so that raw Go type fields are **optional** instead. This is useful when most of your fields have sensible zero values and you only want to require a few specific ones.

```go
boa.Init(
    boa.WithDefaultOptional(),
)
```

#### Override Precedence

Explicit annotations always take precedence over the global default:

| Mechanism | Behavior |
|-----------|----------|
| `Required[T]` wrapper | Always required |
| `Optional[T]` wrapper | Always optional |
| `required:"true"` / `req:"true"` tag | Always required |
| `optional:"true"` / `opt:"true"` tag | Always optional |
| Raw Go type (no tag) | Follows global default |

#### Example

```go
boa.Init(boa.WithDefaultOptional())

type Params struct {
    Name   string            `descr:"user name"`           // optional (global default)
    Port   int               `descr:"port" required:"true"` // required (explicit tag)
    Debug  boa.Optional[bool] `descr:"debug mode"`          // optional (wrapper)
    Output boa.Required[string] `descr:"output file"`       // required (wrapper)
}
```

## Without Init

If you don't call `boa.Init()`, all behavior remains unchanged from previous versions. Raw Go type fields default to required, maintaining full backwards compatibility.
