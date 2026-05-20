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

| Method    | Purpose                                     |
|-----------|---------------------------------------------|
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
goos: darwin
goarch: arm64
pkg: github.com/theapemachine/errnie
cpu: Apple M4 Max
BenchmarkDoes/success-16                            	 1000000000	 0.6399 ns/op      0 B/op    0 allocs/op
BenchmarkDoes/error-16                              	 1000000000	 0.6391 ns/op      0 B/op    0 allocs/op
BenchmarkDoes/string_success-16                     	 1000000000	 0.7664 ns/op      0 B/op    0 allocs/op
BenchmarkResultOr/success-16                        	 1000000000	 0.6417 ns/op      0 B/op    0 allocs/op
BenchmarkResultOr/error-16                          	 1000000000	 0.9117 ns/op      0 B/op    0 allocs/op
BenchmarkResultOr/chained_error-16                  	 716525034	  .550  ns/op	   0 B/op	 0 allocs/op
BenchmarkResultValue/success-16                     	 1000000000	 0.5161 ns/op      0 B/op    0 allocs/op
BenchmarkResultValue/error-16                       	 1000000000	 0.5193 ns/op      0 B/op    0 allocs/op
BenchmarkResultValue/string_success-16              	 1000000000	 0.6460 ns/op      0 B/op    0 allocs/op
BenchmarkResultErr/success-16                       	 1000000000	 0.6508 ns/op      0 B/op    0 allocs/op
BenchmarkResultErr/error-16                         	 1000000000	 0.6515 ns/op      0 B/op    0 allocs/op
BenchmarkResultErr/custom_error-16                  	 1000000000	 0.6589 ns/op      0 B/op    0 allocs/op
BenchmarkNewElasticPostWriter/without_auth-16         	     125133	   9979 ns/op  22069 B/op  647 allocs/op
BenchmarkNewElasticPostWriter/with_auth-16            	     121344	  10010 ns/op  22303 B/op  651 allocs/op
BenchmarkElasticPostWriterWrite/empty_payload-16      	  436279089	  2.717 ns/op      0 B/op    0 allocs/op
BenchmarkElasticPostWriterWrite/json_payload-16       	      38526	  30874 ns/op   5569 B/op   67 allocs/op
BenchmarkE/without_cause-16                           	   60908037	  19.08 ns/op     96 B/op    1 allocs/op
BenchmarkE/with_cause-16                              	   64959230	  18.91 ns/op     96 B/op    1 allocs/op
BenchmarkErrnieErrorOperation-16                      	 1000000000	  1.026 ns/op      0 B/op    0 allocs/op
BenchmarkErrnieErrorWith/two_fields-16                	  138482296	  8.643 ns/op      0 B/op    0 allocs/op
BenchmarkErrnieErrorError-16                          	  931117060	  1.303 ns/op      0 B/op    0 allocs/op
BenchmarkErrnieErrorUnwrap-16                         	 1000000000	 0.6540 ns/op      0 B/op    0 allocs/op
BenchmarkCombine/all_nil-16                           	  395448062	  3.138 ns/op      0 B/op    0 allocs/op
BenchmarkCombine/single_error-16                      	  292563076	  4.006 ns/op      0 B/op    0 allocs/op
BenchmarkCombine/multiple_errors-16                   	   67949688	  17.27 ns/op     32 B/op    1 allocs/op
BenchmarkAsErrnie-16                                  	  273330852	  4.368 ns/op      0 B/op    0 allocs/op
BenchmarkIsKind-16                                    	  294363700	  4.067 ns/op      0 B/op    0 allocs/op
BenchmarkIsNotFound-16                                	  273817381	  4.345 ns/op      0 B/op    0 allocs/op
BenchmarkIsContext-16                                 	  181802181	  6.602 ns/op      0 B/op    0 allocs/op
BenchmarkKindString-16                                	 1000000000	 0.9072 ns/op      0 B/op    0 allocs/op
BenchmarkHotpathErrorReturn/nil_error-16              	  762642186	  1.571 ns/op      0 B/op    0 allocs/op
BenchmarkHotpathErrorReturn/typed_error-16            	   10155147	  118.4 ns/op      0 B/op    0 allocs/op
BenchmarkHotpathErrorReturn/suppressed_typed_error-16 	  555781771	  2.145 ns/op      0 B/op    0 allocs/op
BenchmarkHotpathDoesReturn/success-16                 	  421653920	  2.845 ns/op      0 B/op    0 allocs/op
BenchmarkHotpathDoesReturn/failure-16                 	 1000000000	  1.123 ns/op      0 B/op    0 allocs/op
BenchmarkHotpathLoggingDisabledCaller/info-16         	    8210484	  144.5 ns/op     56 B/op    2 allocs/op
BenchmarkHotpathLoggingSuppressedCheck/not_suppressed-16 1000000000	 0.6424 ns/op      0 B/op    0 allocs/op
BenchmarkHotpathLoggingSuppressedCheck/suppressed-16     1000000000	 0.6405 ns/op     0 B/op	 0 allocs/op
BenchmarkHotpathCombineCleanup-16                          68692951	  17.02 ns/op    32 B/op	 1 allocs/op
BenchmarkApply-16                                          32264822	  37.83 ns/op    64 B/op	 2 allocs/op
BenchmarkBuildWriter/stdout_only-16                        42158145	  28.20 ns/op    64 B/op	 2 allocs/op
BenchmarkBuildWriter/stdout_and_file-16                    18574533	  62.62 ns/op   200 B/op	 4 allocs/op
BenchmarkParseLogLevel/debug-16                           185859067	  6.418 ns/op     0 B/op	 0 allocs/op
BenchmarkParseLogLevel/default-16                         100000000	  11.37 ns/op     0 B/op	 0 allocs/op
BenchmarkNewLogger-16                                     155489390	  7.734 ns/op     8 B/op	 1 allocs/op
BenchmarkError/nil_error-16                               776331172	  1.550 ns/op     0 B/op	 0 allocs/op
BenchmarkError/non-nil_error-16                             7933128	  150.8 ns/op    56 B/op	 2 allocs/op
BenchmarkError/suppressed_non-nil_error-16                556747665	  2.163 ns/op     0 B/op	 0 allocs/op
BenchmarkWarn/enabled-16                                    8086954	  147.1 ns/op    56 B/op	 2 allocs/op
BenchmarkWarn/suppressed-16                               724749097	  1.648 ns/op     0 B/op	 0 allocs/op
BenchmarkInfo/enabled-16                                    8206581	  146.8 ns/op    56 B/op	 2 allocs/op
BenchmarkInfo/suppressed-16                               721928270	  1.652 ns/op     0 B/op	 0 allocs/op
BenchmarkDebug/enabled-16                                   8206884	  145.6 ns/op    56 B/op	 2 allocs/op
BenchmarkDebug/suppressed-16                              730278855	  1.644 ns/op     0 B/op	 0 allocs/op
BenchmarkTrace/enabled-16                                   8013890	  148.0 ns/op    56 B/op	 2 allocs/op
BenchmarkTrace/suppressed-16                              720418563	  1.673 ns/op     0 B/op	 0 allocs/op
BenchmarkSuppressLogging/suppress_and_restore-16           85586419	  13.88 ns/op    16 B/op	 1 allocs/op
BenchmarkLogControllerSuppress/suppress_and_restore-16     89261080	  13.50 ns/op    16 B/op	 1 allocs/op
BenchmarkLogControllerSuppressed/not_suppressed-16       1000000000	 0.6487 ns/op     0 B/op	 0 allocs/op
BenchmarkLogControllerSuppressed/suppressed-16           1000000000	 0.8086 ns/op     0 B/op	 0 allocs/op
BenchmarkLoggingSuppressed/not_suppressed-16             1000000000	 0.6473 ns/op     0 B/op	 0 allocs/op
BenchmarkMissingDependency/absent_nil_interface-16        928865662	  1.307 ns/op     0 B/op	 0 allocs/op
BenchmarkMissingDependency/absent_typed_nil_pointer-16    468857982	  2.160 ns/op     0 B/op	 0 allocs/op
BenchmarkMissingDependency/present_pointer-16             530263695	  2.193 ns/op     0 B/op	 0 allocs/op
BenchmarkRequire/success-16                                24744328	  49.57 ns/op     0 B/op	 0 allocs/op
BenchmarkRequire/missing_dependency-16                     15565909	  77.38 ns/op    32 B/op	 2 allocs/op
PASS
coverage: 72.1% of statements
ok  	github.com/theapemachine/errnie	81.056s
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
