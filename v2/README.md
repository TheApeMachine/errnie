# errnie

Errnie started out as a way to emulate the `rescue` method found in Ruby.

It mutated a few times over into performing highly opinionated error handling as a side-gig.

Eventually I got tied of having to inject it into everything, so its (final?) state now is to be a highly opinionated ambient context handling the things I am tired of handling.

## Usage Patterns

This package provides a suggested ambient context pattern, because I got sick of passing these common things around to every object as an injected dependency. Personally I need this everywhere at all times.

Also it has this concept of collecting errors and making more "complicated" decisions and evaluations of program state.

### Logging Errors

It really just logging anything, but it does have some extra functionality around errors as well.

```go
// Log error to the console (default output) with a log level of ERROR.
err := somepackge.SomeMethod()
errnie.Ambient().Log(errnie.ERROR, err)

// Or...
errnie.Ambient().Log(errnie.ERROR, somepackage.SomeMethod())

// Because...
if ok := errnie.Ambient().Log(errnie.ERROR, somepackage.SomeMethod()); !ok {
    return
}
```

## Things to Know

**You cannot/should not use this library without a (Viper)[https://github.com/spf13/viper] based config workflow.**

errnie assumes a certain amount of configurations to be set such to give it half a change of actually functioning.

Personally I have had a lot of success with an approach like in the code below.

```go
// cmd/root.go
package cmd

//go:embed conf/*
var embedded embed.FS

home, _ := os.UserHomeDir()

// Disregard error handling, for brevity.
if _, err := os.Stat(slug); os.IsNotExist(err) {
    fs, err := embedded.Open("cfg/" + cfgFile)
    defer fs.Close()
    buf, _ := io.ReadAll(fs)
    _ = ioutil.WriteFile(home+"/"+cfgFile, buf, 0644)
}
```

Embed default config into the binary, (re)write it when not present from the embedded file system with sane defaults (including the errnie config).

No more config file handling after that.
