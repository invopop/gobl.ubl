// Package terms exposes the EN16931 Business Term → UBL mapping data that
// backs this converter's documentation. The mapping lives as a YAML data
// file (terms/en16931-ubl.yaml) and is embedded here so downstream tools —
// notably gobl.docs — can import it and render it without vendoring the file.
//
// The schema is deliberately generic: it describes a tree of EN16931 Business
// Groups (BG-*) and Business Terms (BT-*), where each term carries the
// target paths it maps to. The same schema is used by gobl.docs for the
// GOBL → BT leg, so the two legs can be joined on the term ID to produce a
// full GOBL → BT → UBL mapping.
package terms

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed en16931-ubl.yaml
var en16931UBL []byte

// Mapping is a single mapping document: a named set of EN16931 terms each
// resolving to one or more target paths (UBL XPaths here, GOBL JSON paths in
// the gobl.docs equivalent).
type Mapping struct {
	Key         string  `yaml:"key"`
	Name        string  `yaml:"name"`
	Description string  `yaml:"description"`
	Terms       []*Term `yaml:"terms"`
}

// Term is one node in the mapping tree. It is either a Business Term (BT-*)
// with Paths, or a Business Group (BG-*) containing nested Terms — and
// occasionally both.
type Term struct {
	ID    string   `yaml:"id"`
	Name  string   `yaml:"name"`
	Paths []string `yaml:"paths"`
	Notes string   `yaml:"notes"`
	Terms []*Term  `yaml:"terms"`
}

// EN16931UBL parses and returns the embedded EN16931 → UBL 2.1 invoice mapping.
func EN16931UBL() (*Mapping, error) {
	return Parse(en16931UBL)
}

// Parse unmarshals a mapping document from YAML. It is exported so callers
// (e.g. gobl.docs) can reuse it for the GOBL → BT leg, which shares this
// schema.
func Parse(data []byte) (*Mapping, error) {
	m := new(Mapping)
	if err := yaml.Unmarshal(data, m); err != nil {
		return nil, fmt.Errorf("parsing terms mapping: %w", err)
	}
	return m, nil
}

// Flatten returns every Business Term in the document keyed by ID, discarding
// the Business Group hierarchy. Business Groups that also carry paths of their
// own are included; pure grouping nodes (no paths) are not.
func (m *Mapping) Flatten() map[string]*Term {
	out := make(map[string]*Term)
	var walk func(ts []*Term)
	walk = func(ts []*Term) {
		for _, t := range ts {
			if len(t.Paths) > 0 || t.Terms == nil {
				out[t.ID] = t
			}
			if len(t.Terms) > 0 {
				walk(t.Terms)
			}
		}
	}
	walk(m.Terms)
	return out
}
