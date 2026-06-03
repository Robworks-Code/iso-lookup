package scan

import (
	"bufio"
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
)

// maxManifestBytes caps how much of a manifest we read; dependency lists live
// near the top and a runaway file should not stall the scan.
const maxManifestBytes = 1 << 20 // 1 MiB

// parsePackageJSONDeps extracts dependency names from the dependencies and
// devDependencies maps. Malformed JSON yields no components rather than an error.
func parsePackageJSONDeps(_ string, data []byte) []Component {
	var pkg struct {
		Dependencies    map[string]json.RawMessage `json:"dependencies"`
		DevDependencies map[string]json.RawMessage `json:"devDependencies"`
	}
	if err := json.Unmarshal(cap1MiB(data), &pkg); err != nil {
		return nil
	}
	var names []string
	for name := range pkg.Dependencies {
		names = append(names, name)
	}
	for name := range pkg.DevDependencies {
		names = append(names, name)
	}
	return depComponents(names)
}

// reRequirement captures the distribution name at the head of a requirements
// line, before any version specifier, extra, or comment.
var reRequirement = regexp.MustCompile(`^([A-Za-z0-9_.\-]+)`)

// parseRequirementsTxt extracts package names from a pip requirements file.
func parseRequirementsTxt(_ string, data []byte) []Component {
	var names []string
	sc := bufio.NewScanner(bytes.NewReader(cap1MiB(data)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		if m := reRequirement.FindString(line); m != "" {
			names = append(names, m)
		}
	}
	return depComponents(names)
}

// reGoRequire captures a module path on a go.mod require line.
var reGoRequire = regexp.MustCompile(`^\s*(?:require\s+)?([a-z0-9.\-]+\.[a-z]{2,}/[^\s]+)\s+v`)

// parseGoMod extracts imported module paths from the require directives.
func parseGoMod(_ string, data []byte) []Component {
	var names []string
	sc := bufio.NewScanner(bytes.NewReader(cap1MiB(data)))
	for sc.Scan() {
		line := sc.Text()
		if m := reGoRequire.FindStringSubmatch(line); m != nil {
			names = append(names, m[1])
		}
	}
	return depComponents(names)
}

// reTomlDep captures a quoted or bare dependency name from a pyproject line,
// covering both PEP 621 (`dependencies = ["fastapi>=0.1"]`) and Poetry
// (`fastapi = "^0.1"`) layouts. This is a forgiving line scan, not a full TOML
// parse, to avoid pulling in a TOML dependency.
var (
	reTomlListDep = regexp.MustCompile(`["']([A-Za-z0-9_.\-]+)`)
	reTomlKeyDep  = regexp.MustCompile(`^\s*([A-Za-z0-9_.\-]+)\s*=`)
)

// parsePyProject extracts dependency names from a pyproject.toml using a
// best-effort line scan of the dependency sections.
func parsePyProject(_ string, data []byte) []Component {
	var names []string
	sc := bufio.NewScanner(bytes.NewReader(cap1MiB(data)))
	inList := false   // inside a `dependencies = [ ... ]` array
	inPoetry := false // inside a `[tool.poetry.dependencies]` table
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]"):
			inPoetry = strings.Contains(trimmed, "poetry") && strings.Contains(trimmed, "dependencies")
			inList = false
		case strings.HasPrefix(trimmed, "dependencies") && strings.Contains(trimmed, "["):
			inList = true
			for _, m := range reTomlListDep.FindAllStringSubmatch(trimmed, -1) {
				names = append(names, m[1])
			}
			if strings.Contains(trimmed, "]") {
				inList = false
			}
		case inList:
			for _, m := range reTomlListDep.FindAllStringSubmatch(trimmed, -1) {
				names = append(names, m[1])
			}
			if strings.Contains(trimmed, "]") {
				inList = false
			}
		case inPoetry && trimmed != "" && !strings.HasPrefix(trimmed, "#"):
			if m := reTomlKeyDep.FindStringSubmatch(trimmed); m != nil && m[1] != "python" {
				names = append(names, m[1])
			}
		}
	}
	return depComponents(names)
}

func cap1MiB(data []byte) []byte {
	if len(data) > maxManifestBytes {
		return data[:maxManifestBytes]
	}
	return data
}
