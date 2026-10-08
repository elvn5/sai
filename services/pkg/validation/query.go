package validation

import (
	"fmt"
	"math"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Query reads query-string parameters with the semantics the NestJS API's
// zod schemas had (z.coerce.number(), z.enum(), .optional(), .default())
// and collects failures in zod's flatten() shape:
//
//	q := validation.NewQuery(c)
//	limit := q.Int("limit", validation.IntRule{Default: 20, Min: ptr(1), Max: ptr(50)})
//	if !q.OK() { return }
type Query struct {
	c    *gin.Context
	errs map[string][]string
}

func NewQuery(c *gin.Context) *Query {
	return &Query{c: c, errs: map[string][]string{}}
}

func (q *Query) fail(name, message string) {
	q.errs[name] = append(q.errs[name], message)
}

// OK writes the 400 response and returns false if any parameter failed.
func (q *Query) OK() bool {
	if len(q.errs) == 0 {
		return true
	}
	q.c.AbortWithStatusJSON(http.StatusBadRequest, Flattened{FormErrors: []string{}, FieldErrors: q.errs})
	return false
}

func (q *Query) raw(name string) (string, bool) {
	return q.c.GetQuery(name)
}

// String is an optional z.string().trim().max(maxLen).
func (q *Query) String(name string, maxLen int) *string {
	raw, ok := q.raw(name)
	if !ok {
		return nil
	}
	s := strings.TrimSpace(raw)
	if len([]rune(s)) > maxLen {
		q.fail(name, fmt.Sprintf("String must contain at most %d character(s)", maxLen))
	}
	return &s
}

// Enum is an optional z.enum(allowed) (an empty value is invalid, as in zod).
func (q *Query) Enum(name string, allowed []string) *string {
	raw, ok := q.raw(name)
	if !ok {
		return nil
	}
	if !slices.Contains(allowed, raw) {
		q.fail(name, enumMessage(allowed, raw))
		return nil
	}
	return &raw
}

// RequiredEnum is a required z.enum(allowed).
func (q *Query) RequiredEnum(name string, allowed []string) string {
	if _, ok := q.raw(name); !ok {
		q.fail(name, "Required")
		return ""
	}
	if v := q.Enum(name, allowed); v != nil {
		return *v
	}
	return ""
}

// EnumDefault is z.enum(allowed).default(def).
func (q *Query) EnumDefault(name string, allowed []string, def string) string {
	if v := q.Enum(name, allowed); v != nil {
		return *v
	}
	return def
}

// RequiredUUID is a required z.string().uuid().
func (q *Query) RequiredUUID(name string) string {
	raw, ok := q.raw(name)
	if !ok {
		q.fail(name, "Required")
		return ""
	}
	if uuid.Validate(raw) != nil {
		q.fail(name, "Invalid uuid")
		return ""
	}
	return raw
}

// Bool is z.enum(['true', 'false']).optional() turned into a boolean.
func (q *Query) Bool(name string) *bool {
	v := q.Enum(name, []string{"true", "false"})
	if v == nil {
		return nil
	}
	b := *v == "true"
	return &b
}

// List is a comma-separated z.array(z.enum(allowed)); empty when absent.
func (q *Query) List(name string, allowed []string) []string {
	raw, ok := q.raw(name)
	if !ok || raw == "" {
		return []string{}
	}
	var out []string
	for _, item := range strings.Split(raw, ",") {
		if item == "" {
			continue
		}
		if !slices.Contains(allowed, item) {
			q.fail(name, enumMessage(allowed, item))
			continue
		}
		out = append(out, item)
	}
	return out
}

// NumberRule describes a z.coerce.number() chain.
type NumberRule struct {
	Integer  bool
	Min, Max *float64
}

// Number is an optional z.coerce.number() with the rule's checks.
func (q *Query) Number(name string, rule NumberRule) *float64 {
	raw, ok := q.raw(name)
	if !ok {
		return nil
	}
	n := jsNumber(raw)
	switch {
	case math.IsNaN(n):
		q.fail(name, "Expected number, received nan")
		return nil
	case rule.Integer && (n != math.Trunc(n) || math.IsInf(n, 0)):
		q.fail(name, "Expected integer, received float")
		return nil
	}
	valid := true
	if rule.Min != nil && n < *rule.Min {
		q.fail(name, "Number must be greater than or equal to "+formatNumber(*rule.Min))
		valid = false
	}
	if rule.Max != nil && n > *rule.Max {
		q.fail(name, "Number must be less than or equal to "+formatNumber(*rule.Max))
		valid = false
	}
	if !valid {
		return nil
	}
	return &n
}

// IntDefault is z.coerce.number().int() with bounds and a default.
func (q *Query) IntDefault(name string, def int, minimum, maximum *float64) int {
	n := q.Number(name, NumberRule{Integer: true, Min: minimum, Max: maximum})
	if n == nil {
		return def
	}
	return int(*n)
}

// Bound is a convenience for NumberRule limits.
func Bound(v float64) *float64 { return &v }

func enumMessage(allowed []string, received string) string {
	quoted := make([]string, len(allowed))
	for i, a := range allowed {
		quoted[i] = "'" + a + "'"
	}
	return fmt.Sprintf("Invalid enum value. Expected %s, received '%s'", strings.Join(quoted, " | "), received)
}

func formatNumber(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

var jsDecimal = regexp.MustCompile(`^[+-]?(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?$`)

// jsNumber mimics JavaScript's Number(string): surrounding whitespace is
// ignored, "" is 0, and anything that isn't a whole numeric literal is NaN.
func jsNumber(raw string) float64 {
	s := strings.TrimSpace(raw)
	switch s {
	case "":
		return 0
	case "Infinity", "+Infinity":
		return math.Inf(1)
	case "-Infinity":
		return math.Inf(-1)
	}
	lower := strings.ToLower(s)
	for prefix, base := range map[string]int{"0x": 16, "0o": 8, "0b": 2} {
		if strings.HasPrefix(lower, prefix) {
			n, err := strconv.ParseUint(s[2:], base, 64)
			if err != nil {
				return math.NaN()
			}
			return float64(n)
		}
	}
	if !jsDecimal.MatchString(s) {
		return math.NaN()
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return math.NaN()
	}
	return n
}
