package scan

import "testing"

func hasComp(comps []Component, name string) bool {
	for _, c := range comps {
		if c.Name == name {
			return true
		}
	}
	return false
}

func TestParsePackageJSONDeps(t *testing.T) {
	data := []byte(`{"dependencies":{"openai":"^4","pg":"^8"},"devDependencies":{"jest":"^29","@angular/core":"^17"}}`)
	got := parsePackageJSONDeps("package.json", data)
	for _, want := range []string{"OpenAI SDK", "PostgreSQL driver", "Angular"} {
		if !hasComp(got, want) {
			t.Errorf("missing %q from %v", want, got)
		}
	}
}

func TestParsePackageJSONMalformed(t *testing.T) {
	if got := parsePackageJSONDeps("package.json", []byte("{not json")); got != nil {
		t.Errorf("malformed JSON should yield nil, got %v", got)
	}
}

func TestParseRequirementsTxt(t *testing.T) {
	data := []byte("# comment\ntorch==2.2.0\npsycopg2-binary>=2.9\n-r other.txt\nfastapi\n")
	got := parseRequirementsTxt("requirements.txt", data)
	if !hasComp(got, "PyTorch") || !hasComp(got, "PostgreSQL driver") {
		t.Errorf("expected PyTorch and PostgreSQL driver, got %v", got)
	}
}

func TestParseGoModAvoidsFalsePositive(t *testing.T) {
	// "lipgloss" contains the substring "pg"; token-aware matching must not
	// classify it as a PostgreSQL driver.
	data := []byte("module x\n\nrequire (\n\tgithub.com/charmbracelet/lipgloss v1.1.0\n\tgithub.com/redis/go-redis/v9 v9.0.0\n)\n")
	got := parseGoMod("go.mod", data)
	if hasComp(got, "PostgreSQL driver") {
		t.Errorf("lipgloss must not match PostgreSQL driver: %v", got)
	}
	if !hasComp(got, "Redis client") {
		t.Errorf("expected Redis client from go-redis: %v", got)
	}
}

func TestParsePyProject(t *testing.T) {
	data := []byte(`[project]
dependencies = [
  "langchain>=0.1",
  "uvicorn",
]
[tool.poetry.dependencies]
python = "^3.11"
anthropic = "^0.20"
`)
	got := parsePyProject("pyproject.toml", data)
	if !hasComp(got, "LangChain") || !hasComp(got, "Anthropic SDK") {
		t.Errorf("expected LangChain and Anthropic SDK, got %v", got)
	}
}
