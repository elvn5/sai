package validation

import (
	"encoding/json"
	"strings"
)

// Trimmed is a string field that zod declared with .trim(): surrounding
// whitespace is removed before validation and storage.
type Trimmed string

func (t *Trimmed) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	*t = Trimmed(strings.TrimSpace(s))
	return nil
}

func (t Trimmed) String() string { return string(t) }
