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

// resolveEntry resolves an index-file path value against docsDir. Absolute
// paths are accepted as-is (the user controls their own index file); relative
// paths are joined to docsDir and rejected if they escape it via "..".
func (l *Library) resolveEntry(v string) (string, bool) {
	if filepath.IsAbs(v) {
		return v, true
	}
	joined := filepath.Join(l.docsDir, v)
	rel, err := filepath.Rel(l.docsDir, joined)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return joined, true
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
						if p, ok := l.resolveEntry(v); ok {
							return p, true
						}
						// Unsafe/traversing entry: fall through to convention scan.
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
		name    string
		stemLen int
		nameLen int
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
