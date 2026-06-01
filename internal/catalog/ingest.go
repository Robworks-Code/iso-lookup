package catalog

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// flexString unmarshals a JSON value that may be either a string or an array
// of strings (joins with ", ").
type flexString string

func (f *flexString) UnmarshalJSON(data []byte) error {
	// try string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = flexString(s)
		return nil
	}
	// try array
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*f = flexString(strings.Join(arr, ", "))
		return nil
	}
	// null or unknown — leave empty
	*f = ""
	return nil
}

// rawDeliverable mirrors the Open Data deliverables JSONL shape.
type rawDeliverable struct {
	ID              int               `json:"id"`
	Reference       string            `json:"reference"`
	Title           map[string]string `json:"title"`
	Scope           map[string]string `json:"scope"`
	Edition         int               `json:"edition"`
	PublicationDate string            `json:"publicationDate"`
	ICSCode         []string          `json:"icsCode"`
	OwnerCommittee  string            `json:"ownerCommittee"`
	CurrentStage    int               `json:"currentStage"`
	Replaces        flexString        `json:"replaces"`
	ReplacedBy      flexString        `json:"replacedBy"`
	Pages           map[string]*int   `json:"pages"`
}

type rawCommittee struct {
	Reference string            `json:"reference"`
	Title     map[string]string `json:"title"`
}

// Ingest parses the three Open Data sources into slim Records.
func Ingest(deliverables, committees, ics io.Reader) ([]Record, error) {
	comNames, err := parseCommittees(committees)
	if err != nil {
		return nil, fmt.Errorf("committees: %w", err)
	}
	icsNames, err := parseICS(ics)
	if err != nil {
		return nil, fmt.Errorf("ics: %w", err)
	}

	var recs []Record
	sc := bufio.NewScanner(deliverables)
	sc.Buffer(make([]byte, 1024*1024), 8*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}
		var d rawDeliverable
		if err := json.Unmarshal(line, &d); err != nil {
			return nil, err
		}
		recs = append(recs, toRecord(d, comNames, icsNames))
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return recs, nil
}

func toRecord(d rawDeliverable, com, ics map[string]string) Record {
	r := Record{
		Reference:     d.Reference,
		Title:         d.Title["en"],
		Scope:         StripHTML(d.Scope["en"]),
		Edition:       d.Edition,
		PublishedDate: d.PublicationDate,
		StageCode:     d.CurrentStage,
		Status:        StageLabel(d.CurrentStage),
		Replaces:      string(d.Replaces),
		ReplacedBy:    string(d.ReplacedBy),
		ID:            d.ID,
		URL:           fmt.Sprintf("https://www.iso.org/standard/%d.html", d.ID),
	}
	if p := d.Pages["en"]; p != nil {
		r.Pages = *p
	}
	if name := com[d.OwnerCommittee]; name != "" {
		r.Committee = d.OwnerCommittee + " — " + name
	} else {
		r.Committee = d.OwnerCommittee
	}
	for _, code := range d.ICSCode {
		if name := ics[code]; name != "" {
			r.ICS = append(r.ICS, code+" "+name)
		} else {
			r.ICS = append(r.ICS, code)
		}
	}
	return r
}

func parseCommittees(rd io.Reader) (map[string]string, error) {
	out := map[string]string{}
	sc := bufio.NewScanner(rd)
	sc.Buffer(make([]byte, 1024*1024), 8*1024*1024)
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		var c rawCommittee
		if err := json.Unmarshal(sc.Bytes(), &c); err != nil {
			return nil, err
		}
		out[c.Reference] = c.Title["en"]
	}
	return out, sc.Err()
}

func parseICS(rd io.Reader) (map[string]string, error) {
	out := map[string]string{}
	r := csv.NewReader(rd)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	for i, row := range rows {
		if i == 0 || len(row) < 3 { // skip header
			continue
		}
		// strip BOM on first data cell if present
		id := strings.TrimPrefix(row[0], "\ufeff")
		out[id] = row[2]
	}
	return out, nil
}
