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

`errnie` trims that noise while staying idiomatic. No magic globals beyond an optional logger, no reflection-heavy frameworks, and no parallel error hierarchy — just helpers that compose with normal Go, `errors.Is`, `errors.As`, and `errors.Join`.

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

### `ErrnieError` — one canonical typed error

Instead of a zoo of `ValidationError`, `IOError`, and `HTTPError` types, errnie uses a single structured error with a `Kind` discriminator. Domain semantics (`NotFound`, `Unauthorized`, `Validation`) map cleanly across REST, gRPC, databases, and queues — translate to HTTP status codes at the boundary, not in core logic.

```go
return errnie.E(
    errnie.Validation,
    "email is invalid",
    err,
).With("field", "email")

// classify anywhere in the stack
if errnie.IsNotFound(err) {
    ...
}

// works through wrapping
var e *errnie.ErrnieError
if errors.As(err, &e) {
    switch e.Kind {
    case errnie.NotFound:
        ...
    case errnie.Validation:
        ...
    }
}
```

| Kind | Typical boundary mapping |
|------|--------------------------|
| `Validation` | 400 |
| `Unauthorized` | 401 |
| `Forbidden` | 403 |
| `NotFound` | 404 |
| `Conflict` | 409 |
| `Timeout` | 408 / 504 |

Constructors and helpers:

```go
errnie.E(kind, message, cause)          // wrap with kind + message
errnie.Combine(errs...)               // nil-safe join; 2-error fast path
errnie.AsErrnie(err)                    // extract typed error
errnie.IsNotFound(err)                  // kind checks via errors.As
errnie.IsContext(err)                   // context.Canceled / DeadlineExceeded
```

`ErrnieError` supports `Unwrap()` for causal chains, optional `Operation("user.load")` metadata, and `With("key", value)` fields for observability. No stack traces by default — keep the hot path fast.

Build and enrich errors with `E`, `Operation`, and `With` before sharing them across goroutines; concurrent mutation is not supported.

Pair with `Does` for ergonomic side effects:

```go
errnie.Does(func() (User, error) {
    return repo.Find(id)
}).Or(func(err error) {
    if errnie.IsNotFound(err) {
        errnie.Info("user absent", "id", id)
        return
    }
    errnie.Error(err, "id", id)
})
```

Cleanup and concurrent shutdown:

```go
return errnie.Combine(
    tx.Rollback(),
    conn.Close(),
)
```

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

**Elasticsearch performance**

The sink uses the official [`go-elasticsearch`](https://github.com/elastic/go-elasticsearch) client with a [fasthttp](https://github.com/valyala/fasthttp) transport (same approach as the [official example](https://github.com/elastic/go-elasticsearch/blob/master/_examples/fasthttp/fasthttp.go)), tuned for log shipping:

- **fasthttp transport** — pooled request/response buffers and lower allocation HTTP
- **No retries** — failed log lines are not retried (avoids amplifying backpressure)
- **Auto-drain responses** — connections return to the pool without reading full bodies on success
- **Pooled request bodies** — `bytes.Reader` reuse per index call
- **Async writer** — logging goroutines never block on Elasticsearch RTT

For debugging client traffic during development, the go-elasticsearch [custom logger example](https://github.com/elastic/go-elasticsearch/blob/master/_examples/logging/custom.go) shows how to plug a transport logger into the underlying client — not recommended on production hot paths.

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

| API                | Package  | Role                                      |
|--------------------|----------|-------------------------------------------|
| `Error`, `Info`, … | `errnie` | Structured logging with return-on-error   |
| `E`, `ErrnieError` | `errnie` | Canonical typed errors with `Kind`        |
| `Combine`          | `errnie` | Nil-safe `errors.Join` helper             |
| `Apply`, `Config`  | `errnie` | Multi-sink logger configuration           |
| `Does`, `Result`   | `errnie` | Typed `(T, error)` wrapper                |
| `Require`          | `errnie` | Constructor dependency validation         |
| `SuppressLogging`  | `errnie` | Scoped log suppression                    |

Built on [phuslu/log](https://github.com/phuslu/log) for fast, structured JSON logging.

---

## Contributing

Issues and PRs welcome at [github.com/TheApeMachine/errnie](https://github.com/TheApeMachine/errnie).

---

## License

See repository for license details.
