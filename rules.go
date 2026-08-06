package errnie

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

var emailPattern = regexp.MustCompile(
	`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$`,
)

/* RuleFunc validates one reflected value for a custom registered rule. */
type RuleFunc func(value reflect.Value, parameter string) error

/* ruleConfigurationError distinguishes invalid tags from invalid runtime data. */
type ruleConfigurationError struct {
	message string
}

/* Error implements error for invalid validation declarations. */
func (err *ruleConfigurationError) Error() string {
	return err.message
}

func compileRule(
	definition ruleDefinition,
	parameter string,
	fieldType reflect.Type,
) (fieldValidator, error) {
	if definition.compile != nil {
		return definition.compile(parameter, fieldType)
	}

	return func(value reflect.Value) error {
		return definition.validate(value, parameter)
	}, nil
}

func nestedType(typeOf reflect.Type) bool {
	return validationType(typeOf, make(map[reflect.Type]struct{}))
}

func validationType(typeOf reflect.Type, visited map[reflect.Type]struct{}) bool {
	for typeOf.Kind() == reflect.Pointer {
		typeOf = typeOf.Elem()
	}

	switch typeOf.Kind() {
	case reflect.Interface:
		return true
	case reflect.Slice, reflect.Array:
		return validationType(typeOf.Elem(), visited)
	case reflect.Struct:
		if _, exists := visited[typeOf]; exists {
			return false
		}

		visited[typeOf] = struct{}{}
		structValidatorType := reflect.TypeFor[StructValidator]()

		if typeOf.Implements(structValidatorType) ||
			reflect.PointerTo(typeOf).Implements(structValidatorType) {
			return true
		}

		for field := range typeOf.Fields() {
			field := field
			tag := field.Tag.Get("validate")

			if tag != "" && tag != "-" {
				return true
			}

			if tag != "-" && field.IsExported() && validationType(field.Type, visited) {
				return true
			}
		}
	}

	return false
}

func parseRule(declaration string) (string, string) {
	name, parameter, _ := strings.Cut(declaration, "=")

	return strings.TrimSpace(name), strings.TrimSpace(parameter)
}

func indirectValue(value reflect.Value) (reflect.Value, bool) {
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return reflect.Value{}, false
		}

		value = value.Elem()
	}

	return value, value.IsValid()
}

func indirectType(typeOf reflect.Type) reflect.Type {
	for typeOf.Kind() == reflect.Pointer {
		typeOf = typeOf.Elem()
	}

	return typeOf
}

func validateRequired(value reflect.Value, _ string) error {
	if !value.IsValid() || value.IsZero() {
		return errors.New("is required")
	}

	if (value.Kind() == reflect.String || value.Kind() == reflect.Array ||
		value.Kind() == reflect.Slice || value.Kind() == reflect.Map) && value.Len() == 0 {
		return errors.New("is required")
	}

	return nil
}

func compileMin(parameter string, fieldType reflect.Type) (fieldValidator, error) {
	return compileBound(parameter, fieldType, true)
}

func compileMax(parameter string, fieldType reflect.Type) (fieldValidator, error) {
	return compileBound(parameter, fieldType, false)
}

func compileBound(
	parameter string,
	fieldType reflect.Type,
	minimum bool,
) (fieldValidator, error) {
	kind := indirectType(fieldType).Kind()

	switch kind {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map:
		bound, err := strconv.Atoi(parameter)

		if err != nil {
			return nil, &ruleConfigurationError{message: "bound must be an integer"}
		}

		return func(value reflect.Value) error {
			value, present := indirectValue(value)

			if !present {
				return nil
			}

			length := value.Len()

			if kind == reflect.String {
				length = utf8.RuneCountInString(value.String())
			}

			return compareLength(kind, length, bound, minimum)
		}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		bound, err := strconv.ParseInt(parameter, 10, 64)

		if err != nil {
			return nil, &ruleConfigurationError{message: "bound must be an integer"}
		}

		return func(value reflect.Value) error {
			value, present := indirectValue(value)

			if !present {
				return nil
			}

			return compareInt(value.Int(), bound, minimum)
		}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		bound, err := strconv.ParseUint(parameter, 10, 64)

		if err != nil {
			return nil, &ruleConfigurationError{message: "bound must be a positive integer"}
		}

		return func(value reflect.Value) error {
			value, present := indirectValue(value)

			if !present {
				return nil
			}

			return compareUint(value.Uint(), bound, minimum)
		}, nil
	case reflect.Float32, reflect.Float64:
		bound, err := strconv.ParseFloat(parameter, 64)

		if err != nil || math.IsNaN(bound) || math.IsInf(bound, 0) {
			return nil, &ruleConfigurationError{message: "bound must be a finite number"}
		}

		return func(value reflect.Value) error {
			value, present := indirectValue(value)

			if !present {
				return nil
			}

			if math.IsNaN(value.Float()) || math.IsInf(value.Float(), 0) {
				return errors.New("must be finite")
			}

			return compareFloat(value.Float(), bound, minimum)
		}, nil
	default:
		return nil, &ruleConfigurationError{
			message: "min and max require a collection or number",
		}
	}
}

