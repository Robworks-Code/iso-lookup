package catalog

import "testing"

func TestStageLabel(t *testing.T) {
	cases := map[int]string{
		6060: "Published",
		9599: "Withdrawn",
		4020: "Enquiry",     // stage-group fallback (40.20)
		1234: "Stage 12.34", // unknown -> raw
	}
	for code, want := range cases {
		if got := StageLabel(code); got != want {
			t.Errorf("StageLabel(%d) = %q, want %q", code, got, want)
		}
	}
}
