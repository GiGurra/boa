# Quickstart

The fastest way to get started with BOA is to let Claude Code build it for you.

## Using Claude Code

1. Create a new directory for your CLI project:

```bash
mkdir my-cli && cd my-cli
go mod init my-cli
```

2. Feed BOA's README to Claude Code and describe what you want:

```bash
claude "$(curl -s https://raw.githubusercontent.com/GiGurra/boa/main/README.md)

Build me a CLI tool that manages TODO items with commands for:
- add: adds a new todo
- list: lists all todos
- done: marks a todo as complete"
```

3. Run your new CLI:

```bash
go run . --help
```

## Example Prompts

Here are some ideas to try:

**File organizer:**
```
Build me a CLI that organizes files in a directory by extension,
with flags for dry-run mode and a target directory.
```

**API client:**
```
Build me a CLI for interacting with a REST API, with subcommands
for GET, POST, PUT, DELETE and flags for headers and auth tokens.
```

**Dev tool:**
```
Build me a CLI that watches a directory for changes and runs
a command when files change, with flags for patterns to include/exclude.
```

## Manual Setup

If you prefer to start from scratch:

```go
package main

import (
    "fmt"
    "github.com/GiGurra/boa/pkg/boa"
)

type Params struct {
    Name string `descr:"Your name"`
}

func main() {
    boa.NewCmdT[Params]("hello").
        WithShort("Say hello").
        WithRunFunc(func(p *Params) {
            fmt.Printf("Hello, %s!\n", p.Name)
        }).
        Run()
}
```

```bash
go get github.com/GiGurra/boa@latest
go run . --name World
```
