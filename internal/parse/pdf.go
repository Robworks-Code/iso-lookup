package parse

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"

	"github.com/Robworks-Code/iso-lookup/internal/segment"
)

func parsePDF(path string) (Document, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return Document{}, err
	}
	defer f.Close()
	var buf bytes.Buffer
	if rd, err := r.GetPlainText(); err == nil {
		buf.ReadFrom(rd)
	}
	raw := buf.String()
	doc := Document{Raw: raw, Title: filepath.Base(path)}
	if strings.TrimSpace(raw) == "" {
		doc.Sections = []Section{{
			Title: "Full text not extractable",
			Body:  "This PDF appears to be image-only or could not be parsed. Use `iso open` for the official page.",
		}}
		return doc, nil
	}
	doc.Sections = segment.Sections(raw)
	return doc, nil
}
