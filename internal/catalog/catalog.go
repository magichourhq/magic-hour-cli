// Package catalog describes CLI operations independently of command parsing.
package catalog

//go:generate go run ../generate -spec ../../api/openapi.json -metadata ../../api/cli.json -out catalog_gen.go

type Operation struct {
	ID      string      `json:"id"`
	Group   string      `json:"group"`
	Name    string      `json:"name"`
	Kind    string      `json:"kind"`
	Path    string      `json:"path"`
	Summary string      `json:"summary"`
	Example string      `json:"example"`
	Fields  []Field     `json:"fields"`
	Schema  Rule        `json:"schema"`
	Files   []FileInput `json:"files,omitempty"`
}

type FileInput struct {
	Path []string `json:"path"`
	Kind string   `json:"kind"`
}

type Field struct {
	Flag      string   `json:"flag"`
	Path      []string `json:"path"`
	Type      string   `json:"type"`
	Help      string   `json:"help"`
	Required  bool     `json:"required"`
	Default   string   `json:"default,omitempty"`
	Enum      []string `json:"enum,omitempty"`
	Minimum   string   `json:"minimum,omitempty"`
	Maximum   string   `json:"maximum,omitempty"`
	MinLength int      `json:"min_length,omitempty"`
	MaxLength int      `json:"max_length,omitempty"`
	MinItems  int      `json:"min_items,omitempty"`
	MaxItems  int      `json:"max_items,omitempty"`
	FileKind  string   `json:"file_kind,omitempty"`
}
