package errnie

import (
	"math"
	"reflect"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestValidateRequired verifies non-zero required semantics and pointer presence.
*/
func TestValidateRequired(t *testing.T) {
	Convey("Given zero and present values", t, func() {
		zero := 0
		presentZero := &zero

		Convey("When required validation is applied", func() {
			Convey("Then zero and empty collections are absent while a non-nil pointer is present", func() {
				So(validateRequired(reflect.ValueOf(zero), ""), ShouldNotBeNil)
				So(validateRequired(reflect.ValueOf([]int{}), ""), ShouldNotBeNil)
				So(validateRequired(reflect.ValueOf(presentZero), ""), ShouldBeNil)
			})
		})
	})
}

/*
TestValidateBound verifies numeric and collection min/max rules.
*/
func TestValidateBound(t *testing.T) {
	Convey("Given numeric and collection values", t, func() {
		Convey("When minimum and maximum rules are applied", func() {
			Convey("Then compatible boundaries pass and violations fail", func() {
				floatMin, err := compileMin("2", reflect.TypeFor[float64]())
				So(err, ShouldBeNil)
				floatMax, err := compileMax("2", reflect.TypeFor[float64]())
				So(err, ShouldBeNil)
				listMin, err := compileMin("2", reflect.TypeFor[[]int]())
				So(err, ShouldBeNil)
				listMax, err := compileMax("1", reflect.TypeFor[[]int]())
				So(err, ShouldBeNil)
				stringMin, err := compileMin("2", reflect.TypeFor[string]())
				So(err, ShouldBeNil)
				stringMax, err := compileMax("2", reflect.TypeFor[string]())
				So(err, ShouldBeNil)
				uintMax, err := compileMax("9007199254740992", reflect.TypeFor[uint64]())
				So(err, ShouldBeNil)

				So(floatMin(reflect.ValueOf(2.5)), ShouldBeNil)
				So(floatMax(reflect.ValueOf(2.5)), ShouldNotBeNil)
				So(floatMin(reflect.ValueOf(math.NaN())), ShouldNotBeNil)
				So(listMin(reflect.ValueOf([]int{1, 2})), ShouldBeNil)
				So(listMax(reflect.ValueOf([]int{1, 2})), ShouldNotBeNil)
				So(stringMin(reflect.ValueOf("gö")), ShouldBeNil)
				So(stringMax(reflect.ValueOf("gö")), ShouldBeNil)
				So(uintMax(reflect.ValueOf(uint64(9_007_199_254_740_993))), ShouldNotBeNil)
			})
		})
	})
}

/*
TestValidateEmail verifies optional and malformed mailbox handling.
*/
func TestValidateEmail(t *testing.T) {
	Convey("Given optional, valid, and malformed email strings", t, func() {
		Convey("When email validation is applied", func() {
			Convey("Then only the malformed address fails", func() {
				So(validateEmail(reflect.ValueOf(""), ""), ShouldBeNil)
				So(validateEmail(reflect.ValueOf("book@example.com"), ""), ShouldBeNil)
				So(validateEmail(reflect.ValueOf("book@localhost"), ""), ShouldNotBeNil)
			})
		})
	})
}

/*
TestValidateOneOf verifies named-value membership for strings and integers.
*/
func TestValidateOneOf(t *testing.T) {
	Convey("Given string and integer allowed sets", t, func() {
		Convey("When membership validation is applied", func() {
			Convey("Then listed values pass and unlisted values fail", func() {
				stringRule, err := compileOneOf("fast|safe", reflect.TypeFor[string]())
				So(err, ShouldBeNil)
				integerRule, err := compileOneOf("1|2", reflect.TypeFor[int]())
				So(err, ShouldBeNil)

				So(stringRule(reflect.ValueOf("safe")), ShouldBeNil)
				So(integerRule(reflect.ValueOf(2)), ShouldBeNil)
				So(stringRule(reflect.ValueOf("unsafe")), ShouldNotBeNil)
			})
		})
	})
}

/*
TestValidateFinite verifies optional pointers and non-finite rejection.
*/
func TestValidateFinite(t *testing.T) {
	Convey("Given finite, absent, and non-finite floating-point values", t, func() {
		var absent *float64
		finite := 0.5
		nonFinite := math.NaN()

		Convey("When finite validation is applied", func() {
			Convey("Then absent optional and finite values pass while NaN fails", func() {
				So(validateFinite(reflect.ValueOf(absent), ""), ShouldBeNil)
				So(validateFinite(reflect.ValueOf(&finite), ""), ShouldBeNil)
				So(validateFinite(reflect.ValueOf(&nonFinite), ""), ShouldNotBeNil)
			})
		})
	})
}

/*
TestValidateNonnegative verifies signed, unsigned, and non-finite numbers.
*/
func TestValidateNonnegative(t *testing.T) {
	Convey("Given nonnegative, negative, unsigned, and infinite values", t, func() {
		Convey("When nonnegative validation is applied", func() {
			Convey("Then only negative and non-finite values fail", func() {
				So(validateNonnegative(reflect.ValueOf(0), ""), ShouldBeNil)
				So(validateNonnegative(reflect.ValueOf(-1), ""), ShouldNotBeNil)
				So(validateNonnegative(reflect.ValueOf(uint(1)), ""), ShouldBeNil)
				So(validateNonnegative(reflect.ValueOf(math.Inf(1)), ""), ShouldNotBeNil)
			})
		})
	})
}

var benchmarkRuleErr error

/*
BenchmarkValidateFinite measures the finite-pointer rule used by measurement
validation.
*/
func BenchmarkValidateFinite(b *testing.B) {
	finite := 0.5
	value := reflect.ValueOf(&finite)

	for b.Loop() {
		benchmarkRuleErr = validateFinite(value, "")
	}
}
