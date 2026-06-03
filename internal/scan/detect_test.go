package scan

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// names returns the detected component names as a set for easy assertions.
func names(d Detection) map[string]Component {
	m := make(map[string]Component, len(d.Components))
	for _, c := range d.Components {
		m[c.Name] = c
	}
	return m
}

func TestDetectGoService(t *testing.T) {
	d, err := Detect("testdata/sample-go-service", DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	got := names(d)
	for _, want := range []string{"Go", "Docker", "GitHub Actions", "Redis client"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing component %q; got %v", want, keys(got))
		}
	}
	if ev := got["Go"].Evidence; len(ev) == 0 || ev[0] != "go.mod" {
		t.Errorf("Go evidence = %v, want [go.mod]", ev)
	}
	if got["GitHub Actions"].Category != CatCICD {
		t.Errorf("GitHub Actions category = %q", got["GitHub Actions"].Category)
	}
}

func TestDetectNodeAI(t *testing.T) {
	d, err := Detect("testdata/sample-node-ai", DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	got := names(d)
	for _, want := range []string{"Node.js", "OpenAI SDK", "JWT", "PostgreSQL driver", "React"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing component %q; got %v", want, keys(got))
		}
	}
}

func TestDetectPythonML(t *testing.T) {
	d, err := Detect("testdata/sample-python-ml", DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	got := names(d)
	// requirements.txt and pyproject.toml signals.
	for _, want := range []string{"Python", "PyTorch", "Hugging Face Transformers", "PostgreSQL driver", "LangChain", "Anthropic SDK"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing component %q; got %v", want, keys(got))
		}
	}
	// Python should be evidenced by both manifests, deduped to one component.
	if ev := got["Python"].Evidence; len(ev) < 2 {
		t.Errorf("Python evidence = %v, want both manifests", ev)
	}
}

func TestDetectSkipsVendored(t *testing.T) {
	d, err := Detect("testdata/sample-vendored", DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	got := names(d)
	if _, ok := got["Go"]; !ok {
		t.Errorf("expected Go from top-level go.mod; got %v", keys(got))
	}
	if _, ok := got["OpenAI SDK"]; ok {
		t.Errorf("OpenAI SDK in node_modules should have been skipped; got %v", keys(got))
	}
}

func TestDetectDepthCap(t *testing.T) {
	root := t.TempDir()
	deep := filepath.Join(root, "a", "b", "c", "d")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deep, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := DefaultOptions()
	opts.MaxDepth = 2
	d, err := Detect(root, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Truncated {
		t.Error("expected Truncated=true when depth cap pruned the tree")
	}
	if len(d.Components) != 0 {
		t.Errorf("deep go.mod should be unreachable at depth 2; got %v", keys(names(d)))
	}
}

func TestDetectDoesNotFollowSymlinkLoop(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A symlink pointing back at the root would loop a follow-symlinks walker.
	if err := os.Symlink(root, filepath.Join(root, "loop")); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}
	done := make(chan Detection, 1)
	go func() {
		d, _ := Detect(root, DefaultOptions())
		done <- d
	}()
	select {
	case d := <-done:
		if _, ok := names(d)["Go"]; !ok {
			t.Error("expected Go component")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Detect did not terminate; symlink loop was followed")
	}
}

func keys(m map[string]Component) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
