package scan

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Options tunes a directory scan. The zero value is not usable; callers should
// start from DefaultOptions and adjust.
type Options struct {
	MaxDepth  int             // directory depth relative to root (0 = unlimited)
	MaxFiles  int             // safety cap on files visited (0 = unlimited)
	SkipDirs  map[string]bool // directory basenames to prune
	ParseDeps bool            // read manifest contents for dependency signals
}

// DefaultOptions returns sensible scan settings: a moderate depth limit, a large
// file cap, the usual vendored/build directories pruned, and manifest parsing on.
func DefaultOptions() Options {
	return Options{
		MaxDepth:  6,
		MaxFiles:  50000,
		ParseDeps: true,
		SkipDirs: map[string]bool{
			"node_modules":   true,
			"vendor":         true,
			".git":           true,
			"dist":           true,
			"build":          true,
			"target":         true,
			".venv":          true,
			"venv":           true,
			"__pycache__":    true,
			".terraform":     true,
			".next":          true,
			".idea":          true,
			".vscode":        true,
			".browser-pilot": true,
			".claude":        true,
		},
	}
}

// Detect walks root and returns the components it recognizes. It never follows
// symlinked directories (so symlink loops are structurally impossible) and skips
// unreadable entries rather than aborting. A depth or file-count cap sets
// Truncated.
func Detect(root string, opts Options) (Detection, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return Detection{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return Detection{}, err
	}
	if !info.IsDir() {
		return Detection{}, &fs.PathError{Op: "scan", Path: abs, Err: fs.ErrInvalid}
	}

	det := Detection{Root: abs}
	byName := map[string]*Component{}

	add := func(tmpl Component, evidence string) {
		c, ok := byName[tmpl.Name]
		if !ok {
			cp := tmpl
			cp.Evidence = nil
			byName[tmpl.Name] = &cp
			c = byName[tmpl.Name]
		}
		if tmpl.Confidence > c.Confidence {
			c.Confidence = tmpl.Confidence
		}
		c.Concerns = unionConcerns(c.Concerns, tmpl.Concerns)
		c.addEvidence(evidence)
	}

	walkErr := filepath.WalkDir(abs, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			// Unreadable directory or file: skip it, keep going.
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(abs, p)
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if rel == "." {
				return nil
			}
			if opts.SkipDirs[d.Name()] {
				return fs.SkipDir
			}
			if opts.MaxDepth > 0 && strings.Count(rel, "/")+1 > opts.MaxDepth {
				det.Truncated = true
				return fs.SkipDir
			}
			return nil
		}

		// Symlinked entries report as non-dir; WalkDir does not descend them.
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		det.FilesSeen++
		if opts.MaxFiles > 0 && det.FilesSeen > opts.MaxFiles {
			det.Truncated = true
			return filepath.SkipAll
		}

		tmpl, dt, ok := matchFile(rel, d.Name())
		if !ok {
			return nil
		}
		add(tmpl, rel)
		if opts.ParseDeps && dt.Parse != nil {
			if data, rerr := os.ReadFile(p); rerr == nil {
				for _, extra := range dt.Parse(rel, data) {
					add(extra, rel)
				}
			}
		}
		return nil
	})
	if walkErr != nil {
		return Detection{}, walkErr
	}

	det.Components = sortedComponents(byName)
	return det, nil
}

// sortedComponents flattens the name-keyed map into a slice ordered by category
// then name, for stable, readable output.
func sortedComponents(byName map[string]*Component) []Component {
	out := make([]Component, 0, len(byName))
	for _, c := range byName {
		out = append(out, *c)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Category != out[j].Category {
			return out[i].Category < out[j].Category
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// unionConcerns merges b into a, preserving order and dropping duplicates.
func unionConcerns(a, b []Concern) []Concern {
	seen := map[Concern]bool{}
	for _, c := range a {
		seen[c] = true
	}
	for _, c := range b {
		if !seen[c] {
			seen[c] = true
			a = append(a, c)
		}
	}
	return a
}
