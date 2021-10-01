package errnie

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSpec(t *testing.T) {
	Convey("Given no instantiation method", t, func() {
		Convey("It should already be available", func() {
			So(ambctx, ShouldNotBeNil)
		})

		Convey("It should be able to log messages", func() {
			err := "error"

			So(ambctx.Log("test").OK, ShouldEqual, false)
			So(ambctx.Log(err).ERR, ShouldEqual, ERROR)
		})
	})
}