func compareLength(kind reflect.Kind, length, bound int, minimum bool) error {
	if kind == reflect.String && minimum && length < bound {
		return fmt.Errorf("must be at least %d characters long", bound)
	}

	if kind == reflect.String && !minimum && length > bound {
		return fmt.Errorf("must be at most %d characters long", bound)
	}

	if minimum && length < bound {
		return fmt.Errorf("must have at least %d items", bound)
	}

	if !minimum && length > bound {
		return fmt.Errorf("must have at most %d items", bound)
	}

	return nil
}

func compareInt(number, bound int64, minimum bool) error {
	if minimum && number < bound {
		return fmt.Errorf("must be at least %d", bound)
	}

	if !minimum && number > bound {
		return fmt.Errorf("must be at most %d", bound)
	}

	return nil
}

func compareUint(number, bound uint64, minimum bool) error {
	if minimum && number < bound {
		return fmt.Errorf("must be at least %d", bound)
	}

	if !minimum && number > bound {
		return fmt.Errorf("must be at most %d", bound)
	}

	return nil
}

func compareFloat(number, bound float64, minimum bool) error {
	if minimum && number < bound {
		return fmt.Errorf("must be at least %s", strconv.FormatFloat(bound, 'f', -1, 64))
	}

	if !minimum && number > bound {
		return fmt.Errorf("must be at most %s", strconv.FormatFloat(bound, 'f', -1, 64))
	}

	return nil
}

func validateEmail(value reflect.Value, _ string) error {
	value, present := indirectValue(value)

	if !present || value.Kind() == reflect.String && value.String() == "" {
		return nil
	}

	if value.Kind() != reflect.String {
		return &ruleConfigurationError{message: "email requires a string"}
	}

	if !emailPattern.MatchString(value.String()) {
		return errors.New("must be a valid email address")
	}

	return nil
}

func compileOneOf(parameter string, fieldType reflect.Type) (fieldValidator, error) {
	if parameter == "" {
		return nil, &ruleConfigurationError{
			message: "oneof requires at least one allowed value",
		}
	}

	kind := indirectType(fieldType).Kind()
	allowed := strings.Split(parameter, "|")

	signed := kind >= reflect.Int && kind <= reflect.Int64
	unsigned := kind >= reflect.Uint && kind <= reflect.Uintptr

	if kind != reflect.String && !signed && !unsigned {
		return nil, &ruleConfigurationError{message: "oneof requires a string or integer"}
	}

	return func(value reflect.Value) error {
		value, present := indirectValue(value)

		if !present {
			return nil
		}

		var actual string

		switch value.Kind() {
		case reflect.String:
			actual = value.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			actual = strconv.FormatInt(value.Int(), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			actual = strconv.FormatUint(value.Uint(), 10)
		}

		if slices.Contains(allowed, actual) {
			return nil
		}

		return fmt.Errorf("must be one of: %s", strings.Join(allowed, ", "))
	}, nil
}

func validateFinite(value reflect.Value, _ string) error {
	value, present := indirectValue(value)

	if !present {
		return nil
	}

	if value.Kind() != reflect.Float32 && value.Kind() != reflect.Float64 {
		return &ruleConfigurationError{message: "finite requires a floating-point number"}
	}

	if math.IsNaN(value.Float()) || math.IsInf(value.Float(), 0) {
		return errors.New("must be finite")
	}

	return nil
}

func validateNonnegative(value reflect.Value, _ string) error {
	value, present := indirectValue(value)

	if !present {
		return nil
	}

	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value.Int() < 0 {
			return errors.New("must not be negative")
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return nil
	case reflect.Float32, reflect.Float64:
		if math.IsNaN(value.Float()) || math.IsInf(value.Float(), 0) {
			return errors.New("must be finite")
		}

		if value.Float() < 0 {
			return errors.New("must not be negative")
		}
	default:
		return &ruleConfigurationError{message: "nonnegative requires a number"}
	}

	return nil
}
