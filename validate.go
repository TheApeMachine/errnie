package errnie

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

var validatorCtx *Validator

func init() {
	validatorCtx = NewValidator()
}

/*
ValidationError identifies the field and rule that rejected one value.
*/
type ValidationError struct {
	Field   string
	Rule    string
	Message string
}

/* Error implements error without copying the structured field failure. */
func (err *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", err.Field, err.Message)
}

/*
ValidationErrors collects every independent failure from one validation pass.
*/
type ValidationErrors []*ValidationError

/* Error implements error with stable field order. */
func (errs ValidationErrors) Error() string {
	var message strings.Builder

	for index, err := range errs {
		if index > 0 {
			message.WriteByte('\n')
		}

		message.WriteString(err.Error())
	}

	return message.String()
}

/* Unwrap exposes individual field failures to errors.Is and errors.As. */
func (errs ValidationErrors) Unwrap() []error {
	unwrapped := make([]error, len(errs))

	for index := range errs {
		unwrapped[index] = errs[index]
	}

	return unwrapped
}

/*
StructValidator adds cross-field rules after tagged fields are validated.
*/
type StructValidator interface {
	ValidateStruct() error
}

/* fieldValidator is a parsed rule closure reused by cached schemas. */
type fieldValidator func(value reflect.Value) error

/* ruleCompiler parses rule parameters once for a concrete field type. */
type ruleCompiler func(parameter string, fieldType reflect.Type) (fieldValidator, error)

/* ruleDefinition holds either a direct rule or its parse-once compiler. */
type ruleDefinition struct {
	validate RuleFunc
	compile  ruleCompiler
}

/* compiledRule retains one named rule closure for a cached field schema. */
type compiledRule struct {
	name     string
	validate fieldValidator
}

/* compiledField retains the work required for one exported struct field. */
type compiledField struct {
	index  int
	name   string
	rules  []compiledRule
	nested bool
}

/* structSchema is the cached validation plan for one concrete struct type. */
type structSchema struct {
	fields []compiledField
	err    error
}

/* visit identifies a pointer on the active recursive traversal path. */
type visit struct {
	typeOf  reflect.Type
	pointer uintptr
}

/*
Validator recursively validates structs using cached compiled field schemas.
It is intentionally unsynchronized, so callers serialize access to an instance.
*/
type Validator struct {
	rules   map[string]ruleDefinition
	schemas map[reflect.Type]*structSchema
}

/* New constructs a Validator with the built-in rules. */
func NewValidator() *Validator {
	return &Validator{
		rules: map[string]ruleDefinition{
			"required":    {validate: validateRequired},
			"min":         {compile: compileMin},
			"max":         {compile: compileMax},
			"email":       {validate: validateEmail},
			"oneof":       {compile: compileOneOf},
			"finite":      {validate: validateFinite},
			"nonnegative": {validate: validateNonnegative},
		},
		schemas: make(map[reflect.Type]*structSchema),
	}
}

/*
RegisterRule adds or replaces a custom rule and invalidates compiled schemas.
*/
func RegisterRule(name string, rule RuleFunc) {
	name = strings.TrimSpace(name)

	if name == "" || rule == nil {
		panic("validation rule requires a name and function")
	}

	validatorCtx.rules[name] = ruleDefinition{validate: rule}
	clear(validatorCtx.schemas)
}

/*
Validate recursively checks tagged fields, nested structs and collections, and
StructValidator rules. Schema mistakes are Internal errors; rejected values are
Validation errors.
*/
func Validate(input any) error {
	value := reflect.ValueOf(input)

	if err := requireStruct(value); err != nil {
		return Err(Validation, err.Error(), nil)
	}

	failures, domainErr, configurationErr := walk(
		value, "", make(map[visit]struct{}),
	)

	if configurationErr != nil {
		return Err(
			Internal,
			"invalid validation rule: "+configurationErr.Error(),
			configurationErr,
		)
	}

	if len(failures) == 0 && domainErr == nil {
		return nil
	}

	var fieldErr error

	if len(failures) > 0 {
		fieldErr = failures
	}

	cause := Combine(fieldErr, domainErr)

	return Err(Validation, "validation failed: "+cause.Error(), cause)
}

