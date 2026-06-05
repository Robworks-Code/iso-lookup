package main

import "testing"

func TestPathArg(t *testing.T) {
	if got := pathArg(nil); got != "." {
		t.Errorf("pathArg(nil) = %q, want .", got)
	}
	if got := pathArg([]string{"./svc"}); got != "./svc" {
		t.Errorf("pathArg([./svc]) = %q", got)
	}
}

func TestPlural(t *testing.T) {
	if plural(1, "standard") != "standard" {
		t.Error("plural(1) should be singular")
	}
	if plural(2, "standard") != "standards" {
		t.Error("plural(2) should add s")
	}
}

func TestScanCommandWiring(t *testing.T) {
	subs := map[string]bool{}
	for _, c := range scanCmd.Commands() {
		subs[c.Name()] = true
	}
	for _, want := range []string{"stack", "why"} {
		if !subs[want] {
			t.Errorf("scan missing subcommand %q", want)
		}
	}
	if scanCmd.PersistentFlags().Lookup("group-by") == nil {
		t.Error("scan missing --group-by flag")
	}
	if f := scanCmd.PersistentFlags().Lookup("group-by"); f != nil && f.DefValue != "category" {
		t.Errorf("--group-by default = %q, want category", f.DefValue)
	}
}
