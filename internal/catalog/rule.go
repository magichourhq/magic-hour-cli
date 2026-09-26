package catalog

import (
	"fmt"
	"math"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"unicode/utf8"
)

// Rule is the supported request schema compiled into Go, including nested JSON
// inputs. Unknown OpenAPI keywords fail generation, not execution.
type Rule struct {
	Type       string          `json:"type"`
	Properties map[string]Rule `json:"properties,omitempty"`
	Required   []string        `json:"required,omitempty"`
	Item       *Rule           `json:"items,omitempty"`
	Enum       []any           `json:"enum,omitempty"`
	Minimum    string          `json:"minimum,omitempty"`
	Maximum    string          `json:"maximum,omitempty"`
	MultipleOf string          `json:"multiple_of,omitempty"`
	MinLength  int             `json:"min_length,omitempty"`
	MaxLength  int             `json:"max_length,omitempty"`
	MinItems   int             `json:"min_items,omitempty"`
	MaxItems   int             `json:"max_items,omitempty"`
	Pattern    string          `json:"pattern,omitempty"`
	Format     string          `json:"format,omitempty"`
	Nullable   bool            `json:"nullable,omitempty"`
}

func (r Rule) validate(value any, path string) error {
	bad := func(message string) error { return fmt.Errorf("%s: %s", path, message) }
	if value == nil && r.Nullable {
		return nil
	}
	if len(r.Enum) > 0 {
		found := false
		for _, v := range r.Enum {
			if reflect.DeepEqual(v, value) {
				found = true
				break
			}
		}
		if !found {
			return bad(fmt.Sprintf("must be one of %v", r.Enum))
		}
	}
	switch r.Type {
	case "object":
		object, ok := value.(map[string]any)
		if !ok {
			return bad("expected an object")
		}
		for _, key := range r.Required {
			if _, ok := object[key]; !ok {
				return bad("missing " + key)
			}
		}
		for key, v := range object {
			child, ok := r.Properties[key]
			if !ok {
				return bad("unknown field " + key)
			}
			if err := child.validate(v, path+"."+key); err != nil {
				return err
			}
		}
	case "array":
		array, ok := value.([]any)
		if !ok {
			return bad("expected an array")
		}
		if len(array) < r.MinItems || (r.MaxItems > 0 && len(array) > r.MaxItems) {
			return bad("invalid number of items")
		}
		for i, v := range array {
			if err := r.Item.validate(v, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	case "string":
		s, ok := value.(string)
		if !ok {
			return bad("expected a string")
		}
		n := utf8.RuneCountInString(s)
		if n < r.MinLength || (r.MaxLength > 0 && n > r.MaxLength) {
			return bad("invalid text length")
		}
		if r.Pattern != "" {
			matches, err := regexp.MatchString(r.Pattern, s)
			if err != nil || !matches {
				return bad("must match " + r.Pattern)
			}
		}
		if r.Format == "uri" {
			u, err := url.ParseRequestURI(s)
			if err != nil || u.Scheme == "" {
				return bad("expected an absolute URI")
			}
		}
	case "number", "integer":
		n, ok := value.(float64)
		if !ok || math.IsNaN(n) || math.IsInf(n, 0) || (r.Type == "integer" && n != math.Trunc(n)) {
			return bad("expected a finite " + r.Type)
		}
		if r.Minimum != "" {
			min, _ := strconv.ParseFloat(r.Minimum, 64)
			if n < min {
				return bad("must be at least " + r.Minimum)
			}
		}
		if r.Maximum != "" {
			max, _ := strconv.ParseFloat(r.Maximum, 64)
			if n > max {
				return bad("must be at most " + r.Maximum)
			}
		}
		if r.MultipleOf != "" {
			step, _ := strconv.ParseFloat(r.MultipleOf, 64)
			if math.Abs(n/step-math.Round(n/step)) > 1e-8 {
				return bad("must be a multiple of " + r.MultipleOf)
			}
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return bad("expected a boolean")
		}
	default:
		return bad("unsupported type " + r.Type)
	}
	return nil
}

func (r Rule) requiredObjects(body map[string]any) {
	for _, key := range r.Required {
		if child, ok := r.Properties[key]; ok && child.Type == "object" && body[key] == nil {
			body[key] = map[string]any{}
		}
	}
	for key, child := range r.Properties {
		if node, ok := body[key].(map[string]any); ok {
			child.requiredObjects(node)
		}
	}
}
