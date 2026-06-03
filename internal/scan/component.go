// Package scan inspects a project folder, detects its technology stack, and maps
// the detected components to relevant ISO standards drawn from the offline
// catalog. Detection produces Components, each carrying one or more normalized
// Concerns; a curated knowledge base (rules.go) maps Concerns to anchor
// standards, which report.go resolves and groups into a Report.
package scan

import (
	"encoding/json"
	"strings"
)

// Confidence expresses how strongly a signal implies its component or
// recommendation. Higher is stronger.
type Confidence int

const (
	Low Confidence = iota
	Medium
	High
)

// String returns the lowercase label used in text and JSON output.
func (c Confidence) String() string {
	switch c {
	case High:
		return "high"
	case Medium:
		return "medium"
	default:
		return "low"
	}
}

// Category buckets a detected component for the "detected stack" listing. It is
// distinct from a report Group header, which may be a component, an ISO domain
// category, or an ICS code depending on --group-by.
type Category string

const (
	CatLanguage      Category = "Languages & Runtimes"
	CatContainers    Category = "Containers & Orchestration"
	CatIaC           Category = "Infrastructure as Code"
	CatCICD          Category = "CI/CD"
	CatAI            Category = "AI / ML"
	CatSecurity      Category = "Security & Identity"
	CatData          Category = "Data & Persistence"
	CatCloud         Category = "Cloud Services"
	CatWeb           Category = "Web / Frontend"
	CatObservability Category = "Observability"
)

// Concern is a normalized topic key that bridges detected components and the
// curated rule table. Components declare the concerns they raise; rules.go keys
// ISO recommendations off these.
type Concern string

const (
	ConcernInfosec       Concern = "infosec"
	ConcernCloud         Concern = "cloud"
	ConcernPrivacy       Concern = "privacy"
	ConcernSDLC          Concern = "sdlc"
	ConcernSWQuality     Concern = "sw_quality"
	ConcernTesting       Concern = "testing"
	ConcernITSM          Concern = "it_service_mgmt"
	ConcernAI            Concern = "ai"
	ConcernQMS           Concern = "quality_mgmt"
	ConcernDevOps        Concern = "devops"
	ConcernAccessibility Concern = "accessibility"
	ConcernContainers    Concern = "containers"
	ConcernIaC           Concern = "iac"
	ConcernData          Concern = "data"
	ConcernCICD          Concern = "cicd"
	ConcernWeb           Concern = "web"
)

// Component is one detected piece of the stack (a language, framework, tool, or
// notable dependency), with the files that revealed it and the concerns it
// raises for the recommendation layer.
type Component struct {
	Name       string
	Category   Category
	Evidence   []string // repo-relative paths that triggered detection
	Confidence Confidence
	Concerns   []Concern
}

// MarshalJSON renders Confidence as its label rather than an integer.
func (c Component) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name       string    `json:"name"`
		Category   Category  `json:"category"`
		Evidence   []string  `json:"evidence"`
		Concerns   []Concern `json:"concerns"`
		Confidence string    `json:"confidence"`
	}{c.Name, c.Category, c.Evidence, c.Concerns, c.Confidence.String()})
}

// Detection is the result of scanning a folder.
type Detection struct {
	Root       string      `json:"root"`
	Components []Component `json:"components"`
	FilesSeen  int         `json:"files_seen"`
	Truncated  bool        `json:"truncated"` // hit a depth or file-count cap
}

// addEvidence merges a path into a component's evidence list, keeping it unique
// and bounded so a deeply nested marker file does not balloon the output.
func (c *Component) addEvidence(path string) {
	for _, e := range c.Evidence {
		if e == path {
			return
		}
	}
	if len(c.Evidence) < maxEvidencePerComponent {
		c.Evidence = append(c.Evidence, path)
	}
}

const maxEvidencePerComponent = 8

// concernSet returns the union of concerns across a set of components, in first-
// seen order.
func concernSet(comps []Component) []Concern {
	seen := map[Concern]bool{}
	var out []Concern
	for _, c := range comps {
		for _, cn := range c.Concerns {
			if !seen[cn] {
				seen[cn] = true
				out = append(out, cn)
			}
		}
	}
	return out
}

// matchesTerm reports whether term (case-insensitive) is contained in any of the
// candidate strings. Used by --component / scan why fuzzy matching.
func matchesTerm(term string, candidates ...string) bool {
	term = strings.ToLower(strings.TrimSpace(term))
	if term == "" {
		return false
	}
	for _, c := range candidates {
		if strings.Contains(strings.ToLower(c), term) {
			return true
		}
	}
	return false
}
