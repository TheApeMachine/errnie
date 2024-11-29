# errnie

A modern, ergonomic error handling and logging package for Go that prioritizes developer experience and robust error management.

## Features

### 🎯 Smart Error Handling

-   **Must Pattern**: Simplified error handling with `Must` and `MustVoid` functions
-   **Safe Operations**: Built-in panic recovery with `SafeMust` and `SafeMustVoid`
-   **Chainable Operations**: Fluent error handling with `Op`, `OpValue`, and `OpPtr`

### 📝 Advanced Logging

-   **Rich Context**: Automatic stack traces and code snippets for errors
-   **Flexible Output**: Console and file logging with environment-based configuration
-   **Pretty Formatting**: Beautiful console output using lipgloss styles
-   **Thread Safety**: Concurrent-safe logging operations

### 🛠️ Developer Tools

-   **Debug Support**: Built-in goroutine monitoring and debug logging
-   **Configurable Levels**: Support for trace, debug, info, warn, and error levels
-   **Raw Object Inspection**: Deep inspection of objects using spew

## Installation

```bash
go get github.com/yourusername/errnie/v3
```

## Quick Start

```go
import "github.com/yourusername/errnie/v3"

// Initialize the logger
v3.InitLogger()

// Use Must pattern for clean error handling
result := v3.Must(someFunction())

// Safe operations with automatic recovery
v3.SafeMustVoid(func() error {
    // Your code here
    return nil
})

// Rich error logging
if err := riskyOperation(); err != nil {
    return v3.Error(err, "failed during risky operation")
}
```

## Configuration

Environment variables:

-   `LOGFILE=true`: Enable file logging
-   `NOCONSOLE=true`: Disable console output
-   `LOGGOROUTINES=true`: Enable goroutine monitoring

Viper configuration:

-   `loglevel`: Set logging level (trace, debug, info, warn, error)

## Advanced Usage

### Chainable Operations

```go
type Config struct {
    Port int
    Host string
}

func (c *Config) SetPort(port int) error { /* ... */ }
func (c *Config) SetHost(host string) error { /* ... */ }

// Chain operations with automatic error handling
config := &Config{}
v3.Must(config,
    v3.OpPtr(Config.SetPort, 8080),
    v3.OpPtr(Config.SetHost, "localhost"),
)
```

### Safe Operations with Fallbacks

```go
result := v3.SafeMust(
    func() (int, error) {
        return riskyComputation()
    },
    func(p interface{}) {
        cleanup()
    },
    func(p interface{}) {
        metrics.RecordPanic(p)
    },
)
```

SafeMust supports optional fallback handlers that execute in order if a panic occurs. Each fallback receives the panic value, allowing for:

-   Cleanup operations
-   Metric recording
-   Custom error handling
-   Resource management
-   Graceful degradation

This makes it ideal for operations that need to maintain system stability even when things go wrong.

## Best Practices

1. Initialize the logger early in your application:

    ```go
    func main() {
        v3.InitLogger()
        // ... rest of your application
    }
    ```

2. Use `Must` for initialization code where errors should halt the program:

    ```go
    config := v3.Must(LoadConfig())
    ```

3. Use `SafeMust` for operations that should gracefully handle failures:

    ```go
    result := v3.SafeMust(nonCriticalOperation)
    ```

4. Leverage the logging levels appropriately:
    ```go
    v3.Debug("Detailed information for debugging")
    v3.Info("General information about program execution")
    v3.Warn("Warning messages for concerning but non-critical issues")
    v3.Error(err, "Critical errors that need attention")
    ```

## License

MIT License - See LICENSE file for details

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.