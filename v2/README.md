# errnie

Errnie started out as a way to emulate the `rescue` method found in Ruby.

It mutated a few times over into performing highly opinionated error handling as a side-gig.

Eventually I got tired of having to inject it into everything, so its final state now is to be a highly opinionated ambient context handling the things I am tired of handling.

It is a relatively thin wrapper around Go's error object and debug package to simplify error handling and prevent putting if statements all over the code while also giving you a much more direct and verbose tracing feature in the console that can be controlled using a config file.

An example of what to expect when turning debug and traching features on.
```go
❯ go run main.go run
 TRACER  🔍 <- tracer.go 51 TraceIn []
 TRACER  🔍 <- tracer.go 51 TraceIn []
 ERRNIE  🐞 -- logging potential errors [[set session to region ]]
  DEBUG  set session to region
 TRACER  🐞 context.go 39 Handles [[<nil>]]
 TRACER  🐞 context.go 54 Handles [[<nil>]]
 ERRNIE  🐞 -- adding potential errors [[<nil>]]
 ERRNIE  🐞 -- finding real errors [[<nil>]]
 TRACER  🐞 context.go 55 Handles [[<nil>]]
 TRACER  👍 context.go 39 Handles [OK]
```

## Things to Know

**You cannot/should not use this library without a (Viper)[https://github.com/spf13/viper] based config workflow.**

While errnie will technically function without using the viper package it is not a recommended approach, as you will have no control
over its output mode, and it will therefore not show any kind of debug or trace information.

To enable tracing and debug messages your apps config file should have the following key/value setup (I usually just put it at the top).

```yaml
errnie:
  debug: true
  trace: true
```

errnie will use viper's `viper.GetViper().GetString("errnie.debug")` (and the same for trace) approach to pick up the global instance of viper, which also
lives as an ambient context in your app if you use it.

These days it is probably the most common approach for Go programs to use both (cobra)[https://github.com/spf13/cobra] and (viper)[https://github.com/spf13/viper] 
to build a CLI into your app, so the chances are high you are already using this.

To add to that standard patterns and introduce a very low-maintance situation around managing config files I have started a while back to embed a default config file into
the binaries for my projects and personally I have had a lot of success with this approach.

It means that even when you (accidentally) remove the config file, or changed it in such a way that you cannot get back to a working state, all you need to do it (remove it)
run the program and it will write a fresh default to your home directory and of course the same holds true for the first time you ever run the project.

Below is an abbreviated example of how to implement this in a standard `cmd/root.go` setup that is common to all projects implementing `cobra`.

This is compatible with Linux, Mac, and Windows.

```yaml
# ~/myproject/cmd/cfg/.myproject.yml
errnie:
  debug: true
  trace: true

myproject:
  someconfigkey:
    somenestedkey: somevalue
```

```go
// ~/myproject/cmd/root.go
package cmd

/* Use meta comment to embed anything in the cfg directory as a mini file system into the binary */
//go:embed cfg/*
var embedded embed.FS

/* MISSING: viper config file command line flags binding logic */

// Disregard error handling, for brevity.
home, _ := os.UserHomeDir()

// Check if config file exists in home directory of current user, otherwise write out the default
// from the embedded file system.
if _, err := os.Stat(slug); os.IsNotExist(err) {
    fs, err := embedded.Open("cfg/" + cfgFile)
    defer fs.Close()
    buf, _ := io.ReadAll(fs)
    _ = ioutil.WriteFile(home+"/"+cfgFile, buf, 0644)
}

/* MISSING: viper config file loading/initialization logic */
```

## Usage Patterns

### Logging Errors

This package provides a suggested ambient context pattern, because I got sick of passing these common things around to every object as an injected dependency. Personally I need this everywhere at all times.

In its simplest form you can use errnie as a simple one-line replacement for the standard way most people handle errors in Go.

```go
// Instead of...
err := somepackage.SomeMethodThatReturnsAnError()

if err != nil {
    fmt.Println(err)
}

// You can just do...
errnie.Log(somepackage.SomeMethodThatReturnsAnError())
```

errnie itself will hold an exported boolean value representing its internal state, i.e. whether or not an actual error event took place. You can access that in implementation using something like the following.

```go
// This allows you to handle logging the error as well as handling it in a concise manner.
if ok := errnie.Log(somepackage.SomeMethodThatReturnsAnError()).OK; !ok {
    // Do some...
}
```

Should you have a need to get to the actual error inside, this is also an option, though ideally this package was designed for this to be a rather unique situation.

```go
// This allows you to handle logging the error as well as handling it in a concise manner.
if err := errnie.Log(somepackage.SomeMethodThatReturnsAnError()).ERR; err != nil {
    // Do some...
}
```

### Handling Errors

Handling errors is pretty much the same as logging them, however the method to do so has a couple of magic features that are good to be aware about from the start.
