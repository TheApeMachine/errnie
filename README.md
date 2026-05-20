# errnie

**Structured logging and error utilities for Go — without the ceremony.**

`errnie` is a small toolkit for the parts of Go that repeat in every project: log an error and return it, validate dependencies in constructors, run a function and decide what to do when it fails. One import, sensible defaults, zero framework lock-in.

```go
return errnie.Error(fmt.Errorf("connect db: %w", err), "host", cfg.Host)
```

Log it. Return it. Move on.

---

## Why errnie?

Go makes you write the same error-handling boilerplate over and over:

```go
if err != nil {
    log.Error().Err(err).Msg(err.Error())
    return err
}
```

And the same constructor guardrails:

```go
if db == nil {
    return nil, errors.New("db is required")
}
if cache == nil {
    return nil, errors.New("cache is required")
}
```

`errnie` trims that noise while staying idiomatic. No magic globals beyond an optional logger, no custom error types you have to learn, no reflection-heavy frameworks. Just helpers that compose with normal Go.

---

## Install

```bash
go get github.com/theapemachine/errnie
```

Requires **Go 1.26+**.

---

## Quick start

```go
package main

import (
    "fmt"

    "github.com/theapemachine/errnie"
)

func loadUser(id string) (string, error) {
    if id == "" {
        return "", fmt.Errorf("id is required")
    }
    return "ernie", nil
}

func main() {
    result := errnie.Does(func() (string, error) {
        return loadUser("ernie")
    }).Or(func(err error) {
        errnie.Warn("load failed", "err", err)
    })

    if err := result.Err(); err != nil {
        return
    }

    errnie.Info("loaded user", "name", result.Value())
}
```

---

## Features

### `Error` — log and return in one line

The workhorse. Logs at error level when `err` is non-nil, then returns the same error so you can chain it into `return` statements.

```go
func Save(ctx context.Context, doc Document) error {
    if err := db.Insert(ctx, doc); err != nil {
        return errnie.Error(err, "doc_id", doc.ID)
    }
    return nil
}
```

Also available: `Info`, `Warn`, `Debug`, and `Trace` with alternating key/value fields.

```go
errnie.Info("server started", "addr", addr, "version", version)
```

---

### `Does` — run, wrap, decide

`Does` executes a function and wraps its `(T, error)` result. No panics, no hidden control flow — just a typed container you can inspect or chain.

```go
result := errnie.Does(func() (*User, error) {
    return repo.Find(id)
}).Or(func(err error) {
    errnie.Warn("find user failed", "id", id, "err", err)
})

if err := result.Err(); err != nil {
    return err
}

return greet(result.Value())
```

| Method   | Purpose                                      |
|----------|----------------------------------------------|
| `Value()` | Returns the value from `fn`                 |
| `Err()`   | Returns the error from `fn`, or `nil`       |
| `Or(fn)`  | Calls `fn(err)` only on failure; chainable  |

The wrapper is effectively free on the hot path — zero allocations when you use named functions and keep the result typed (see [Benchmarks](#benchmarks)).

---

### `Require` — fail fast in constructors

Validates required dependencies after options are applied. Catches the Go interface-nil trap (typed nil pointers in `any` slots) and reports missing names in stable sorted order.

```go
func NewService(db *sql.DB, cache *redis.Client) (*Service, error) {
    if err := errnie.Require(map[string]any{
        "db":    db,
        "cache": cache,
    }); err != nil {
        return nil, err
    }
    return &Service{db: db, cache: cache}, nil
}
// → "cache is required"
```

---

### Logging configuration

Call `Apply` after loading config (e.g. from Viper) to reconfigure the global logger. By default, logs go to stdout.

```go
var cfg errnie.Config
// load cfg from Viper, env, etc.

errnie.Apply(&cfg)
```

`Config` supports stdout, optional file output, and Elasticsearch indexing via the official [`go-elasticsearch`](https://github.com/elastic/go-elasticsearch) client.

Example YAML (mapstructure tags):

```yaml
level: info

file:
  active: true
  path: /var/log/myapp/app.log

elasticsearch:
  active: true
  url: https://localhost:9200
  index: myapp-logs
  username: elastic
  password: changeme
```

When multiple sinks are active, each log entry is written to all of them. Elasticsearch writes are async with a bounded buffer — if the queue fills, entries are discarded rather than blocking your app.

Supported log levels: `trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic`.

---

### `SuppressLogging` — quiet during tests

Disable errnie logging for a scope and restore it when done. Useful in tests or REPL sessions where expected errors would clutter output.

```go
restore := errnie.SuppressLogging()
defer restore()

// errnie.Error(err) returns err but does not log
_ = errnie.Error(fmt.Errorf("expected failure"))
```

---

## Benchmarks

The `Does` / `Result` API is designed to disappear at compile time:

```
BenchmarkDoes/success          ~0.3 ns/op    0 allocs/op
BenchmarkResultOr/success      ~0.3 ns/op    0 allocs/op
BenchmarkResultValue/success   ~0.3 ns/op    0 allocs/op
BenchmarkResultErr/success     ~0.3 ns/op    0 allocs/op
```

Run them yourself:

```bash
go test -bench=. -benchmem -run='^$'
```

**Tips for zero allocations in production:**

- Pass named functions to `Does`, not inline closures
- Keep `Result[T]` typed — avoid assigning to `any`
- Errors from `errors.New` / `fmt.Errorf` allocate on their own; that's expected

---

## Testing

Tests use [GoConvey](https://github.com/smartystreets/goconvey) with BDD-style `Given` / `When` / `Then` blocks.

```bash
go test ./...
```

---

## What's in the box

| API                | Package        | Role                                      |
|--------------- ----|----------------|-------------------------------------------|
| `Error`, `Info`, … | `errnie`       | Structured logging with return-on-error   |
| `Apply`, `Config`  | `errnie`       | Multi-sink logger configuration           |
| `Does`, `Result`   | `errnie`       | Typed `(T, error)` wrapper                |
| `Require`          | `errnie`       | Constructor dependency validation         |
| `SuppressLogging`  | `errnie`       | Scoped log suppression                    |

Built on [phuslu/log](https://github.com/phuslu/log) for fast, structured JSON logging.

---

## Contributing

Issues and PRs welcome at [github.com/TheApeMachine/errnie](https://github.com/TheApeMachine/errnie).

---

## License

See repository for license details.
