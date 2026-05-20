package errnie

import (
	"sync"
	"sync/atomic"
)

/*
LogController controls errnie logger emission. Suppression depth is tracked with
atomics so the read path used by every log call avoids mutex contention.
*/
type LogController struct {
	suppressed atomic.Int32
}

var defaultLogController = &LogController{}

/*
SuppressLogging disables errnie logging until the returned restore function is
called.
*/
func SuppressLogging() func() {
	return defaultLogController.Suppress()
}

/*
Suppress disables logging for this controller.
*/
func (controller *LogController) Suppress() func() {
	if controller == nil {
		return func() {}
	}

	controller.suppressed.Add(1)

	var once sync.Once

	return func() {
		once.Do(func() {
			if controller.suppressed.Load() == 0 {
				return
			}

			controller.suppressed.Add(-1)
		})
	}
}

/*
Suppressed reports whether logging is currently disabled.
*/
func (controller *LogController) Suppressed() bool {
	if controller == nil {
		return false
	}

	return controller.suppressed.Load() > 0
}

/*
loggingSuppressed reports whether the package-level log controller has disabled
emission. Used by Error, Warn, Info, Debug, and Trace before writing.
*/
func loggingSuppressed() bool {
	return defaultLogController.suppressed.Load() > 0
}
