package scan

import (
	"path"
	"regexp"
	"strings"
)

// detector matches a single file and emits a component template. Exactly one of
// Base, Glob, or PathHas is set. When Parse is non-nil and dependency parsing is
// enabled, the file's contents are additionally scanned for dependency signals.
type detector struct {
	Base    string    // exact basename, e.g. "go.mod"
	Glob    string    // basename glob, e.g. "docker-compose*.y*ml"
	PathHas string    // forward-slash path substring, e.g. ".github/workflows/"
	Emit    Component // template (Evidence filled in per match)
	Parse   func(rel string, data []byte) []Component
}

// comp builds a Component template for the detector tables.
func comp(name string, cat Category, conf Confidence, concerns ...Concern) Component {
	return Component{Name: name, Category: cat, Confidence: conf, Concerns: concerns}
}

// fileDetectors maps marker files to components. Order is not significant;
// matches are deduplicated by component name during the walk.
var fileDetectors = []detector{
	{Base: "go.mod", Emit: comp("Go", CatLanguage, High, ConcernSDLC, ConcernSWQuality), Parse: parseGoMod},
	{Base: "package.json", Emit: comp("Node.js", CatLanguage, High, ConcernSDLC, ConcernSWQuality, ConcernWeb), Parse: parsePackageJSONDeps},
	{Base: "requirements.txt", Emit: comp("Python", CatLanguage, High, ConcernSDLC, ConcernSWQuality), Parse: parseRequirementsTxt},
	{Base: "pyproject.toml", Emit: comp("Python", CatLanguage, High, ConcernSDLC, ConcernSWQuality), Parse: parsePyProject},
	{Base: "Cargo.toml", Emit: comp("Rust", CatLanguage, High, ConcernSDLC, ConcernSWQuality)},
	{Base: "pom.xml", Emit: comp("Java (Maven)", CatLanguage, High, ConcernSDLC, ConcernSWQuality)},
	{Base: "build.gradle", Emit: comp("Java/Kotlin (Gradle)", CatLanguage, High, ConcernSDLC, ConcernSWQuality)},
	{Base: "Gemfile", Emit: comp("Ruby", CatLanguage, High, ConcernSDLC, ConcernSWQuality)},
	{Base: "composer.json", Emit: comp("PHP", CatLanguage, High, ConcernSDLC, ConcernSWQuality)},

	{Base: "Dockerfile", Emit: comp("Docker", CatContainers, High, ConcernContainers)},
	{Glob: "docker-compose*.y*ml", Emit: comp("Docker Compose", CatContainers, High, ConcernContainers, ConcernData)},
	{Base: "Chart.yaml", Emit: comp("Helm", CatContainers, Medium, ConcernContainers, ConcernCloud)},
	{PathHas: "k8s/", Emit: comp("Kubernetes", CatContainers, Medium, ConcernContainers, ConcernCloud)},
	{PathHas: "kubernetes/", Emit: comp("Kubernetes", CatContainers, Medium, ConcernContainers, ConcernCloud)},

	{Glob: "*.tf", Emit: comp("Terraform", CatIaC, High, ConcernIaC, ConcernCloud)},
	{Base: "Pulumi.yaml", Emit: comp("Pulumi", CatIaC, High, ConcernIaC, ConcernCloud)},
	{Base: "cloudformation.yaml", Emit: comp("CloudFormation", CatIaC, Medium, ConcernIaC, ConcernCloud)},

	{PathHas: ".github/workflows/", Emit: comp("GitHub Actions", CatCICD, High, ConcernCICD, ConcernDevOps, ConcernTesting)},
	{Base: ".gitlab-ci.yml", Emit: comp("GitLab CI", CatCICD, High, ConcernCICD, ConcernDevOps, ConcernTesting)},
	{Base: "Jenkinsfile", Emit: comp("Jenkins", CatCICD, High, ConcernCICD, ConcernDevOps)},
	{Base: ".circleci/config.yml", Emit: comp("CircleCI", CatCICD, High, ConcernCICD, ConcernDevOps)},

	{Base: "openapi.yaml", Emit: comp("OpenAPI", CatWeb, Medium, ConcernWeb, ConcernSWQuality)},
	{Base: "openapi.json", Emit: comp("OpenAPI", CatWeb, Medium, ConcernWeb, ConcernSWQuality)},
}