/* walk recursively validates one reflected value while cutting pointer cycles. */
func walk(
	value reflect.Value,
	path string,
	visited map[visit]struct{},
) (ValidationErrors, error, error) {
	if !value.IsValid() {
		return nil, nil, nil
	}

	kind := value.Kind()

	if (kind == reflect.Interface || kind == reflect.Pointer) && value.IsNil() {
		return nil, nil, nil
	}

	if kind == reflect.Interface {
		return walk(value.Elem(), path, visited)
	}

	if kind == reflect.Pointer {
		identity := visit{typeOf: value.Type(), pointer: value.Pointer()}

		if _, exists := visited[identity]; exists {
			return nil, nil, nil
		}

		visited[identity] = struct{}{}
		failures, domainErr, configurationErr := walk(
			value.Elem(), path, visited,
		)
		delete(visited, identity)

		return failures, domainErr, configurationErr
	}

	switch kind {
	case reflect.Struct:
		return validateStruct(value, path, visited)
	case reflect.Slice, reflect.Array:
		var failures ValidationErrors
		var domainErr error

		for index := range value.Len() {
			itemPath := fmt.Sprintf("%s[%d]", path, index)
			nested, nestedErr, configurationErr := walk(
				value.Index(index), itemPath, visited,
			)
			failures = append(failures, nested...)
			domainErr = Combine(domainErr, nestedErr)

			if configurationErr != nil {
				return nil, nil, configurationErr
			}
		}

		return failures, domainErr, nil
	default:
		return nil, nil, nil
	}
}

/* validateStruct applies a cached schema and then recurses into its fields. */
func validateStruct(
	value reflect.Value,
	path string,
	visited map[visit]struct{},
) (ValidationErrors, error, error) {
	schema := schema(value.Type())

	if schema.err != nil {
		return nil, nil, schema.err
	}

	var failures ValidationErrors
	var domainErr error

	for _, field := range schema.fields {
		fieldValue := value.Field(field.index)
		fieldPath := joinPath(path, field.name)

		for _, rule := range field.rules {
			if err := rule.validate(fieldValue); err != nil {
				var configuration *ruleConfigurationError

				if errors.As(err, &configuration) {
					return nil, nil, fmt.Errorf("%s: %w", fieldPath, err)
				}

				failures = append(failures, &ValidationError{
					Field: fieldPath, Rule: rule.name, Message: err.Error(),
				})
			}
		}

		if field.nested {
			nested, nestedErr, configurationErr := walk(
				fieldValue, fieldPath, visited,
			)
			failures = append(failures, nested...)
			domainErr = Combine(domainErr, nestedErr)

			if configurationErr != nil {
				return nil, nil, configurationErr
			}
		}
	}

	return failures, Combine(domainErr, structValidation(value)), nil
}

/* schema returns one parse-once compiled schema for a struct type. */
func schema(typeOf reflect.Type) *structSchema {
	if compiled, exists := validatorCtx.schemas[typeOf]; exists {
		return compiled
	}

	compiled := compile(typeOf)
	validatorCtx.schemas[typeOf] = compiled

	return compiled
}

/* compile parses field tags and rule parameters outside the validation path. */
func compile(typeOf reflect.Type) *structSchema {
	schema := &structSchema{}

	for index := range typeOf.NumField() {
		fieldType := typeOf.Field(index)
		tag := fieldType.Tag.Get("validate")

		if !fieldType.IsExported() && tag != "" && tag != "-" {
			schema.err = fmt.Errorf(
				"field %s is unexported but has a validate tag", fieldType.Name,
			)

			return schema
		}

		if !fieldType.IsExported() || tag == "-" {
			continue
		}

		field := compiledField{
			index: index, name: fieldType.Name, nested: nestedType(fieldType.Type),
		}

		for declaration := range strings.SplitSeq(tag, ",") {
			name, parameter := parseRule(strings.TrimSpace(declaration))

			if name == "" {
				continue
			}

			definition, exists := validatorCtx.rules[name]

			if !exists {
				schema.err = fmt.Errorf("%s uses unknown rule %q", field.name, name)

				return schema
			}

			compiled, err := compileRule(definition, parameter, fieldType.Type)

			if err != nil {
				schema.err = fmt.Errorf("%s: %w", field.name, err)

				return schema
			}

			field.rules = append(field.rules, compiledRule{name: name, validate: compiled})
		}

		if len(field.rules) > 0 || field.nested {
			schema.fields = append(schema.fields, field)
		}
	}

	return schema
}

func joinPath(prefix, field string) string {
	if prefix == "" {
		return field
	}

	return prefix + "." + field
}

func requireStruct(value reflect.Value) error {
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return errors.New("cannot validate nil pointer")
		}

		value = value.Elem()
	}

	if !value.IsValid() || value.Kind() != reflect.Struct {
		return errors.New("validate expects a struct")
	}

	return nil
}

func structValidation(value reflect.Value) error {
	if value.CanAddr() && value.Addr().CanInterface() {
		if validator, ok := value.Addr().Interface().(StructValidator); ok {
			return validator.ValidateStruct()
		}
	}

	if value.CanInterface() {
		if validator, ok := value.Interface().(StructValidator); ok {
			return validator.ValidateStruct()
		}
	}

	return nil
}
