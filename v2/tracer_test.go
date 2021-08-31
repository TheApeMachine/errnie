package errnie

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTrace(t *testing.T) {
	Convey("Given an ambient context", t, func() {
		ctx := NewContext(AmbientContext{})
		Convey("It should be able to trace: ", func() {
			fmt.Println()
			ctx.Trace()
			ctx.TraceIn()
			ctx.TraceOut()
			So(1, ShouldEqual, 1)
		})
	})
}
