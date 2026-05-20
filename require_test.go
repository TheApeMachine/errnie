package errnie

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestMissingDependency verifies nil-interface and typed-nil detection used by
Require.
*/
func TestMissingDependency(t *testing.T) {
	Convey("Given a nil interface value", t, func() {
		var value any

		Convey("When missingDependency is called", func() {
			absent := missingDependency(value)

			Convey("Then it should report the dependency as missing", func() {
				So(absent, ShouldBeTrue)
			})
		})
	})

	Convey("Given a typed nil pointer stored in an any slot", t, func() {
		var pointer *int
		var value any = pointer

		Convey("When missingDependency is called", func() {
			absent := missingDependency(value)

			Convey("Then it should report the dependency as missing", func() {
				So(absent, ShouldBeTrue)
			})
		})
	})

	Convey("Given typed nil map, slice, channel, and function values", t, func() {
		var (
			mapping map[string]int
			items   []int
			ch      chan int
			fn      func()
		)

		Convey("When missingDependency is called for each", func() {
			Convey("Then each should be reported as missing", func() {
				So(missingDependency(any(mapping)), ShouldBeTrue)
				So(missingDependency(any(items)), ShouldBeTrue)
				So(missingDependency(any(ch)), ShouldBeTrue)
				So(missingDependency(any(fn)), ShouldBeTrue)
			})
		})
	})

	Convey("Given present dependencies", t, func() {
		value := 42
		pointer := &value
		mapping := map[string]int{"ok": 1}
		items := []int{1}
		ch := make(chan int)
		fn := func() {}

		Convey("When missingDependency is called", func() {
			Convey("Then each should be reported as present", func() {
				So(missingDependency(value), ShouldBeFalse)
				So(missingDependency(pointer), ShouldBeFalse)
				So(missingDependency(mapping), ShouldBeFalse)
				So(missingDependency(items), ShouldBeFalse)
				So(missingDependency(ch), ShouldBeFalse)
				So(missingDependency(fn), ShouldBeFalse)
			})
		})
	})
}

/*
TestRequire verifies dependency validation and stable missing-name reporting.
*/
func TestRequire(t *testing.T) {
	Convey("Given an empty dependency map", t, func() {
		Convey("When Require is called", func() {
			err := Require(map[string]any{})

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("Given all dependencies are present", t, func() {
		value := 1

		Convey("When Require is called", func() {
			err := Require(map[string]any{
				"cache": &value,
				"db":    &value,
			})

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("Given a single missing dependency", t, func() {
		value := 1

		Convey("When Require is called", func() {
			err := Require(map[string]any{
				"cache": &value,
				"db":    nil,
			})

			Convey("Then it should return a clear required error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "db is required")
			})
		})
	})

	Convey("Given multiple missing dependencies", t, func() {
		Convey("When Require is called", func() {
			err := Require(map[string]any{
				"cache": nil,
				"db":    nil,
			})

			Convey("Then it should report the first missing name in sorted order", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "cache is required")
			})
		})
	})

	Convey("Given a typed nil pointer dependency", t, func() {
		var pointer *int

		Convey("When Require is called", func() {
			err := Require(map[string]any{
				"pool": pointer,
			})

			Convey("Then it should treat typed nil as missing", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "pool is required")
			})
		})
	})
}

var (
	benchmarkRequirePresent = 1
	benchmarkRequireErr     error
)

/*
BenchmarkMissingDependency measures missingDependency for present and absent
values.
*/
func BenchmarkMissingDependency(b *testing.B) {
	var pointer *int
	present := &benchmarkRequirePresent

	b.Run("absent nil interface", func(b *testing.B) {
		var value any
		for range b.N {
			benchmarkRequireMissingSink = missingDependency(value)
		}
	})

	b.Run("absent typed nil pointer", func(b *testing.B) {
		for range b.N {
			benchmarkRequireMissingSink = missingDependency(pointer)
		}
	})

	b.Run("present pointer", func(b *testing.B) {
		for range b.N {
			benchmarkRequireMissingSink = missingDependency(present)
		}
	})
}

/*
BenchmarkRequire measures Require on success and failure paths.
*/
func BenchmarkRequire(b *testing.B) {
	present := map[string]any{
		"cache": &benchmarkRequirePresent,
		"db":    &benchmarkRequirePresent,
	}

	b.Run("success", func(b *testing.B) {
		for range b.N {
			benchmarkRequireErr = Require(present)
		}
	})

	b.Run("missing dependency", func(b *testing.B) {
		absent := map[string]any{
			"cache": &benchmarkRequirePresent,
			"db":    nil,
		}

		for range b.N {
			benchmarkRequireErr = Require(absent)
		}
	})
}

var (
	benchmarkRequireMissingSink bool
)
