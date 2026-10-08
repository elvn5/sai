// Package validation binds and validates JSON request bodies with
// go-playground/validator (`binding` struct tags), and reports failures in
// exactly the shape the NestJS API's ZodValidationPipe produced —
// zod's error.flatten() with zod's wording:
//
//	{"formErrors": [], "fieldErrors": {"filename": ["Required"]}}
//
// so clients see identical 400 responses after an endpoint moves to Go.
package validation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/elvn5/sai/services/pkg/httperr"
)

// Flattened mirrors zod's error.flatten().
type Flattened struct {
	FormErrors  []string            `json:"formErrors"`
	FieldErrors map[string][]string `json:"fieldErrors"`
}

var (
	validate = newValidator()
	// Custom pattern tags → zod's custom regex messages.
	patternMessages sync.Map
)

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.SetTagName("binding")
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "-" {
			return ""
		}
		return name
	})
	return v
}

// RegisterPattern adds a `binding:"<tag>"` rule that requires a string to
// match re, reported with message (zod's z.string().regex(re, message)).
// Call it from an init function.
func RegisterPattern(tag string, re *regexp.Regexp, message string) {
	patternMessages.Store(tag, message)
	if err := validate.RegisterValidation(tag, func(fl validator.FieldLevel) bool {
		return re.MatchString(fl.Field().String())
	}); err != nil {
		panic(fmt.Sprintf("validation: register %q: %v", tag, err))
	}
}

// BindJSON decodes the request body into dst (a pointer to a struct) and
// validates it. On failure it writes the 400 response and returns false.
//
// Like NestJS, a missing or non-JSON body is treated as an empty object, so
// required fields report "Required".
func BindJSON(c *gin.Context, dst any) bool {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		httperr.Abort(c, http.StatusBadRequest, "Could not read request body")
		return false
	}
	if len(bytes.TrimSpace(body)) == 0 || !strings.Contains(c.ContentType(), "json") {
		body = []byte("{}")
	}

	var raw any
	if err := json.Unmarshal(body, &raw); err != nil {
		httperr.Abort(c, http.StatusBadRequest, "Invalid JSON body")
		return false
	}
	if _, ok := raw.(map[string]any); !ok {
		abort(c, Flattened{
			FormErrors:  []string{"Expected object, received " + jsonKind(raw)},
			FieldErrors: map[string][]string{},
		})
		return false
	}

	result := Flattened{FormErrors: []string{}, FieldErrors: map[string][]string{}}

	// encoding/json keeps decoding after a type mismatch and reports only
	// the first one; record it and skip that field's validator errors so
	// it isn't also reported as "Required".
	var typeErr *json.UnmarshalTypeError
	if err := json.Unmarshal(body, dst); err != nil {
		if !errors.As(err, &typeErr) {
			httperr.Abort(c, http.StatusBadRequest, "Invalid JSON body")
			return false
		}
		typeErr.Field = topLevel(typeErr.Field)
		result.FieldErrors[typeErr.Field] = []string{typeMismatchMessage(typeErr)}
	}

	var verrs validator.ValidationErrors
	if err := validate.Struct(dst); err != nil && errors.As(err, &verrs) {
		for _, fe := range verrs {
			field := fieldPath(fe)
			if typeErr != nil && field == typeErr.Field {
				continue
			}
			result.FieldErrors[field] = append(result.FieldErrors[field], message(fe))
		}
	}

	if len(result.FieldErrors) > 0 {
		abort(c, result)
		return false
	}
	return true
}

func abort(c *gin.Context, f Flattened) {
	c.AbortWithStatusJSON(http.StatusBadRequest, f)
}

// fieldPath is the top-level property an error belongs to, as zod's
// flatten() reports it: "Dto.attachments[0].url" → "attachments".
func fieldPath(fe validator.FieldError) string {
	_, path, found := strings.Cut(fe.Namespace(), ".")
	if !found {
		return fe.Field()
	}
	return topLevel(path)
}

func topLevel(path string) string {
	if i := strings.IndexAny(path, ".["); i >= 0 {
		return path[:i]
	}
	return path
}

func message(fe validator.FieldError) string {
	kind := fe.Kind()
	if kind == reflect.Pointer {
		kind = fe.Type().Elem().Kind()
	}
	switch fe.Tag() {
	case "required":
		return "Required"
	case "min", "gte":
		return boundMessage(kind, "at least", "greater than or equal to", fe.Param())
	case "max", "lte":
		return boundMessage(kind, "at most", "less than or equal to", fe.Param())
	case "oneof":
		options := strings.Fields(fe.Param())
		for i, o := range options {
			options[i] = "'" + o + "'"
		}
		return fmt.Sprintf("Invalid enum value. Expected %s, received '%v'", strings.Join(options, " | "), fe.Value())
	case "uuid", "uuid4":
		return "Invalid uuid"
	case "url", "http_url":
		return "Invalid url"
	case "email":
		return "Invalid email"
	}
	if msg, ok := patternMessages.Load(fe.Tag()); ok {
		return msg.(string)
	}
	return "Invalid input"
}

func boundMessage(kind reflect.Kind, stringWord, numberWord, n string) string {
	switch kind {
	case reflect.String:
		return fmt.Sprintf("String must contain %s %s character(s)", stringWord, n)
	case reflect.Slice, reflect.Array:
		return fmt.Sprintf("Array must contain %s %s element(s)", stringWord, n)
	default:
		return fmt.Sprintf("Number must be %s %s", numberWord, n)
	}
}

func typeMismatchMessage(e *json.UnmarshalTypeError) string {
	want, got := zodType(e.Type), zodKindFromJSON(e.Value)
	// A fractional number for an integer field: z.number().int().
	if want == "number" && got == "number" && isInteger(e.Type) {
		return "Expected integer, received float"
	}
	return fmt.Sprintf("Expected %s, received %s", want, got)
}

func isInteger(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	}
	return false
}

// zodType names a Go destination type the way zod names expected types.
func zodType(t reflect.Type) string {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Struct, reflect.Map:
		return "object"
	default:
		return "number"
	}
}

// zodKindFromJSON maps encoding/json's UnmarshalTypeError.Value
// ("string", "number", "bool", "array", "object") to zod's wording.
func zodKindFromJSON(value string) string {
	if value == "bool" {
		return "boolean"
	}
	if strings.HasPrefix(value, "number") {
		return "number"
	}
	return value
}

func jsonKind(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case []any:
		return "array"
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	default:
		return "object"
	}
}
