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
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		if canon(stem) == want || strings.Contains(want, canon(stem)) || strings.Contains(canon(stem), want) {
			return filepath.Join(l.docsDir, name), true
		}
	}
	return "", false
}
