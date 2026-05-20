package errnie

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestSuppressLogging verifies package-level scoped log suppression.
*/
func TestSuppressLogging(t *testing.T) {
	Convey("Given logging is enabled", t, func() {
		Convey("When SuppressLogging is called and restored", func() {
			So(loggingSuppressed(), ShouldBeFalse)

			restore := SuppressLogging()
			So(loggingSuppressed(), ShouldBeTrue)

			restore()

			Convey("Then logging should be enabled again", func() {
				So(loggingSuppressed(), ShouldBeFalse)
			})
		})
	})

	Convey("Given nested SuppressLogging calls", t, func() {
		Convey("When restore is called in stack order", func() {
			outer := SuppressLogging()
			inner := SuppressLogging()
			So(loggingSuppressed(), ShouldBeTrue)

			inner()
			So(loggingSuppressed(), ShouldBeTrue)

			outer()

			Convey("Then logging should be enabled only after all restores", func() {
				So(loggingSuppressed(), ShouldBeFalse)
			})
		})
	})
}

/*
TestLogControllerSuppress verifies scoped suppression on a LogController.
*/
func TestLogControllerSuppress(t *testing.T) {
	Convey("Given a nil LogController", t, func() {
		var controller *LogController

		Convey("When Suppress is called", func() {
			restore := controller.Suppress()

			Convey("Then it should return a no-op restore function", func() {
				So(restore, ShouldNotBeNil)
				So(func() { restore() }, ShouldNotPanic)
			})
		})
	})

	Convey("Given a LogController with nested Suppress calls", t, func() {
		controller := &LogController{}

		Convey("When Suppress restore functions are called in order", func() {
			outer := controller.Suppress()
			inner := controller.Suppress()
			So(controller.Suppressed(), ShouldBeTrue)

			inner()
			So(controller.Suppressed(), ShouldBeTrue)

			outer()

			Convey("Then the controller should be unsuppressed", func() {
				So(controller.Suppressed(), ShouldBeFalse)
			})
		})
	})

	Convey("Given a restore function called more than once", t, func() {
		controller := &LogController{}
		restore := controller.Suppress()

		Convey("When restore is invoked twice", func() {
			restore()
			restore()

			Convey("Then suppression should only be decremented once", func() {
				So(controller.Suppressed(), ShouldBeFalse)
			})
		})
	})
}

/*
TestLogControllerSuppressed verifies Suppressed on nil and active controllers.
*/
func TestLogControllerSuppressed(t *testing.T) {
	Convey("Given a nil LogController", t, func() {
		var controller *LogController

		Convey("When Suppressed is called", func() {
			suppressed := controller.Suppressed()

			Convey("Then it should report false", func() {
				So(suppressed, ShouldBeFalse)
			})
		})
	})

	Convey("Given an active LogController", t, func() {
		controller := &LogController{}

		Convey("When Suppressed is called before and during suppression", func() {
			So(controller.Suppressed(), ShouldBeFalse)

			restore := controller.Suppress()
			defer restore()

			Convey("Then it should report true while suppressed", func() {
				So(controller.Suppressed(), ShouldBeTrue)
			})
		})
	})
}

/*
TestLoggingSuppressed verifies the package-level loggingSuppressed helper.
*/
func TestLoggingSuppressed(t *testing.T) {
	Convey("Given the default log controller", t, func() {
		restore := SuppressLogging()
		defer restore()

		Convey("When loggingSuppressed is called", func() {
			suppressed := loggingSuppressed()

			Convey("Then it should reflect controller state", func() {
				So(suppressed, ShouldBeTrue)
			})
		})
	})
}

/*
BenchmarkSuppressLogging measures package-level suppress and restore.
*/
func BenchmarkSuppressLogging(b *testing.B) {
	b.Run("suppress and restore", func(b *testing.B) {
		for range b.N {
			restore := SuppressLogging()
			restore()
		}
	})
}

/*
BenchmarkLogControllerSuppress measures controller-level suppress and restore.
*/
func BenchmarkLogControllerSuppress(b *testing.B) {
	controller := &LogController{}

	b.Run("suppress and restore", func(b *testing.B) {
		for range b.N {
			restore := controller.Suppress()
			restore()
		}
	})
}

/*
BenchmarkLogControllerSuppressed measures Suppressed on an active controller.
*/
func BenchmarkLogControllerSuppressed(b *testing.B) {
	controller := &LogController{}

	b.Run("not suppressed", func(b *testing.B) {
		for range b.N {
			benchmarkLoggingSuppressedSink = controller.Suppressed()
		}
	})

	b.Run("suppressed", func(b *testing.B) {
		restore := controller.Suppress()
		defer restore()

		b.ResetTimer()
		for range b.N {
			benchmarkLoggingSuppressedSink = controller.Suppressed()
		}
	})
}

/*
BenchmarkLoggingSuppressed measures the package-level loggingSuppressed helper.
*/
func BenchmarkLoggingSuppressed(b *testing.B) {
	b.Run("not suppressed", func(b *testing.B) {
		for range b.N {
			benchmarkLoggingSuppressedSink = loggingSuppressed()
		}
	})
}

var benchmarkLoggingSuppressedSink bool
