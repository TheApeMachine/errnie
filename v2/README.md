# errnie

Errnie started out as a way to emulate the `rescue` method found in Ruby.

It mutated a few times over into performing highly opinionated error handling as a side-gig.

Eventually I got tied of having to inject it into everything, so its (final?) state now is to be a highly opinionated ambient context handling the things I am tired of handling.

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
    fs, err := embedded.Open("conf/" + cfgFile)
    defer fs.Close()
    buf, _ := io.ReadAll(fs)
    _ = ioutil.WriteFile(home+"/"+cfgFile, buf, 0644)
}
```

Embed default config into the binary, (re)write it when not present.

No more config file handling after that.