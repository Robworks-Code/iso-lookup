package catalog

import "fmt"

// exactStage maps full ISO harmonized stage codes (stage*100+substage) to labels.
var exactStage = map[int]string{
	6060: "Published",
	6000: "Publication",
	9599: "Withdrawn",
	9060: "Review completed",
	9020: "Under review",
	1099: "New project approved",
}

// stageGroup maps the stage prefix (code/100) to a coarse label.
var stageGroup = map[int]string{
	0:  "Preliminary",
	10: "Proposal",
	20: "Preparatory",
	30: "Committee",
	40: "Enquiry",
	50: "Approval",
	60: "Publication",
	90: "Review",
	95: "Withdrawal",
}

// StageLabel converts an ISO stage code to a human-readable status.
func StageLabel(code int) string {
	if s, ok := exactStage[code]; ok {
		return s
	}
	if s, ok := stageGroup[code/100]; ok {
		return s
	}
	return fmt.Sprintf("Stage %02d.%02d", code/100, code%100)
}
