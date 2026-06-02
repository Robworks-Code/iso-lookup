package catalog

// Record is the slim, display-ready metadata for one ISO deliverable.
type Record struct {
	Reference     string
	Title         string
	Scope         string // HTML stripped to plain text
	Edition       int
	PublishedDate string
	StageCode     int
	Status        string // StageLabel(StageCode)
	ICS           []string
	Committee     string
	Replaces      string
	ReplacedBy    string
	Pages         int
	ID            int
	URL           string
}
