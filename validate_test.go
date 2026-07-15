package errnie

import (
	"errors"
	"math"
	"reflect"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/* validationFixture exercises every built-in rule through one cached schema. */
type validationFixture struct {
	Name  string   `validate:"required"`
	Count int      `validate:"min=1,max=3"`
	Ratio *float64 `validate:"finite"`
	Email string   `validate:"email"`
	Mode  string   `validate:"oneof=fast|safe"`
}

/* crossFieldFixture exercises StructValidator after tagged field validation. */
type crossFieldFixture struct {
	Name    string `validate:"required"`
	From    int
	Through int
}

/* nestedChildFixture supplies one tagged field for recursive path assertions. */
type nestedChildFixture struct {
	Name string `validate:"required"`
}

/* nestedFixture covers structs, slices, and aliased pointers during recursion. */
type nestedFixture struct {
	Child  nestedChildFixture
	Items  []nestedChildFixture
	Alias  *nestedChildFixture
	Repeat *nestedChildFixture
}

/* recursiveFixture proves cyclic pointers terminate on the active path. */
type recursiveFixture struct {
	Name string `validate:"required"`
	Next *recursiveFixture
}

/*
ValidateStruct rejects a backwards fixture interval so Validator cross-field
dispatch is exercised without coupling the test to an external domain type.
*/
func (fixture crossFieldFixture) ValidateStruct() error {
	if fixture.Through < fixture.From {
		return errors.New("interval ends before it starts")
	}

	return nil
}

/*
TestValidationErrors verifies stable aggregate rendering and error traversal.
*/
func TestValidationErrors(t *testing.T) {
	Convey("Given two structured field failures", t, func() {
		failures := ValidationErrors{
			{Field: "Name", Rule: "required", Message: "is required"},
			{Field: "Mode", Rule: "oneof", Message: "must be one of: fast, safe"},
		}

		Convey("When the aggregate is rendered and unwrapped", func() {
			Convey("Then order and field identity remain inspectable", func() {
				So(failures.Error(), ShouldEqual,
					"Name: is required\nMode: must be one of: fast, safe")
				So(failures.Unwrap(), ShouldHaveLength, 2)

				var failure *ValidationError
				So(errors.As(failures, &failure), ShouldBeTrue)
				So(failure.Field, ShouldEqual, "Name")
			})
		})
	})
}

/*
TestValidatorValidate verifies field tags, cross-field validation, error kinds,
and custom rule registration through the public Validator surface.
*/
func TestValidatorValidate(t *testing.T) {
	Convey("Given a fully valid tagged struct", t, func() {
		ratio := 0.5
		fixture := validationFixture{
			Name: "book", Count: 2, Ratio: &ratio,
			Email: "book@example.com", Mode: "safe",
		}

		Convey("When it is validated", func() {
			err := Validate(&fixture)

			Convey("Then every built-in rule accepts it", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("Given independent invalid fields", t, func() {
		nonFinite := math.Inf(1)
		fixture := validationFixture{
			Count: 4, Ratio: &nonFinite,
			Email: "not-an-address", Mode: "unsafe",
		}

		Convey("When it is validated", func() {
			err := Validate(fixture)

			Convey("Then all data failures are returned as validation errors", func() {
				So(err, ShouldNotBeNil)
				So(IsValidation(err), ShouldBeTrue)
				So(err.Error(), ShouldContainSubstring, "Name: is required")
				So(err.Error(), ShouldContainSubstring, "Count: must be at most 3")
				So(err.Error(), ShouldContainSubstring, "Ratio: must be finite")

				var failures ValidationErrors
				So(errors.As(err, &failures), ShouldBeTrue)
				So(failures, ShouldHaveLength, 5)
			})
		})
	})

	Convey("Given a value with a backwards cross-field interval", t, func() {
		fixture := crossFieldFixture{Name: "book", From: 2, Through: 1}

		Convey("When it is validated", func() {
			err := Validate(fixture)

			Convey("Then its StructValidator invariant is preserved as a validation error", func() {
				So(err, ShouldNotBeNil)
				So(IsValidation(err), ShouldBeTrue)
				So(err.Error(), ShouldContainSubstring, "interval ends before it starts")
			})
		})
	})

	Convey("Given invalid validator inputs", t, func() {
		var fixture *validationFixture

		Convey("When nil and non-struct values are validated", func() {
			Convey("Then both fail explicitly as validation errors", func() {
				nilErr := Validate(fixture)
				valueErr := Validate("book")
				So(IsValidation(nilErr), ShouldBeTrue)
				So(nilErr.Error(), ShouldContainSubstring, "cannot validate nil pointer")
				So(IsValidation(valueErr), ShouldBeTrue)
				So(valueErr.Error(), ShouldContainSubstring, "validate expects a struct")
			})
		})
	})

	Convey("Given malformed rule declarations", t, func() {
		type unknownFixture struct {
			Value string `validate:"missing"`
		}
		type parameterFixture struct {
			Value int `validate:"min=nope"`
		}

		Convey("When their schemas are evaluated", func() {
			unknownErr := Validate(unknownFixture{Value: "value"})
			parameterErr := Validate(parameterFixture{Value: 1})

			Convey("Then programmer errors are not mislabeled as bad runtime data", func() {
				So(IsInternal(unknownErr), ShouldBeTrue)
				So(unknownErr.Error(), ShouldContainSubstring, "unknown rule")
				So(IsInternal(parameterErr), ShouldBeTrue)
				So(parameterErr.Error(), ShouldContainSubstring, "bound must be an integer")
			})
		})
	})

	Convey("Given a validation tag on an unexported field", t, func() {
		type hiddenFixture struct {
			hidden string `validate:"required"`
		}

		Convey("When its schema is evaluated", func() {
			err := Validate(hiddenFixture{})

			Convey("Then the silently unreachable rule is a configuration error", func() {
				So(IsInternal(err), ShouldBeTrue)
				So(err.Error(), ShouldContainSubstring, "unexported but has a validate tag")
			})
		})
	})

	Convey("Given a registered domain rule", t, func() {
		type evenFixture struct {
			Value int `validate:"even"`
		}

		RegisterRule("even", func(value reflect.Value, _ string) error {
			if value.Int()%2 != 0 {
				return errors.New("must be even")
			}

			return nil
		})

		Convey("When matching and non-matching values are validated", func() {
			Convey("Then the custom rule participates in normal validation", func() {
				So(Validate(evenFixture{Value: 2}), ShouldBeNil)
				So(Validate(evenFixture{Value: 3}).Error(),
					ShouldContainSubstring, "Value: must be even")
			})
		})
	})

	Convey("Given an empty custom rule registration", t, func() {
		Convey("When registration is attempted", func() {
			Convey("Then the programmer error fails immediately", func() {
				So(func() { RegisterRule("", nil) }, ShouldPanic)
			})
		})
	})
}

/*
TestValidatorRecursive verifies nested paths, collections, and pointer cycles.
*/
func TestValidatorRecursive(t *testing.T) {
	Convey("Given invalid nested structs and a struct slice", t, func() {
		shared := &nestedChildFixture{}
		fixture := nestedFixture{
			Items:  []nestedChildFixture{{}},
			Child:  *shared,
			Alias:  shared,
			Repeat: shared,
		}

		Convey("When the root value is validated", func() {
			err := Validate(&fixture)

			Convey("Then every nested failure retains its full path", func() {
				So(IsValidation(err), ShouldBeTrue)
				So(err.Error(), ShouldContainSubstring, "Child.Name: is required")
				So(err.Error(), ShouldContainSubstring, "Items[0].Name: is required")
				So(err.Error(), ShouldContainSubstring, "Alias.Name: is required")
				So(err.Error(), ShouldContainSubstring, "Repeat.Name: is required")
			})
		})
	})

	Convey("Given a self-referential valid struct", t, func() {
		fixture := &recursiveFixture{Name: "root"}
		fixture.Next = fixture

		Convey("When it is recursively validated", func() {
			Convey("Then the pointer cycle terminates without duplicating work", func() {
				So(Validate(fixture), ShouldBeNil)
			})
		})
	})
}

/* TestValidatorSchemaCache verifies parse-once reuse and invalidation. */
func TestValidatorSchemaCache(t *testing.T) {
	Convey("Given a configured validator and valid fixture", t, func() {
		ratio := 0.5
		fixture := validationFixture{
			Name: "book", Count: 2, Ratio: &ratio,
			Email: "book@example.com", Mode: "safe",
		}
		typeOf := reflect.TypeOf(fixture)

		Convey("When the same type is validated repeatedly", func() {
			So(Validate(&fixture), ShouldBeNil)
			first, firstExists := validatorCtx.schemas[typeOf]
			So(Validate(&fixture), ShouldBeNil)
			second, secondExists := validatorCtx.schemas[typeOf]

			Convey("Then its compiled schema is reused", func() {
				So(firstExists, ShouldBeTrue)
				So(secondExists, ShouldBeTrue)
				So(first, ShouldEqual, second)
			})
		})

		Convey("When a custom rule changes the registry", func() {
			So(Validate(&fixture), ShouldBeNil)
			RegisterRule("custom", func(reflect.Value, string) error {
				return nil
			})
			_, exists := validatorCtx.schemas[typeOf]

			Convey("Then stale compiled schemas are removed", func() {
				So(exists, ShouldBeFalse)
			})
		})
	})
}

var benchmarkValidationErr error

/*
BenchmarkValidatorValidate measures the real tagged struct path on accepted and
rejected data.
*/
func BenchmarkValidatorValidate(b *testing.B) {
	ratio := 0.5
	valid := validationFixture{
		Name: "book", Count: 2, Ratio: &ratio,
		Email: "book@example.com", Mode: "safe",
	}
	invalid := validationFixture{Count: 4, Email: "invalid", Mode: "unsafe"}

	b.Run("valid", func(b *testing.B) {
		for b.Loop() {
			benchmarkValidationErr = Validate(&valid)
		}
	})

	b.Run("invalid", func(b *testing.B) {
		for b.Loop() {
			benchmarkValidationErr = Validate(&invalid)
		}
	})
}