// depSignals maps a dependency-name substring to a component template. Manifest
// parsers consult this table so that, e.g., an "openai" dependency in any
// ecosystem raises the AI concern.
var depSignals = []struct {
	Sub string
	C   Component
}{
	{"torch", comp("PyTorch", CatAI, High, ConcernAI)},
	{"tensorflow", comp("TensorFlow", CatAI, High, ConcernAI)},
	{"transformers", comp("Hugging Face Transformers", CatAI, High, ConcernAI)},
	{"scikit-learn", comp("scikit-learn", CatAI, Medium, ConcernAI)},
	{"openai", comp("OpenAI SDK", CatAI, High, ConcernAI)},
	{"anthropic", comp("Anthropic SDK", CatAI, High, ConcernAI)},
	{"langchain", comp("LangChain", CatAI, High, ConcernAI)},
	{"llama-index", comp("LlamaIndex", CatAI, High, ConcernAI)},

	{"jsonwebtoken", comp("JWT", CatSecurity, High, ConcernInfosec)},
	{"jose", comp("JWT/JOSE", CatSecurity, Medium, ConcernInfosec)},
	{"pyjwt", comp("JWT", CatSecurity, High, ConcernInfosec)},
	{"oauth", comp("OAuth", CatSecurity, High, ConcernInfosec, ConcernPrivacy)},
	{"passport", comp("Passport auth", CatSecurity, Medium, ConcernInfosec)},
	{"bcrypt", comp("Password hashing", CatSecurity, Medium, ConcernInfosec)},
	{"argon2", comp("Password hashing", CatSecurity, Medium, ConcernInfosec)},
	{"keycloak", comp("Keycloak (IAM)", CatSecurity, High, ConcernInfosec, ConcernPrivacy)},

	{"postgres", comp("PostgreSQL driver", CatData, High, ConcernData)},
	{"psycopg", comp("PostgreSQL driver", CatData, High, ConcernData)},
	{"pg", comp("PostgreSQL driver", CatData, High, ConcernData)},
	{"mysql", comp("MySQL driver", CatData, High, ConcernData)},
	{"mongodb", comp("MongoDB driver", CatData, High, ConcernData)},
	{"pymongo", comp("MongoDB driver", CatData, High, ConcernData)},
	{"redis", comp("Redis client", CatData, Medium, ConcernData)},
	{"sqlalchemy", comp("SQLAlchemy ORM", CatData, Medium, ConcernData)},
	{"prisma", comp("Prisma ORM", CatData, Medium, ConcernData)},

	{"aws-sdk", comp("AWS SDK", CatCloud, High, ConcernCloud)},
	{"boto3", comp("AWS SDK (boto3)", CatCloud, High, ConcernCloud)},
	{"@azure", comp("Azure SDK", CatCloud, High, ConcernCloud)},
	{"google-cloud", comp("Google Cloud SDK", CatCloud, High, ConcernCloud)},

	{"stripe", comp("Payments (Stripe)", CatSecurity, Medium, ConcernPrivacy, ConcernInfosec)},
	{"opentelemetry", comp("OpenTelemetry", CatObservability, Medium, ConcernITSM)},
	{"prom-client", comp("Prometheus client", CatObservability, Medium, ConcernITSM)},
	{"sentry", comp("Sentry", CatObservability, Low, ConcernITSM)},

	{"react", comp("React", CatWeb, Medium, ConcernWeb, ConcernAccessibility)},
	{"vue", comp("Vue", CatWeb, Medium, ConcernWeb, ConcernAccessibility)},
	{"@angular", comp("Angular", CatWeb, Medium, ConcernWeb, ConcernAccessibility)},
	{"svelte", comp("Svelte", CatWeb, Medium, ConcernWeb, ConcernAccessibility)},
}

// matchFile returns the component template for a file, or ok=false. rel is the
// repo-relative, forward-slash path; base is its basename.
func matchFile(rel, base string) (Component, *detector, bool) {
	for i := range fileDetectors {
		d := &fileDetectors[i]
		switch {
		case d.Base != "":
			// Base may itself contain a separator (e.g. ".circleci/config.yml").
			if base == d.Base || strings.HasSuffix(rel, "/"+d.Base) || rel == d.Base {
				return d.Emit, d, true
			}
		case d.Glob != "":
			if ok, _ := path.Match(d.Glob, base); ok {
				return d.Emit, d, true
			}
		case d.PathHas != "":
			if strings.Contains(rel+"/", d.PathHas) {
				return d.Emit, d, true
			}
		}
	}
	return Component{}, nil, false
}

// depComponents scans a dependency-name list against depSignals. Matching is
// token-aware to avoid false positives: a signal containing punctuation (e.g.
// "aws-sdk", "@azure") matches as a substring of the whole name, while a plain
// alphanumeric signal matches a path/name token — exactly for very short signals
// (so "pg" matches the npm package "pg" but not "lipgloss"), as a substring
// otherwise (so "torch" still matches "pytorch").
func depComponents(deps []string) []Component {
	var out []Component
	seen := map[string]bool{}
	for _, dep := range deps {
		low := strings.ToLower(dep)
		tokens := reDepToken.Split(low, -1)
		for _, sig := range depSignals {
			if seen[sig.C.Name] {
				continue
			}
			if depSignalMatches(sig.Sub, low, tokens) {
				seen[sig.C.Name] = true
				out = append(out, sig.C)
			}
		}
	}
	return out
}

var reDepToken = regexp.MustCompile(`[^a-z0-9]+`)

func depSignalMatches(sig, full string, tokens []string) bool {
	if reDepToken.MatchString(sig) {
		// Signal carries punctuation; match against the whole name.
		return strings.Contains(full, sig)
	}
	for _, tok := range tokens {
		if len(sig) <= 3 {
			if tok == sig {
				return true
			}
		} else if strings.Contains(tok, sig) {
			return true
		}
	}
	return false
}
