package errnie

import (
	"github.com/pterm/pterm"
	"github.com/spf13/viper"
)

/* Panic is a pretty printed panic line. */
func (collector Collector) Panic(msg string) { pterm.Fatal.Println(msg); panic(nil) }

/* Fatal is a pretty printed fatal line. */
func (collector Collector) Fatal(msg string) { pterm.Fatal.Println(msg) }

/* Critical is a pretty printed critical line. */
func (collector Collector) Critical(msg string) { pterm.Error.Println(msg) }

/* Error is a pretty printed error line. */
func (collector Collector) Error(msg string) { pterm.Error.Println(msg) }

/* Info is a pretty printed info line. */
func (collector Collector) Info(msg string) { pterm.Info.Println(msg) }

/* Warn is a pretty printed warning line. */
func (collector Collector) Warn(msg string) { pterm.Warning.Println(msg) }

/* Debug is a pretty printed info line. */
func (collector Collector) Debug(msg string) {
	// Don't print noise if we are not in debug mode.
	if !viper.GetBool("debug") {
		return
	}
	pterm.Debug.Println(msg)
}
