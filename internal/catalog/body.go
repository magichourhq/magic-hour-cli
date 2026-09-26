package catalog

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"unicode/utf8"
)

// Body validates flag values before uploads or generation can have side effects.
// Only CLI defaults are materialized; unspecified optional fields stay omitted.
func (op Operation) Body(values map[string][]string) (map[string]any, error) {
	body := map[string]any{}
	known := map[string]bool{}
	for _, f := range op.Fields {
		known[f.Flag] = true
		v, present := values[f.Flag]
		if !present && f.Default != "" {
			v, present = []string{f.Default}, true
		}
		if !present {
			if f.Required {
				return nil, fmt.Errorf("missing --%s\nExample: %s", f.Flag, op.Example)
			}
			continue
		}
		if f.Type != "array" && len(v) != 1 {
			return nil, fmt.Errorf("--%s expects one value", f.Flag)
		}
		var value any
		if f.Type == "array" {
			if len(v) < f.MinItems || (f.MaxItems > 0 && len(v) > f.MaxItems) {
				return nil, fmt.Errorf("--%s has an invalid number of items (min %d, max %d)", f.Flag, f.MinItems, f.MaxItems)
			}
			for _, item := range v {
				if err := f.validateText(item); err != nil {
					return nil, err
				}
			}
			value = v
		} else {
			var err error
			value, err = f.scalar(v[0])
			if err != nil {
				return nil, err
			}
		}
		node := body
		for _, part := range f.Path[:len(f.Path)-1] {
			if node[part] == nil {
				node[part] = map[string]any{}
			}
			node = node[part].(map[string]any)
		}
		node[f.Path[len(f.Path)-1]] = value
	}
	for flag := range values {
		if !known[flag] {
			return nil, fmt.Errorf("unknown input --%s", flag)
		}
	}
	return body, nil
}

func (f Field) scalar(s string) (any, error) {
	if len(f.Enum) > 0 && !slices.Contains(f.Enum, s) {
		return nil, fmt.Errorf("--%s must be one of %v", f.Flag, f.Enum)
	}
	switch f.Type {
	case "string":
		return s, f.validateText(s)
	case "boolean":
		v, err := strconv.ParseBool(s)
		if err != nil {
			return nil, fmt.Errorf("--%s must be a boolean", f.Flag)
		}
		return v, nil
	case "integer", "number":
		v, err := strconv.ParseFloat(s, 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || (f.Type == "integer" && math.Trunc(v) != v) {
			return nil, fmt.Errorf("--%s must be a finite %s", f.Flag, f.Type)
		}
		if f.Minimum != "" {
			n, _ := strconv.ParseFloat(f.Minimum, 64)
			if v < n {
				return nil, fmt.Errorf("--%s must be at least %s", f.Flag, f.Minimum)
			}
		}
		if f.Maximum != "" {
			n, _ := strconv.ParseFloat(f.Maximum, 64)
			if v > n {
				return nil, fmt.Errorf("--%s must be at most %s", f.Flag, f.Maximum)
			}
		}
		return v, nil
	}
	return nil, fmt.Errorf("unsupported field type %s", f.Type)
}

func (f Field) validateText(s string) error {
	n := utf8.RuneCountInString(s)
	if n < f.MinLength || (f.MaxLength > 0 && n > f.MaxLength) {
		return fmt.Errorf("--%s has an invalid text length (min %d, max %d)", f.Flag, f.MinLength, f.MaxLength)
	}
	return nil
}
