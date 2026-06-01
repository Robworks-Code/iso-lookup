package library

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Library resolves an ISO reference to a local file path.
type Library struct {
	docsDir   string
	indexFile string
}

// New constructs a Library. indexFile may be empty.
func New(docsDir, indexFile string) *Library {
	return &Library{docsDir: docsDir, indexFile: indexFile}
}

type indexFileShape struct {
	Entries map[string]string `yaml:"entries"`
}

var reNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func canon(s string) string {
	return reNonAlnum.ReplaceAllString(strings.ToLower(s), "")
}

// Find returns the local file path for ref, index override first.
func (l *Library) Find(ref string) (string, bool) {
	if l.docsDir == "" {
		return "", false
	}
	if l.indexFile != "" {
		if b, err := os.ReadFile(l.indexFile); err == nil {
			var idx indexFileShape
			if yaml.Unmarshal(b, &idx) == nil {
				for k, v := range idx.Entries {
					if canon(k) == canon(ref) {
						if filepath.IsAbs(v) {
							return v, true
						}
						return filepath.Join(l.docsDir, v), true
					}
				}
			}
		}
	}
	want := canon(ref)
	entries, err := os.ReadDir(l.docsDir)
	if err != nil {
		return "", false
	}

	// First pass: exact canonical match.
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		if canon(stem) == want {
			return filepath.Join(l.docsDir, name), true
		}
	}

	// Second pass: collect substring-containment candidates and pick the most
	// specific (longest canon(stem)), breaking ties by shortest filename then
	// lexical order. Require canon(stem) length >= 4 to avoid junk matches.
	type candidate struct {
		name     string
		stemLen  int
		nameLen  int
	}
	var candidates []candidate
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		cs := canon(stem)
		if len(cs) < 4 {
			continue
		}
		if strings.Contains(want, cs) || strings.Contains(cs, want) {
			candidates = append(candidates, candidate{name: name, stemLen: len(cs), nameLen: len(name)})
		}
	}
	if len(candidates) == 0 {
		return "", false
	}
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.stemLen > best.stemLen {
			best = c
		} else if c.stemLen == best.stemLen {
			if c.nameLen < best.nameLen || (c.nameLen == best.nameLen && c.name < best.name) {
				best = c
			}
		}
	}
	return filepath.Join(l.docsDir, best.name), true
}
